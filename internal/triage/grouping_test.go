package triage

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		UIDs:     []uint32{1, 2, 3, 4, 5},
		Messages: nil,
	}

	assert.Equal(t, "test@example.com", g.Address)
	assert.Equal(t, 5, g.Count)
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
