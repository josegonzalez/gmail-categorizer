package triage

import (
	"context"
	"errors"
	"testing"

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
		{UID: 1, To: []imap.Address{{Mailbox: "user1", Host: "example.com"}}},
		{UID: 2, To: []imap.Address{{Mailbox: "user1", Host: "example.com"}}},
		{UID: 3, To: []imap.Address{{Mailbox: "user2", Host: "example.com"}}},
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
	assert.Equal(t, 1, groupings[1].Count)
	assert.Equal(t, "user2@example.com", groupings[1].Address)
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
	assert.Equal(t, 1, groupings[1].Count)
	assert.Equal(t, "admin@other.com", groupings[1].Address)
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
	var archivedUIDs []uint32
	var destFolder string

	client := &mockClient{
		ArchiveToFolderFunc: func(ctx context.Context, uids []uint32, folder string) error {
			archivedUIDs = uids
			destFolder = folder
			return nil
		},
	}

	triager := NewTriager(client, []string{})
	grouping := &Grouping{
		Address: "test@example.com",
		Count:   3,
		UIDs:    []uint32{1, 2, 3},
	}

	result, err := triager.Archive(context.Background(), grouping)

	require.NoError(t, err)
	assert.Equal(t, 3, result.ArchivedCount)
	assert.Equal(t, "automated/test", result.DestinationFolder)
	assert.Equal(t, []uint32{1, 2, 3}, archivedUIDs)
	assert.Equal(t, "automated/test", destFolder)
}

func TestTriager_Archive_Error(t *testing.T) {
	expectedErr := errors.New("archive failed")
	client := &mockClient{
		ArchiveToFolderFunc: func(ctx context.Context, uids []uint32, folder string) error {
			return expectedErr
		},
	}

	triager := NewTriager(client, []string{})
	grouping := &Grouping{
		Address: "test@example.com",
		Count:   1,
		UIDs:    []uint32{1},
	}

	result, err := triager.Archive(context.Background(), grouping)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "archiving")
	assert.Nil(t, result)
}
