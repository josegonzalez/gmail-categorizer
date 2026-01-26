// Package config handles configuration loading from multiple sources.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/term"
)

// Config holds all application configuration.
type Config struct {
	IMAP    IMAPConfig    `mapstructure:"imap"`
	Output  OutputConfig  `mapstructure:"output"`
	Special SpecialConfig `mapstructure:"special"`
}

// IMAPConfig contains IMAP connection settings.
type IMAPConfig struct {
	Server   string        `mapstructure:"server"`
	Port     int           `mapstructure:"port"`
	Username string        `mapstructure:"username"`
	Password string        `mapstructure:"password"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Mailbox  string        `mapstructure:"mailbox"`
}

// OutputConfig controls output formatting.
type OutputConfig struct {
	Format    string `mapstructure:"format"`
	SortBy    string `mapstructure:"sort_by"`
	SortOrder string `mapstructure:"sort_order"`
	Limit     int    `mapstructure:"limit"`
}

// SpecialConfig defines addresses that receive special handling.
type SpecialConfig struct {
	GroupByFrom []string `mapstructure:"group_by_from"`
}

// Output format constants.
const (
	FormatTable = "table"
	FormatJSON  = "json"
	FormatCSV   = "csv"
)

// Sort field constants.
const (
	SortByCount   = "count"
	SortByAddress = "address"
)

// Sort order constants.
const (
	SortAsc  = "asc"
	SortDesc = "desc"
)

// SetDefaults configures default values in Viper.
func SetDefaults(v *viper.Viper) {
	// IMAP defaults
	v.SetDefault("imap.server", "imap.gmail.com")
	v.SetDefault("imap.port", 993)
	v.SetDefault("imap.timeout", "30s")
	v.SetDefault("imap.mailbox", "INBOX")

	// Output defaults
	v.SetDefault("output.format", FormatTable)
	v.SetDefault("output.sort_by", SortByCount)
	v.SetDefault("output.sort_order", SortDesc)
	v.SetDefault("output.limit", 0)

	// Special addresses that trigger grouping by sender
	v.SetDefault("special.group_by_from", []string{"admin@", "hi@", "email@"})
}

// Load reads configuration from all sources with precedence:
// CLI flags > environment variables > config file > defaults.
func Load(v *viper.Viper, configPath string) (*Config, error) {
	SetDefaults(v)

	// Environment variables
	v.SetEnvPrefix("GMAIL_CATEGORIZER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicitly bind environment variables for nested keys
	v.BindEnv("imap.username")
	v.BindEnv("imap.password")
	v.BindEnv("imap.server")
	v.BindEnv("imap.port")
	v.BindEnv("imap.mailbox")
	v.BindEnv("output.format")
	v.BindEnv("output.sort_by")
	v.BindEnv("output.sort_order")
	v.BindEnv("output.limit")

	// Config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Search in default locations
		v.SetConfigName("gmail-categorizer")
		v.SetConfigType("yaml")

		// Add search paths
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(home)
			v.AddConfigPath(filepath.Join(home, ".config"))
		}
		v.AddConfigPath(".")
	}

	// Read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return error if it's not a "file not found" error
			if configPath != "" {
				return nil, fmt.Errorf("reading config file: %w", err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// ReadPasswordFromStdin securely reads a password from terminal input.
func ReadPasswordFromStdin() (string, error) {
	fmt.Fprint(os.Stderr, "Enter App Password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after hidden input
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return string(password), nil
}

// Validate checks the configuration for required fields and valid values.
func (c *Config) Validate() error {
	if c.IMAP.Username == "" {
		return fmt.Errorf("username is required (set via --username, GMAIL_STATS_IMAP_USERNAME, or config file)")
	}

	if c.IMAP.Password == "" {
		return fmt.Errorf("password is required (set via --password, GMAIL_STATS_IMAP_PASSWORD, stdin, or config file)")
	}

	// Validate output format
	switch c.Output.Format {
	case FormatTable, FormatJSON, FormatCSV:
		// valid
	default:
		return fmt.Errorf("invalid output format %q (must be table, json, or csv)", c.Output.Format)
	}

	// Validate sort field
	switch c.Output.SortBy {
	case SortByCount, SortByAddress:
		// valid
	default:
		return fmt.Errorf("invalid sort field %q (must be count or address)", c.Output.SortBy)
	}

	// Validate sort order
	switch c.Output.SortOrder {
	case SortAsc, SortDesc:
		// valid
	default:
		return fmt.Errorf("invalid sort order %q (must be asc or desc)", c.Output.SortOrder)
	}

	return nil
}
