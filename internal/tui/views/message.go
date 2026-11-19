package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/tui/styles"
)

// MessageHeaderLines is how many lines RenderMessage uses around the body,
// so callers can size the body viewport to fit the terminal.
const MessageHeaderLines = 9

// RenderMessage renders a single message: its headers, the scrollable body
// and the key help.
func RenderMessage(msg *imap.Message, body string, scrollPercent float64, width int) string {
	var b strings.Builder

	if width <= 0 {
		width = 80
	}

	subject := "(no subject)"
	if msg != nil && msg.Subject != "" {
		subject = msg.Subject
	}
	b.WriteString(styles.TitleStyle.Render(truncate(subject, width)))
	b.WriteString("\n")

	label := lipgloss.NewStyle().Foreground(styles.MutedColor).Width(6)
	if msg != nil {
		b.WriteString(label.Render("From") + truncate(imap.FormatAddresses(msg.From), width-6) + "\n")
		b.WriteString(label.Render("To") + truncate(imap.FormatAddresses(msg.To), width-6) + "\n")
		if !msg.Date.IsZero() {
			b.WriteString(label.Render("Date") + msg.Date.Format("Mon, Jan 2 2006 3:04 PM") + "\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(body)
	b.WriteString("\n\n")

	help := "↑/↓ scroll • pgup/pgdn page • a archive all • esc back • q quit"
	if scrollPercent >= 0 {
		help = fmt.Sprintf("%3.f%% • %s", scrollPercent*100, help)
	}
	b.WriteString(styles.HelpStyle.Render(help))

	return b.String()
}
