package main

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

func TestNewStatsCmd(t *testing.T) {
	cmd := newStatsCmd()

	assert.Equal(t, "stats", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check flags are registered
	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("username"))
	assert.NotNil(t, flags.Lookup("password"))
	assert.NotNil(t, flags.Lookup("mailbox"))
	assert.NotNil(t, flags.Lookup("keychain"))
	assert.NotNil(t, flags.Lookup("delete-keychain"))
	assert.NotNil(t, flags.Lookup("format"))
	assert.NotNil(t, flags.Lookup("limit"))
	assert.NotNil(t, flags.Lookup("sort-by"))
	assert.NotNil(t, flags.Lookup("sort-order"))
}

func TestNewTriageCmd(t *testing.T) {
	cmd := newTriageCmd()

	assert.Equal(t, "triage", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check flags are registered
	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("username"))
	assert.NotNil(t, flags.Lookup("password"))
	assert.NotNil(t, flags.Lookup("keychain"))
	assert.NotNil(t, flags.Lookup("delete-keychain"))
}

func TestNewMailboxesCmd(t *testing.T) {
	cmd := newMailboxesCmd()

	assert.Equal(t, "mailboxes", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check flags are registered
	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("username"))
	assert.NotNil(t, flags.Lookup("password"))
	assert.NotNil(t, flags.Lookup("keychain"))
}

func TestResolvePassword_AlreadySet(t *testing.T) {
	cfg := &config.Config{
		IMAP: config.IMAPConfig{
			Username: "test@gmail.com",
			Password: "existing-password",
		},
	}

	err := resolvePassword(cfg, false)
	require.NoError(t, err)
	assert.Equal(t, "existing-password", cfg.IMAP.Password)
}

func TestHandleDeleteKeychain_EmptyUsername(t *testing.T) {
	err := handleDeleteKeychain("")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

func TestBindIMAPFlags(t *testing.T) {
	cmd := newStatsCmd()
	localV := viper.New()

	// Set a flag value
	err := cmd.Flags().Set("username", "test@example.com")
	require.NoError(t, err)

	// Bind flags - need to bind to our local viper
	localV.BindPFlag("imap.username", cmd.Flags().Lookup("username"))
	localV.BindPFlag("imap.password", cmd.Flags().Lookup("password"))

	// Check that the binding works
	assert.Equal(t, "test@example.com", localV.GetString("imap.username"))
}

func TestVersion(t *testing.T) {
	// Version should be set (either default "dev" or build-injected)
	assert.NotEmpty(t, version)
}

// mockIMAPClient implements imap.Client for testing
type mockIMAPClient struct {
	ConnectFunc           func(ctx context.Context) error
	LoginFunc             func(username, password string) error
	SelectMailboxFunc     func(name string) (*imap.MailboxInfo, error)
	FetchMessagesFunc     func(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error
	FetchMessagesWithUIDsFunc func(ctx context.Context, uids []uint32, handler imap.MessageHandler) error
	MoveMessagesFunc      func(ctx context.Context, uids []uint32, destMailbox string) error
	ArchiveToFolderFunc   func(ctx context.Context, uids []uint32, folder string) error
	CreateMailboxFunc     func(ctx context.Context, name string) error
	ListMailboxesFunc     func(ctx context.Context) ([]string, error)
	LogoutFunc            func() error
	CloseFunc             func() error
}

func (m *mockIMAPClient) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}
	return nil
}

func (m *mockIMAPClient) Login(username, password string) error {
	if m.LoginFunc != nil {
		return m.LoginFunc(username, password)
	}
	return nil
}

func (m *mockIMAPClient) SelectMailbox(name string) (*imap.MailboxInfo, error) {
	if m.SelectMailboxFunc != nil {
		return m.SelectMailboxFunc(name)
	}
	return &imap.MailboxInfo{Name: name}, nil
}

func (m *mockIMAPClient) FetchMessages(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
	if m.FetchMessagesFunc != nil {
		return m.FetchMessagesFunc(ctx, handler, progress)
	}
	return nil
}

func (m *mockIMAPClient) FetchMessagesWithUIDs(ctx context.Context, uids []uint32, handler imap.MessageHandler) error {
	if m.FetchMessagesWithUIDsFunc != nil {
		return m.FetchMessagesWithUIDsFunc(ctx, uids, handler)
	}
	return nil
}

func (m *mockIMAPClient) MoveMessages(ctx context.Context, uids []uint32, destMailbox string) error {
	if m.MoveMessagesFunc != nil {
		return m.MoveMessagesFunc(ctx, uids, destMailbox)
	}
	return nil
}

func (m *mockIMAPClient) ArchiveToFolder(ctx context.Context, uids []uint32, folder string) error {
	if m.ArchiveToFolderFunc != nil {
		return m.ArchiveToFolderFunc(ctx, uids, folder)
	}
	return nil
}

func (m *mockIMAPClient) CreateMailbox(ctx context.Context, name string) error {
	if m.CreateMailboxFunc != nil {
		return m.CreateMailboxFunc(ctx, name)
	}
	return nil
}

func (m *mockIMAPClient) ListMailboxes(ctx context.Context) ([]string, error) {
	if m.ListMailboxesFunc != nil {
		return m.ListMailboxesFunc(ctx)
	}
	return []string{}, nil
}

func (m *mockIMAPClient) Logout() error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc()
	}
	return nil
}

func (m *mockIMAPClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func TestStatsCmd_Flags(t *testing.T) {
	cmd := newStatsCmd()

	// Test default values
	format, _ := cmd.Flags().GetString("format")
	assert.Equal(t, "table", format)

	mailbox, _ := cmd.Flags().GetString("mailbox")
	assert.Equal(t, "INBOX", mailbox)

	limit, _ := cmd.Flags().GetInt("limit")
	assert.Equal(t, 0, limit)

	sortBy, _ := cmd.Flags().GetString("sort-by")
	assert.Equal(t, "count", sortBy)

	sortOrder, _ := cmd.Flags().GetString("sort-order")
	assert.Equal(t, "desc", sortOrder)
}

func TestTriageCmd_Flags(t *testing.T) {
	cmd := newTriageCmd()

	// Check boolean flags default to false
	keychain, _ := cmd.Flags().GetBool("keychain")
	assert.False(t, keychain)

	deleteKeychain, _ := cmd.Flags().GetBool("delete-keychain")
	assert.False(t, deleteKeychain)
}
