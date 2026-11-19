package imap

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
)

// Client defines the interface for IMAP operations.
type Client interface {
	Connect(ctx context.Context) error
	Login(ctx context.Context, username, password string) error
	SelectMailbox(ctx context.Context, name string) (*MailboxInfo, error)
	FetchMessages(ctx context.Context, handler MessageHandler, progress ProgressCallback) error
	FetchMessagesWithUIDs(ctx context.Context, uids []uint32, handler MessageHandler) error
	FetchMessageBody(ctx context.Context, uid uint32) (*MessageBody, error)
	MoveMessages(ctx context.Context, uids []uint32, destMailbox string) error
	ArchiveToFolder(ctx context.Context, uids []uint32, folder string) error
	CreateMailbox(ctx context.Context, name string) error
	ListMailboxes(ctx context.Context) ([]string, error)
	Noop(ctx context.Context) error
	Logout() error
	Close() error
}

// gmailClient implements the Client interface for Gmail IMAP.
type gmailClient struct {
	config  *config.IMAPConfig
	client  *imapclient.Client
	mailbox *imap.SelectData
}

// NewClient creates a new Gmail IMAP client.
func NewClient(cfg *config.IMAPConfig) Client {
	return &gmailClient{config: cfg}
}

// requireConnected returns an error if the client is not connected.
func (c *gmailClient) requireConnected() error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}
	return nil
}

// defaultCommandTimeout applies when the config sets no timeout.
const defaultCommandTimeout = 30 * time.Second

// commandTimeout is how long short commands (connecting, logging in,
// selecting a mailbox, NOOP and LOGOUT) may wait for the server.
func (c *gmailClient) commandTimeout() time.Duration {
	if c.config == nil || c.config.Timeout <= 0 {
		return defaultCommandTimeout
	}
	return c.config.Timeout
}

// guard runs fn, which blocks on the IMAP connection, and gives up when ctx
// is cancelled or, if timeout is non-zero, when timeout passes. Giving up
// closes the connection: that is the only way to unblock a command waiting on
// the server, and a command abandoned part way leaves the session in an
// unknown state anyway. Long operations such as fetching the whole inbox pass
// a zero timeout so they can only be cancelled.
func (c *gmailClient) guard(ctx context.Context, timeout time.Duration, op string, fn func() error) error {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	client := c.client
	stop := context.AfterFunc(ctx, func() {
		if client != nil {
			_ = client.Close()
		}
	})
	err := fn()
	if !stop() {
		// ctx ended while fn was running and the connection was closed.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			server := "the server"
			if c.config != nil && c.config.Server != "" {
				server = c.config.Server
			}
			return fmt.Errorf("%s: no response from %s within %s: %w", op, server, timeout, ctx.Err())
		}
		return fmt.Errorf("%s: %w", op, ctx.Err())
	}
	return err
}

// Connect establishes a TLS connection to the IMAP server and waits for its
// greeting.
func (c *gmailClient) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.config.Server, c.config.Port)
	timeout := c.commandTimeout()

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: timeout},
		Config:    &tls.Config{NextProtos: []string{"imap"}},
	}
	conn, err := dialer.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("connecting to %s: no response within %s: %w", addr, timeout, err)
		}
		return fmt.Errorf("connecting to %s: %w", addr, err)
	}

	c.client = imapclient.New(conn, &imapclient.Options{
		// Enable debug logging if needed
		// DebugWriter: os.Stderr,
	})

	err = c.guard(ctx, timeout, "waiting for server greeting", func() error {
		return c.client.WaitGreeting()
	})
	if err != nil {
		_ = c.client.Close()
		c.client = nil
		return fmt.Errorf("connecting to %s: %w", addr, err)
	}

	return nil
}

