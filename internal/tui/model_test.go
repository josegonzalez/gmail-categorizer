package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

func TestNewModel(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}

	model := NewModel(ctx, triager)

	assert.Equal(t, ViewLoading, model.view)
	assert.NotNil(t, model.keys)
	assert.NotNil(t, model.spinner)
	assert.False(t, model.quitting)
	assert.Equal(t, 0, model.groupingsCursor)
	assert.Nil(t, model.groupings)
	assert.NotNil(t, model.checkedGroupings)
	assert.Empty(t, model.checkedGroupings)
}

func TestModel_Init(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{
		LoadGroupingsFunc: func(ctx context.Context) ([]*triage.Grouping, error) {
			return []*triage.Grouping{}, nil
		},
	}

	model := NewModel(ctx, triager)
	cmd := model.Init()

	// Init should return a batch command
	require.NotNil(t, cmd)
}

func TestModel_Update_WindowSizeMsg(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
	assert.Nil(t, cmd)
}

func TestModel_Update_KeyMsg_Quit_Loading(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd) // Should be tea.Quit
}

func TestModel_Update_KeyMsg_CtrlC_Loading(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_Update_GroupingsLoadedMsg(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)

	groupings := []*triage.Grouping{
		{Address: "test@example.com", Count: 5},
	}
	msg := groupingsLoadedMsg{groupings: groupings}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
	assert.Equal(t, groupings, m.groupings)
	assert.Nil(t, cmd)
}

func TestModel_Update_GroupingsLoadedMsg_Empty(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)

	msg := groupingsLoadedMsg{groupings: []*triage.Grouping{}}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewError, m.view)
	assert.NotNil(t, m.err)
	assert.Contains(t, m.err.Error(), "no messages")
	assert.Nil(t, cmd)
}

func TestModel_Update_GroupingsLoadedMsg_ClearsChecked(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.checkedGroupings = map[int]bool{0: true, 1: true}

	groupings := []*triage.Grouping{
		{Address: "test@example.com", Count: 5},
	}
	msg := groupingsLoadedMsg{groupings: groupings}
	newModel, _ := model.Update(msg)

	m := newModel.(Model)
	assert.Empty(t, m.checkedGroupings)
}

func TestModel_Update_MessagesLoadedMsg(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading

	msg := messagesLoadedMsg{}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewSubjects, m.view)
	assert.Nil(t, cmd)
}

func TestModel_Update_ArchiveResultMsg(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)

	result := &triage.TriageResult{ArchivedCount: 10, DestinationFolder: "automated/test"}
	msg := archiveResultMsg{result: result}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewResult, m.view)
	assert.Equal(t, result, m.result)
	assert.Nil(t, cmd)
}

func TestModel_Update_BatchArchiveResultMsg(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)

	results := []batchArchiveEntry{
		{Address: "a@test.com", Result: &triage.TriageResult{ArchivedCount: 5, DestinationFolder: "automated/a"}},
		{Address: "b@test.com", Result: &triage.TriageResult{ArchivedCount: 3, DestinationFolder: "automated/b"}},
	}
	msg := batchArchiveResultMsg{results: results}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewResult, m.view)
	assert.Equal(t, results, m.archiveResults)
	assert.Nil(t, cmd)
}

func TestModel_Update_ErrMsg(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)

	testErr := errors.New("test error")
	msg := errMsg{err: testErr}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewError, m.view)
	assert.Equal(t, testErr, m.err)
	assert.Nil(t, cmd)
}

func TestModel_Update_SpinnerTickMsg_Loading(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading

	msg := spinner.TickMsg{}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	// Spinner update should return a command
	assert.NotNil(t, cmd)
}

func TestModel_Update_SpinnerTickMsg_NotLoading(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings

	msg := spinner.TickMsg{}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
	assert.Nil(t, cmd)
}

