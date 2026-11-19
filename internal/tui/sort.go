package tui

import (
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

// SortMode is the message sort order, shared with the web UI.
type SortMode = triage.SortMode

// Re-export the sort modes for use within the tui package.
const (
	SortDateDesc    = triage.SortDateDesc
	SortDateAsc     = triage.SortDateAsc
	SortSubjectAsc  = triage.SortSubjectAsc
	SortSubjectDesc = triage.SortSubjectDesc
)

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
