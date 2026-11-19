package tui

import (
	"context"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/josegonzalez/gmail-categorizer/internal/browser"
	"github.com/josegonzalez/gmail-categorizer/internal/imap"
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
	ViewMessage
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

	// username is the Gmail account, used to build Gmail search links.
	username string
	// openURL opens a URL in the browser; swapped out in tests.
	openURL func(url string) error

	// Groupings view
	groupings        []*triage.Grouping
	groupingsCursor  int
	checkedGroupings map[int]bool
	groupingFilter   GroupingFilter
	previewOffset    int
	status           string

	// Subjects view
	selectedGrouping *triage.Grouping
	subjectsCursor   int
	subjectsOffset   int
	subjectsSortMode SortMode

	// Message view
	selectedMessage *imap.Message
	messageViewport viewport.Model

	// Confirm view
	confirmAction     string
	confirmReturnView ViewState

	// Loading view
	loadingAction string

	// Result view
	result         *triage.TriageResult
	archiveResults []triage.BatchResult
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
		openURL:          browser.Open,
	}
}

// WithOptions returns a copy of the model configured with opts.
func (m Model) WithOptions(opts Options) Model {
	m.username = opts.Username
	return m
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
		if m.view == ViewMessage {
			m.resizeMessageViewport()
		}
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
		m.previewOffset = 0
		m.view = ViewGroupings
		if len(m.groupings) == 0 {
			m.err = fmt.Errorf("no messages in inbox")
			m.view = ViewError
		}
		return m, nil

	case messagesLoadedMsg:
		m.view = ViewSubjects
		if m.selectedGrouping != nil {
			triage.SortMessages(m.selectedGrouping.Messages, m.subjectsSortMode)
		}
		return m, nil

	case bodyLoadedMsg:
		m.view = ViewMessage
		m.resizeMessageViewport()
		m.messageViewport.SetContent(msg.body.PlainText())
		m.messageViewport.GotoTop()
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
	case ViewMessage:
		return m.handleMessageKeys(msg)
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
	m.status = ""

	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case msg.String() == "up" || msg.String() == "k":
		if m.groupingsCursor > 0 {
			m.groupingsCursor--
			m.previewOffset = 0
		}

	case msg.String() == "down" || msg.String() == "j":
		if m.groupingsCursor < len(filtered)-1 {
			m.groupingsCursor++
			m.previewOffset = 0
		}

	case msg.String() == "K":
		if m.previewOffset > 0 {
			m.previewOffset--
		}

	case msg.String() == "J":
		if len(filtered) > 0 && m.previewOffset < views.MaxPreviewOffset(filtered[m.groupingsCursor]) {
			m.previewOffset++
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
			m.confirmReturnView = ViewGroupings
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

	case msg.String() == "g":
		if len(filtered) > 0 {
			g := filtered[m.groupingsCursor]
			if err := m.openURL(g.GmailSearchURL(m.username)); err != nil {
				m.status = fmt.Sprintf("Could not open Gmail: %v", err)
			} else {
				m.status = fmt.Sprintf("Opened Gmail search for %s", g.Address)
			}
		}

	case msg.String() == "f":
		if m.groupingFilter == FilterAll {
			m.groupingFilter = FilterSpecial
		} else {
			m.groupingFilter = FilterAll
		}
		m.groupingsCursor = 0
		m.previewOffset = 0
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
		if m.selectedGrouping != nil {
			// Keep the groupings preview newest first after a subject sort.
			triage.SortMessages(m.selectedGrouping.Messages, SortDateDesc)
		}
		m.view = ViewGroupings
		m.selectedGrouping = nil
		m.subjectsSortMode = SortDateDesc
		m.previewOffset = 0

	case msg.String() == "up" || msg.String() == "k":
		if m.subjectsCursor > 0 {
			m.subjectsCursor--
		}

	case msg.String() == "down" || msg.String() == "j":
		if m.selectedGrouping != nil && m.subjectsCursor < len(m.selectedGrouping.Messages)-1 {
			m.subjectsCursor++
		}

	case msg.String() == "enter":
		if m.selectedGrouping != nil && m.subjectsCursor < len(m.selectedGrouping.Messages) {
			m.selectedMessage = m.selectedGrouping.Messages[m.subjectsCursor]
			m.view = ViewLoading
			m.loadingAction = "loading-body"
			return m, tea.Batch(m.spinner.Tick, m.loadMessageBody)
		}

	case msg.String() == "s":
		m.subjectsSortMode = m.subjectsSortMode.Next()
		if m.selectedGrouping != nil {
			triage.SortMessages(m.selectedGrouping.Messages, m.subjectsSortMode)
			m.subjectsCursor = 0
		}

	case msg.String() == "a":
		m.confirmAction = "archive"
		m.confirmReturnView = ViewSubjects
		m.view = ViewConfirm
	}

	return m, nil
}

