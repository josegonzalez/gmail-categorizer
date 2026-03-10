package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
	"github.com/josegonzalez/gmail-categorizer/internal/tui/views"
)

// ViewState represents the current view in the TUI.
type ViewState int

const (
	ViewLoading ViewState = iota
	ViewGroupings
	ViewSubjects
	ViewConfirm
	ViewResult
	ViewError
)

// Model represents the main TUI application state.
type Model struct {
	triager  triage.Triager
	ctx      context.Context
	keys     KeyMap
	width    int
	height   int
	view     ViewState
	err      error
	spinner  spinner.Model
	quitting bool

	// Groupings view
	groupings       []*triage.Grouping
	groupingsCursor int

	// Subjects view
	selectedGrouping *triage.Grouping
	subjectsCursor   int
	subjectsOffset   int
	subjectsSortMode SortMode

	// Confirm view
	confirmAction string

	// Loading view
	loadingAction string

	// Result view
	result *triage.TriageResult
}

// NewModel creates a new TUI model.
func NewModel(ctx context.Context, triager triage.Triager) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return Model{
		triager: triager,
		ctx:     ctx,
		keys:    DefaultKeyMap(),
		view:    ViewLoading,
		spinner: s,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadGroupings,
	)
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.view == ViewLoading {
			// Only allow quit during loading
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
		return m.handleKeyPress(msg)

	case groupingsLoadedMsg:
		m.groupings = msg.groupings
		m.view = ViewGroupings
		if len(m.groupings) == 0 {
			m.err = fmt.Errorf("no messages in inbox")
			m.view = ViewError
		}
		return m, nil

	case messagesLoadedMsg:
		m.view = ViewSubjects
		if m.selectedGrouping != nil {
			sortMessages(m.selectedGrouping.Messages, m.subjectsSortMode)
		}
		return m, nil

	case archiveResultMsg:
		m.result = msg.result
		m.view = ViewResult
		return m, nil

	case errMsg:
		m.err = msg.err
		m.view = ViewError
		return m, nil

	case spinner.TickMsg:
		if m.view == ViewLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles key presses based on current view.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewGroupings:
		return m.handleGroupingsKeys(msg)
	case ViewSubjects:
		return m.handleSubjectsKeys(msg)
	case ViewConfirm:
		return m.handleConfirmKeys(msg)
	case ViewResult:
		return m.handleResultKeys(msg)
	case ViewError:
		return m.handleErrorKeys(msg)
	}
	return m, nil
}

// handleGroupingsKeys handles keys in the groupings view.
func (m Model) handleGroupingsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case msg.String() == "up" || msg.String() == "k":
		if m.groupingsCursor > 0 {
			m.groupingsCursor--
		}

	case msg.String() == "down" || msg.String() == "j":
		if m.groupingsCursor < len(m.groupings)-1 {
			m.groupingsCursor++
		}

	case msg.String() == "enter":
		if len(m.groupings) > 0 {
			m.selectedGrouping = m.groupings[m.groupingsCursor]
			m.subjectsCursor = 0
			m.subjectsOffset = 0
			m.view = ViewLoading
			m.loadingAction = "loading"
			return m, tea.Batch(m.spinner.Tick, m.loadMessages)
		}

	case msg.String() == "a":
		if len(m.groupings) > 0 {
			m.selectedGrouping = m.groupings[m.groupingsCursor]
			m.confirmAction = "archive"
			m.view = ViewConfirm
		}
	}

	return m, nil
}

