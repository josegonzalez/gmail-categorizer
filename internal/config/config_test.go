package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDefaults(t *testing.T) {
	v := viper.New()
	SetDefaults(v)

	assert.Equal(t, "imap.gmail.com", v.GetString("imap.server"))
	assert.Equal(t, 993, v.GetInt("imap.port"))
	assert.Equal(t, "INBOX", v.GetString("imap.mailbox"))
	assert.Equal(t, "table", v.GetString("output.format"))
	assert.Equal(t, "count", v.GetString("output.sort_by"))
	assert.Equal(t, "desc", v.GetString("output.sort_order"))
	assert.Equal(t, 0, v.GetInt("output.limit"))
	assert.Equal(t, []string{"admin@", "hi@", "email@"}, v.GetStringSlice("special.group_by_from"))
}

func TestLoad_Defaults(t *testing.T) {
	v := viper.New()
	cfg, err := Load(v, "")

	require.NoError(t, err)
	assert.Equal(t, "imap.gmail.com", cfg.IMAP.Server)
	assert.Equal(t, 993, cfg.IMAP.Port)
	assert.Equal(t, "INBOX", cfg.IMAP.Mailbox)
	assert.Equal(t, FormatTable, cfg.Output.Format)
	assert.Equal(t, SortByCount, cfg.Output.SortBy)
	assert.Equal(t, SortDesc, cfg.Output.SortOrder)
}

func TestLoad_EnvOverrides(t *testing.T) {
	// Set environment variables
	t.Setenv("GMAIL_CATEGORIZER_IMAP_USERNAME", "test@gmail.com")
	t.Setenv("GMAIL_CATEGORIZER_IMAP_PASSWORD", "secret123")
	t.Setenv("GMAIL_CATEGORIZER_OUTPUT_FORMAT", "json")

	v := viper.New()
	cfg, err := Load(v, "")

	require.NoError(t, err)
	assert.Equal(t, "test@gmail.com", cfg.IMAP.Username)
	assert.Equal(t, "secret123", cfg.IMAP.Password)
	assert.Equal(t, "json", cfg.Output.Format)
}

func TestLoad_ConfigFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gmail-categorizer.yaml")

	configContent := `
imap:
  server: custom.imap.com
  port: 995
  username: config@example.com
  mailbox: "All Mail"
output:
  format: csv
  sort_by: address
  limit: 50
special:
  group_by_from:
    - support@
    - info@
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	v := viper.New()
	cfg, err := Load(v, configPath)

	require.NoError(t, err)
	assert.Equal(t, "custom.imap.com", cfg.IMAP.Server)
	assert.Equal(t, 995, cfg.IMAP.Port)
	assert.Equal(t, "config@example.com", cfg.IMAP.Username)
	assert.Equal(t, "All Mail", cfg.IMAP.Mailbox)
	assert.Equal(t, "csv", cfg.Output.Format)
	assert.Equal(t, "address", cfg.Output.SortBy)
	assert.Equal(t, 50, cfg.Output.Limit)
	assert.Equal(t, []string{"support@", "info@"}, cfg.Special.GroupByFrom)
}

func TestLoad_MissingConfigFile(t *testing.T) {
	v := viper.New()
	cfg, err := Load(v, "/nonexistent/path/config.yaml")

	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				IMAP: IMAPConfig{
					Username: "user@gmail.com",
					Password: "password",
				},
				Output: OutputConfig{
					Format:    FormatTable,
					SortBy:    SortByCount,
					SortOrder: SortDesc,
				},
			},
			expectError: false,
		},
		{
			name: "missing username",
			config: Config{
				IMAP: IMAPConfig{
					Password: "password",
				},
				Output: OutputConfig{
					Format:    FormatTable,
					SortBy:    SortByCount,
					SortOrder: SortDesc,
				},
			},
			expectError: true,
			errorMsg:    "username is required",
		},
		{
			name: "missing password",
			config: Config{
				IMAP: IMAPConfig{
					Username: "user@gmail.com",
				},
				Output: OutputConfig{
					Format:    FormatTable,
					SortBy:    SortByCount,
					SortOrder: SortDesc,
				},
			},
			expectError: true,
			errorMsg:    "password is required",
		},
		{
			name: "invalid format",
			config: Config{
				IMAP: IMAPConfig{
					Username: "user@gmail.com",
					Password: "password",
				},
				Output: OutputConfig{
					Format:    "xml",
					SortBy:    SortByCount,
					SortOrder: SortDesc,
				},
			},
			expectError: true,
			errorMsg:    "invalid output format",
		},
		{
			name: "invalid sort field",
			config: Config{
				IMAP: IMAPConfig{
					Username: "user@gmail.com",
					Password: "password",
				},
				Output: OutputConfig{
					Format:    FormatTable,
					SortBy:    "invalid",
					SortOrder: SortDesc,
				},
			},
			expectError: true,
			errorMsg:    "invalid sort field",
		},
		{
			name: "invalid sort order",
			config: Config{
				IMAP: IMAPConfig{
					Username: "user@gmail.com",
					Password: "password",
				},
				Output: OutputConfig{
					Format:    FormatTable,
					SortBy:    SortByCount,
					SortOrder: "random",
				},
			},
			expectError: true,
			errorMsg:    "invalid sort order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_ValidFormats(t *testing.T) {
	baseConfig := Config{
		IMAP: IMAPConfig{
			Username: "user@gmail.com",
			Password: "password",
		},
		Output: OutputConfig{
			SortBy:    SortByCount,
			SortOrder: SortDesc,
		},
	}

	formats := []string{FormatTable, FormatJSON, FormatCSV}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			cfg := baseConfig
			cfg.Output.Format = format
			assert.NoError(t, cfg.Validate())
		})
	}
}