func TestModel_HandleGroupingsKeys_Navigation(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 1},
		{Address: "b@test.com", Count: 2},
		{Address: "c@test.com", Count: 3},
	}
	model.groupingsCursor = 1

	// Test down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ := model.handleGroupingsKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, 2, m.groupingsCursor)

	// Test up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, _ = m.handleGroupingsKeys(msg)
	m = newModel.(Model)
	assert.Equal(t, 1, m.groupingsCursor)
}

func TestModel_HandleGroupingsKeys_Quit(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := model.handleGroupingsKeys(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_HandleGroupingsKeys_Enter(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "test@example.com", Count: 5},
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.handleGroupingsKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Equal(t, model.groupings[0], m.selectedGrouping)
	assert.NotNil(t, cmd)
}

func TestModel_HandleGroupingsKeys_Archive(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "test@example.com", Count: 5},
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := model.handleGroupingsKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewConfirm, m.view)
	assert.Equal(t, model.groupings[0], m.selectedGrouping)
	assert.Equal(t, "archive", m.confirmAction)
}

func TestModel_HandleGroupingsKeys_Toggle(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 1},
		{Address: "b@test.com", Count: 2},
	}
	model.groupingsCursor = 0

	// Toggle on
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	newModel, _ := model.handleGroupingsKeys(msg)
	m := newModel.(Model)
	assert.True(t, m.checkedGroupings[0])
	assert.Equal(t, 0, m.groupingsCursor) // Cursor stays

	// Toggle off
	newModel, _ = m.handleGroupingsKeys(msg)
	m = newModel.(Model)
	assert.False(t, m.checkedGroupings[0])
	assert.Equal(t, 0, m.groupingsCursor)
}

func TestModel_HandleGroupingsKeys_Toggle_EmptyGroupings(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	newModel, _ := model.handleGroupingsKeys(msg)
	m := newModel.(Model)
	assert.Empty(t, m.checkedGroupings)
}

func TestModel_HandleGroupingsKeys_BatchArchive(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 5},
		{Address: "b@test.com", Count: 3},
		{Address: "c@test.com", Count: 2},
	}
	model.checkedGroupings = map[int]bool{0: true, 2: true}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := model.handleGroupingsKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewConfirm, m.view)
	assert.Equal(t, "batch-archive", m.confirmAction)
	// Should NOT set selectedGrouping for batch
	assert.Nil(t, m.selectedGrouping)
}

func TestModel_HandleSubjectsKeys_Navigation(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{
		Messages: []*imap.Message{
			{Subject: "A"},
			{Subject: "B"},
			{Subject: "C"},
		},
	}
	model.subjectsCursor = 1

	// Test down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ := model.handleSubjectsKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, 2, m.subjectsCursor)

	// Test up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, _ = m.handleSubjectsKeys(msg)
	m = newModel.(Model)
	assert.Equal(t, 1, m.subjectsCursor)
}

func TestModel_HandleSubjectsKeys_Back(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := model.handleSubjectsKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
	assert.Nil(t, m.selectedGrouping)
}

func TestModel_HandleConfirmKeys_Yes(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "archive"
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	newModel, cmd := model.handleConfirmKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Equal(t, "archiving", m.loadingAction)
	assert.NotNil(t, cmd)
}

func TestModel_HandleConfirmKeys_Yes_BatchArchive(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "batch-archive"
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 5},
		{Address: "b@test.com", Count: 3},
	}
	model.checkedGroupings = map[int]bool{0: true, 1: true}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	newModel, cmd := model.handleConfirmKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Equal(t, "batch-archiving", m.loadingAction)
	assert.NotNil(t, cmd)
}

func TestModel_HandleConfirmKeys_No(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "archive"
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ := model.handleConfirmKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
}

func TestModel_HandleConfirmKeys_No_BatchArchive(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "batch-archive"

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ := model.handleConfirmKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
}

func TestModel_HandleConfirmKeys_No_WithMessages(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "archive"
	model.selectedGrouping = &triage.Grouping{
		Address:  "test@example.com",
		Messages: []*imap.Message{{Subject: "test"}},
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ := model.handleConfirmKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewSubjects, m.view)
}

