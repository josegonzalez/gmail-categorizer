package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

func TestNewAggregator(t *testing.T) {
	agg := NewAggregator([]string{"admin@", "Hi@", "EMAIL@"})
	require.NotNil(t, agg)

	// Verify it can process messages
	err := agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "user", Host: "example.com"}},
	})
	assert.NoError(t, err)
}

func TestAggregator_BasicCounting(t *testing.T) {
	agg := NewAggregator([]string{})

	// Add messages to the same address
	agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "user", Host: "example.com"}},
	})
	agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "user", Host: "example.com"}},
	})
	agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "other", Host: "example.com"}},
	})

	results := agg.Results("INBOX")

	assert.Equal(t, 3, results.TotalMessages)
	assert.Equal(t, 2, len(results.ByToAddress))

	// Find the user@example.com entry
	var userCount, otherCount int
	for _, stat := range results.ByToAddress {
		if stat.Address == "user@example.com" {
			userCount = stat.Count
		} else if stat.Address == "other@example.com" {
			otherCount = stat.Count
		}
	}
	assert.Equal(t, 2, userCount)
	assert.Equal(t, 1, otherCount)
}

func TestAggregator_SpecialAddress(t *testing.T) {
	agg := NewAggregator([]string{"admin@"})

	// Email to admin@ should be grouped by sender
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "admin", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "sender1", Host: "other.com"}},
	})
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "admin", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "sender1", Host: "other.com"}},
	})
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "admin", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "sender2", Host: "other.com"}},
	})

	results := agg.Results("INBOX")

	assert.Equal(t, 3, results.TotalMessages)
	assert.Equal(t, 0, len(results.ByToAddress), "admin@ emails should not appear in regular stats")
	assert.Equal(t, 2, len(results.ByFromAddress), "should have 2 unique senders")

	// Verify sender counts
	senderCounts := make(map[string]int)
	for _, stat := range results.ByFromAddress {
		assert.Equal(t, "admin@example.com", stat.ToAddress)
		senderCounts[stat.FromAddress] = stat.Count
	}
	assert.Equal(t, 2, senderCounts["sender1@other.com"])
	assert.Equal(t, 1, senderCounts["sender2@other.com"])
}

func TestAggregator_SpecialAddressCaseInsensitive(t *testing.T) {
	agg := NewAggregator([]string{"admin@"})

	// Different cases should all be treated as special
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "ADMIN", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "sender", Host: "other.com"}},
	})
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "Admin", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "sender", Host: "other.com"}},
	})

	results := agg.Results("INBOX")

	assert.Equal(t, 0, len(results.ByToAddress))
	assert.Equal(t, 1, len(results.ByFromAddress))
	assert.Equal(t, 2, results.ByFromAddress[0].Count)
}

func TestAggregator_MultipleRecipients(t *testing.T) {
	agg := NewAggregator([]string{})

	// Message to multiple recipients
	agg.Process(&imap.Message{
		To: []imap.Address{
			{Mailbox: "user1", Host: "example.com"},
			{Mailbox: "user2", Host: "example.com"},
		},
	})

	results := agg.Results("INBOX")

	assert.Equal(t, 1, results.TotalMessages)
	assert.Equal(t, 2, len(results.ByToAddress))
}

func TestAggregator_MixedSpecialAndRegular(t *testing.T) {
	agg := NewAggregator([]string{"admin@", "hi@"})

	// Regular email
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "user", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "sender", Host: "other.com"}},
	})
	// Special email
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "admin", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "sender", Host: "other.com"}},
	})
	// Another special email
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "hi", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "sender", Host: "other.com"}},
	})

	results := agg.Results("INBOX")

	assert.Equal(t, 3, results.TotalMessages)
	assert.Equal(t, 1, len(results.ByToAddress))
	assert.Equal(t, 2, len(results.ByFromAddress))
}

func TestAggregator_GmailPlusAddressing(t *testing.T) {
	agg := NewAggregator([]string{})

	// Emails with + addressing should be normalized
	agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "user+newsletter", Host: "example.com"}},
	})
	agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "user+shopping", Host: "example.com"}},
	})
	agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "user", Host: "example.com"}},
	})

	results := agg.Results("INBOX")

	assert.Equal(t, 3, results.TotalMessages)
	assert.Equal(t, 1, len(results.ByToAddress))
	assert.Equal(t, "user@example.com", results.ByToAddress[0].Address)
	assert.Equal(t, 3, results.ByToAddress[0].Count)
}

