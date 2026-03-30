package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
	"github.com/josegonzalez/gmail-categorizer/internal/tui/styles"
)

// RenderGroupings renders the groupings list view.
func RenderGroupings(groupings []*triage.Grouping, cursor int, width, height int, checked map[int]bool, filterMode int, specialCount int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Email Groupings"))
	b.WriteString("\n")

	var subtitle string
	if filterMode == 1 {
		// FilterSpecial
		subtitle = fmt.Sprintf("%d special groupings (grouped by sender) • f show all", len(groupings))
	} else {
		// FilterAll
		subtitle = fmt.Sprintf("%d groupings in inbox", len(groupings))
		if specialCount > 0 {
			subtitle += fmt.Sprintf(" (%d special)", specialCount)
		}
	}
	checkedCount := len(checked)
	if checkedCount > 0 {
		subtitle += fmt.Sprintf(" • %d selected", checkedCount)
	}
	b.WriteString(lipgloss.NewStyle().Foreground(styles.MutedColor).Render(subtitle))
	b.WriteString("\n\n")

	visible := CalculateVisibleRange(len(groupings), cursor, height, 8)

	// Calculate max address width and whether any single-email subjects exist
	maxAddrWidth := 0
	hasSubjects := false
	for i := visible.Start; i < visible.End; i++ {
		if len(groupings[i].Address) > maxAddrWidth {
			maxAddrWidth = len(groupings[i].Address)
		}
		if groupings[i].Count == 1 && groupings[i].Subject != "" {
			hasSubjects = true
		}
	}

	for i := visible.Start; i < visible.End; i++ {
		g := groupings[i]
		checkbox := "[ ]"
		if checked[i] {
			checkbox = "[x]"
		}

		addr := g.Address
		if hasSubjects {
			addr = fmt.Sprintf("%-*s", maxAddrWidth, g.Address)
		}

		line := fmt.Sprintf("%s %s  %s", checkbox, styles.CountStyle.Render(fmt.Sprintf("%4d", g.Count)), addr)

		if g.GroupedByFrom {
			line += "  " + styles.SpecialMarkerStyle.Render("*")
		}

		if g.Count == 1 && g.Subject != "" {
			// Fixed overhead: checkbox(3) + space(1) + count(4) + spacing(2) + addr + spacing(2) + marker(~3)
			overhead := 12 + maxAddrWidth + 4
			maxSubjectWidth := width - overhead
			if maxSubjectWidth < 10 {
				maxSubjectWidth = 10
			}
			subject := g.Subject
			if len(subject) > maxSubjectWidth {
				subject = subject[:maxSubjectWidth-3] + "..."
			}
			line += "  " + styles.SubjectStyle.Render(subject)
		}

		if i == cursor {
			b.WriteString(styles.SelectedStyle.Render(line))
		} else {
			b.WriteString(styles.NormalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("↑/↓ navigate • space toggle • enter view • a archive • f filter • q quit"))

	return b.String()
}