func TestModel_HandleResultKeys_Continue(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}
	model.result = &triage.TriageResult{ArchivedCount: 5}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.handleResultKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Nil(t, m.selectedGrouping)
	assert.Nil(t, m.result)
	assert.Nil(t, m.archiveResults)
	assert.Empty(t, m.checkedGroupings)
	assert.NotNil(t, cmd)
}

func TestModel_HandleResultKeys_Continue_ClearsBatchState(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult
	model.archiveResults = []batchArchiveEntry{
		{Address: "a@test.com", Result: &triage.TriageResult{ArchivedCount: 5}},
	}
	model.checkedGroupings = map[int]bool{0: true}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.handleResultKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Nil(t, m.archiveResults)
	assert.Empty(t, m.checkedGroupings)
	assert.NotNil(t, cmd)
}

func TestModel_HandleResultKeys_Quit(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := model.handleResultKeys(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_HandleErrorKeys(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewError
	model.err = errors.New("test error")

	// Any key should quit
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.handleErrorKeys(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_View_Loading(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading

	view := model.View()
	assert.Contains(t, view, "Loading")
}

func TestModel_View_Loading_WithSelectedGrouping(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}
	model.loadingAction = "archiving"

	view := model.View()
	assert.Contains(t, view, "Archiving")
	assert.Contains(t, view, "test@example.com")
}

func TestModel_View_Loading_BatchArchiving(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading
	model.loadingAction = "batch-archiving"
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 5},
		{Address: "b@test.com", Count: 3},
		{Address: "c@test.com", Count: 2},
	}
	model.checkedGroupings = map[int]bool{0: true, 2: true}

	view := model.View()
	assert.Contains(t, view, "Archiving 2 groupings")
}

func TestModel_View_Groupings(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.width = 80
	model.height = 24
	model.groupings = []*triage.Grouping{
		{Address: "test@example.com", Count: 5},
	}

	view := model.View()
	assert.Contains(t, view, "test@example.com")
	assert.Contains(t, view, "5")
}

func TestModel_View_Subjects(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.width = 80
	model.height = 24
	model.selectedGrouping = &triage.Grouping{
		Address: "test@example.com",
		Messages: []*imap.Message{
			{Subject: "Test Subject"},
		},
	}

	view := model.View()
	assert.Contains(t, view, "Test Subject")
}

func TestModel_View_Subjects_NoGrouping(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = nil

	view := model.View()
	assert.Contains(t, view, "No grouping selected")
}

func TestModel_View_Confirm(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "archive"
	model.selectedGrouping = &triage.Grouping{
		Address: "test@example.com",
		Count:   5,
	}

	view := model.View()
	assert.Contains(t, view, "test@example.com")
	assert.Contains(t, view, "5")
}

func TestModel_View_Confirm_Batch(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "batch-archive"
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 5},
		{Address: "b@test.com", Count: 3},
	}
	model.checkedGroupings = map[int]bool{0: true, 1: true}

	view := model.View()
	assert.Contains(t, view, "Batch Archive")
	assert.Contains(t, view, "a@test.com")
	assert.Contains(t, view, "b@test.com")
}

func TestModel_View_Result(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult
	model.result = &triage.TriageResult{
		ArchivedCount:     10,
		DestinationFolder: "automated/test",
	}

	view := model.View()
	assert.Contains(t, view, "10")
	assert.Contains(t, view, "automated/test")
}

func TestModel_View_Result_Batch(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult
	model.archiveResults = []batchArchiveEntry{
		{Address: "a@test.com", Result: &triage.TriageResult{ArchivedCount: 5, DestinationFolder: "automated/a"}},
		{Address: "b@test.com", Result: &triage.TriageResult{ArchivedCount: 3, DestinationFolder: "automated/b"}},
	}

	view := model.View()
	assert.Contains(t, view, "Batch Archive Complete")
	assert.Contains(t, view, "automated/a")
	assert.Contains(t, view, "automated/b")
}

