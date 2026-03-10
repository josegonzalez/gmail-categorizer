package views

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

func TestRenderSubjects(t *testing.T) {
	messages := []*imap.Message{
		{Subject: "First Subject"},
		{Subject: "Second Subject"},
		{Subject: "Third Subject"},
	}

	result := RenderSubjects("test@example.com", messages, 1, 80, 24, "date ↓")

	// Check title
	assert.Contains(t, result, "Messages for test@example.com")

	// Check count
	assert.Contains(t, result, "3 messages")

	// Check subjects
	assert.Contains(t, result, "First Subject")
	assert.Contains(t, result, "Second Subject")
	assert.Contains(t, result, "Third Subject")

	// Check sort indicator
	assert.Contains(t, result, "sorted by date ↓")

	// Check help text
	assert.Contains(t, result, "navigate")
	assert.Contains(t, result, "s sort")
	assert.Contains(t, result, "archive")
	assert.Contains(t, result, "back")
}

func TestRenderSubjects_SortIndicator(t *testing.T) {
	messages := []*imap.Message{
		{Subject: "Test"},
	}

	result := RenderSubjects("test@example.com", messages, 0, 80, 24, "subject A→Z")
	assert.Contains(t, result, "sorted by subject A→Z")
}

func TestRenderSubjects_NoSubject(t *testing.T) {
	messages := []*imap.Message{
		{Subject: ""},
	}

	result := RenderSubjects("test@example.com", messages, 0, 80, 24, "date ↓")

	assert.Contains(t, result, "(no subject)")
}

func TestRenderSubjects_Truncation(t *testing.T) {
	longSubject := strings.Repeat("a", 200)
	messages := []*imap.Message{
		{Subject: longSubject},
	}

	result := RenderSubjects("test@example.com", messages, 0, 80, 24, "date ↓")

	// Should contain truncation indicator
	assert.Contains(t, result, "...")
	// Should not contain the full subject
	assert.NotContains(t, result, longSubject)
}

func TestRenderSubjects_EmptyMessages(t *testing.T) {
	messages := []*imap.Message{}

	result := RenderSubjects("test@example.com", messages, 0, 80, 24, "date ↓")

	assert.Contains(t, result, "0 messages")
}

func TestRenderSubjects_SmallWidth(t *testing.T) {
	messages := []*imap.Message{
		{Subject: "A moderately long subject line"},
	}

	// Very small width - should use minimum
	result := RenderSubjects("test@example.com", messages, 0, 10, 24, "date ↓")
	assert.Contains(t, result, "Messages for test@example.com")
}

func TestRenderSubjects_CursorPositions(t *testing.T) {
	messages := []*imap.Message{
		{Subject: "First"},
		{Subject: "Second"},
		{Subject: "Third"},
	}

	// All cursor positions should render without error
	for i := 0; i < 3; i++ {
		result := RenderSubjects("test@example.com", messages, i, 80, 24, "date ↓")
		assert.NotEmpty(t, result)
	}
}
