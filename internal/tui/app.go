package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

// Run starts the TUI application.
func Run(ctx context.Context, triager triage.Triager) error {
	model := NewModel(ctx, triager)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