func TestModel_View_Result_Batch_WithFailure(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult
	model.archiveResults = []batchArchiveEntry{
		{Address: "a@test.com", Result: &triage.TriageResult{ArchivedCount: 5, DestinationFolder: "automated/a"}},
		{Address: "b@test.com", Err: errors.New("timeout")},
	}

	view := model.View()
	assert.Contains(t, view, "Batch Archive Complete")
	assert.Contains(t, view, "automated/a")
	assert.Contains(t, view, "timeout")
}

func TestModel_View_Error(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewError
	model.err = errors.New("something went wrong")

	view := model.View()
	assert.Contains(t, view, "something went wrong")
}

func TestModel_View_Quitting(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.quitting = true

	view := model.View()
	assert.Empty(t, view)
}

func TestModel_View_UnknownState(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewState(999) // Unknown state

	view := model.View()
	assert.Contains(t, view, "Unknown")
}

func TestModel_LoadGroupings(t *testing.T) {
	ctx := context.Background()
	groupings := []*triage.Grouping{
		{Address: "test@example.com", Count: 5},
	}
	triager := &mockTriager{
		LoadGroupingsFunc: func(ctx context.Context) ([]*triage.Grouping, error) {
			return groupings, nil
		},
	}

	model := NewModel(ctx, triager)
	msg := model.loadGroupings()

	loadedMsg, ok := msg.(groupingsLoadedMsg)
	require.True(t, ok)
	assert.Equal(t, groupings, loadedMsg.groupings)
}

func TestModel_LoadGroupings_Error(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{
		LoadGroupingsFunc: func(ctx context.Context) ([]*triage.Grouping, error) {
			return nil, errors.New("load failed")
		},
	}

	model := NewModel(ctx, triager)
	msg := model.loadGroupings()

	errMsgResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Contains(t, errMsgResult.err.Error(), "load failed")
}

func TestModel_LoadMessages(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{
		LoadMessagesFunc: func(ctx context.Context, g *triage.Grouping) error {
			g.Messages = []*imap.Message{{Subject: "Test"}}
			return nil
		},
	}

	model := NewModel(ctx, triager)
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := model.loadMessages()

	_, ok := msg.(messagesLoadedMsg)
	require.True(t, ok)
}

func TestModel_LoadMessages_NilGrouping(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}

	model := NewModel(ctx, triager)
	model.selectedGrouping = nil

	msg := model.loadMessages()

	errMsgResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Contains(t, errMsgResult.err.Error(), "no grouping selected")
}

func TestModel_LoadMessages_Error(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{
		LoadMessagesFunc: func(ctx context.Context, g *triage.Grouping) error {
			return errors.New("load failed")
		},
	}

	model := NewModel(ctx, triager)
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := model.loadMessages()

	errMsgResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Contains(t, errMsgResult.err.Error(), "load failed")
}

func TestModel_DoArchive(t *testing.T) {
	ctx := context.Background()
	result := &triage.TriageResult{ArchivedCount: 10, DestinationFolder: "automated/test"}
	triager := &mockTriager{
		ArchiveFunc: func(ctx context.Context, g *triage.Grouping) (*triage.TriageResult, error) {
			return result, nil
		},
	}

	model := NewModel(ctx, triager)
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := model.doArchive()

	archiveMsg, ok := msg.(archiveResultMsg)
	require.True(t, ok)
	assert.Equal(t, result, archiveMsg.result)
}

func TestModel_DoArchive_NilGrouping(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}

	model := NewModel(ctx, triager)
	model.selectedGrouping = nil

	msg := model.doArchive()

	errMsgResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Contains(t, errMsgResult.err.Error(), "no grouping selected")
}

