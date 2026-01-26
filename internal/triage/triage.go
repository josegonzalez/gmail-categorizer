package triage

import (
	"context"
	"fmt"
	"sort"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/pkg/mailaddr"
)

// Triager provides email triage operations.
type Triager interface {
	LoadGroupings(ctx context.Context) ([]*Grouping, error)
	LoadMessages(ctx context.Context, g *Grouping) error
	Archive(ctx context.Context, g *Grouping) (*TriageResult, error)
}

// triager is the concrete implementation of Triager.
type triager struct {
	client      imap.Client
	groupByFrom []string
}

// NewTriager creates a new Triager with the given IMAP client.
func NewTriager(client imap.Client, groupByFrom []string) Triager {
	return &triager{
		client:      client,
		groupByFrom: groupByFrom,
	}
}

// LoadGroupings fetches all inbox messages and groups them by recipient address.
func (t *triager) LoadGroupings(ctx context.Context) ([]*Grouping, error) {
	// Group messages by recipient
	groupMap := make(map[string]*Grouping)

	handler := func(msg *imap.Message) error {
		address := t.extractGroupingAddress(msg)
		if address == "" {
			return nil
		}

		if g, exists := groupMap[address]; exists {
			g.Count++
			g.UIDs = append(g.UIDs, msg.UID)
		} else {
			groupMap[address] = &Grouping{
				Address: address,
				Count:   1,
				UIDs:    []uint32{msg.UID},
			}
		}
		return nil
	}

	if err := t.client.FetchMessages(ctx, handler, nil); err != nil {
		return nil, fmt.Errorf("fetching messages: %w", err)
	}

	// Convert map to slice and sort by count
	groupings := make([]*Grouping, 0, len(groupMap))
	for _, g := range groupMap {
		groupings = append(groupings, g)
	}

	sort.Slice(groupings, func(i, j int) bool {
		return groupings[i].Count > groupings[j].Count
	})

	return groupings, nil
}

// extractGroupingAddress determines the address to use for grouping.
func (t *triager) extractGroupingAddress(msg *imap.Message) string {
	// Check if sender should trigger group-by-from behavior
	if len(msg.From) > 0 {
		fromAddr := msg.From[0].String()
		for _, prefix := range t.groupByFrom {
			if mailaddr.HasPrefix(fromAddr, prefix) {
				return fromAddr
			}
		}
	}

	// Otherwise, use the To address
	if len(msg.To) > 0 {
		return mailaddr.Normalize(msg.To[0].String())
	}

	return ""
}

// LoadMessages fetches full message details for a grouping.
func (t *triager) LoadMessages(ctx context.Context, g *Grouping) error {
	if len(g.Messages) > 0 {
		// Already loaded
		return nil
	}

	g.Messages = make([]*imap.Message, 0, len(g.UIDs))

	handler := func(msg *imap.Message) error {
		g.Messages = append(g.Messages, msg)
		return nil
	}

	return t.client.FetchMessagesWithUIDs(ctx, g.UIDs, handler)
}

// Archive archives all messages in the grouping to a folder based on the address.
func (t *triager) Archive(ctx context.Context, g *Grouping) (*TriageResult, error) {
	destFolder := g.DestinationFolder()

	// Archive to the folder (creates if needed)
	if err := t.client.ArchiveToFolder(ctx, g.UIDs, destFolder); err != nil {
		return nil, fmt.Errorf("archiving messages: %w", err)
	}

	return &TriageResult{
		ArchivedCount:     len(g.UIDs),
		DestinationFolder: destFolder,
	}, nil
}
