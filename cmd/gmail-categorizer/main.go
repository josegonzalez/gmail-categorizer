// Package main provides the CLI entry point for gmail-stats.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/keychain"
	"github.com/josegonzalez/gmail-categorizer/internal/output"
	"github.com/josegonzalez/gmail-categorizer/internal/stats"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
	"github.com/josegonzalez/gmail-categorizer/internal/tui"
)

var (
	version = "dev"
	cfgFile string
	v       = viper.New()
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "gmail-stats",
		Short:   "Gmail inbox management CLI",
		Long:    `A CLI tool for Gmail inbox statistics and email triage.`,
		Version: version,
	}

	// Persistent flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: $HOME/gmail-stats.yaml)")

	// Add subcommands
	rootCmd.AddCommand(newStatsCmd())
	rootCmd.AddCommand(newTriageCmd())
	rootCmd.AddCommand(newMailboxesCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// Stats command variables
var (
	statsUseKeychain    bool
	statsDeleteKeychain bool
)

func newStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Generate statistics from Gmail inbox",
		Long:  `Connects to Gmail via IMAP and generates statistics about email distribution by recipient address.`,
		RunE:  runStats,
	}

	// IMAP flags
	cmd.Flags().StringP("username", "u", "", "Gmail username/email address")
	cmd.Flags().StringP("password", "p", "", "Gmail app password (prefer stdin or env var)")
	cmd.Flags().StringP("mailbox", "m", "INBOX", "Mailbox to analyze")

	// Keychain flags
	cmd.Flags().BoolVarP(&statsUseKeychain, "keychain", "k", false, "Use OS keychain to store/retrieve password")
	cmd.Flags().BoolVar(&statsDeleteKeychain, "delete-keychain", false, "Delete stored password from keychain and exit")

	// Output flags
	cmd.Flags().StringP("format", "f", "table", "Output format: table, json, csv")
	cmd.Flags().IntP("limit", "l", 0, "Limit number of results (0 = unlimited)")
	cmd.Flags().String("sort-by", "count", "Sort by: count, address")
	cmd.Flags().String("sort-order", "desc", "Sort order: asc, desc")

	return cmd
}

// Triage command variables
var (
	triageUseKeychain    bool
	triageDeleteKeychain bool
)

func newTriageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "triage",
		Short: "Interactive email triage with TUI",
		Long:  `Launch an interactive TUI for triaging emails by grouping, viewing subjects, and batch archiving.`,
		RunE:  runTriage,
	}

	// IMAP flags
	cmd.Flags().StringP("username", "u", "", "Gmail username/email address")
	cmd.Flags().StringP("password", "p", "", "Gmail app password (prefer stdin or env var)")

	// Keychain flags
	cmd.Flags().BoolVarP(&triageUseKeychain, "keychain", "k", false, "Use OS keychain to store/retrieve password")
	cmd.Flags().BoolVar(&triageDeleteKeychain, "delete-keychain", false, "Delete stored password from keychain and exit")

	return cmd
}

// Mailboxes command variables
var (
	mailboxesUseKeychain bool
)

func newMailboxesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailboxes",
		Short: "List all mailboxes/labels",
		Long:  `Lists all available mailboxes (Gmail labels) in the account.`,
		RunE:  runMailboxes,
	}

	// IMAP flags
	cmd.Flags().StringP("username", "u", "", "Gmail username/email address")
	cmd.Flags().StringP("password", "p", "", "Gmail app password (prefer stdin or env var)")

	// Keychain flags
	cmd.Flags().BoolVarP(&mailboxesUseKeychain, "keychain", "k", false, "Use OS keychain to store/retrieve password")

	return cmd
}