// Login authenticates with the IMAP server.
func (c *gmailClient) Login(ctx context.Context, username, password string) error {
	if err := c.requireConnected(); err != nil {
		return err
	}

	err := c.guard(ctx, c.commandTimeout(), "authenticating", func() error {
		return c.client.Login(username, password).Wait()
	})
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

// SelectMailbox selects a mailbox for reading.
func (c *gmailClient) SelectMailbox(ctx context.Context, name string) (*MailboxInfo, error) {
	if err := c.requireConnected(); err != nil {
		return nil, err
	}

	var data *imap.SelectData
	err := c.guard(ctx, c.commandTimeout(), "selecting mailbox", func() error {
		var err error
		data, err = c.client.Select(name, nil).Wait()
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("selecting mailbox %q: %w", name, err)
	}

	c.mailbox = data

	return &MailboxInfo{
		Name:        name,
		NumMessages: data.NumMessages,
		UIDValidity: data.UIDValidity,
	}, nil
}

// fetchMessages retrieves all messages from the selected mailbox.
func (c *gmailClient) fetchMessages(ctx context.Context, handler MessageHandler, progress ProgressCallback) error {
	if err := c.requireConnected(); err != nil {
		return err
	}
	if c.mailbox == nil {
		return fmt.Errorf("no mailbox selected")
	}

	if c.mailbox.NumMessages == 0 {
		return nil
	}

	// Fetch all messages
	var seqSet imap.SeqSet
	seqSet.AddRange(1, c.mailbox.NumMessages)

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		UID:      true,
	}

	fetchCmd := c.client.Fetch(seqSet, fetchOptions)

	var count uint32
	for {
		select {
		case <-ctx.Done():
			fetchCmd.Close()
			return ctx.Err()
		default:
		}

		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		parsed, err := c.parseMessage(msg)
		if err != nil {
			// Skip messages that fail to parse
			continue
		}

		if err := handler(parsed); err != nil {
			fetchCmd.Close()
			return fmt.Errorf("handling message: %w", err)
		}

		count++
		if progress != nil && count%100 == 0 {
			progress(count, c.mailbox.NumMessages)
		}
	}

	if err := fetchCmd.Close(); err != nil {
		return fmt.Errorf("fetching messages: %w", err)
	}

	// Final progress update
	if progress != nil {
		progress(count, c.mailbox.NumMessages)
	}

	return nil
}

// parseMessage converts an IMAP message to our Message type.
func (c *gmailClient) parseMessage(msg *imapclient.FetchMessageData) (*Message, error) {
	var envelope *imap.Envelope
	var uid imap.UID
	var seqNum uint32

	for {
		item := msg.Next()
		if item == nil {
			break
		}

		switch data := item.(type) {
		case imapclient.FetchItemDataEnvelope:
			envelope = data.Envelope
		case imapclient.FetchItemDataUID:
			uid = data.UID
		}
	}

	seqNum = msg.SeqNum

	if envelope == nil {
		return nil, fmt.Errorf("no envelope data")
	}

	parsed := &Message{
		UID:     uint32(uid),
		SeqNum:  seqNum,
		Subject: envelope.Subject,
		Date:    envelope.Date,
		From:    convertAddresses(envelope.From),
		To:      convertAddresses(envelope.To),
	}

	return parsed, nil
}

// convertAddresses converts IMAP addresses to our Address type.
func convertAddresses(addrs []imap.Address) []Address {
	result := make([]Address, 0, len(addrs))
	for _, a := range addrs {
		result = append(result, Address{
			Name:    a.Name,
			Mailbox: a.Mailbox,
			Host:    a.Host,
		})
	}
	return result
}

// Noop sends a NOOP command, which fails when the connection has dropped or
// the server stops responding.
func (c *gmailClient) Noop(ctx context.Context) error {
	if err := c.requireConnected(); err != nil {
		return err
	}
	err := c.guard(ctx, c.commandTimeout(), "checking connection", func() error {
		return c.client.Noop().Wait()
	})
	if err != nil {
		return fmt.Errorf("checking connection: %w", err)
	}
	return nil
}

// Logout logs out from the IMAP server. It gives up after the command
// timeout so a stalled server cannot block shutdown.
func (c *gmailClient) Logout() error {
	if c.client == nil {
		return nil
	}

	return c.guard(context.Background(), c.commandTimeout(), "logging out", func() error {
		return c.client.Logout().Wait()
	})
}

// Close closes the connection to the IMAP server.
func (c *gmailClient) Close() error {
	if c.client == nil {
		return nil
	}

	return c.client.Close()
}

// WithTimeout wraps a context with the configured timeout.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return context.WithTimeout(ctx, timeout)
}

