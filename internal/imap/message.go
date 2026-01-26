// Package imap provides IMAP client functionality for Gmail.
package imap

import (
	"time"

	"github.com/josegonzalez/gmail-categorizer/pkg/mailaddr"
)

// Message represents a parsed email message with relevant headers.
type Message struct {
	UID     uint32
	SeqNum  uint32
	From    []Address
	To      []Address
	Subject string
	Date    time.Time
}

// Address represents an email address with optional display name.
type Address struct {
	Name    string
	Mailbox string // local part
	Host    string // domain
}

// String returns the full email address (mailbox@host).
func (a Address) String() string {
	return mailaddr.Format(a.Mailbox, a.Host)
}

// MailboxInfo contains metadata about a mailbox.
type MailboxInfo struct {
	Name        string
	NumMessages uint32
	NumUnseen   uint32
}

// MessageHandler is called for each message during fetch operations.
type MessageHandler func(msg *Message) error

// ProgressCallback is called to report fetch progress.
type ProgressCallback func(current, total uint32)
