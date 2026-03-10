package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
	"github.com/josegonzalez/gmail-categorizer/internal/tui/styles"
)

// RenderGroupings renders the groupings list view.
func RenderGroupings(groupings []*triage.Grouping, cursor int, width, height int, checked map[int]bool) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Email Groupings"))
	b.WriteString("\n")

	subtitle := fmt.Sprintf("%d groupings in inbox", len(groupings))
	checkedCount := len(checked)
	if checkedCount > 0 {
		subtitle += fmt.Sprintf(" • %d selected", checkedCount)
	}
	b.WriteString(lipgloss.NewStyle().Foreground(styles.MutedColor).Render(subtitle))
	b.WriteString("\n\n")

	visible := CalculateVisibleRange(len(groupings), cursor, height, 8)

	for i := visible.Start; i < visible.End; i++ {
		g := groupings[i]
		checkbox := "[ ]"
		if checked[i] {
			checkbox = "[x]"
		}
		line := fmt.Sprintf("%s %s  %s", checkbox, styles.CountStyle.Render(fmt.Sprintf("%4d", g.Count)), g.Address)

		if i == cursor {
			b.WriteString(styles.SelectedStyle.Render(line))
		} else {
			b.WriteString(styles.NormalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("↑/↓ navigate • space toggle • enter view • a archive • q quit"))

	return b.String()
}
