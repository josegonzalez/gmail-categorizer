package imap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
)

// Client defines the interface for IMAP operations.
type Client interface {
	Connect(ctx context.Context) error
	Login(username, password string) error
	SelectMailbox(name string) (*MailboxInfo, error)
	FetchMessages(ctx context.Context, handler MessageHandler, progress ProgressCallback) error
	FetchMessagesWithUIDs(ctx context.Context, uids []uint32, handler MessageHandler) error
	MoveMessages(ctx context.Context, uids []uint32, destMailbox string) error
	ArchiveToFolder(ctx context.Context, uids []uint32, folder string) error
	CreateMailbox(ctx context.Context, name string) error
	ListMailboxes(ctx context.Context) ([]string, error)
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

// Connect establishes a TLS connection to the IMAP server.
func (c *gmailClient) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.config.Server, c.config.Port)

	options := &imapclient.Options{
		// Enable debug logging if needed
		// DebugWriter: os.Stderr,
	}

	var err error
	c.client, err = imapclient.DialTLS(addr, options)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", addr, err)
	}

	return nil
}

// Login authenticates with the IMAP server.
func (c *gmailClient) Login(username, password string) error {
	if err := c.requireConnected(); err != nil {
		return err
	}

	cmd := c.client.Login(username, password)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

// SelectMailbox selects a mailbox for reading.
func (c *gmailClient) SelectMailbox(name string) (*MailboxInfo, error) {
	if err := c.requireConnected(); err != nil {
		return nil, err
	}

	cmd := c.client.Select(name, nil)
	data, err := cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("selecting mailbox %q: %w", name, err)
	}

	c.mailbox = data

	return &MailboxInfo{
		Name:        name,
		NumMessages: data.NumMessages,
	}, nil
}

// FetchMessages retrieves all messages from the selected mailbox.
func (c *gmailClient) FetchMessages(ctx context.Context, handler MessageHandler, progress ProgressCallback) error {
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

// Logout logs out from the IMAP server.
func (c *gmailClient) Logout() error {
	if c.client == nil {
		return nil
	}

	cmd := c.client.Logout()
	return cmd.Wait()
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

// FetchMessagesWithUIDs retrieves specific messages by UID.
func (c *gmailClient) FetchMessagesWithUIDs(ctx context.Context, uids []uint32, handler MessageHandler) error {
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

// ListMailboxes returns all available mailboxes.
func (c *gmailClient) ListMailboxes(ctx context.Context) ([]string, error) {
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

// MoveMessages moves messages by UID to another mailbox.
func (c *gmailClient) MoveMessages(ctx context.Context, uids []uint32, destMailbox string) error {
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

// CreateMailbox creates a new mailbox/folder.
// It handles "already exists" errors gracefully.
func (c *gmailClient) CreateMailbox(ctx context.Context, name string) error {
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
