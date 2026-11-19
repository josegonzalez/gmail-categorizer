package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
	"github.com/josegonzalez/gmail-categorizer/internal/tui/styles"
)

// PreviewLines is how many subjects the expanded cursor row shows at once.
const PreviewLines = 5

// GroupingsView holds everything needed to render the groupings list.
type GroupingsView struct {
	Groupings    []*triage.Grouping
	Cursor       int
	Width        int
	Height       int
	Checked      map[int]bool
	FilterMode   int
	SpecialCount int
	// PreviewOffset is the first subject shown in the expanded cursor row.
	PreviewOffset int
	// Status is a one-line message shown under the subtitle, such as a
	// browser launch error.
	Status string
	// Now is used to format dates; the zero value means time.Now().
	Now time.Time
}

// PreviewHasExpansion reports whether the grouping expands into a subject
// list when the cursor is on it. Single-message groupings already show their
// subject inline.
func PreviewHasExpansion(g *triage.Grouping) bool {
	return g != nil && g.Count > 1 && len(g.Messages) > 0
}

// MaxPreviewOffset returns the largest valid PreviewOffset for a grouping.
func MaxPreviewOffset(g *triage.Grouping) int {
	if !PreviewHasExpansion(g) || len(g.Messages) <= PreviewLines {
		return 0
	}
	return len(g.Messages) - PreviewLines
}

// expandedLines returns how many extra lines the expanded row takes.
func expandedLines(g *triage.Grouping) int {
	if !PreviewHasExpansion(g) {
		return 0
	}
	n := len(g.Messages)
	if n > PreviewLines {
		// Subjects plus the "more" line.
		return PreviewLines + 1
	}
	return n
}

// RenderGroupings renders the groupings list view.
func RenderGroupings(v GroupingsView) string {
	var b strings.Builder
	groupings := v.Groupings
	now := v.Now
	if now.IsZero() {
		now = time.Now()
	}

	b.WriteString(styles.TitleStyle.Render("Email Groupings"))
	b.WriteString("\n")

	var subtitle string
	if v.FilterMode == 1 {
		// FilterSpecial
		subtitle = fmt.Sprintf("%d special groupings (grouped by sender) • f show all", len(groupings))
	} else {
		// FilterAll
		subtitle = fmt.Sprintf("%d groupings in inbox", len(groupings))
		if v.SpecialCount > 0 {
			subtitle += fmt.Sprintf(" (%d special)", v.SpecialCount)
		}
	}
	if checkedCount := len(v.Checked); checkedCount > 0 {
		subtitle += fmt.Sprintf(" • %d selected", checkedCount)
	}
	b.WriteString(lipgloss.NewStyle().Foreground(styles.MutedColor).Render(subtitle))
	b.WriteString("\n")
	if v.Status != "" {
		b.WriteString(styles.WarningStyle.Render(v.Status))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	var cursorGrouping *triage.Grouping
	if v.Cursor >= 0 && v.Cursor < len(groupings) {
		cursorGrouping = groupings[v.Cursor]
	}
	headerOffset := 8 + expandedLines(cursorGrouping)
	if v.Status != "" {
		headerOffset++
	}
	visible := CalculateVisibleRange(len(groupings), v.Cursor, v.Height, headerOffset)

	// Column widths across the visible rows.
	maxAddrWidth, maxFolderWidth := 0, 0
	hasSubjects := false
	for i := visible.Start; i < visible.End; i++ {
		g := groupings[i]
		maxAddrWidth = max(maxAddrWidth, ansi.StringWidth(g.Address))
		maxFolderWidth = max(maxFolderWidth, ansi.StringWidth(g.DestinationFolder()))
		if g.Count == 1 && g.Subject != "" {
			hasSubjects = true
		}
	}

	for i := visible.Start; i < visible.End; i++ {
		g := groupings[i]
		checkbox := "[ ]"
		if v.Checked[i] {
			checkbox = "[x]"
		}

		marker := "   "
		if g.GroupedByFrom {
			marker = "  " + styles.SpecialMarkerStyle.Render("*")
		}

		folder := g.DestinationFolder()
		if hasSubjects {
			folder = padRight(folder, maxFolderWidth)
		}

		line := fmt.Sprintf("%s %s  %s%s  %s",
			checkbox,
			styles.CountStyle.Render(fmt.Sprintf("%4d", g.Count)),
			padRight(g.Address, maxAddrWidth),
			marker,
			styles.SubjectStyle.Render(folder),
		)

		if g.Count == 1 && g.Subject != "" {
			// checkbox(3) + space + count(4) + 2 + addr + marker(3) + 2 + folder + 2 + padding(2)
			overhead := 18 + maxAddrWidth + maxFolderWidth
			line += "  " + styles.SubjectStyle.Render(truncate(g.Subject, max(v.Width-overhead, 10)))
		}

		if i == v.Cursor {
			b.WriteString(styles.SelectedStyle.Render(line))
		} else {
			b.WriteString(styles.NormalStyle.Render(line))
		}
		b.WriteString("\n")

		if i == v.Cursor {
			b.WriteString(renderPreview(g, v.PreviewOffset, v.Width, now))
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("↑/↓ navigate • J/K scroll subjects • space toggle • enter view • a archive • g open in Gmail • f filter • q quit"))

	return b.String()
}

// renderPreview renders the expanded subject list under the cursor row.
func renderPreview(g *triage.Grouping, offset, width int, now time.Time) string {
	if !PreviewHasExpansion(g) {
		return ""
	}
	offset = min(max(offset, 0), MaxPreviewOffset(g))
	end := min(offset+PreviewLines, len(g.Messages))

	const indent = "         "
	dates := make([]string, 0, end-offset)
	dateWidth := 0
	for _, msg := range g.Messages[offset:end] {
		d := triage.ShortDate(msg.Date, now)
		dates = append(dates, d)
		dateWidth = max(dateWidth, ansi.StringWidth(d))
	}

	if width <= 0 {
		width = 80
	}
	subjectWidth := max(width-len(indent)-dateWidth-4, 10)

	var b strings.Builder
	for i, msg := range g.Messages[offset:end] {
		subject := msg.Subject
		if subject == "" {
			subject = "(no subject)"
		}
		row := indent + padRight(truncate(subject, subjectWidth), subjectWidth) + "  " + padLeft(dates[i], dateWidth)
		b.WriteString(styles.PreviewStyle.Render(row))
		b.WriteString("\n")
	}

	if len(g.Messages) > PreviewLines {
		var parts []string
		if offset > 0 {
			parts = append(parts, fmt.Sprintf("↑ %d above", offset))
		}
		if below := len(g.Messages) - end; below > 0 {
			parts = append(parts, fmt.Sprintf("↓ %d more", below))
		}
		b.WriteString(styles.SubjectStyle.Render(indent + strings.Join(parts, "  ") + "  (J/K scroll)"))
		b.WriteString("\n")
	}

	return b.String()
}

// truncate shortens s to at most width display cells, adding "..." when cut.
func truncate(s string, width int) string {
	if ansi.StringWidth(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "...")
}

// padRight pads s with spaces to width display cells.
func padRight(s string, width int) string {
	if pad := width - ansi.StringWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// padLeft pads s with leading spaces to width display cells.
func padLeft(s string, width int) string {
	if pad := width - ansi.StringWidth(s); pad > 0 {
		return strings.Repeat(" ", pad) + s
	}
	return s
}
