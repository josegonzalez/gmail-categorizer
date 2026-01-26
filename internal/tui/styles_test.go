package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/josegonzalez/gmail-categorizer/internal/tui/styles"
)

func TestSpinnerStyle(t *testing.T) {
	// Verify that SpinnerStyle is re-exported correctly from styles package
	assert.Equal(t, styles.SpinnerStyle, SpinnerStyle)

	// Verify it can render without panic
	result := SpinnerStyle.Render("test")
	assert.NotEmpty(t, result)
}
