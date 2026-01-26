package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/tui/styles"
)

// RenderSubjects renders the subjects list view.
func RenderSubjects(address string, messages []*imap.Message, cursor int, width, height int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render(fmt.Sprintf("Messages for %s", address)))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.MutedColor).Render(fmt.Sprintf("%d messages", len(messages))))
	b.WriteString("\n\n")

	visible := CalculateVisibleRange(len(messages), cursor, height, 8)

	maxWidth := width - 4
	if maxWidth < 20 {
		maxWidth = 60
	}

	for i := visible.Start; i < visible.End; i++ {
		msg := messages[i]
		subject := msg.Subject
		if subject == "" {
			subject = "(no subject)"
		}

		// Truncate if too long
		if len(subject) > maxWidth {
			subject = subject[:maxWidth-3] + "..."
		}

		if i == cursor {
			b.WriteString(styles.SelectedStyle.Render(subject))
		} else {
			b.WriteString(styles.NormalStyle.Render(subject))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("↑/↓ navigate • a archive all • esc back • q quit"))

	return b.String()
}