// handleSubjectsKeys handles keys in the subjects view.
func (m Model) handleSubjectsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case msg.String() == "esc" || msg.String() == "backspace":
		m.view = ViewGroupings
		m.selectedGrouping = nil
		m.subjectsSortMode = SortDateDesc

	case msg.String() == "up" || msg.String() == "k":
		if m.subjectsCursor > 0 {
			m.subjectsCursor--
		}

	case msg.String() == "down" || msg.String() == "j":
		if m.selectedGrouping != nil && m.subjectsCursor < len(m.selectedGrouping.Messages)-1 {
			m.subjectsCursor++
		}

	case msg.String() == "s":
		m.subjectsSortMode = (m.subjectsSortMode + 1) % 4
		if m.selectedGrouping != nil {
			sortMessages(m.selectedGrouping.Messages, m.subjectsSortMode)
			m.subjectsCursor = 0
		}

	case msg.String() == "a":
		m.confirmAction = "archive"
		m.view = ViewConfirm
	}

	return m, nil
}

// handleConfirmKeys handles keys in the confirm view.
func (m Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "y" || msg.String() == "enter":
		m.view = ViewLoading
		m.loadingAction = "archiving"
		return m, tea.Batch(m.spinner.Tick, m.doArchive)

	case msg.String() == "n" || msg.String() == "esc":
		// Go back to previous view
		if m.selectedGrouping != nil && len(m.selectedGrouping.Messages) > 0 {
			m.view = ViewSubjects
		} else {
			m.view = ViewGroupings
		}
	}

	return m, nil
}

// handleResultKeys handles keys in the result view.
func (m Model) handleResultKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case msg.String() == "enter" || msg.String() == "esc":
		// Reset and reload groupings
		m.selectedGrouping = nil
		m.result = nil
		m.view = ViewLoading
		m.loadingAction = ""
		return m, tea.Batch(m.spinner.Tick, m.loadGroupings)
	}

	return m, nil
}

// handleErrorKeys handles keys in the error view.
func (m Model) handleErrorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "enter":
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// Message types for async operations
type groupingsLoadedMsg struct{ groupings []*triage.Grouping }
type messagesLoadedMsg struct{}
type archiveResultMsg struct{ result *triage.TriageResult }
type errMsg struct{ err error }

// loadGroupings loads email groupings asynchronously.
func (m Model) loadGroupings() tea.Msg {
	groupings, err := m.triager.LoadGroupings(m.ctx)
	if err != nil {
		return errMsg{err}
	}
	return groupingsLoadedMsg{groupings}
}

// loadMessages loads messages for the selected grouping.
func (m Model) loadMessages() tea.Msg {
	if m.selectedGrouping == nil {
		return errMsg{fmt.Errorf("no grouping selected")}
	}
	if err := m.triager.LoadMessages(m.ctx, m.selectedGrouping); err != nil {
		return errMsg{err}
	}
	return messagesLoadedMsg{}
}

// doArchive performs the archive operation.
func (m Model) doArchive() tea.Msg {
	if m.selectedGrouping == nil {
		return errMsg{fmt.Errorf("no grouping selected")}
	}
	result, err := m.triager.Archive(m.ctx, m.selectedGrouping)
	if err != nil {
		return errMsg{err}
	}
	return archiveResultMsg{result}
}

// View renders the current view.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	switch m.view {
	case ViewLoading:
		msg := "Loading..."
		if m.selectedGrouping != nil {
			switch m.loadingAction {
			case "archiving":
				msg = fmt.Sprintf("Archiving messages for %s...", m.selectedGrouping.Address)
			case "loading":
				msg = fmt.Sprintf("Loading messages for %s...", m.selectedGrouping.Address)
			}
		}
		return views.RenderLoading(m.spinner.View(), msg)

	case ViewGroupings:
		return views.RenderGroupings(m.groupings, m.groupingsCursor, m.width, m.height)

	case ViewSubjects:
		if m.selectedGrouping == nil {
			return "No grouping selected"
		}
		return views.RenderSubjects(m.selectedGrouping.Address, m.selectedGrouping.Messages, m.subjectsCursor, m.width, m.height, sortModeLabel(m.subjectsSortMode))

	case ViewConfirm:
		return views.RenderConfirm(m.selectedGrouping)

	case ViewResult:
		return views.RenderResult(m.result)

	case ViewError:
		return views.RenderError(m.err)

	default:
		return "Unknown view"
	}
}
