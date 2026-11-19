package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

// Options configures the TUI.
type Options struct {
	// Username is the Gmail account, used to open Gmail searches in the
	// right account.
	Username string
}

// Run starts the TUI application.
func Run(ctx context.Context, triager triage.Triager, opts Options) error {
	model := NewModel(ctx, triager).WithOptions(opts)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
