package web

import (
	"context"
	"errors"
	"sync"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

// mockClient is an in-memory imap.Client holding an inbox of messages.
type mockClient struct {
	mu       sync.Mutex
	messages []*imap.Message
	bodies   map[uint32]*imap.MessageBody

	noopErr    error
	fetchErr   error
	bodyErr    error
	archiveErr map[string]error

	// failMoveCall makes the nth MoveMessages call (1-based) fail.
	failMoveCall int
	moveCalls    int

	fetchCalls int
	bodyCalls  []uint32
	archived   map[string][]uint32
	closed     bool
}

func (m *mockClient) Connect(ctx context.Context) error                          { return nil }
func (m *mockClient) Login(ctx context.Context, username, password string) error { return nil }

func (m *mockClient) SelectMailbox(ctx context.Context, name string) (*imap.MailboxInfo, error) {
	return &imap.MailboxInfo{Name: name, NumMessages: uint32(len(m.messages))}, nil
}

func (m *mockClient) FetchMessages(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
	m.mu.Lock()
	m.fetchCalls++
	msgs := append([]*imap.Message(nil), m.messages...)
	err := m.fetchErr
	m.mu.Unlock()
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		// Hand out copies, as a real fetch would.
		c := *msg
		if err := handler(&c); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockClient) FetchMessagesWithUIDs(ctx context.Context, uids []uint32, handler imap.MessageHandler) error {
	return nil
}

func (m *mockClient) FetchMessageBody(ctx context.Context, uid uint32) (*imap.MessageBody, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bodyCalls = append(m.bodyCalls, uid)
	if m.bodyErr != nil {
		return nil, m.bodyErr
	}
	if b, ok := m.bodies[uid]; ok {
		return b, nil
	}
	return &imap.MessageBody{}, nil
}

func (m *mockClient) ArchiveToFolder(ctx context.Context, uids []uint32, folder string) error {
	return m.MoveMessages(ctx, uids, folder)
}

func (m *mockClient) MoveMessages(ctx context.Context, uids []uint32, folder string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.moveCalls++
	if m.failMoveCall == m.moveCalls {
		return errors.New("connection reset")
	}
	if err := m.archiveErr[folder]; err != nil {
		return err
	}
	if m.archived == nil {
		m.archived = make(map[string][]uint32)
	}
	m.archived[folder] = append(m.archived[folder], uids...)

	// Remove archived messages from the inbox.
	gone := make(map[uint32]bool, len(uids))
	for _, uid := range uids {
		gone[uid] = true
	}
	kept := m.messages[:0]
	for _, msg := range m.messages {
		if !gone[msg.UID] {
			kept = append(kept, msg)
		}
	}
	m.messages = kept
	return nil
}

func (m *mockClient) CreateMailbox(ctx context.Context, name string) error { return nil }

func (m *mockClient) ListMailboxes(ctx context.Context) ([]string, error) { return nil, nil }

func (m *mockClient) Noop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.noopErr
}

func (m *mockClient) Logout() error { return nil }

func (m *mockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}
