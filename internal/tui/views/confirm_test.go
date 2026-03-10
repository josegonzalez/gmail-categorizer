package views

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

func TestRenderConfirm(t *testing.T) {
	grouping := &triage.Grouping{
		Address: "test@example.com",
		Count:   10,
		UIDs:    []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	result := RenderConfirm(grouping)

	// Check title
	assert.Contains(t, result, "Confirm Archive")

	// Check message count
	assert.Contains(t, result, "10 messages")

	// Check address
	assert.Contains(t, result, "test@example.com")

	// Check destination folder
	assert.Contains(t, result, "automated/test")

	// Check confirmation prompt
	assert.Contains(t, result, "Continue?")
	assert.Contains(t, result, "y/n")
}

func TestRenderConfirm_SingleMessage(t *testing.T) {
	grouping := &triage.Grouping{
		Address: "single@example.com",
		Count:   1,
		UIDs:    []uint32{1},
	}

	result := RenderConfirm(grouping)

	assert.Contains(t, result, "1 messages")
	assert.Contains(t, result, "single@example.com")
}

func TestRenderBatchConfirm(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "newsletter@example.com", Count: 15},
		{Address: "alerts@bank.com", Count: 8},
		{Address: "digest@medium.com", Count: 3},
	}

	result := RenderBatchConfirm(groupings)

	assert.Contains(t, result, "Confirm Batch Archive")
	assert.Contains(t, result, "26 messages from 3 groupings")
	assert.Contains(t, result, "newsletter@example.com")
	assert.Contains(t, result, "alerts@bank.com")
	assert.Contains(t, result, "digest@medium.com")
	assert.Contains(t, result, "automated/newsletter")
	assert.Contains(t, result, "automated/alerts")
	assert.Contains(t, result, "automated/digest")
	assert.Contains(t, result, "Continue?")
	assert.Contains(t, result, "y/n")
}

func TestRenderBatchConfirm_SingleGrouping(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "test@example.com", Count: 5},
	}

	result := RenderBatchConfirm(groupings)

	assert.Contains(t, result, "5 messages from 1 groupings")
	assert.Contains(t, result, "test@example.com")
}

func TestRenderBatchResult_AllSucceeded(t *testing.T) {
	results := []BatchResultEntry{
		{Address: "newsletter@example.com", ArchivedCount: 15, DestinationFolder: "automated/newsletter"},
		{Address: "alerts@bank.com", ArchivedCount: 8, DestinationFolder: "automated/alerts"},
		{Address: "digest@medium.com", ArchivedCount: 3, DestinationFolder: "automated/digest"},
	}

	output := RenderBatchResult(results)

	assert.Contains(t, output, "Batch Archive Complete")
	assert.Contains(t, output, "15 messages")
	assert.Contains(t, output, "automated/newsletter")
	assert.Contains(t, output, "8 messages")
	assert.Contains(t, output, "automated/alerts")
	assert.Contains(t, output, "3 messages")
	assert.Contains(t, output, "automated/digest")
	assert.Contains(t, output, "26 messages archived across 3 groupings")
	assert.Contains(t, output, "enter")
	assert.Contains(t, output, "continue")
}

func TestRenderBatchResult_PartialFailure(t *testing.T) {
	results := []BatchResultEntry{
		{Address: "newsletter@example.com", ArchivedCount: 15, DestinationFolder: "automated/newsletter"},
		{Address: "alerts@bank.com", Err: errors.New("connection timeout")},
		{Address: "digest@medium.com", ArchivedCount: 3, DestinationFolder: "automated/digest"},
	}

	output := RenderBatchResult(results)

	assert.Contains(t, output, "Batch Archive Complete")
	assert.Contains(t, output, "15 messages")
	assert.Contains(t, output, "✗ alerts@bank.com: connection timeout")
	assert.Contains(t, output, "3 messages")
	assert.Contains(t, output, "18 messages archived (2 succeeded, 1 failed)")
}

func TestRenderBatchResult_AllFailed(t *testing.T) {
	results := []BatchResultEntry{
		{Address: "a@example.com", Err: errors.New("error 1")},
		{Address: "b@example.com", Err: errors.New("error 2")},
	}

	output := RenderBatchResult(results)

	assert.Contains(t, output, "✗ a@example.com: error 1")
	assert.Contains(t, output, "✗ b@example.com: error 2")
	assert.Contains(t, output, "0 messages archived (0 succeeded, 2 failed)")
}

func TestRenderResult(t *testing.T) {
	result := &triage.TriageResult{
		ArchivedCount:     25,
		DestinationFolder: "automated/sender",
	}

	output := RenderResult(result)

	// Check title
	assert.Contains(t, output, "Archive Complete")

	// Check archived count
	assert.Contains(t, output, "25")

	// Check destination
	assert.Contains(t, output, "automated/sender")

	// Check help text
	assert.Contains(t, output, "enter")
	assert.Contains(t, output, "continue")
}

func TestRenderResult_ZeroArchived(t *testing.T) {
	result := &triage.TriageResult{
		ArchivedCount:     0,
		DestinationFolder: "automated/empty",
	}

	output := RenderResult(result)

	assert.Contains(t, output, "0")
	assert.Contains(t, output, "automated/empty")
}

func TestRenderError(t *testing.T) {
	err := errors.New("connection failed: timeout")

	result := RenderError(err)

	// Check title
	assert.Contains(t, result, "Error")

	// Check error message
	assert.Contains(t, result, "connection failed: timeout")

	// Check help text
	assert.Contains(t, result, "Press any key to exit")
}

func TestRenderError_LongMessage(t *testing.T) {
	err := errors.New("this is a very long error message that might wrap around the screen if it's too long to fit on a single line")

	result := RenderError(err)

	assert.Contains(t, result, "this is a very long error message")
}

func TestRenderLoading(t *testing.T) {
	spinnerView := "⣾"
	message := "Loading messages..."

	result := RenderLoading(spinnerView, message)

	assert.Contains(t, result, "⣾")
	assert.Contains(t, result, "Loading messages...")
}

func TestRenderLoading_EmptySpinner(t *testing.T) {
	result := RenderLoading("", "Processing...")

	assert.Contains(t, result, "Processing...")
}

func TestRenderLoading_DifferentMessages(t *testing.T) {
	messages := []string{
		"Loading...",
		"Connecting to server...",
		"Fetching emails...",
		"Archiving messages...",
	}

	for _, msg := range messages {
		result := RenderLoading("⣾", msg)
		assert.Contains(t, result, msg)
	}
}
