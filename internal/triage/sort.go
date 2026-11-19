package triage

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

// sortModeCount is the number of sort modes, used to cycle through them.
const sortModeCount = 4

// Next returns the sort mode that follows m, wrapping around.
func (m SortMode) Next() SortMode {
	return (m + 1) % sortModeCount
}

// Key returns the identifier used for the sort mode in URLs.
func (m SortMode) Key() string {
	switch m {
	case SortDateAsc:
		return "date-asc"
	case SortSubjectAsc:
		return "subject-asc"
	case SortSubjectDesc:
		return "subject-desc"
	default:
		return "date-desc"
	}
}

// ParseSortMode converts a key produced by SortMode.Key back into a SortMode.
// Unknown or empty keys return SortDateDesc.
func ParseSortMode(key string) SortMode {
	switch key {
	case "date-asc":
		return SortDateAsc
	case "subject-asc":
		return SortSubjectAsc
	case "subject-desc":
		return SortSubjectDesc
	default:
		return SortDateDesc
	}
}

// SortMessages sorts messages in place based on the given sort mode.
func SortMessages(messages []*imap.Message, mode SortMode) {
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
