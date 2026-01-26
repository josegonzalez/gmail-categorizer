package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
	"github.com/josegonzalez/gmail-categorizer/internal/stats"
)

func createTestStats() *stats.Statistics {
	return &stats.Statistics{
		TotalMessages: 100,
		MailboxName:   "INBOX",
		ProcessedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		ByToAddress: []stats.AddressStat{
			{Address: "user1@example.com", Count: 50},
			{Address: "user2@example.com", Count: 30},
			{Address: "user3@example.com", Count: 20},
		},
		ByFromAddress: []stats.SpecialAddressStat{
			{ToAddress: "admin@example.com", FromAddress: "sender1@test.com", Count: 15},
			{ToAddress: "admin@example.com", FromAddress: "sender2@test.com", Count: 10},
		},
	}
}

func TestNewFormatter(t *testing.T) {
	opts := Options{SortBy: "count", SortOrder: "desc"}

	tests := []struct {
		format      string
		expectError bool
	}{
		{"table", false},
		{"json", false},
		{"csv", false},
		{"xml", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			f, err := NewFormatter(tt.format, opts)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, f)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, f)
			}
		})
	}
}

func TestTableFormatter(t *testing.T) {
	stats := createTestStats()
	formatter := NewTableFormatter(Options{
		SortBy:    config.SortByCount,
		SortOrder: config.SortDesc,
	})

	var buf bytes.Buffer
	err := formatter.Format(&buf, stats)

	require.NoError(t, err)
	output := buf.String()

	// Check header info
	assert.Contains(t, output, "INBOX")
	assert.Contains(t, output, "100")

	// Check address data
	assert.Contains(t, output, "user1@example.com")
	assert.Contains(t, output, "50")
	assert.Contains(t, output, "user2@example.com")
	assert.Contains(t, output, "30")

	// Check special address data
	assert.Contains(t, output, "admin@example.com")
	assert.Contains(t, output, "sender1@test.com")
	assert.Contains(t, output, "15")
}

func TestTableFormatter_EmptyStats(t *testing.T) {
	stats := &stats.Statistics{
		TotalMessages: 0,
		MailboxName:   "INBOX",
		ProcessedAt:   time.Now(),
		ByToAddress:   []stats.AddressStat{},
		ByFromAddress: []stats.SpecialAddressStat{},
	}

	formatter := NewTableFormatter(Options{})

	var buf bytes.Buffer
	err := formatter.Format(&buf, stats)

	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "INBOX")
	assert.Contains(t, output, "0")
}

func TestJSONFormatter(t *testing.T) {
	inputStats := createTestStats()
	formatter := NewJSONFormatter(Options{
		SortBy:    config.SortByCount,
		SortOrder: config.SortDesc,
	})

	var buf bytes.Buffer
	err := formatter.Format(&buf, inputStats)

	require.NoError(t, err)

	// Parse the output
	var output stats.Statistics
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, 100, output.TotalMessages)
	assert.Equal(t, "INBOX", output.MailboxName)
	assert.Equal(t, 3, len(output.ByToAddress))
	assert.Equal(t, 2, len(output.ByFromAddress))

	// Verify sorting (descending by count)
	assert.Equal(t, 50, output.ByToAddress[0].Count)
	assert.Equal(t, 30, output.ByToAddress[1].Count)
	assert.Equal(t, 20, output.ByToAddress[2].Count)
}

func TestJSONFormatter_Limit(t *testing.T) {
	inputStats := createTestStats()
	formatter := NewJSONFormatter(Options{
		SortBy:    config.SortByCount,
		SortOrder: config.SortDesc,
		Limit:     2,
	})

	var buf bytes.Buffer
	err := formatter.Format(&buf, inputStats)

	require.NoError(t, err)

	var output stats.Statistics
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, 2, len(output.ByToAddress))
	assert.Equal(t, 2, len(output.ByFromAddress))
}

func TestCSVFormatter(t *testing.T) {
	inputStats := createTestStats()
	formatter := NewCSVFormatter(Options{
		SortBy:    config.SortByCount,
		SortOrder: config.SortDesc,
	})

	var buf bytes.Buffer
	err := formatter.Format(&buf, inputStats)

	require.NoError(t, err)

	// Parse the CSV
	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Header + 3 regular + 2 special = 6 rows
	assert.Equal(t, 6, len(records))

	// Check header
	assert.Equal(t, []string{"type", "to_address", "from_address", "count"}, records[0])

	// Check first regular record (sorted by count desc)
	assert.Equal(t, "regular", records[1][0])
	assert.Equal(t, "user1@example.com", records[1][1])
	assert.Equal(t, "", records[1][2])
	assert.Equal(t, "50", records[1][3])

	// Check special record
	assert.Equal(t, "special", records[4][0])
	assert.Equal(t, "admin@example.com", records[4][1])
	assert.Contains(t, []string{"sender1@test.com", "sender2@test.com"}, records[4][2])
}

func TestCSVFormatter_EmptyStats(t *testing.T) {
	inputStats := &stats.Statistics{
		TotalMessages: 0,
		MailboxName:   "INBOX",
		ProcessedAt:   time.Now(),
		ByToAddress:   []stats.AddressStat{},
		ByFromAddress: []stats.SpecialAddressStat{},
	}

	formatter := NewCSVFormatter(Options{})

	var buf bytes.Buffer
	err := formatter.Format(&buf, inputStats)

	require.NoError(t, err)

	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Only header
	assert.Equal(t, 1, len(records))
}

func TestFormatter_SortByAddress(t *testing.T) {
	inputStats := &stats.Statistics{
		TotalMessages: 3,
		MailboxName:   "INBOX",
		ProcessedAt:   time.Now(),
		ByToAddress: []stats.AddressStat{
			{Address: "charlie@example.com", Count: 10},
			{Address: "alice@example.com", Count: 20},
			{Address: "bob@example.com", Count: 15},
		},
	}

	formatter := NewJSONFormatter(Options{
		SortBy:    config.SortByAddress,
		SortOrder: config.SortAsc,
	})

	var buf bytes.Buffer
	err := formatter.Format(&buf, inputStats)

	require.NoError(t, err)

	var output stats.Statistics
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	// Should be sorted alphabetically
	assert.Equal(t, "alice@example.com", output.ByToAddress[0].Address)
	assert.Equal(t, "bob@example.com", output.ByToAddress[1].Address)
	assert.Equal(t, "charlie@example.com", output.ByToAddress[2].Address)
}

func TestFormatter_DoesNotModifyOriginal(t *testing.T) {
	inputStats := createTestStats()
	originalOrder := make([]stats.AddressStat, len(inputStats.ByToAddress))
	copy(originalOrder, inputStats.ByToAddress)

	formatter := NewJSONFormatter(Options{
		SortBy:    config.SortByAddress,
		SortOrder: config.SortAsc,
		Limit:     1,
	})

	var buf bytes.Buffer
	err := formatter.Format(&buf, inputStats)
	require.NoError(t, err)

	// Original should be unchanged
	assert.Equal(t, len(originalOrder), len(inputStats.ByToAddress))
	for i, stat := range originalOrder {
		assert.Equal(t, stat.Address, inputStats.ByToAddress[i].Address)
		assert.Equal(t, stat.Count, inputStats.ByToAddress[i].Count)
	}
}

