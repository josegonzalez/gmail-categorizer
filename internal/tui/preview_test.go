package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
	"github.com/josegonzalez/gmail-categorizer/internal/tui/views"
)

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// groupingWithMessages builds a grouping whose envelopes are newest first,
// as LoadGroupings returns them.
func groupingWithMessages(address string, n int) *triage.Grouping {
	now := time.Now()
	g := &triage.Grouping{Address: address, Count: n}
	for i := 0; i < n; i++ {
		uid := uint32(i + 1)
		g.UIDs = append(g.UIDs, uid)
		g.Messages = append(g.Messages, &imap.Message{
			UID:     uid,
			Subject: fmt.Sprintf("Subject %02d", i),
			Date:    now.Add(-time.Duration(i) * time.Hour),
		})
	}
	return g
}

func groupingsModel(groupings ...*triage.Grouping) Model {
	m := NewModel(context.Background(), &mockTriager{})
	m.view = ViewGroupings
	m.groupings = groupings
	m.width = 100
	m.height = 40
	return m
}

func TestGroupingsKeys_ScrollPreview(t *testing.T) {
	m := groupingsModel(groupingWithMessages("a@example.com", 8))

	newModel, _ := m.handleGroupingsKeys(runeKey('J'))
	m = newModel.(Model)
	assert.Equal(t, 1, m.previewOffset)

	// Clamped at len(messages) - PreviewLines.
	for i := 0; i < 10; i++ {
		newModel, _ = m.handleGroupingsKeys(runeKey('J'))
		m = newModel.(Model)
	}
	assert.Equal(t, 8-views.PreviewLines, m.previewOffset)

	newModel, _ = m.handleGroupingsKeys(runeKey('K'))
	m = newModel.(Model)
	assert.Equal(t, 8-views.PreviewLines-1, m.previewOffset)

	for i := 0; i < 10; i++ {
		newModel, _ = m.handleGroupingsKeys(runeKey('K'))
		m = newModel.(Model)
	}
	assert.Equal(t, 0, m.previewOffset)
}

func TestGroupingsKeys_ScrollPreview_ShortListDoesNotScroll(t *testing.T) {
	m := groupingsModel(groupingWithMessages("a@example.com", 3))

	newModel, _ := m.handleGroupingsKeys(runeKey('J'))
	m = newModel.(Model)
	assert.Equal(t, 0, m.previewOffset)
}

func TestGroupingsKeys_CursorMoveResetsPreview(t *testing.T) {
	m := groupingsModel(
		groupingWithMessages("a@example.com", 8),
		groupingWithMessages("b@example.com", 8),
	)
	m.previewOffset = 2

	newModel, _ := m.handleGroupingsKeys(runeKey('j'))
	m = newModel.(Model)
	assert.Equal(t, 1, m.groupingsCursor)
	assert.Equal(t, 0, m.previewOffset)

	m.previewOffset = 2
	newModel, _ = m.handleGroupingsKeys(runeKey('k'))
	m = newModel.(Model)
	assert.Equal(t, 0, m.groupingsCursor)
	assert.Equal(t, 0, m.previewOffset)

	m.previewOffset = 2
	newModel, _ = m.handleGroupingsKeys(runeKey('f'))
	m = newModel.(Model)
	assert.Equal(t, 0, m.previewOffset)
}

func TestGroupingsKeys_OpenGmail(t *testing.T) {
	g := groupingWithMessages("user+tag@example.com", 2)
	m := groupingsModel(g).WithOptions(Options{Username: "me@example.com"})

	var opened string
	m.openURL = func(url string) error {
		opened = url
		return nil
	}

	newModel, cmd := m.handleGroupingsKeys(runeKey('g'))
	m = newModel.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, g.GmailSearchURL("me@example.com"), opened)
	assert.Contains(t, m.status, "Opened Gmail search for user+tag@example.com")
	assert.Contains(t, m.View(), "Opened Gmail search")
}