// handleMessageKeys handles keys in the message view.
func (m Model) handleMessageKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "esc", "backspace":
		m.view = ViewSubjects
		m.selectedMessage = nil
		return m, nil

	case "a":
		m.confirmAction = "archive"
		m.confirmReturnView = ViewMessage
		m.view = ViewConfirm
		return m, nil
	}

	var cmd tea.Cmd
	m.messageViewport, cmd = m.messageViewport.Update(msg)
	return m, cmd
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
		m.view = m.confirmReturnView
		if m.view == ViewConfirm || m.view == ViewLoading {
			m.view = ViewGroupings
		}
		if m.view == ViewGroupings && m.confirmAction == "archive" {
			// The single-archive target came from the cursor, not a
			// subjects view the user opened.
			m.selectedGrouping = nil
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
		m.selectedMessage = nil
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

// resizeMessageViewport sizes the body viewport to the terminal.
func (m *Model) resizeMessageViewport() {
	width, height := m.width, m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	bodyHeight := max(height-views.MessageHeaderLines, 3)
	if m.messageViewport.Width == 0 && m.messageViewport.Height == 0 {
		m.messageViewport = viewport.New(width, bodyHeight)
	} else {
		m.messageViewport.Width = width
		m.messageViewport.Height = bodyHeight
	}
	m.messageViewport.Style = lipgloss.NewStyle().Width(width)
}

// Message types for async operations
type groupingsLoadedMsg struct{ groupings []*triage.Grouping }
type messagesLoadedMsg struct{}
type bodyLoadedMsg struct{ body *imap.MessageBody }
type archiveResultMsg struct{ result *triage.TriageResult }
type batchArchiveResultMsg struct{ results []triage.BatchResult }
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

// loadMessageBody loads the body of the selected message.
func (m Model) loadMessageBody() tea.Msg {
	if m.selectedGrouping == nil || m.selectedMessage == nil {
		return errMsg{fmt.Errorf("no message selected")}
	}
	body, err := m.triager.LoadMessageBody(m.ctx, m.selectedGrouping, m.selectedMessage.UID)
	if err != nil {
		return errMsg{err}
	}
	return bodyLoadedMsg{body}
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
	return batchArchiveResultMsg{triage.ArchiveBatch(m.ctx, m.triager, groupings)}
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
		} else if m.loadingAction == "loading-body" {
			msg = "Loading message..."
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
		return views.RenderGroupings(views.GroupingsView{
			Groupings:     filtered,
			Cursor:        m.groupingsCursor,
			Width:         m.width,
			Height:        m.height,
			Checked:       filteredChecked,
			FilterMode:    int(m.groupingFilter),
			SpecialCount:  specialCount,
			PreviewOffset: m.previewOffset,
			Status:        m.status,
		})

	case ViewSubjects:
		if m.selectedGrouping == nil {
			return "No grouping selected"
		}
		return views.RenderSubjects(m.selectedGrouping.Address, m.selectedGrouping.Messages, m.subjectsCursor, m.width, m.height, sortModeLabel(m.subjectsSortMode))

	case ViewMessage:
		return views.RenderMessage(m.selectedMessage, m.messageViewport.View(), m.messageViewport.ScrollPercent(), m.width)

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
