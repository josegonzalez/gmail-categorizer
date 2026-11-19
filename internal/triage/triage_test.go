package triage

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

func TestNewTriager(t *testing.T) {
	client := &mockClient{}
	groupByFrom := []string{"admin@", "hi@"}

	triager := NewTriager(client, groupByFrom)

	require.NotNil(t, triager)
}

func TestTriager_LoadGroupings_GroupByTo(t *testing.T) {
	messages := []*imap.Message{
		{UID: 1, Subject: "Hello", To: []imap.Address{{Mailbox: "user1", Host: "example.com"}}},
		{UID: 2, Subject: "World", To: []imap.Address{{Mailbox: "user1", Host: "example.com"}}},
		{UID: 3, Subject: "Single Email", To: []imap.Address{{Mailbox: "user2", Host: "example.com"}}},
	}

	client := &mockClient{
		FetchMessagesFunc: func(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
			for _, msg := range messages {
				if err := handler(msg); err != nil {
					return err
				}
			}
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	groupings, err := triager.LoadGroupings(context.Background())

	require.NoError(t, err)
	require.Equal(t, 2, len(groupings))

	// Should be sorted by count descending
	assert.Equal(t, 2, groupings[0].Count)
	assert.Equal(t, "user1@example.com", groupings[0].Address)
	assert.False(t, groupings[0].GroupedByFrom)
	assert.Equal(t, "", groupings[0].Subject)
	assert.Equal(t, 1, groupings[1].Count)
	assert.Equal(t, "user2@example.com", groupings[1].Address)
	assert.False(t, groupings[1].GroupedByFrom)
	assert.Equal(t, "Single Email", groupings[1].Subject)
}

func TestTriager_LoadGroupings_SubjectClearedOnMultiple(t *testing.T) {
	messages := []*imap.Message{
		{UID: 1, Subject: "First", To: []imap.Address{{Mailbox: "user", Host: "example.com"}}},
		{UID: 2, Subject: "Second", To: []imap.Address{{Mailbox: "user", Host: "example.com"}}},
	}

	client := &mockClient{
		FetchMessagesFunc: func(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
			for _, msg := range messages {
				if err := handler(msg); err != nil {
					return err
				}
			}
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	groupings, err := triager.LoadGroupings(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, len(groupings))
	assert.Equal(t, 2, groupings[0].Count)
	assert.Equal(t, "", groupings[0].Subject)
}

func TestTriager_LoadGroupings_EmptyInbox(t *testing.T) {
	client := &mockClient{
		FetchMessagesFunc: func(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	groupings, err := triager.LoadGroupings(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, len(groupings))
}

func TestTriager_LoadGroupings_GroupByFrom(t *testing.T) {
	// In triage, groupByFrom prefixes are matched against FROM addresses
	// If FROM starts with the prefix, grouping is by FROM address
	messages := []*imap.Message{
		{
			UID:  1,
			To:   []imap.Address{{Mailbox: "user", Host: "example.com"}},
			From: []imap.Address{{Mailbox: "admin", Host: "company.com"}},
		},
		{
			UID:  2,
			To:   []imap.Address{{Mailbox: "user", Host: "example.com"}},
			From: []imap.Address{{Mailbox: "admin", Host: "company.com"}},
		},
		{
			UID:  3,
			To:   []imap.Address{{Mailbox: "user", Host: "example.com"}},
			From: []imap.Address{{Mailbox: "admin", Host: "other.com"}},
		},
	}

	client := &mockClient{
		FetchMessagesFunc: func(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
			for _, msg := range messages {
				if err := handler(msg); err != nil {
					return err
				}
			}
			return nil
		},
	}

	triager := NewTriager(client, []string{"admin@"})
	groupings, err := triager.LoadGroupings(context.Background())

	require.NoError(t, err)
	require.Equal(t, 2, len(groupings))

	// Grouped by FROM address since FROM starts with "admin@"
	assert.Equal(t, 2, groupings[0].Count)
	assert.Equal(t, "admin@company.com", groupings[0].Address)
	assert.True(t, groupings[0].GroupedByFrom)
	assert.Equal(t, 1, groupings[1].Count)
	assert.Equal(t, "admin@other.com", groupings[1].Address)
	assert.True(t, groupings[1].GroupedByFrom)
}

func TestTriager_LoadGroupings_Error(t *testing.T) {
	expectedErr := errors.New("fetch failed")
	client := &mockClient{
		FetchMessagesFunc: func(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
			return expectedErr
		},
	}

	triager := NewTriager(client, []string{})
	groupings, err := triager.LoadGroupings(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch")
	assert.Nil(t, groupings)
}

func TestTriager_LoadGroupings_EmptyToAddress(t *testing.T) {
	messages := []*imap.Message{
		{UID: 1, To: []imap.Address{}},
		{UID: 2, To: []imap.Address{{Mailbox: "user", Host: "example.com"}}},
	}

	client := &mockClient{
		FetchMessagesFunc: func(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
			for _, msg := range messages {
				if err := handler(msg); err != nil {
					return err
				}
			}
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	groupings, err := triager.LoadGroupings(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, len(groupings))
	assert.Equal(t, "user@example.com", groupings[0].Address)
}

func TestTriager_LoadMessages(t *testing.T) {
	messages := []*imap.Message{
		{UID: 1, Subject: "Subject 1"},
		{UID: 2, Subject: "Subject 2"},
	}

	client := &mockClient{
		FetchMessagesWithUIDsFunc: func(ctx context.Context, uids []uint32, handler imap.MessageHandler) error {
			for _, msg := range messages {
				if err := handler(msg); err != nil {
					return err
				}
			}
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	grouping := &Grouping{
		Address: "test@example.com",
		Count:   2,
		UIDs:    []uint32{1, 2},
	}

	err := triager.LoadMessages(context.Background(), grouping)

	require.NoError(t, err)
	assert.Equal(t, 2, len(grouping.Messages))
	assert.Equal(t, "Subject 1", grouping.Messages[0].Subject)
	assert.Equal(t, "Subject 2", grouping.Messages[1].Subject)
}

func TestTriager_LoadMessages_AlreadyLoaded(t *testing.T) {
	fetchCalled := false
	client := &mockClient{
		FetchMessagesWithUIDsFunc: func(ctx context.Context, uids []uint32, handler imap.MessageHandler) error {
			fetchCalled = true
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	grouping := &Grouping{
		Address:  "test@example.com",
		Count:    1,
		UIDs:     []uint32{1},
		Messages: []*imap.Message{{UID: 1, Subject: "Already loaded"}},
	}

	err := triager.LoadMessages(context.Background(), grouping)

	require.NoError(t, err)
	assert.False(t, fetchCalled, "Should not fetch when messages already loaded")
	assert.Equal(t, 1, len(grouping.Messages))
}

func TestTriager_LoadMessages_Error(t *testing.T) {
	expectedErr := errors.New("fetch failed")
	client := &mockClient{
		FetchMessagesWithUIDsFunc: func(ctx context.Context, uids []uint32, handler imap.MessageHandler) error {
			return expectedErr
		},
	}

	triager := NewTriager(client, []string{})
	grouping := &Grouping{
		Address: "test@example.com",
		Count:   1,
		UIDs:    []uint32{1},
	}

	err := triager.LoadMessages(context.Background(), grouping)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestTriager_Archive(t *testing.T) {
	var created []string
	var moved [][]uint32
	var destFolder string

	client := &mockClient{
		CreateMailboxFunc: func(ctx context.Context, name string) error {
			created = append(created, name)
			return nil
		},
		MoveMessagesFunc: func(ctx context.Context, uids []uint32, folder string) error {
			moved = append(moved, append([]uint32(nil), uids...))
			destFolder = folder
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	grouping := &Grouping{
		Address:  "test@example.com",
		Count:    3,
		UIDs:     []uint32{1, 2, 3},
		Messages: []*imap.Message{{UID: 1}, {UID: 2}, {UID: 3}},
	}

	var progress []int
	result, err := triager.Archive(context.Background(), grouping, func(n int) { progress = append(progress, n) })

	require.NoError(t, err)
	assert.Equal(t, 3, result.ArchivedCount)
	assert.Equal(t, "automated/test", result.DestinationFolder)
	assert.Equal(t, []string{"automated/test"}, created)
	assert.Equal(t, [][]uint32{{1, 2, 3}}, moved, "small groupings move in one chunk")
	assert.Equal(t, "automated/test", destFolder)
	assert.Equal(t, []int{3}, progress)

	// The grouping now holds what is left in the inbox: nothing.
	assert.Equal(t, 0, grouping.Count)
	assert.Empty(t, grouping.UIDs)
	assert.Empty(t, grouping.Messages)
}

func TestTriager_Archive_NilProgress(t *testing.T) {
	triager := NewTriager(&mockClient{}, []string{})
	grouping := &Grouping{Address: "test@example.com", Count: 1, UIDs: []uint32{1}}

	result, err := triager.Archive(context.Background(), grouping, nil)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ArchivedCount)
}

func TestTriager_Archive_Empty(t *testing.T) {
	called := false
	client := &mockClient{
		CreateMailboxFunc: func(ctx context.Context, name string) error {
			called = true
			return nil
		},
	}
	triager := NewTriager(client, []string{})

	result, err := triager.Archive(context.Background(), &Grouping{Address: "test@example.com"}, nil)

	require.NoError(t, err)
	assert.Equal(t, 0, result.ArchivedCount)
	assert.False(t, called, "nothing to archive means no server calls")
}

func TestTriager_Archive_ChunksLargeGroupings(t *testing.T) {
	var chunks []int
	client := &mockClient{
		MoveMessagesFunc: func(ctx context.Context, uids []uint32, folder string) error {
			chunks = append(chunks, len(uids))
			return nil
		},
	}

	grouping := &Grouping{Address: "test@example.com"}
	for i := 1; i <= 1000; i++ {
		grouping.UIDs = append(grouping.UIDs, uint32(i))
	}
	grouping.Count = len(grouping.UIDs)

	var progress []int
	result, err := NewTriager(client, nil).Archive(context.Background(), grouping, func(n int) { progress = append(progress, n) })

	require.NoError(t, err)
	assert.Equal(t, 1000, result.ArchivedCount)
	assert.Len(t, chunks, archiveSteps)
	for _, c := range chunks {
		assert.Equal(t, 50, c)
	}
	require.Len(t, progress, archiveSteps)
	assert.Equal(t, 50, progress[0])
	assert.Equal(t, 1000, progress[len(progress)-1])
}

func TestArchiveChunkSize(t *testing.T) {
	assert.Equal(t, minArchiveChunk, archiveChunkSize(1))
	assert.Equal(t, minArchiveChunk, archiveChunkSize(500))
	assert.Equal(t, 26, archiveChunkSize(501))
	assert.Equal(t, 50, archiveChunkSize(1000))
	assert.Equal(t, 501, archiveChunkSize(10001))
}

func TestTriager_Archive_CreateMailboxError(t *testing.T) {
	expectedErr := errors.New("no permission")
	moveCalled := false
	client := &mockClient{
		CreateMailboxFunc: func(ctx context.Context, name string) error { return expectedErr },
		MoveMessagesFunc: func(ctx context.Context, uids []uint32, folder string) error {
			moveCalled = true
			return nil
		},
	}
	grouping := &Grouping{Address: "test@example.com", Count: 1, UIDs: []uint32{1}}

	result, err := NewTriager(client, nil).Archive(context.Background(), grouping, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Contains(t, err.Error(), "archiving")
	assert.Nil(t, result)
	assert.False(t, moveCalled)
	assert.Equal(t, []uint32{1}, grouping.UIDs, "nothing was moved")
}

func TestTriager_Archive_PartialFailureKeepsUnmovedMessages(t *testing.T) {
	expectedErr := errors.New("connection reset")
	calls := 0
	client := &mockClient{
		MoveMessagesFunc: func(ctx context.Context, uids []uint32, folder string) error {
			calls++
			if calls == 2 {
				return expectedErr
			}
			return nil
		},
	}

	grouping := &Grouping{Address: "test@example.com"}
	for i := 1; i <= 60; i++ {
		grouping.UIDs = append(grouping.UIDs, uint32(i))
		grouping.Messages = append(grouping.Messages, &imap.Message{UID: uint32(i), Subject: fmt.Sprintf("s%d", i)})
	}
	grouping.Count = 60

	result, err := NewTriager(client, nil).Archive(context.Background(), grouping, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	require.NotNil(t, result)
	assert.Equal(t, 25, result.ArchivedCount, "the first chunk was moved")
	assert.Equal(t, 35, grouping.Count)
	assert.Len(t, grouping.Messages, 35)
	assert.Equal(t, uint32(26), grouping.UIDs[0])
	assert.Equal(t, uint32(26), grouping.Messages[0].UID)
}

func TestTriager_LoadGroupings_KeepsEnvelopesNewestFirst(t *testing.T) {
	now := time.Now()
	to := []imap.Address{{Mailbox: "user", Host: "example.com"}}
	messages := []*imap.Message{
		{UID: 1, Subject: "Oldest", Date: now.Add(-2 * time.Hour), To: to},
		{UID: 2, Subject: "Newest", Date: now, To: to},
		{UID: 3, Subject: "Middle", Date: now.Add(-1 * time.Hour), To: to},
	}

	fetchByUIDCalled := false
	client := &mockClient{
		FetchMessagesFunc: func(ctx context.Context, handler imap.MessageHandler, progress imap.ProgressCallback) error {
			for _, msg := range messages {
				if err := handler(msg); err != nil {
					return err
				}
			}
			return nil
		},
		FetchMessagesWithUIDsFunc: func(ctx context.Context, uids []uint32, handler imap.MessageHandler) error {
			fetchByUIDCalled = true
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	groupings, err := triager.LoadGroupings(context.Background())
	require.NoError(t, err)
	require.Len(t, groupings, 1)

	g := groupings[0]
	require.Len(t, g.Messages, 3)
	assert.Equal(t, "Newest", g.Messages[0].Subject)
	assert.Equal(t, "Middle", g.Messages[1].Subject)
	assert.Equal(t, "Oldest", g.Messages[2].Subject)
	assert.Equal(t, "", g.Subject, "subject is still cleared for multi-message groupings")

	require.NoError(t, triager.LoadMessages(context.Background(), g))
	assert.False(t, fetchByUIDCalled, "LoadMessages should reuse envelopes from LoadGroupings")
}

func TestTriager_LoadMessageBody(t *testing.T) {
	var fetchedUID uint32
	client := &mockClient{
		FetchMessageBodyFunc: func(ctx context.Context, uid uint32) (*imap.MessageBody, error) {
			fetchedUID = uid
			return &imap.MessageBody{Text: "hello"}, nil
		},
	}

	triager := NewTriager(client, []string{})
	g := &Grouping{Address: "test@example.com", UIDs: []uint32{7, 8}}

	body, err := triager.LoadMessageBody(context.Background(), g, 8)

	require.NoError(t, err)
	assert.Equal(t, "hello", body.Text)
	assert.Equal(t, uint32(8), fetchedUID)
}

func TestTriager_LoadMessageBody_UnknownUID(t *testing.T) {
	called := false
	client := &mockClient{
		FetchMessageBodyFunc: func(ctx context.Context, uid uint32) (*imap.MessageBody, error) {
			called = true
			return nil, nil
		},
	}

	triager := NewTriager(client, []string{})
	g := &Grouping{Address: "test@example.com", UIDs: []uint32{7}}

	_, err := triager.LoadMessageBody(context.Background(), g, 99)

	require.Error(t, err)
	assert.False(t, called, "should not fetch a UID outside the grouping")
}

func TestTriager_LoadMessageBody_Error(t *testing.T) {
	expectedErr := errors.New("fetch failed")
	client := &mockClient{
		FetchMessageBodyFunc: func(ctx context.Context, uid uint32) (*imap.MessageBody, error) {
			return nil, expectedErr
		},
	}

	triager := NewTriager(client, []string{})
	g := &Grouping{Address: "test@example.com", UIDs: []uint32{7}}

	_, err := triager.LoadMessageBody(context.Background(), g, 7)

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestArchiveBatch(t *testing.T) {
	expectedErr := errors.New("archive failed")
	client := &mockClient{
		MoveMessagesFunc: func(ctx context.Context, uids []uint32, folder string) error {
			if folder == "automated/bad" {
				return expectedErr
			}
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	groupings := []*Grouping{
		{Address: "good@example.com", Count: 2, UIDs: []uint32{1, 2}},
		{Address: "bad@example.com", Count: 1, UIDs: []uint32{3}},
		{Address: "also@example.com", Count: 1, UIDs: []uint32{4}},
	}

	var progress []BatchProgress
	results := ArchiveBatch(context.Background(), triager, groupings, func(p BatchProgress) { progress = append(progress, p) })

	require.Len(t, results, 3)
	assert.Equal(t, "good@example.com", results[0].Address)
	assert.NoError(t, results[0].Err)
	assert.Equal(t, 2, results[0].Result.ArchivedCount)
	assert.Equal(t, "bad@example.com", results[1].Address)
	assert.ErrorIs(t, results[1].Err, expectedErr)
	assert.Equal(t, 0, results[1].Result.ArchivedCount)
	assert.Equal(t, "also@example.com", results[2].Address)
	assert.NoError(t, results[2].Err, "a failure should not stop later groupings")

	require.NotEmpty(t, progress)
	assert.Equal(t, BatchProgress{Groupings: 3, Messages: 4}, progress[0], "reported before starting")
	last := progress[len(progress)-1]
	assert.Equal(t, 3, last.DoneGroupings)
	assert.Equal(t, 4, last.DoneMessages, "a failed grouping still counts as done")
	assert.Equal(t, "also@example.com", last.Current)
	for i := 1; i < len(progress); i++ {
		assert.GreaterOrEqual(t, progress[i].DoneMessages, progress[i-1].DoneMessages, "progress never goes backwards")
	}
}

func TestArchiveBatch_Empty(t *testing.T) {
	triager := NewTriager(&mockClient{}, []string{})
	assert.Empty(t, ArchiveBatch(context.Background(), triager, nil, nil))
}
