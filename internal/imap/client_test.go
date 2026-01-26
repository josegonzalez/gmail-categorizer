package imap

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server:   "imap.gmail.com",
		Port:     993,
		Username: "test@gmail.com",
		Password: "password",
		Timeout:  30 * time.Second,
	}

	client := NewClient(cfg)
	require.NotNil(t, client)
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "positive timeout",
			timeout:         60 * time.Second,
			expectedTimeout: 60 * time.Second,
		},
		{
			name:            "zero timeout uses default",
			timeout:         0,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "negative timeout uses default",
			timeout:         -5 * time.Second,
			expectedTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			newCtx, cancel := WithTimeout(ctx, tt.timeout)
			defer cancel()

			deadline, ok := newCtx.Deadline()
			require.True(t, ok, "context should have deadline")

			// Check that deadline is approximately expectedTimeout from now
			expectedDeadline := time.Now().Add(tt.expectedTimeout)
			diff := deadline.Sub(expectedDeadline)
			assert.Less(t, diff.Abs(), time.Second, "deadline should be within 1 second of expected")
		})
	}
}

func TestGmailClient_RequireConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg).(*gmailClient)

	// Client is not connected, so all operations should fail
	err := client.requireConnected()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGmailClient_Login_NotConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.Login("user", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGmailClient_SelectMailbox_NotConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	info, err := client.SelectMailbox("INBOX")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
	assert.Nil(t, info)
}

func TestGmailClient_FetchMessages_NotConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.FetchMessages(context.Background(), func(msg *Message) error { return nil }, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGmailClient_FetchMessagesWithUIDs_NotConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.FetchMessagesWithUIDs(context.Background(), []uint32{1, 2, 3}, func(msg *Message) error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGmailClient_FetchMessagesWithUIDs_EmptyUIDs(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.FetchMessagesWithUIDs(context.Background(), []uint32{}, func(msg *Message) error { return nil })
	// Empty UIDs still requires connection check first
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGmailClient_MoveMessages_NotConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.MoveMessages(context.Background(), []uint32{1}, "Archive")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGmailClient_MoveMessages_EmptyUIDs(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.MoveMessages(context.Background(), []uint32{}, "Archive")
	// Empty UIDs still requires connection check first
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGmailClient_CreateMailbox_NotConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.CreateMailbox(context.Background(), "NewFolder")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGmailClient_ListMailboxes_NotConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	mailboxes, err := client.ListMailboxes(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
	assert.Nil(t, mailboxes)
}

func TestGmailClient_ArchiveToFolder_EmptyUIDs(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.ArchiveToFolder(context.Background(), []uint32{}, "Archive")
	// Empty UIDs should return nil without requiring connection
	assert.NoError(t, err)
}

func TestGmailClient_Logout_NotConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.Logout()
	// Logout on nil client should return nil
	assert.NoError(t, err)
}

func TestGmailClient_Close_NotConnected(t *testing.T) {
	cfg := &config.IMAPConfig{
		Server: "imap.gmail.com",
		Port:   993,
	}

	client := NewClient(cfg)
	err := client.Close()
	// Close on nil client should return nil
	assert.NoError(t, err)
}

func TestConvertAddresses(t *testing.T) {
	// Testing the unexported function via reflection is not ideal,
	// but we can test it indirectly through Address.String()
	addr := Address{
		Name:    "John Doe",
		Mailbox: "john",
		Host:    "example.com",
	}
	assert.Equal(t, "john@example.com", addr.String())
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"Hello World", "world", true},
		{"HELLO", "hello", true},
		{"test", "xyz", false},
		{"", "", true},
		{"test", "", true},
	}

	for _, tt := range tests {
		result := contains(tt.s, tt.substr)
		assert.Equal(t, tt.expected, result, "contains(%q, %q)", tt.s, tt.substr)
	}
}