// fetchMessagesWithUIDs retrieves specific messages by UID.
func (c *gmailClient) fetchMessagesWithUIDs(ctx context.Context, uids []uint32, handler MessageHandler) error {
	if err := c.requireConnected(); err != nil {
		return err
	}
	if len(uids) == 0 {
		return nil
	}

	var uidSet imap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(imap.UID(uid))
	}

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		UID:      true,
	}

	fetchCmd := c.client.Fetch(uidSet, fetchOptions)

	for {
		select {
		case <-ctx.Done():
			fetchCmd.Close()
			return ctx.Err()
		default:
		}

		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		parsed, err := c.parseMessage(msg)
		if err != nil {
			continue
		}

		if err := handler(parsed); err != nil {
			fetchCmd.Close()
			return fmt.Errorf("handling message: %w", err)
		}
	}

	return fetchCmd.Close()
}

// fetchMessageBody retrieves and parses the body of a single message by UID.
// It uses BODY.PEEK so viewing a message does not mark it as read, and fetches
// at most MaxBodySize bytes.
func (c *gmailClient) fetchMessageBody(ctx context.Context, uid uint32) (*MessageBody, error) {
	if err := c.requireConnected(); err != nil {
		return nil, err
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(uid))

	section := &imap.FetchItemBodySection{
		Peek:    true,
		Partial: &imap.SectionPartial{Offset: 0, Size: MaxBodySize},
	}
	fetchCmd := c.client.Fetch(uidSet, &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{section},
	})

	var body *MessageBody
	for {
		if err := ctx.Err(); err != nil {
			fetchCmd.Close()
			return nil, err
		}

		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		for {
			item := msg.Next()
			if item == nil {
				break
			}
			data, ok := item.(imapclient.FetchItemDataBodySection)
			if !ok || data.Literal == nil || body != nil {
				continue
			}
			parsed, err := parseBody(io.LimitReader(data.Literal, MaxBodySize))
			if err != nil {
				fetchCmd.Close()
				return nil, err
			}
			body = parsed
		}
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("fetching message %d: %w", uid, err)
	}
	if body == nil {
		return nil, fmt.Errorf("message %d not found", uid)
	}
	return body, nil
}

// listMailboxes returns all available mailboxes.
func (c *gmailClient) listMailboxes(ctx context.Context) ([]string, error) {
	if err := c.requireConnected(); err != nil {
		return nil, err
	}

	listCmd := c.client.List("", "*", nil)

	var mailboxes []string
	for {
		select {
		case <-ctx.Done():
			listCmd.Close()
			return nil, ctx.Err()
		default:
		}

		mbox := listCmd.Next()
		if mbox == nil {
			break
		}
		mailboxes = append(mailboxes, mbox.Mailbox)
	}

	if err := listCmd.Close(); err != nil {
		return nil, fmt.Errorf("listing mailboxes: %w", err)
	}

	return mailboxes, nil
}

// moveMessages moves messages by UID to another mailbox.
func (c *gmailClient) moveMessages(ctx context.Context, uids []uint32, destMailbox string) error {
	if err := c.requireConnected(); err != nil {
		return err
	}
	if len(uids) == 0 {
		return nil
	}

	var uidSet imap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(imap.UID(uid))
	}

	// Copy to destination
	copyCmd := c.client.Copy(uidSet, destMailbox)
	if _, err := copyCmd.Wait(); err != nil {
		return fmt.Errorf("copying messages to %q: %w", destMailbox, err)
	}

	// Mark as deleted in current mailbox
	storeFlags := imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Flags:  []imap.Flag{imap.FlagDeleted},
		Silent: true,
	}
	storeCmd := c.client.Store(uidSet, &storeFlags, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("marking messages as deleted: %w", err)
	}

	// Expunge deleted messages
	expungeCmd := c.client.Expunge()
	if err := expungeCmd.Close(); err != nil {
		return fmt.Errorf("expunging messages: %w", err)
	}

	return nil
}

