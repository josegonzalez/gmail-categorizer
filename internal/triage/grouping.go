// Package triage provides email triage business logic.
package triage

import (
	"net/url"
	"strings"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/pkg/mailaddr"
)

// Grouping represents a collection of emails grouped by recipient address.
type Grouping struct {
	Address string
	Count   int
	// Subject is set only when the grouping holds a single message.
	Subject string
	UIDs    []uint32
	// Messages holds the envelope of every message in the grouping, newest
	// first once LoadGroupings returns.
	Messages      []*imap.Message
	GroupedByFrom bool
}

// DestinationFolder returns the archive folder path for this grouping.
func (g *Grouping) DestinationFolder() string {
	return "automated/" + mailaddr.LocalPart(g.Address)
}

// HasUID reports whether uid belongs to this grouping.
func (g *Grouping) HasUID(uid uint32) bool {
	for _, u := range g.UIDs {
		if u == uid {
			return true
		}
	}
	return false
}

// removeUIDs drops the given messages from the grouping, keeping UIDs,
// Messages and Count in step.
func (g *Grouping) removeUIDs(uids []uint32) {
	if len(uids) == 0 {
		return
	}
	gone := make(map[uint32]bool, len(uids))
	for _, uid := range uids {
		gone[uid] = true
	}

	keptUIDs := g.UIDs[:0]
	for _, uid := range g.UIDs {
		if !gone[uid] {
			keptUIDs = append(keptUIDs, uid)
		}
	}
	g.UIDs = keptUIDs

	keptMessages := g.Messages[:0]
	for _, msg := range g.Messages {
		if !gone[msg.UID] {
			keptMessages = append(keptMessages, msg)
		}
	}
	g.Messages = keptMessages
	g.Count = len(g.UIDs)
	if g.Count != 1 {
		g.Subject = ""
	} else if len(g.Messages) == 1 {
		g.Subject = g.Messages[0].Subject
	}
}

// GmailSearchQuery returns a Gmail search that finds this grouping's inbox
// messages. Groupings made by sender search on the sender; the rest search on
// the address the mail was delivered to, which is what a catch-all inbox
// groups on.
func (g *Grouping) GmailSearchQuery() string {
	if g.GroupedByFrom {
		return "from:" + g.Address + " in:inbox"
	}
	return "deliveredto:" + g.Address + " in:inbox"
}

// GmailSearchURL returns a link that opens GmailSearchQuery in the Gmail web
// UI for the given account.
func (g *Grouping) GmailSearchURL(username string) string {
	u := "https://mail.google.com/mail/"
	if username != "" {
		u += "?authuser=" + url.QueryEscape(username)
	}
	// Gmail reads "+" in the search fragment as a space, so escape it too.
	query := strings.ReplaceAll(url.PathEscape(g.GmailSearchQuery()), "+", "%2B")
	return u + "#search/" + query
}

// TriageResult contains the result of a triage operation.
type TriageResult struct {
	ArchivedCount     int
	DestinationFolder string
}

// BatchResult holds the result of archiving one grouping in a batch.
type BatchResult struct {
	Address string
	Result  *TriageResult
	Err     error
}
