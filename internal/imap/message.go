// Package imap provides IMAP client functionality for Gmail.
package imap

import (
	"strings"
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

// FormatAddresses formats addresses as "Name <mailbox@host>", comma
// separated, falling back to whichever of the name or address is present.
func FormatAddresses(addrs []Address) string {
	parts := make([]string, 0, len(addrs))
	for _, a := range addrs {
		addr := a.String()
		switch {
		case a.Name != "" && addr != "":
			parts = append(parts, a.Name+" <"+addr+">")
		case addr != "":
			parts = append(parts, addr)
		case a.Name != "":
			parts = append(parts, a.Name)
		}
	}
	return strings.Join(parts, ", ")
}

// MailboxInfo contains metadata about a mailbox.
type MailboxInfo struct {
	Name        string
	NumMessages uint32
	NumUnseen   uint32
	// UIDValidity changes when the server renumbers UIDs, which makes any
	// cached UIDs from this mailbox invalid.
	UIDValidity uint32
}

// MessageHandler is called for each message during fetch operations.
type MessageHandler func(msg *Message) error

// ProgressCallback is called to report fetch progress.
type ProgressCallback func(current, total uint32)