func TestModel_DoArchive_Error(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{
		ArchiveFunc: func(ctx context.Context, g *triage.Grouping) (*triage.TriageResult, error) {
			return nil, errors.New("archive failed")
		},
	}

	model := NewModel(ctx, triager)
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := model.doArchive()

	errMsgResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Contains(t, errMsgResult.err.Error(), "archive failed")
}

func TestModel_CheckedGroupingsList(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 1},
		{Address: "b@test.com", Count: 2},
		{Address: "c@test.com", Count: 3},
	}
	model.checkedGroupings = map[int]bool{2: true, 0: true}

	result := model.checkedGroupingsList()

	require.Len(t, result, 2)
	// Should be in index order
	assert.Equal(t, "a@test.com", result[0].Address)
	assert.Equal(t, "c@test.com", result[1].Address)
}

func TestModel_CheckedGroupingsList_Empty(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 1},
	}

	result := model.checkedGroupingsList()
	assert.Empty(t, result)
}

func TestModel_DoBatchArchive_Success(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{
		ArchiveFunc: func(ctx context.Context, g *triage.Grouping) (*triage.TriageResult, error) {
			return &triage.TriageResult{
				ArchivedCount:     g.Count,
				DestinationFolder: g.DestinationFolder(),
			}, nil
		},
	}
	model := NewModel(ctx, triager)
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 5},
		{Address: "b@test.com", Count: 3},
	}
	model.checkedGroupings = map[int]bool{0: true, 1: true}

	msg := model.doBatchArchive()

	batchMsg, ok := msg.(batchArchiveResultMsg)
	require.True(t, ok)
	require.Len(t, batchMsg.results, 2)
	assert.Equal(t, "a@test.com", batchMsg.results[0].Address)
	assert.Equal(t, 5, batchMsg.results[0].Result.ArchivedCount)
	assert.Nil(t, batchMsg.results[0].Err)
	assert.Equal(t, "b@test.com", batchMsg.results[1].Address)
	assert.Equal(t, 3, batchMsg.results[1].Result.ArchivedCount)
	assert.Nil(t, batchMsg.results[1].Err)
}

func TestModel_DoBatchArchive_PartialFailure(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	triager := &mockTriager{
		ArchiveFunc: func(ctx context.Context, g *triage.Grouping) (*triage.TriageResult, error) {
			callCount++
			if callCount == 2 {
				return nil, errors.New("connection timeout")
			}
			return &triage.TriageResult{
				ArchivedCount:     g.Count,
				DestinationFolder: g.DestinationFolder(),
			}, nil
		},
	}
	model := NewModel(ctx, triager)
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 5},
		{Address: "b@test.com", Count: 3},
		{Address: "c@test.com", Count: 2},
	}
	model.checkedGroupings = map[int]bool{0: true, 1: true, 2: true}

	msg := model.doBatchArchive()

	batchMsg, ok := msg.(batchArchiveResultMsg)
	require.True(t, ok)
	require.Len(t, batchMsg.results, 3)

	assert.Nil(t, batchMsg.results[0].Err)
	assert.NotNil(t, batchMsg.results[1].Err)
	assert.Contains(t, batchMsg.results[1].Err.Error(), "connection timeout")
	assert.Nil(t, batchMsg.results[2].Err)
}

func TestModel_DoBatchArchive_NoneChecked(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 5},
	}

	msg := model.doBatchArchive()

	errMsgResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Contains(t, errMsgResult.err.Error(), "no groupings selected")
}

func TestModel_HandleGroupingsKeys_BoundaryNavigation(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "a@test.com", Count: 1},
		{Address: "b@test.com", Count: 2},
	}

	// At top, can't go further up
	model.groupingsCursor = 0
	msg := tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ := model.handleGroupingsKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, 0, m.groupingsCursor)

	// At bottom, can't go further down
	m.groupingsCursor = 1
	msg = tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ = m.handleGroupingsKeys(msg)
	m = newModel.(Model)
	assert.Equal(t, 1, m.groupingsCursor)
}

