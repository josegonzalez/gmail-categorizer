package views

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

func TestRenderGroupings(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "user1@example.com", Count: 50},
		{Address: "user2@example.com", Count: 30},
		{Address: "user3@example.com", Count: 10},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 1, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})

	// Check title
	assert.Contains(t, result, "Email Groupings")

	// Check count display
	assert.Contains(t, result, "3 groupings")

	// Check addresses
	assert.Contains(t, result, "user1@example.com")
	assert.Contains(t, result, "user2@example.com")
	assert.Contains(t, result, "user3@example.com")

	// Check counts
	assert.Contains(t, result, "50")
	assert.Contains(t, result, "30")
	assert.Contains(t, result, "10")

	// Check help text
	assert.Contains(t, result, "navigate")
	assert.Contains(t, result, "quit")
	assert.Contains(t, result, "space toggle")
	assert.Contains(t, result, "f filter")

	// Check checkboxes are unchecked
	assert.Contains(t, result, "[ ]")
}

func TestRenderGroupings_Empty(t *testing.T) {
	groupings := []*triage.Grouping{}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})

	assert.Contains(t, result, "Email Groupings")
	assert.Contains(t, result, "0 groupings")
}

func TestRenderGroupings_SingleItem(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "only@example.com", Count: 100},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})

	assert.Contains(t, result, "only@example.com")
	assert.Contains(t, result, "100")
	assert.Contains(t, result, "1 groupings")
}

func TestRenderGroupings_CursorAtDifferentPositions(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "a@example.com", Count: 1},
		{Address: "b@example.com", Count: 2},
		{Address: "c@example.com", Count: 3},
	}

	// Cursor at beginning
	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})
	assert.Contains(t, result, "a@example.com")

	// Cursor in middle
	result = RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 1, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})
	assert.Contains(t, result, "b@example.com")

	// Cursor at end
	result = RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 2, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})
	assert.Contains(t, result, "c@example.com")
}

func TestRenderGroupings_SmallHeight(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "a@example.com", Count: 1},
		{Address: "b@example.com", Count: 2},
	}

	// Very small height
	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 5, Checked: nil, FilterMode: 0, SpecialCount: 0})
	assert.Contains(t, result, "Email Groupings")
}

func TestRenderGroupings_WithCheckedItems(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "a@example.com", Count: 15},
		{Address: "b@example.com", Count: 12},
		{Address: "c@example.com", Count: 8},
	}

	checked := map[int]bool{0: true, 2: true}
	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 1, Width: 80, Height: 24, Checked: checked, FilterMode: 0, SpecialCount: 0})

	// Check that checked items show [x]
	assert.Contains(t, result, "[x]")
	// Check that unchecked items show [ ]
	assert.Contains(t, result, "[ ]")
	// Check selected count in subtitle
	assert.Contains(t, result, "2 selected")
}

func TestRenderGroupings_NoSelectedCount_WhenNoneChecked(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "a@example.com", Count: 1},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})
	assert.NotContains(t, result, "selected")

	result = RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: map[int]bool{}, FilterMode: 0, SpecialCount: 0})
	assert.NotContains(t, result, "selected")
}

func TestRenderGroupings_SpecialMarker(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30, GroupedByFrom: false},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 1})

	assert.Contains(t, result, "*")
	assert.Contains(t, result, "1 special")
}

func TestRenderGroupings_FilterAll_WithSpecials(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 2})

	assert.Contains(t, result, "3 groupings in inbox")
	assert.Contains(t, result, "(2 special)")
}

func TestRenderGroupings_FilterAll_NoSpecials(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "user1@example.com", Count: 30},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})

	assert.Contains(t, result, "1 groupings in inbox")
	assert.NotContains(t, result, "special")
}

func TestRenderGroupings_WithSubject(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "user1@example.com", Count: 50},
		{Address: "user2@example.com", Count: 1, Subject: "Weekly Newsletter"},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})

	assert.Contains(t, result, "Weekly Newsletter")
	assert.Contains(t, result, "user1@example.com")
	assert.Contains(t, result, "user2@example.com")
}

func TestRenderGroupings_SubjectTruncation(t *testing.T) {
	longSubject := strings.Repeat("a", 200)
	groupings := []*triage.Grouping{
		{Address: "user1@example.com", Count: 1, Subject: longSubject},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})

	assert.Contains(t, result, "...")
	assert.NotContains(t, result, longSubject)
}

func TestRenderGroupings_NoSubjectPadding_WhenNoSingleEmails(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "user1@example.com", Count: 50},
		{Address: "user2@example.com", Count: 30},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 0, SpecialCount: 0})

	assert.Contains(t, result, "user1@example.com")
	assert.Contains(t, result, "user2@example.com")
	// No subject text should appear
	assert.NotContains(t, result, "...")
}

func TestRenderGroupings_FilterSpecial(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 80, Height: 24, Checked: nil, FilterMode: 1, SpecialCount: 2})

	assert.Contains(t, result, "2 special groupings (grouped by sender)")
	assert.Contains(t, result, "f show all")
}

func previewGrouping(address string, n int, now time.Time) *triage.Grouping {
	g := &triage.Grouping{Address: address, Count: n}
	for i := 0; i < n; i++ {
		g.Messages = append(g.Messages, &imap.Message{
			UID:     uint32(i + 1),
			Subject: fmt.Sprintf("Subject %02d", i),
			Date:    now.Add(-time.Duration(i) * 24 * time.Hour),
		})
	}
	return g
}

