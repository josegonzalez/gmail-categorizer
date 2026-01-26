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