func TestModel_HandleSubjectsKeys_BoundaryNavigation(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{
		Messages: []*imap.Message{
			{Subject: "A"},
			{Subject: "B"},
		},
	}

	// At top, can't go further up
	model.subjectsCursor = 0
	msg := tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ := model.handleSubjectsKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, 0, m.subjectsCursor)

	// At bottom, can't go further down
	m.subjectsCursor = 1
	msg = tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ = m.handleSubjectsKeys(msg)
	m = newModel.(Model)
	assert.Equal(t, 1, m.subjectsCursor)
}

func TestModel_HandleSubjectsKeys_Quit(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{
		Messages: []*imap.Message{{Subject: "Test"}},
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := model.handleSubjectsKeys(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_HandleSubjectsKeys_Archive(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{
		Address:  "test@example.com",
		Messages: []*imap.Message{{Subject: "Test"}},
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := model.handleSubjectsKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewConfirm, m.view)
	assert.Equal(t, "archive", m.confirmAction)
}

func TestModel_HandleSubjectsKeys_BackWithBackspace(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ := model.handleSubjectsKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
	assert.Nil(t, m.selectedGrouping)
}

func TestModel_Update_KeyMsg_NonLoading(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "test@example.com", Count: 5},
	}

	// Test that Update properly dispatches to handleKeyPress
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_HandleResultKeys_Esc(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}
	model.result = &triage.TriageResult{ArchivedCount: 5}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, cmd := model.handleResultKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Nil(t, m.selectedGrouping)
	assert.Nil(t, m.result)
	assert.NotNil(t, cmd)
}

func TestModel_HandleErrorKeys_Q(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewError
	model.err = errors.New("test error")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := model.handleErrorKeys(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_HandleErrorKeys_CtrlC(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewError
	model.err = errors.New("test error")

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := model.handleErrorKeys(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_HandleErrorKeys_OtherKey(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewError
	model.err = errors.New("test error")

	// Non-quit key should also quit from error view
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	newModel, _ := model.handleErrorKeys(msg)

	m := newModel.(Model)
	// Any key other than q/ctrl+c/enter doesn't quit
	assert.False(t, m.quitting)
}

func TestModel_HandleKeyPress_DefaultCase(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading // A view that doesn't have a specific handler in handleKeyPress switch

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	newModel, cmd := model.handleKeyPress(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Nil(t, cmd)
}

func TestModel_HandleGroupingsKeys_CtrlC(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := model.handleGroupingsKeys(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_HandleSubjectsKeys_CtrlC(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{
		Messages: []*imap.Message{{Subject: "Test"}},
	}

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := model.handleSubjectsKeys(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_HandleResultKeys_CtrlC(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := model.handleResultKeys(msg)

	m := newModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)
}

func TestModel_HandleConfirmKeys_Enter(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "archive"
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.handleConfirmKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Equal(t, "archiving", m.loadingAction)
	assert.NotNil(t, cmd)
}

func TestModel_HandleConfirmKeys_Esc(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "archive"
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := model.handleConfirmKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
}

func TestModel_HandleGroupingsKeys_EmptyGroupings(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{}

	// Enter on empty groupings should do nothing
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.handleGroupingsKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)

	// Archive on empty groupings should do nothing
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ = m.handleGroupingsKeys(msg)

	m = newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
}

func TestModel_View_Loading_WithLoadingAction(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}
	model.loadingAction = "loading"

	view := model.View()
	assert.Contains(t, view, "Loading")
	assert.Contains(t, view, "test@example.com")
}

func TestModel_HandleSubjectsKeys_NilGrouping(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = nil

	// Down with nil grouping should do nothing
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.handleSubjectsKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, 0, m.subjectsCursor)
}

// Integration tests that go through Update to exercise handleKeyPress
func TestModel_Update_GroupingsKeyPress(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "test@example.com", Count: 5},
	}

	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(msg)

	// Should have processed through handleKeyPress -> handleGroupingsKeys
	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
}

func TestModel_Update_SubjectsKeyPress(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{
		Address:  "test@example.com",
		Messages: []*imap.Message{{Subject: "Test"}},
	}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := model.Update(msg)

	// Should have processed through handleKeyPress -> handleSubjectsKeys
	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
}

func TestModel_Update_ConfirmKeyPress(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewConfirm
	model.confirmAction = "archive"
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ := model.Update(msg)

	// Should have processed through handleKeyPress -> handleConfirmKeys
	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
}

func TestModel_Update_ResultKeyPress(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult
	model.result = &triage.TriageResult{ArchivedCount: 5}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, _ := model.Update(msg)

	// Should have processed through handleKeyPress -> handleResultKeys
	m := newModel.(Model)
	assert.True(t, m.quitting)
}

func TestModel_Update_ErrorKeyPress(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewError
	model.err = errors.New("test error")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.Update(msg)

	// Should have processed through handleKeyPress -> handleErrorKeys
	m := newModel.(Model)
	assert.True(t, m.quitting)
}

func TestModel_Update_LoadingOtherKey(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading

	// Non-quit key during loading should be ignored
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	newModel, cmd := model.Update(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.False(t, m.quitting)
	assert.Nil(t, cmd)
}

func TestModel_HandleSubjectsKeys_Sort(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{
		Messages: []*imap.Message{
			{Subject: "B"},
			{Subject: "A"},
		},
	}

	// First press: cycle from SortDateDesc to SortDateAsc
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	newModel, _ := model.handleSubjectsKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, SortDateAsc, m.subjectsSortMode)
	assert.Equal(t, 0, m.subjectsCursor)

	// Second press: SortSubjectAsc
	newModel, _ = m.handleSubjectsKeys(msg)
	m = newModel.(Model)
	assert.Equal(t, SortSubjectAsc, m.subjectsSortMode)

	// Third press: SortSubjectDesc
	newModel, _ = m.handleSubjectsKeys(msg)
	m = newModel.(Model)
	assert.Equal(t, SortSubjectDesc, m.subjectsSortMode)

	// Fourth press: wraps back to SortDateDesc
	newModel, _ = m.handleSubjectsKeys(msg)
	m = newModel.(Model)
	assert.Equal(t, SortDateDesc, m.subjectsSortMode)
}

func TestModel_SortModeDefaultOnMessageLoad(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewLoading
	model.selectedGrouping = &triage.Grouping{
		Messages: []*imap.Message{
			{Subject: "Test"},
		},
	}

	msg := messagesLoadedMsg{}
	newModel, _ := model.Update(msg)
	m := newModel.(Model)
	assert.Equal(t, SortDateDesc, m.subjectsSortMode)
}

func TestModel_SortModeResetOnBack(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewSubjects
	model.selectedGrouping = &triage.Grouping{Address: "test@example.com"}
	model.subjectsSortMode = SortSubjectAsc

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := model.handleSubjectsKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, ViewGroupings, m.view)
	assert.Equal(t, SortDateDesc, m.subjectsSortMode)
}

func TestModel_Update_DefaultMsgType(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)

	// Unknown message type should return model unchanged
	type unknownMsg struct{}
	newModel, cmd := model.Update(unknownMsg{})

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Nil(t, cmd)
}

func TestModel_HandleGroupingsKeys_Filter_Toggle(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}
	model.groupingsCursor = 1
	model.checkedGroupings = map[int]bool{0: true}

	// Press f to toggle to special filter
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	newModel, _ := model.handleGroupingsKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, FilterSpecial, m.groupingFilter)
	assert.Equal(t, 0, m.groupingsCursor)
	assert.Empty(t, m.checkedGroupings)

	// Press f again to toggle back to all
	newModel, _ = m.handleGroupingsKeys(msg)
	m = newModel.(Model)
	assert.Equal(t, FilterAll, m.groupingFilter)
	assert.Equal(t, 0, m.groupingsCursor)
}