func TestGroupingsKeys_OpenGmail_Error(t *testing.T) {
	m := groupingsModel(groupingWithMessages("a@example.com", 2))
	m.openURL = func(url string) error { return errors.New("no browser") }

	newModel, _ := m.handleGroupingsKeys(runeKey('g'))
	m = newModel.(Model)

	assert.Equal(t, ViewGroupings, m.view, "a launch failure keeps the TUI running")
	assert.Contains(t, m.status, "Could not open Gmail: no browser")

	// The status clears on the next key.
	newModel, _ = m.handleGroupingsKeys(runeKey('j'))
	m = newModel.(Model)
	assert.Empty(t, m.status)
}

func TestGroupingsKeys_OpenGmail_Empty(t *testing.T) {
	m := groupingsModel()
	called := false
	m.openURL = func(url string) error {
		called = true
		return nil
	}

	m.handleGroupingsKeys(runeKey('g'))
	assert.False(t, called)
}

func TestView_Groupings_ShowsPreviewForCursorRow(t *testing.T) {
	m := groupingsModel(
		groupingWithMessages("a@example.com", 8),
		groupingWithMessages("b@example.com", 8),
	)

	out := m.View()
	assert.Contains(t, out, "Subject 00")
	assert.Contains(t, out, "Subject 04")
	assert.NotContains(t, out, "Subject 05")
	assert.Contains(t, out, "↓ 3 more")
	assert.Contains(t, out, "automated/a")
	assert.Contains(t, out, "automated/b")
}

func TestSubjectsKeys_EnterLoadsBody(t *testing.T) {
	g := groupingWithMessages("a@example.com", 3)
	var loadedUID uint32
	triager := &mockTriager{
		LoadMessageBodyFunc: func(ctx context.Context, gr *triage.Grouping, uid uint32) (*imap.MessageBody, error) {
			loadedUID = uid
			return &imap.MessageBody{Text: "line one\nline two"}, nil
		},
	}
	m := NewModel(context.Background(), triager)
	m.view = ViewSubjects
	m.selectedGrouping = g
	m.subjectsCursor = 1

	newModel, cmd := m.handleSubjectsKeys(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)
	require.NotNil(t, cmd)
	assert.Equal(t, ViewLoading, m.view)
	assert.Equal(t, "loading-body", m.loadingAction)
	assert.Contains(t, m.View(), "Loading message...")

	msg := m.loadMessageBody()
	assert.Equal(t, g.Messages[1].UID, loadedUID)

	newModel, _ = m.Update(msg)
	m = newModel.(Model)
	assert.Equal(t, ViewMessage, m.view)

	out := m.View()
	assert.Contains(t, out, "Subject 01")
	assert.Contains(t, out, "line one")
	assert.Contains(t, out, "line two")
}

