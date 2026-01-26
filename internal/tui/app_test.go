package tui

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

func TestRun_CreatesModel(t *testing.T) {
	// We can't fully test Run since it starts an interactive TUI,
	// but we can verify the function signature and that it compiles.
	// In a real test environment, we might use a mock tea.Program.

	ctx := context.Background()
	triager := &mockTriager{
		LoadGroupingsFunc: func(ctx context.Context) ([]*triage.Grouping, error) {
			return []*triage.Grouping{}, nil
		},
	}

	// Verify that NewModel works with the triager
	model := NewModel(ctx, triager)
	assert.NotNil(t, model)
	assert.Equal(t, ViewLoading, model.view)
}

// Note: Full integration testing of Run() would require mocking tea.Program
// which is beyond the scope of unit tests. The function primarily creates
// a model and starts a bubbletea program, which is tested through model_test.go.
