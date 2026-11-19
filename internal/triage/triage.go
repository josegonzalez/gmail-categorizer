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
	LoadMessageBody(ctx context.Context, g *Grouping, uid uint32) (*imap.MessageBody, error)
	Archive(ctx context.Context, g *Grouping, progress ProgressFunc) (*TriageResult, error)
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
// Each grouping keeps the envelopes it was built from in Messages, newest
// first, so callers can list subjects without fetching again.
func (t *triager) LoadGroupings(ctx context.Context) ([]*Grouping, error) {
	// Group messages by recipient
	groupMap := make(map[string]*Grouping)

	handler := func(msg *imap.Message) error {
		address, isGroupedByFrom := t.extractGroupingAddress(msg)
		if address == "" {
			return nil
		}

		if g, exists := groupMap[address]; exists {
			g.Count++
			g.UIDs = append(g.UIDs, msg.UID)
			g.Messages = append(g.Messages, msg)
			g.Subject = ""
		} else {
			groupMap[address] = &Grouping{
				Address:       address,
				Count:         1,
				Subject:       msg.Subject,
				UIDs:          []uint32{msg.UID},
				Messages:      []*imap.Message{msg},
				GroupedByFrom: isGroupedByFrom,
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
		SortMessages(g.Messages, SortDateDesc)
		groupings = append(groupings, g)
	}

	sort.Slice(groupings, func(i, j int) bool {
		return groupings[i].Count > groupings[j].Count
	})

	return groupings, nil
}

// extractGroupingAddress determines the address to use for grouping.
// Returns the address and whether it was grouped by the FROM field.
func (t *triager) extractGroupingAddress(msg *imap.Message) (string, bool) {
	// Check if sender should trigger group-by-from behavior
	if len(msg.From) > 0 {
		fromAddr := msg.From[0].String()
		for _, prefix := range t.groupByFrom {
			if mailaddr.HasPrefix(fromAddr, prefix) {
				return fromAddr, true
			}
		}
	}

	// Otherwise, use the To address
	if len(msg.To) > 0 {
		return mailaddr.Normalize(msg.To[0].String()), false
	}

	return "", false
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

// LoadMessageBody fetches the body of one message in the grouping.
func (t *triager) LoadMessageBody(ctx context.Context, g *Grouping, uid uint32) (*imap.MessageBody, error) {
	if !g.HasUID(uid) {
		return nil, fmt.Errorf("message %d is not in grouping %s", uid, g.Address)
	}
	body, err := t.client.FetchMessageBody(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("loading message body: %w", err)
	}
	return body, nil
}

// ProgressFunc is called as a grouping is archived, with the number of its
// messages moved so far.
type ProgressFunc func(archived int)

// Archive chunk sizes: each grouping is moved in about archiveSteps chunks so
// progress can be reported, but never fewer than minArchiveChunk messages at
// a time, since every chunk costs several round trips to the server.
const (
	archiveSteps    = 20
	minArchiveChunk = 25
)

// archiveChunkSize returns how many messages to move per IMAP command.
func archiveChunkSize(total int) int {
	return max(minArchiveChunk, (total+archiveSteps-1)/archiveSteps)
}

// Archive moves all messages in the grouping to a folder based on the
// address, creating the folder if needed. Messages are moved in chunks and
// progress, if non-nil, is called after each one.
//
// The grouping is updated to hold only the messages still in the inbox: it is
// empty after a successful archive, and after a failure part way it keeps the
// messages that were not moved. On failure the result reports how many
// messages were moved before the error.
func (t *triager) Archive(ctx context.Context, g *Grouping, progress ProgressFunc) (*TriageResult, error) {
	destFolder := g.DestinationFolder()
	result := &TriageResult{DestinationFolder: destFolder}

	if len(g.UIDs) == 0 {
		return result, nil
	}

	if err := t.client.CreateMailbox(ctx, destFolder); err != nil {
		return nil, fmt.Errorf("archiving messages: ensuring folder exists: %w", err)
	}

	uids := append([]uint32(nil), g.UIDs...)
	chunk := archiveChunkSize(len(uids))
	for start := 0; start < len(uids); start += chunk {
		end := min(start+chunk, len(uids))
		if err := t.client.MoveMessages(ctx, uids[start:end], destFolder); err != nil {
			g.removeUIDs(uids[:start])
			return result, fmt.Errorf("archiving messages: %w", err)
		}
		result.ArchivedCount = end
		if progress != nil {
			progress(end)
		}
	}

	g.removeUIDs(uids)
	return result, nil
}

// BatchProgress describes how far a batch archive has got.
type BatchProgress struct {
	// Current is the grouping being archived.
	Current string
	// Groupings is the number of groupings in the batch, and DoneGroupings
	// how many have finished, successfully or not.
	Groupings     int
	DoneGroupings int
	// Messages is the number of messages in the batch, and DoneMessages how
	// many have been moved or skipped because their grouping failed.
	Messages     int
	DoneMessages int
}

// BatchProgressFunc is called as a batch archive progresses.
type BatchProgressFunc func(BatchProgress)

// ArchiveBatch archives each grouping in order, carrying on past failures so
// one bad grouping does not stop the rest. progress, if non-nil, is called
// once before starting and then as messages are moved.
func ArchiveBatch(ctx context.Context, t Triager, groupings []*Grouping, progress BatchProgressFunc) []BatchResult {
	state := BatchProgress{Groupings: len(groupings)}
	for _, g := range groupings {
		state.Messages += len(g.UIDs)
	}
	report := func() {
		if progress != nil {
			progress(state)
		}
	}
	report()

	results := make([]BatchResult, 0, len(groupings))
	for _, g := range groupings {
		state.Current = g.Address
		before := state.DoneMessages
		total := len(g.UIDs)

		result, err := t.Archive(ctx, g, func(archived int) {
			state.DoneMessages = before + archived
			report()
		})
		results = append(results, BatchResult{
			Address: g.Address,
			Result:  result,
			Err:     err,
		})

		// A failed grouping's remaining messages count as done so the bar
		// still reaches the end.
		state.DoneMessages = before + total
		state.DoneGroupings++
		report()
	}
	return results
}