func TestSubjectsKeys_EnterBodyError(t *testing.T) {
	triager := &mockTriager{
		LoadMessageBodyFunc: func(ctx context.Context, g *triage.Grouping, uid uint32) (*imap.MessageBody, error) {
			return nil, errors.New("fetch failed")
		},
	}
	m := NewModel(context.Background(), triager)
	m.view = ViewSubjects
	m.selectedGrouping = groupingWithMessages("a@example.com", 1)

	newModel, _ := m.handleSubjectsKeys(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	newModel, _ = m.Update(m.loadMessageBody())
	m = newModel.(Model)
	assert.Equal(t, ViewError, m.view)
	assert.EqualError(t, m.err, "fetch failed")
}

func TestLoadMessageBody_NoSelection(t *testing.T) {
	m := NewModel(context.Background(), &mockTriager{})
	msg := m.loadMessageBody()
	_, ok := msg.(errMsg)
	assert.True(t, ok)
}

func TestSubjectsKeys_BackRestoresNewestFirst(t *testing.T) {
	g := groupingWithMessages("a@example.com", 3)
	m := NewModel(context.Background(), &mockTriager{})
	m.view = ViewSubjects
	m.selectedGrouping = g

	// Sort oldest first in the subjects view, then go back.
	newModel, _ := m.handleSubjectsKeys(runeKey('s'))
	m = newModel.(Model)
	require.Equal(t, "Subject 02", g.Messages[0].Subject)

	newModel, _ = m.handleSubjectsKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	assert.Equal(t, ViewGroupings, m.view)
	assert.Equal(t, "Subject 00", g.Messages[0].Subject)
	assert.Equal(t, "Subject 02", g.Messages[2].Subject)
}

func messageViewModel(t *testing.T, text string) Model {
	t.Helper()
	g := groupingWithMessages("a@example.com", 1)
	m := NewModel(context.Background(), &mockTriager{})
	m.width = 80
	m.height = 20
	m.selectedGrouping = g
	m.selectedMessage = g.Messages[0]
	newModel, _ := m.Update(bodyLoadedMsg{body: &imap.MessageBody{Text: text}})
	return newModel.(Model)
}

func TestMessageKeys_Scroll(t *testing.T) {
	lines := make([]string, 60)
	for i := range lines {
		lines[i] = fmt.Sprintf("body line %d", i)
	}
	m := messageViewModel(t, strings.Join(lines, "\n"))
	require.Equal(t, ViewMessage, m.view)
	assert.Equal(t, 20-views.MessageHeaderLines, m.messageViewport.Height)
	assert.Equal(t, 0, m.messageViewport.YOffset)

	newModel, _ := m.handleMessageKeys(runeKey('j'))
	m = newModel.(Model)
	assert.Equal(t, 1, m.messageViewport.YOffset)

	newModel, _ = m.handleMessageKeys(tea.KeyMsg{Type: tea.KeyPgDown})
	m = newModel.(Model)
	assert.Greater(t, m.messageViewport.YOffset, 1)

	newModel, _ = m.handleMessageKeys(runeKey('k'))
	m = newModel.(Model)
	assert.Equal(t, ViewMessage, m.view)
}

func TestMessageKeys_EscBack(t *testing.T) {
	m := messageViewModel(t, "hello")

	newModel, _ := m.handleMessageKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	assert.Equal(t, ViewSubjects, m.view)
	assert.Nil(t, m.selectedMessage)
}

func TestMessageKeys_ArchiveConfirmAndCancel(t *testing.T) {
	m := messageViewModel(t, "hello")

	newModel, _ := m.handleMessageKeys(runeKey('a'))
	m = newModel.(Model)
	assert.Equal(t, ViewConfirm, m.view)
	assert.Equal(t, "archive", m.confirmAction)

	newModel, _ = m.handleConfirmKeys(runeKey('n'))
	m = newModel.(Model)
	assert.Equal(t, ViewMessage, m.view)
}

func TestMessageKeys_Quit(t *testing.T) {
	m := messageViewModel(t, "hello")

	newModel, cmd := m.handleMessageKeys(runeKey('q'))
	m = newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestMessageView_Resize(t *testing.T) {
	m := messageViewModel(t, "hello")

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = newModel.(Model)

	assert.Equal(t, 120, m.messageViewport.Width)
	assert.Equal(t, 50-views.MessageHeaderLines, m.messageViewport.Height)
}

func TestMessageView_HTMLBodyShownAsText(t *testing.T) {
	g := groupingWithMessages("a@example.com", 1)
	m := NewModel(context.Background(), &mockTriager{})
	m.selectedGrouping = g
	m.selectedMessage = g.Messages[0]

	newModel, _ := m.Update(bodyLoadedMsg{body: &imap.MessageBody{HTML: "<p>Hello &amp; welcome</p><script>x()</script>"}})
	m = newModel.(Model)

	out := m.View()
	assert.Contains(t, out, "Hello & welcome")
	assert.NotContains(t, out, "x()")
}

func TestWithOptions(t *testing.T) {
	m := NewModel(context.Background(), &mockTriager{}).WithOptions(Options{Username: "me@example.com"})
	assert.Equal(t, "me@example.com", m.username)
	assert.NotNil(t, m.openURL)
}
