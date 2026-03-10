package views

import (
	"fmt"
	"strings"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
	"github.com/josegonzalez/gmail-categorizer/internal/tui/styles"
)

// RenderConfirm renders the confirmation dialog.
func RenderConfirm(grouping *triage.Grouping) string {
	var b strings.Builder

	b.WriteString(styles.WarningStyle.Render("Confirm Archive"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("This will archive %d messages from:\n", grouping.Count))
	b.WriteString(fmt.Sprintf("  %s\n", grouping.Address))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Messages will be moved to %s.\n", grouping.DestinationFolder()))
	b.WriteString("\n")
	b.WriteString("Continue? (y/n)")

	return styles.WarningBoxStyle.Render(b.String())
}

// RenderBatchConfirm renders the batch archive confirmation dialog.
func RenderBatchConfirm(groupings []*triage.Grouping) string {
	var b strings.Builder

	totalMessages := 0
	for _, g := range groupings {
		totalMessages += g.Count
	}

	b.WriteString(styles.WarningStyle.Render("Confirm Batch Archive"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("This will archive %d messages from %d groupings:\n\n", totalMessages, len(groupings)))

	for _, g := range groupings {
		b.WriteString(fmt.Sprintf("  %4d  %s → %s\n", g.Count, g.Address, g.DestinationFolder()))
	}

	b.WriteString("\n")
	b.WriteString("Continue? (y/n)")

	return styles.WarningBoxStyle.Render(b.String())
}

// BatchResultEntry contains the result for a single grouping in a batch archive.
type BatchResultEntry struct {
	Address           string
	ArchivedCount     int
	DestinationFolder string
	Err               error
}

// RenderBatchResult renders the batch archive result view.
func RenderBatchResult(results []BatchResultEntry) string {
	var b strings.Builder

	b.WriteString(styles.SuccessStyle.Render("Batch Archive Complete"))
	b.WriteString("\n\n")

	totalArchived := 0
	succeeded := 0
	failed := 0

	for _, r := range results {
		if r.Err != nil {
			failed++
			b.WriteString(fmt.Sprintf("  ✗ %s: %s\n", r.Address, r.Err.Error()))
		} else {
			succeeded++
			totalArchived += r.ArchivedCount
			b.WriteString(fmt.Sprintf("  %4d messages → %s\n", r.ArchivedCount, r.DestinationFolder))
		}
	}

	b.WriteString("\n")
	if failed > 0 {
		b.WriteString(fmt.Sprintf("Total: %d messages archived (%d succeeded, %d failed)", totalArchived, succeeded, failed))
	} else {
		b.WriteString(fmt.Sprintf("Total: %d messages archived across %d groupings", totalArchived, len(results)))
	}
	b.WriteString("\n\n")
	b.WriteString(styles.HelpStyle.Render("Press enter to continue or q to quit"))

	return styles.BoxStyle.Render(b.String())
}

// RenderResult renders the operation result.
func RenderResult(result *triage.TriageResult) string {
	var b strings.Builder

	b.WriteString(styles.SuccessStyle.Render("Archive Complete"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Archived %d messages to %s\n", result.ArchivedCount, result.DestinationFolder))
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("Press enter to continue or q to quit"))

	return styles.BoxStyle.Render(b.String())
}

// RenderError renders an error message.
func RenderError(err error) string {
	var b strings.Builder

	b.WriteString(styles.ErrorStyle.Render("Error"))
	b.WriteString("\n\n")
	b.WriteString(err.Error())
	b.WriteString("\n\n")
	b.WriteString(styles.HelpStyle.Render("Press any key to exit"))

	return styles.BoxStyle.Render(b.String())
}

// RenderLoading renders a loading message.
func RenderLoading(spinnerView string, message string) string {
	return fmt.Sprintf("%s %s", spinnerView, message)
}