func runMailboxes(cmd *cobra.Command, args []string) error {
	// Bind flags to viper at runtime (avoids conflicts between subcommands)
	bindIMAPFlags(cmd)

	// Load configuration
	cfg, err := config.Load(v, cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Handle password retrieval with keychain support
	if err := resolvePassword(cfg, mailboxesUseKeychain); err != nil {
		return err
	}

	// Validate configuration
	if cfg.IMAP.Username == "" {
		return fmt.Errorf("username is required (set via --username, GMAIL_STATS_IMAP_USERNAME, or config file)")
	}
	if cfg.IMAP.Password == "" {
		return fmt.Errorf("password is required (set via --password, GMAIL_STATS_IMAP_PASSWORD, stdin, or config file)")
	}

	ctx := context.Background()

	client, cleanup, err := connectAndLogin(ctx, &cfg.IMAP)
	if err != nil {
		return err
	}
	defer cleanup()

	// List mailboxes
	mailboxes, err := client.ListMailboxes(ctx)
	if err != nil {
		return fmt.Errorf("listing mailboxes: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d mailboxes:\n\n", len(mailboxes))

	for _, mbox := range mailboxes {
		fmt.Println(mbox)
	}

	return nil
}

// bindIMAPFlags binds the common IMAP flags to viper.
func bindIMAPFlags(cmd *cobra.Command) {
	v.BindPFlag("imap.username", cmd.Flags().Lookup("username"))
	v.BindPFlag("imap.password", cmd.Flags().Lookup("password"))
}

// connectAndLogin creates an IMAP client, connects to the server, and authenticates.
// It returns the client, a cleanup function, and any error.
func connectAndLogin(ctx context.Context, cfg *config.IMAPConfig) (imap.Client, func(), error) {
	client := imap.NewClient(cfg)

	fmt.Fprintln(os.Stderr, "Connecting to IMAP server...")
	if err := client.Connect(ctx); err != nil {
		return nil, nil, err
	}

	fmt.Fprintln(os.Stderr, "Authenticating...")
	if err := client.Login(cfg.Username, cfg.Password); err != nil {
		client.Close()
		return nil, nil, err
	}

	cleanup := func() {
		client.Logout()
		client.Close()
	}

	return client, cleanup, nil
}

// resolvePassword handles password retrieval from various sources including keychain.
func resolvePassword(cfg *config.Config, useKeychain bool) error {
	// If password is already set (via flag, env, or config file), use it
	if cfg.IMAP.Password != "" {
		if useKeychain && cfg.IMAP.Username != "" {
			if err := keychain.Set(cfg.IMAP.Username, cfg.IMAP.Password); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not store password in keychain: %v\n", err)
			} else {
				fmt.Fprintln(os.Stderr, "Password stored in keychain.")
			}
		}
		return nil
	}

	// If keychain flag is set, try to retrieve from keychain
	if useKeychain && cfg.IMAP.Username != "" {
		password, err := keychain.Get(cfg.IMAP.Username)
		if err != nil {
			return fmt.Errorf("reading from keychain: %w", err)
		}
		if password != "" {
			cfg.IMAP.Password = password
			fmt.Fprintln(os.Stderr, "Using password from keychain.")
			return nil
		}
	}

	// Prompt for password
	password, err := config.ReadPasswordFromStdin()
	if err != nil {
		return fmt.Errorf("reading password: %w", err)
	}
	cfg.IMAP.Password = password

	// If keychain flag is set, store the password
	if useKeychain && cfg.IMAP.Username != "" {
		if err := keychain.Set(cfg.IMAP.Username, password); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not store password in keychain: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "Password stored in keychain.")
		}
	}

	return nil
}

