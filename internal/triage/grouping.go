// Package triage provides email triage business logic.
package triage

import (
	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/pkg/mailaddr"
)

// Grouping represents a collection of emails grouped by recipient address.
type Grouping struct {
	Address  string
	Count    int
	UIDs     []uint32
	Messages []*imap.Message
}

// DestinationFolder returns the archive folder path for this grouping.
func (g *Grouping) DestinationFolder() string {
	return "automated/" + mailaddr.LocalPart(g.Address)
}

// TriageResult contains the result of a triage operation.
type TriageResult struct {
	ArchivedCount     int
	DestinationFolder string
}
