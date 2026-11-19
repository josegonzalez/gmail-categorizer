package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
	"github.com/josegonzalez/gmail-categorizer/internal/web"
)

// Web command variables
var (
	webUseKeychain    bool
	webDeleteKeychain bool
)

func newWebCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Serve a web UI for email triage",
		Long:  `Start a local web server for browsing groupings, previewing messages and batch archiving from a browser.`,
		RunE:  runWeb,
	}

	// IMAP flags
	cmd.Flags().StringP("username", "u", "", "Gmail username/email address")
	cmd.Flags().StringP("password", "p", "", "Gmail app password (prefer stdin or env var)")

	// Keychain flags
	cmd.Flags().BoolVarP(&webUseKeychain, "keychain", "k", false, "Use OS keychain to store/retrieve password")
	cmd.Flags().BoolVar(&webDeleteKeychain, "delete-keychain", false, "Delete stored password from keychain and exit")

	// Server flags
	cmd.Flags().String("listen", config.DefaultWebListen, "Address to listen on (host:port)")

	return cmd
}

func runWeb(cmd *cobra.Command, args []string) error {
	// Bind flags to viper at runtime (avoids conflicts between subcommands)
	bindIMAPFlags(cmd)
	v.BindPFlag("web.listen", cmd.Flags().Lookup("listen"))

	cfg, err := config.Load(v, cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if webDeleteKeychain {
		return handleDeleteKeychain(cfg.IMAP.Username)
	}

	if err := resolvePassword(cfg, webUseKeychain); err != nil {
		return err
	}

	if cfg.IMAP.Username == "" {
		return fmt.Errorf("username is required (set via --username, GMAIL_CATEGORIZER_IMAP_USERNAME, or config file)")
	}
	if cfg.IMAP.Password == "" {
		return fmt.Errorf("password is required (set via --password, GMAIL_CATEGORIZER_IMAP_PASSWORD, stdin, or config file)")
	}

	// The first Ctrl+C cancels ctx, which stops any IMAP command in flight and
	// shuts the server down. Restoring the default handler right away means a
	// second Ctrl+C always exits immediately.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	context.AfterFunc(ctx, stop)

	session := web.NewSession(
		inboxDialer(&cfg.IMAP),
		func(client imap.Client) triage.Triager {
			return triage.NewTriager(client, cfg.Special.GroupByFrom)
		},
	)
	defer session.Close()

	server, err := web.NewServer(session, web.Options{Username: cfg.IMAP.Username})
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Loading inbox...")
	if err := server.Load(ctx); err != nil {
		return err
	}

	listener, err := net.Listen("tcp", cfg.Web.Listen)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", cfg.Web.Listen, err)
	}
	if !isLoopback(listener.Addr()) {
		fmt.Fprintf(os.Stderr, "Warning: listening on %s, which is reachable from other machines. The web UI has no authentication; anyone who can reach it can read and archive your mail.\n", listener.Addr())
	}

	httpServer := &http.Server{
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.Serve(listener)
	}()

	fmt.Fprintf(os.Stderr, "Serving the web UI at http://%s (press Ctrl+C to stop)\n", listener.Addr())

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serving web UI: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	fmt.Fprintln(os.Stderr, "Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down web UI: %w", err)
	}
	return nil
}

// inboxDialer returns a dial function that connects, logs in and selects the
// inbox, for the web session to call on start and after a dropped connection.
func inboxDialer(cfg *config.IMAPConfig) web.DialFunc {
	return func(ctx context.Context) (imap.Client, *imap.MailboxInfo, error) {
		client, cleanup, err := connectAndLogin(ctx, cfg)
		if err != nil {
			return nil, nil, err
		}

		info, err := client.SelectMailbox(ctx, "INBOX")
		if err != nil {
			cleanup()
			return nil, nil, err
		}

		return client, info, nil
	}
}

// isLoopback reports whether addr only accepts local connections.
func isLoopback(addr net.Addr) bool {
	tcp, ok := addr.(*net.TCPAddr)
	if !ok {
		return false
	}
	return tcp.IP.IsLoopback()
}