func runTriage(cmd *cobra.Command, args []string) error {
	// Bind flags to viper at runtime (avoids conflicts between subcommands)
	bindIMAPFlags(cmd)

	// Load configuration
	cfg, err := config.Load(v, cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Handle --delete-keychain flag
	if triageDeleteKeychain {
		return handleDeleteKeychain(cfg.IMAP.Username)
	}

	// Handle password retrieval with keychain support
	if err := resolvePassword(cfg, triageUseKeychain); err != nil {
		return err
	}

	// Validate configuration
	if cfg.IMAP.Username == "" {
		return fmt.Errorf("username is required (set via --username, GMAIL_STATS_IMAP_USERNAME, or config file)")
	}
	if cfg.IMAP.Password == "" {
		return fmt.Errorf("password is required (set via --password, GMAIL_STATS_IMAP_PASSWORD, stdin, or config file)")
	}

	ctx := context.Background()

	client, cleanup, err := connectAndLogin(ctx, &cfg.IMAP)
	if err != nil {
		return err
	}
	defer cleanup()

	// Select INBOX
	mailboxInfo, err := client.SelectMailbox("INBOX")
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Found %d messages in %s\n", mailboxInfo.NumMessages, mailboxInfo.Name)

	if mailboxInfo.NumMessages == 0 {
		fmt.Fprintln(os.Stderr, "No messages to process.")
		return nil
	}

	// Create triager
	triager := triage.NewTriager(client, cfg.Special.GroupByFrom)

	// Run TUI
	return tui.Run(ctx, triager)
}


func runStats(cmd *cobra.Command, args []string) error {
	// Bind flags to viper at runtime (avoids conflicts between subcommands)
	bindIMAPFlags(cmd)
	v.BindPFlag("imap.mailbox", cmd.Flags().Lookup("mailbox"))
	v.BindPFlag("output.format", cmd.Flags().Lookup("format"))
	v.BindPFlag("output.limit", cmd.Flags().Lookup("limit"))
	v.BindPFlag("output.sort_by", cmd.Flags().Lookup("sort-by"))
	v.BindPFlag("output.sort_order", cmd.Flags().Lookup("sort-order"))

	// Load configuration
	cfg, err := config.Load(v, cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Handle --delete-keychain flag
	if statsDeleteKeychain {
		return handleDeleteKeychain(cfg.IMAP.Username)
	}

	// Handle password retrieval with keychain support
	if err := resolvePassword(cfg, statsUseKeychain); err != nil {
		return err
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	ctx := context.Background()

	client, cleanup, err := connectAndLogin(ctx, &cfg.IMAP)
	if err != nil {
		return err
	}
	defer cleanup()

	// Select mailbox
	mailboxInfo, err := client.SelectMailbox(cfg.IMAP.Mailbox)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Found %d messages in %s\n", mailboxInfo.NumMessages, mailboxInfo.Name)

	if mailboxInfo.NumMessages == 0 {
		fmt.Fprintln(os.Stderr, "No messages to process.")
		return nil
	}

	// Create aggregator
	aggregator := stats.NewAggregator(cfg.Special.GroupByFrom)

	// Progress callback
	progress := func(current, total uint32) {
		fmt.Fprintf(os.Stderr, "\rProcessing messages: %d/%d", current, total)
	}

	// Fetch and process messages
	fmt.Fprintln(os.Stderr, "Fetching messages...")
	if err := client.FetchMessages(ctx, aggregator.Process, progress); err != nil {
		return fmt.Errorf("fetching messages: %w", err)
	}

	fmt.Fprintln(os.Stderr, "\nDone!")
	fmt.Fprintln(os.Stderr)

	// Get results
	results := aggregator.Results(mailboxInfo.Name)

	// Create formatter
	formatter, err := output.NewFormatter(cfg.Output.Format, output.Options{
		SortBy:    cfg.Output.SortBy,
		SortOrder: cfg.Output.SortOrder,
		Limit:     cfg.Output.Limit,
	})
	if err != nil {
		return err
	}

	// Output results
	return formatter.Format(os.Stdout, results)
}


// handleDeleteKeychain removes the stored password from the keychain.
func handleDeleteKeychain(username string) error {
	if username == "" {
		return fmt.Errorf("username is required to delete keychain entry (use --username or config file)")
	}

	exists, err := keychain.Exists(username)
	if err != nil {
		return fmt.Errorf("checking keychain: %w", err)
	}

	if !exists {
		fmt.Fprintf(os.Stderr, "No password stored in keychain for %s\n", username)
		return nil
	}

	if err := keychain.Delete(username); err != nil {
		return fmt.Errorf("deleting from keychain: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Password for %s deleted from keychain.\n", username)
	return nil
}
