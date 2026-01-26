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
