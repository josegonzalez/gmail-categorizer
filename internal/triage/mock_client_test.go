package triage

import (
	"context"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

// mockClient implements imap.Client for testing.
type mockClient struct {
	ConnectFunc             func(ctx context.Context) error
	LoginFunc               func(username, password string) error
	SelectMailboxFunc       func(name string) (*imap.MailboxInfo, error)
	FetchMessagesFunc       func(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error
	FetchMessagesWithUIDsFunc func(ctx context.Context, uids []uint32, handler imap.MessageHandler) error
	MoveMessagesFunc        func(ctx context.Context, uids []uint32, destMailbox string) error
	ArchiveToFolderFunc     func(ctx context.Context, uids []uint32, folder string) error
	CreateMailboxFunc       func(ctx context.Context, name string) error
	ListMailboxesFunc       func(ctx context.Context) ([]string, error)
	LogoutFunc              func() error
	CloseFunc               func() error
}

func (m *mockClient) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}
	return nil
}

func (m *mockClient) Login(username, password string) error {
	if m.LoginFunc != nil {
		return m.LoginFunc(username, password)
	}
	return nil
}

func (m *mockClient) SelectMailbox(name string) (*imap.MailboxInfo, error) {
	if m.SelectMailboxFunc != nil {
		return m.SelectMailboxFunc(name)
	}
	return &imap.MailboxInfo{Name: name, NumMessages: 0}, nil
}

func (m *mockClient) FetchMessages(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
	if m.FetchMessagesFunc != nil {
		return m.FetchMessagesFunc(ctx, handler, progress)
	}
	return nil
}

func (m *mockClient) FetchMessagesWithUIDs(ctx context.Context, uids []uint32, handler imap.MessageHandler) error {
	if m.FetchMessagesWithUIDsFunc != nil {
		return m.FetchMessagesWithUIDsFunc(ctx, uids, handler)
	}
	return nil
}

func (m *mockClient) MoveMessages(ctx context.Context, uids []uint32, destMailbox string) error {
	if m.MoveMessagesFunc != nil {
		return m.MoveMessagesFunc(ctx, uids, destMailbox)
	}
	return nil
}

func (m *mockClient) ArchiveToFolder(ctx context.Context, uids []uint32, folder string) error {
	if m.ArchiveToFolderFunc != nil {
		return m.ArchiveToFolderFunc(ctx, uids, folder)
	}
	return nil
}

func (m *mockClient) CreateMailbox(ctx context.Context, name string) error {
	if m.CreateMailboxFunc != nil {
		return m.CreateMailboxFunc(ctx, name)
	}
	return nil
}

func (m *mockClient) ListMailboxes(ctx context.Context) ([]string, error) {
	if m.ListMailboxesFunc != nil {
		return m.ListMailboxesFunc(ctx)
	}
	return []string{}, nil
}

func (m *mockClient) Logout() error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc()
	}
	return nil
}

func (m *mockClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
