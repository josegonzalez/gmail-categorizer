package tui

import (
	"context"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

// mockTriager implements triage.Triager for testing.
type mockTriager struct {
	LoadGroupingsFunc func(ctx context.Context) ([]*triage.Grouping, error)
	LoadMessagesFunc  func(ctx context.Context, g *triage.Grouping) error
	ArchiveFunc       func(ctx context.Context, g *triage.Grouping) (*triage.TriageResult, error)
}

func (m *mockTriager) LoadGroupings(ctx context.Context) ([]*triage.Grouping, error) {
	if m.LoadGroupingsFunc != nil {
		return m.LoadGroupingsFunc(ctx)
	}
	return []*triage.Grouping{}, nil
}

func (m *mockTriager) LoadMessages(ctx context.Context, g *triage.Grouping) error {
	if m.LoadMessagesFunc != nil {
		return m.LoadMessagesFunc(ctx, g)
	}
	return nil
}

func (m *mockTriager) Archive(ctx context.Context, g *triage.Grouping) (*triage.TriageResult, error) {
	if m.ArchiveFunc != nil {
		return m.ArchiveFunc(ctx, g)
	}
	return &triage.TriageResult{ArchivedCount: 0, DestinationFolder: ""}, nil
}