func TestRenderGroupings_ExpandsOnlyCursorRow(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	groupings := []*triage.Grouping{
		previewGrouping("first@example.com", 3, now),
		previewGrouping("second@example.com", 3, now.Add(-time.Hour)),
	}
	// Give the second grouping distinct subjects.
	for _, msg := range groupings[1].Messages {
		msg.Subject = "Other " + msg.Subject
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 0, Width: 100, Height: 40, Now: now})

	assert.Contains(t, result, "Subject 00")
	assert.Contains(t, result, "Subject 02")
	assert.NotContains(t, result, "Other")
	// Dates: today shows the time, earlier days the month and day.
	assert.Contains(t, result, "12:00 PM")
	assert.Contains(t, result, "Nov 18")
	// Three messages fit, so no scroll hint.
	assert.NotContains(t, result, "more")
}

func TestRenderGroupings_PreviewScrollHints(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	groupings := []*triage.Grouping{previewGrouping("a@example.com", 9, now)}

	top := RenderGroupings(GroupingsView{Groupings: groupings, Width: 100, Height: 40, Now: now})
	assert.Contains(t, top, "Subject 04")
	assert.NotContains(t, top, "Subject 05")
	assert.Contains(t, top, "↓ 4 more")
	assert.NotContains(t, top, "above")
	assert.Contains(t, top, "(J/K scroll)")

	scrolled := RenderGroupings(GroupingsView{Groupings: groupings, Width: 100, Height: 40, Now: now, PreviewOffset: 2})
	assert.NotContains(t, scrolled, "Subject 01")
	assert.Contains(t, scrolled, "Subject 02")
	assert.Contains(t, scrolled, "Subject 06")
	assert.Contains(t, scrolled, "↑ 2 above")
	assert.Contains(t, scrolled, "↓ 2 more")

	// Offsets past the end are clamped.
	clamped := RenderGroupings(GroupingsView{Groupings: groupings, Width: 100, Height: 40, Now: now, PreviewOffset: 99})
	assert.Contains(t, clamped, "Subject 08")
	assert.Contains(t, clamped, "↑ 4 above")
	assert.NotContains(t, clamped, "more")
}

func TestRenderGroupings_SingleMessageDoesNotExpand(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	g := previewGrouping("one@example.com", 1, now)
	g.Subject = "Only subject"
	g.Messages[0].Subject = "Only subject"

	result := RenderGroupings(GroupingsView{Groupings: []*triage.Grouping{g}, Width: 120, Height: 40, Now: now})

	assert.Equal(t, 1, strings.Count(result, "Only subject"), "the inline subject is not repeated in a preview")
}

func TestRenderGroupings_ShowsDestinationFolder(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "notifications@example.com", Count: 2},
		{Address: "billing@example.com", Count: 1, Subject: "Invoice"},
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Width: 120, Height: 40})

	assert.Contains(t, result, "automated/notifications")
	assert.Contains(t, result, "automated/billing")
	assert.Contains(t, result, "Invoice")
}

func TestRenderGroupings_Status(t *testing.T) {
	groupings := []*triage.Grouping{{Address: "a@example.com", Count: 1}}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Width: 80, Height: 24, Status: "Could not open Gmail: boom"})

	assert.Contains(t, result, "Could not open Gmail: boom")
}

func TestRenderGroupings_HelpMentionsNewKeys(t *testing.T) {
	result := RenderGroupings(GroupingsView{Width: 80, Height: 24})
	assert.Contains(t, result, "J/K scroll subjects")
	assert.Contains(t, result, "g open in Gmail")
}

func TestRenderGroupings_CursorStaysVisibleWithPreview(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	var groupings []*triage.Grouping
	for i := 0; i < 30; i++ {
		groupings = append(groupings, previewGrouping(fmt.Sprintf("user%02d@example.com", i), 8, now))
	}

	result := RenderGroupings(GroupingsView{Groupings: groupings, Cursor: 29, Width: 100, Height: 24, Now: now})

	assert.Contains(t, result, "user29@example.com")
	assert.Contains(t, result, "Subject 00")
	assert.LessOrEqual(t, strings.Count(result, "\n"), 24, "output fits the terminal height")
}

func TestRenderGroupings_TruncatesMultibyteSafely(t *testing.T) {
	g := &triage.Grouping{Address: "a@example.com", Count: 1, Subject: strings.Repeat("é", 200)}

	result := RenderGroupings(GroupingsView{Groupings: []*triage.Grouping{g}, Width: 80, Height: 24})

	assert.True(t, utf8.ValidString(result))
	assert.Contains(t, result, "...")
}

func TestMaxPreviewOffset(t *testing.T) {
	now := time.Now()
	assert.Equal(t, 0, MaxPreviewOffset(nil))
	assert.Equal(t, 0, MaxPreviewOffset(previewGrouping("a@example.com", 1, now)))
	assert.Equal(t, 0, MaxPreviewOffset(previewGrouping("a@example.com", PreviewLines, now)))
	assert.Equal(t, 3, MaxPreviewOffset(previewGrouping("a@example.com", PreviewLines+3, now)))
}