func TestModel_FilteredGroupings(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.groupings = []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}

	// FilterAll returns all
	model.groupingFilter = FilterAll
	filtered := model.filteredGroupings()
	assert.Len(t, filtered, 3)

	// FilterSpecial returns only GroupedByFrom
	model.groupingFilter = FilterSpecial
	filtered = model.filteredGroupings()
	assert.Len(t, filtered, 2)
	assert.Equal(t, "admin@company.com", filtered[0].Address)
	assert.Equal(t, "hi@startup.io", filtered[1].Address)
}

func TestModel_FilteredGroupingIndices(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.groupings = []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}

	// FilterAll returns sequential indices
	model.groupingFilter = FilterAll
	indices := model.filteredGroupingIndices()
	assert.Equal(t, []int{0, 1, 2}, indices)

	// FilterSpecial returns only indices of GroupedByFrom
	model.groupingFilter = FilterSpecial
	indices = model.filteredGroupingIndices()
	assert.Equal(t, []int{0, 2}, indices)
}

func TestModel_HandleGroupingsKeys_NavigationRespectsFilter(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}
	model.groupingFilter = FilterSpecial
	model.groupingsCursor = 0

	// Down should work within filtered bounds (2 items)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ := model.handleGroupingsKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, 1, m.groupingsCursor)

	// Down again should NOT go beyond filtered length
	newModel, _ = m.handleGroupingsKeys(msg)
	m = newModel.(Model)
	assert.Equal(t, 1, m.groupingsCursor)
}

