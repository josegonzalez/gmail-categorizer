package tui

import (
	"sort"
	"strings"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

// SortMode represents the sort order for messages.
type SortMode int

const (
	SortDateDesc    SortMode = iota // newest first
	SortDateAsc                     // oldest first
	SortSubjectAsc                  // A→Z
	SortSubjectDesc                 // Z→A
)

// sortMessages sorts messages in place based on the given sort mode.
func sortMessages(messages []*imap.Message, mode SortMode) {
	sort.SliceStable(messages, func(i, j int) bool {
		switch mode {
		case SortDateAsc:
			return messages[i].Date.Before(messages[j].Date)
		case SortSubjectAsc:
			return strings.ToLower(messages[i].Subject) < strings.ToLower(messages[j].Subject)
		case SortSubjectDesc:
			return strings.ToLower(messages[i].Subject) > strings.ToLower(messages[j].Subject)
		default: // SortDateDesc
			return messages[i].Date.After(messages[j].Date)
		}
	})
}

// sortModeLabel returns a display label for the given sort mode.
func sortModeLabel(mode SortMode) string {
	switch mode {
	case SortDateAsc:
		return "date ↑"
	case SortSubjectAsc:
		return "subject A→Z"
	case SortSubjectDesc:
		return "subject Z→A"
	default:
		return "date ↓"
	}
}
