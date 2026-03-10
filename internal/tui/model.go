package tui

import (
	"context"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
	"github.com/josegonzalez/gmail-categorizer/internal/tui/views"
)

// GroupingFilter represents the filter mode for the groupings view.
type GroupingFilter int

const (
	FilterAll     GroupingFilter = iota
	FilterSpecial                // Show only GroupedByFrom groupings
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

// batchArchiveEntry holds the result of archiving a single grouping in a batch.
type batchArchiveEntry struct {
	Address string
	Result  *triage.TriageResult
	Err     error
}

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
	groupings        []*triage.Grouping
	groupingsCursor  int
	checkedGroupings map[int]bool
	groupingFilter   GroupingFilter

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
	result         *triage.TriageResult
	archiveResults []batchArchiveEntry
}

// NewModel creates a new TUI model.
func NewModel(ctx context.Context, triager triage.Triager) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return Model{
		triager:          triager,
		ctx:              ctx,
		keys:             DefaultKeyMap(),
		view:             ViewLoading,
		spinner:          s,
		checkedGroupings: make(map[int]bool),
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
		m.checkedGroupings = make(map[int]bool)
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

	case batchArchiveResultMsg:
		m.archiveResults = msg.results
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
	filtered := m.filteredGroupings()
	indices := m.filteredGroupingIndices()

	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case msg.String() == "up" || msg.String() == "k":
		if m.groupingsCursor > 0 {
			m.groupingsCursor--
		}

	case msg.String() == "down" || msg.String() == "j":
		if m.groupingsCursor < len(filtered)-1 {
			m.groupingsCursor++
		}

	case msg.String() == " ":
		if len(filtered) > 0 {
			realIdx := indices[m.groupingsCursor]
			if m.checkedGroupings[realIdx] {
				delete(m.checkedGroupings, realIdx)
			} else {
				m.checkedGroupings[realIdx] = true
			}
		}

	case msg.String() == "enter":
		if len(filtered) > 0 {
			realIdx := indices[m.groupingsCursor]
			m.selectedGrouping = m.groupings[realIdx]
			m.subjectsCursor = 0
			m.subjectsOffset = 0
			m.view = ViewLoading
			m.loadingAction = "loading"
			return m, tea.Batch(m.spinner.Tick, m.loadMessages)
		}

	case msg.String() == "a":
		if len(filtered) > 0 {
			if len(m.checkedGroupings) > 0 {
				// Batch archive
				m.confirmAction = "batch-archive"
				m.view = ViewConfirm
			} else {
				// Single archive (existing behavior)
				realIdx := indices[m.groupingsCursor]
				m.selectedGrouping = m.groupings[realIdx]
				m.confirmAction = "archive"
				m.view = ViewConfirm
			}
		}

	case msg.String() == "f":
		if m.groupingFilter == FilterAll {
			m.groupingFilter = FilterSpecial
		} else {
			m.groupingFilter = FilterAll
		}
		m.groupingsCursor = 0
		m.checkedGroupings = make(map[int]bool)
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
		if m.confirmAction == "batch-archive" {
			m.loadingAction = "batch-archiving"
			return m, tea.Batch(m.spinner.Tick, m.doBatchArchive)
		}
		m.loadingAction = "archiving"
		return m, tea.Batch(m.spinner.Tick, m.doArchive)

	case msg.String() == "n" || msg.String() == "esc":
		if m.confirmAction == "batch-archive" {
			m.view = ViewGroupings
		} else if m.selectedGrouping != nil && len(m.selectedGrouping.Messages) > 0 {
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
		m.archiveResults = nil
		m.checkedGroupings = make(map[int]bool)
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
type batchArchiveResultMsg struct{ results []batchArchiveEntry }
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

// filteredGroupings returns groupings matching the current filter.
func (m Model) filteredGroupings() []*triage.Grouping {
	if m.groupingFilter == FilterAll {
		return m.groupings
	}
	result := make([]*triage.Grouping, 0)
	for _, g := range m.groupings {
		if g.GroupedByFrom {
			result = append(result, g)
		}
	}
	return result
}

// filteredGroupingIndices returns indices into m.groupings for filtered items.
func (m Model) filteredGroupingIndices() []int {
	if m.groupingFilter == FilterAll {
		indices := make([]int, len(m.groupings))
		for i := range m.groupings {
			indices[i] = i
		}
		return indices
	}
	indices := make([]int, 0)
	for i, g := range m.groupings {
		if g.GroupedByFrom {
			indices = append(indices, i)
		}
	}
	return indices
}

// checkedGroupingsList returns checked groupings in index order.
func (m Model) checkedGroupingsList() []*triage.Grouping {
	indices := make([]int, 0, len(m.checkedGroupings))
	for idx := range m.checkedGroupings {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	result := make([]*triage.Grouping, 0, len(indices))
	for _, idx := range indices {
		if idx < len(m.groupings) {
			result = append(result, m.groupings[idx])
		}
	}
	return result
}

// doBatchArchive archives each checked grouping sequentially.
func (m Model) doBatchArchive() tea.Msg {
	groupings := m.checkedGroupingsList()
	if len(groupings) == 0 {
		return errMsg{fmt.Errorf("no groupings selected")}
	}

	results := make([]batchArchiveEntry, 0, len(groupings))
	for _, g := range groupings {
		result, err := m.triager.Archive(m.ctx, g)
		results = append(results, batchArchiveEntry{
			Address: g.Address,
			Result:  result,
			Err:     err,
		})
	}
	return batchArchiveResultMsg{results}
}

// View renders the current view.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	switch m.view {
	case ViewLoading:
		msg := "Loading..."
		if m.loadingAction == "batch-archiving" {
			checkedGroupings := m.checkedGroupingsList()
			msg = fmt.Sprintf("Archiving %d groupings...", len(checkedGroupings))
		} else if m.selectedGrouping != nil {
			switch m.loadingAction {
			case "archiving":
				msg = fmt.Sprintf("Archiving messages for %s...", m.selectedGrouping.Address)
			case "loading":
				msg = fmt.Sprintf("Loading messages for %s...", m.selectedGrouping.Address)
			}
		}
		return views.RenderLoading(m.spinner.View(), msg)

	case ViewGroupings:
		filtered := m.filteredGroupings()
		indices := m.filteredGroupingIndices()
		// Remap checked from real indices to filtered indices
		filteredChecked := make(map[int]bool)
		for fi, realIdx := range indices {
			if m.checkedGroupings[realIdx] {
				filteredChecked[fi] = true
			}
		}
		specialCount := 0
		for _, g := range m.groupings {
			if g.GroupedByFrom {
				specialCount++
			}
		}
		return views.RenderGroupings(filtered, m.groupingsCursor, m.width, m.height, filteredChecked, int(m.groupingFilter), specialCount)

	case ViewSubjects:
		if m.selectedGrouping == nil {
			return "No grouping selected"
		}
		return views.RenderSubjects(m.selectedGrouping.Address, m.selectedGrouping.Messages, m.subjectsCursor, m.width, m.height, sortModeLabel(m.subjectsSortMode))

	case ViewConfirm:
		if m.confirmAction == "batch-archive" {
			return views.RenderBatchConfirm(m.checkedGroupingsList())
		}
		return views.RenderConfirm(m.selectedGrouping)

	case ViewResult:
		if m.archiveResults != nil {
			entries := make([]views.BatchResultEntry, len(m.archiveResults))
			for i, r := range m.archiveResults {
				entries[i] = views.BatchResultEntry{
					Address: r.Address,
					Err:     r.Err,
				}
				if r.Result != nil {
					entries[i].ArchivedCount = r.Result.ArchivedCount
					entries[i].DestinationFolder = r.Result.DestinationFolder
				}
			}
			return views.RenderBatchResult(entries)
		}
		return views.RenderResult(m.result)

	case ViewError:
		return views.RenderError(m.err)

	default:
		return "Unknown view"
	}
}