func TestAggregator_EmptyAddresses(t *testing.T) {
	agg := NewAggregator([]string{})

	// Message with empty To
	agg.Process(&imap.Message{
		To: []imap.Address{},
	})
	// Message with empty mailbox
	agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "", Host: "example.com"}},
	})

	results := agg.Results("INBOX")

	assert.Equal(t, 2, results.TotalMessages)
	assert.Equal(t, 0, len(results.ByToAddress))
}

func TestAggregator_SpecialAddress_EmptyFromAddress(t *testing.T) {
	agg := NewAggregator([]string{"admin@"})

	// Special address with empty From - should not count as special
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "admin", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "", Host: "other.com"}},
	})
	// Special address with valid From
	agg.Process(&imap.Message{
		To:   []imap.Address{{Mailbox: "admin", Host: "example.com"}},
		From: []imap.Address{{Mailbox: "sender", Host: "other.com"}},
	})

	results := agg.Results("INBOX")

	assert.Equal(t, 2, results.TotalMessages)
	assert.Equal(t, 0, len(results.ByToAddress))
	assert.Equal(t, 1, len(results.ByFromAddress))
	assert.Equal(t, 1, results.ByFromAddress[0].Count)
}

func TestAggregator_Reset(t *testing.T) {
	agg := NewAggregator([]string{})

	agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "user", Host: "example.com"}},
	})

	results := agg.Results("INBOX")
	assert.Equal(t, 1, results.TotalMessages)

	agg.Reset()

	results = agg.Results("INBOX")
	assert.Equal(t, 0, results.TotalMessages)
	assert.Equal(t, 0, len(results.ByToAddress))
}

func TestAggregator_ResultsMetadata(t *testing.T) {
	agg := NewAggregator([]string{})

	before := time.Now()
	agg.Process(&imap.Message{
		To: []imap.Address{{Mailbox: "user", Host: "example.com"}},
	})

	results := agg.Results("TestMailbox")
	after := time.Now()

	assert.Equal(t, "TestMailbox", results.MailboxName)
	assert.True(t, results.ProcessedAt.After(before) || results.ProcessedAt.Equal(before))
	assert.True(t, results.ProcessedAt.Before(after) || results.ProcessedAt.Equal(after))
}

func TestSortByCount(t *testing.T) {
	stats := []AddressStat{
		{Address: "a@example.com", Count: 5},
		{Address: "b@example.com", Count: 10},
		{Address: "c@example.com", Count: 1},
	}

	// Sort descending
	SortByCount(stats, true)
	assert.Equal(t, 10, stats[0].Count)
	assert.Equal(t, 5, stats[1].Count)
	assert.Equal(t, 1, stats[2].Count)

	// Sort ascending
	SortByCount(stats, false)
	assert.Equal(t, 1, stats[0].Count)
	assert.Equal(t, 5, stats[1].Count)
	assert.Equal(t, 10, stats[2].Count)
}

func TestSortByAddress(t *testing.T) {
	stats := []AddressStat{
		{Address: "charlie@example.com", Count: 1},
		{Address: "alice@example.com", Count: 2},
		{Address: "bob@example.com", Count: 3},
	}

	// Sort ascending
	SortByAddress(stats, false)
	assert.Equal(t, "alice@example.com", stats[0].Address)
	assert.Equal(t, "bob@example.com", stats[1].Address)
	assert.Equal(t, "charlie@example.com", stats[2].Address)

	// Sort descending
	SortByAddress(stats, true)
	assert.Equal(t, "charlie@example.com", stats[0].Address)
	assert.Equal(t, "bob@example.com", stats[1].Address)
	assert.Equal(t, "alice@example.com", stats[2].Address)
}

func TestSortSpecialByCount(t *testing.T) {
	stats := []SpecialAddressStat{
		{ToAddress: "admin@example.com", FromAddress: "a@test.com", Count: 5},
		{ToAddress: "admin@example.com", FromAddress: "b@test.com", Count: 10},
		{ToAddress: "admin@example.com", FromAddress: "c@test.com", Count: 1},
	}

	SortSpecialByCount(stats, true)
	assert.Equal(t, 10, stats[0].Count)
	assert.Equal(t, 5, stats[1].Count)
	assert.Equal(t, 1, stats[2].Count)
}

func TestSortSpecialByFrom(t *testing.T) {
	stats := []SpecialAddressStat{
		{ToAddress: "admin@example.com", FromAddress: "charlie@test.com", Count: 1},
		{ToAddress: "admin@example.com", FromAddress: "alice@test.com", Count: 2},
		{ToAddress: "admin@example.com", FromAddress: "bob@test.com", Count: 3},
	}

	SortSpecialByFrom(stats, false)
	assert.Equal(t, "alice@test.com", stats[0].FromAddress)
	assert.Equal(t, "bob@test.com", stats[1].FromAddress)
	assert.Equal(t, "charlie@test.com", stats[2].FromAddress)
}