func TestModel_HandleGroupingsKeys_EnterWithFilter(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}
	model.groupingFilter = FilterSpecial
	model.groupingsCursor = 1 // Second filtered item = hi@startup.io (real index 2)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.handleGroupingsKeys(msg)

	m := newModel.(Model)
	assert.Equal(t, ViewLoading, m.view)
	assert.Equal(t, "hi@startup.io", m.selectedGrouping.Address)
	assert.NotNil(t, cmd)
}

func TestModel_HandleGroupingsKeys_ToggleWithFilter(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}
	model.groupingFilter = FilterSpecial
	model.groupingsCursor = 1 // Second filtered item = real index 2

	// Toggle on
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	newModel, _ := model.handleGroupingsKeys(msg)
	m := newModel.(Model)
	assert.True(t, m.checkedGroupings[2])  // Real index 2
	assert.False(t, m.checkedGroupings[1]) // Real index 1 not affected
}

func TestModel_HandleGroupingsKeys_ArchiveWithFilter(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.groupings = []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}
	model.groupingFilter = FilterSpecial
	model.groupingsCursor = 1 // Second filtered item = real index 2

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := model.handleGroupingsKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, ViewConfirm, m.view)
	assert.Equal(t, "archive", m.confirmAction)
	assert.Equal(t, "hi@startup.io", m.selectedGrouping.Address)
}

func TestModel_View_GroupingsWithFilter(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewGroupings
	model.width = 80
	model.height = 24
	model.groupings = []*triage.Grouping{
		{Address: "admin@company.com", Count: 42, GroupedByFrom: true},
		{Address: "user1@example.com", Count: 30},
		{Address: "hi@startup.io", Count: 15, GroupedByFrom: true},
	}
	model.groupingFilter = FilterSpecial

	view := model.View()
	assert.Contains(t, view, "admin@company.com")
	assert.Contains(t, view, "hi@startup.io")
	assert.NotContains(t, view, "user1@example.com")
	assert.Contains(t, view, "special groupings")
}

func TestModel_FilterPersistsAfterResult(t *testing.T) {
	ctx := context.Background()
	triager := &mockTriager{}
	model := NewModel(ctx, triager)
	model.view = ViewResult
	model.groupingFilter = FilterSpecial
	model.result = &triage.TriageResult{ArchivedCount: 5}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.handleResultKeys(msg)
	m := newModel.(Model)
	assert.Equal(t, FilterSpecial, m.groupingFilter)
}
