package views

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

func TestRenderGroupings(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "user1@example.com", Count: 50},
		{Address: "user2@example.com", Count: 30},
		{Address: "user3@example.com", Count: 10},
	}

	result := RenderGroupings(groupings, 1, 80, 24)

	// Check title
	assert.Contains(t, result, "Email Groupings")

	// Check count display
	assert.Contains(t, result, "3 groupings")

	// Check addresses
	assert.Contains(t, result, "user1@example.com")
	assert.Contains(t, result, "user2@example.com")
	assert.Contains(t, result, "user3@example.com")

	// Check counts
	assert.Contains(t, result, "50")
	assert.Contains(t, result, "30")
	assert.Contains(t, result, "10")

	// Check help text
	assert.Contains(t, result, "navigate")
	assert.Contains(t, result, "quit")
}

func TestRenderGroupings_Empty(t *testing.T) {
	groupings := []*triage.Grouping{}

	result := RenderGroupings(groupings, 0, 80, 24)

	assert.Contains(t, result, "Email Groupings")
	assert.Contains(t, result, "0 groupings")
}

func TestRenderGroupings_SingleItem(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "only@example.com", Count: 100},
	}

	result := RenderGroupings(groupings, 0, 80, 24)

	assert.Contains(t, result, "only@example.com")
	assert.Contains(t, result, "100")
	assert.Contains(t, result, "1 groupings")
}

func TestRenderGroupings_CursorAtDifferentPositions(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "a@example.com", Count: 1},
		{Address: "b@example.com", Count: 2},
		{Address: "c@example.com", Count: 3},
	}

	// Cursor at beginning
	result := RenderGroupings(groupings, 0, 80, 24)
	assert.Contains(t, result, "a@example.com")

	// Cursor in middle
	result = RenderGroupings(groupings, 1, 80, 24)
	assert.Contains(t, result, "b@example.com")

	// Cursor at end
	result = RenderGroupings(groupings, 2, 80, 24)
	assert.Contains(t, result, "c@example.com")
}

func TestRenderGroupings_SmallHeight(t *testing.T) {
	groupings := []*triage.Grouping{
		{Address: "a@example.com", Count: 1},
		{Address: "b@example.com", Count: 2},
	}

	// Very small height
	result := RenderGroupings(groupings, 0, 80, 5)
	assert.Contains(t, result, "Email Groupings")
}