// createMailbox creates a new mailbox/folder.
// It handles "already exists" errors gracefully.
func (c *gmailClient) createMailbox(ctx context.Context, name string) error {
	if err := c.requireConnected(); err != nil {
		return err
	}

	cmd := c.client.Create(name, nil)
	err := cmd.Wait()
	if err != nil {
		// Check if the error is "mailbox already exists" - this is acceptable
		errStr := err.Error()
		if contains(errStr, "already exists") || contains(errStr, "ALREADYEXISTS") || contains(errStr, "Duplicate folder") {
			return nil
		}
		return fmt.Errorf("creating mailbox %q: %w", name, err)
	}

	return nil
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ArchiveToFolder archives messages by moving them to a specified folder.
// It creates the folder if it doesn't exist.
func (c *gmailClient) ArchiveToFolder(ctx context.Context, uids []uint32, folder string) error {
	if len(uids) == 0 {
		return nil
	}

	// Create the folder if it doesn't exist
	if err := c.CreateMailbox(ctx, folder); err != nil {
		return fmt.Errorf("ensuring folder exists: %w", err)
	}

	// Move messages to the folder
	return c.MoveMessages(ctx, uids, folder)
}

// FetchMessages is cancelled with ctx.
func (c *gmailClient) FetchMessages(ctx context.Context, handler MessageHandler, progress ProgressCallback) error {
	if err := c.requireConnected(); err != nil {
		return err
	}
	return c.guard(ctx, 0, "fetching messages", func() error {
		return c.fetchMessages(ctx, handler, progress)
	})
}

// FetchMessagesWithUIDs is cancelled with ctx.
func (c *gmailClient) FetchMessagesWithUIDs(ctx context.Context, uids []uint32, handler MessageHandler) error {
	if err := c.requireConnected(); err != nil {
		return err
	}
	return c.guard(ctx, 0, "fetching messages", func() error {
		return c.fetchMessagesWithUIDs(ctx, uids, handler)
	})
}

// MoveMessages is cancelled with ctx.
func (c *gmailClient) MoveMessages(ctx context.Context, uids []uint32, destMailbox string) error {
	if err := c.requireConnected(); err != nil {
		return err
	}
	return c.guard(ctx, 0, "moving messages", func() error {
		return c.moveMessages(ctx, uids, destMailbox)
	})
}

// CreateMailbox is cancelled with ctx and times out after the command timeout.
func (c *gmailClient) CreateMailbox(ctx context.Context, name string) error {
	if err := c.requireConnected(); err != nil {
		return err
	}
	return c.guard(ctx, c.commandTimeout(), "creating mailbox", func() error {
		return c.createMailbox(ctx, name)
	})
}

// FetchMessageBody is cancelled with ctx.
func (c *gmailClient) FetchMessageBody(ctx context.Context, uid uint32) (*MessageBody, error) {
	if err := c.requireConnected(); err != nil {
		return nil, err
	}
	var result *MessageBody
	err := c.guard(ctx, 0, "fetching message body", func() error {
		var err error
		result, err = c.fetchMessageBody(ctx, uid)
		return err
	})
	return result, err
}

// ListMailboxes is cancelled with ctx.
func (c *gmailClient) ListMailboxes(ctx context.Context) ([]string, error) {
	if err := c.requireConnected(); err != nil {
		return nil, err
	}
	var result []string
	err := c.guard(ctx, 0, "listing mailboxes", func() error {
		var err error
		result, err = c.listMailboxes(ctx)
		return err
	})
	return result, err
}
