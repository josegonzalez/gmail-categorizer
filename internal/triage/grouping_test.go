package triage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

func TestGrouping_DestinationFolder(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "simple address",
			address:  "user@example.com",
			expected: "automated/user",
		},
		{
			name:     "address with plus addressing",
			address:  "user+tag@example.com",
			expected: "automated/user+tag",
		},
		{
			name:     "address with subdomain",
			address:  "admin@mail.example.com",
			expected: "automated/admin",
		},
		{
			name:     "local part only",
			address:  "localonly",
			expected: "automated/localonly",
		},
		{
			name:     "empty address",
			address:  "",
			expected: "automated/",
		},
		{
			name:     "address with multiple @ signs",
			address:  "user@sub@example.com",
			expected: "automated/user@sub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Grouping{
				Address: tt.address,
				Count:   10,
				UIDs:    []uint32{1, 2, 3},
			}
			result := g.DestinationFolder()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGrouping_Fields(t *testing.T) {
	g := &Grouping{
		Address:  "test@example.com",
		Count:    5,
		Subject:  "",
		UIDs:     []uint32{1, 2, 3, 4, 5},
		Messages: nil,
	}

	assert.Equal(t, "test@example.com", g.Address)
	assert.Equal(t, 5, g.Count)
	assert.Equal(t, "", g.Subject)
	assert.Equal(t, 5, len(g.UIDs))
	assert.Nil(t, g.Messages)
}

func TestTriageResult(t *testing.T) {
	result := &TriageResult{
		ArchivedCount:     10,
		DestinationFolder: "automated/test",
	}

	assert.Equal(t, 10, result.ArchivedCount)
	assert.Equal(t, "automated/test", result.DestinationFolder)
}

func TestGrouping_HasUID(t *testing.T) {
	g := &Grouping{UIDs: []uint32{1, 5, 9}}
	assert.True(t, g.HasUID(5))
	assert.False(t, g.HasUID(2))
	assert.False(t, (&Grouping{}).HasUID(1))
}

func TestGrouping_GmailSearchQuery(t *testing.T) {
	byTo := &Grouping{Address: "notifications@example.com"}
	assert.Equal(t, "deliveredto:notifications@example.com in:inbox", byTo.GmailSearchQuery())

	byFrom := &Grouping{Address: "alerts@monitor.example", GroupedByFrom: true}
	assert.Equal(t, "from:alerts@monitor.example in:inbox", byFrom.GmailSearchQuery())
}

func TestGrouping_GmailSearchURL(t *testing.T) {
	g := &Grouping{Address: "user+tag@example.com"}

	assert.Equal(t,
		"https://mail.google.com/mail/?authuser=me%40example.com#search/deliveredto:user%2Btag@example.com%20in:inbox",
		g.GmailSearchURL("me@example.com"))
	assert.Equal(t,
		"https://mail.google.com/mail/#search/deliveredto:user%2Btag@example.com%20in:inbox",
		g.GmailSearchURL(""))
}

func TestGrouping_removeUIDs(t *testing.T) {
	g := &Grouping{
		Count:    3,
		UIDs:     []uint32{1, 2, 3},
		Messages: []*imap.Message{{UID: 3, Subject: "c"}, {UID: 2, Subject: "b"}, {UID: 1, Subject: "a"}},
	}

	g.removeUIDs([]uint32{1, 3})

	assert.Equal(t, 1, g.Count)
	assert.Equal(t, []uint32{2}, g.UIDs)
	require.Len(t, g.Messages, 1)
	assert.Equal(t, "b", g.Subject, "a single remaining message shows its subject")

	g.removeUIDs(nil)
	assert.Equal(t, 1, g.Count)
}
