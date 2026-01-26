package views

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateVisibleRange(t *testing.T) {
	tests := []struct {
		name         string
		totalItems   int
		cursor       int
		height       int
		headerOffset int
		expected     VisibleRange
	}{
		{
			name:         "basic case - cursor at beginning",
			totalItems:   100,
			cursor:       0,
			height:       20,
			headerOffset: 5,
			expected:     VisibleRange{Start: 0, End: 15},
		},
		{
			name:         "cursor in middle of list",
			totalItems:   100,
			cursor:       50,
			height:       20,
			headerOffset: 5,
			expected:     VisibleRange{Start: 36, End: 51},
		},
		{
			name:         "cursor near end of list",
			totalItems:   100,
			cursor:       98,
			height:       20,
			headerOffset: 5,
			expected:     VisibleRange{Start: 84, End: 99},
		},
		{
			name:         "cursor at last item",
			totalItems:   100,
			cursor:       99,
			height:       20,
			headerOffset: 5,
			expected:     VisibleRange{Start: 85, End: 100},
		},
		{
			name:         "small list - fewer items than height",
			totalItems:   5,
			cursor:       2,
			height:       20,
			headerOffset: 5,
			expected:     VisibleRange{Start: 0, End: 5},
		},
		{
			name:         "single item",
			totalItems:   1,
			cursor:       0,
			height:       20,
			headerOffset: 5,
			expected:     VisibleRange{Start: 0, End: 1},
		},
		{
			name:         "empty list",
			totalItems:   0,
			cursor:       0,
			height:       20,
			headerOffset: 5,
			expected:     VisibleRange{Start: 0, End: 0},
		},
		{
			name:         "very small height - uses minimum",
			totalItems:   100,
			cursor:       0,
			height:       3,
			headerOffset: 5,
			expected:     VisibleRange{Start: 0, End: 10},
		},
		{
			name:         "height equals header offset - uses minimum",
			totalItems:   100,
			cursor:       0,
			height:       5,
			headerOffset: 5,
			expected:     VisibleRange{Start: 0, End: 10},
		},
		{
			name:         "negative visible items - uses minimum",
			totalItems:   100,
			cursor:       0,
			height:       2,
			headerOffset: 5,
			expected:     VisibleRange{Start: 0, End: 10},
		},
		{
			name:         "cursor exactly at visible boundary",
			totalItems:   50,
			cursor:       15,
			height:       20,
			headerOffset: 5,
			expected:     VisibleRange{Start: 1, End: 16},
		},
		{
			name:         "large offset",
			totalItems:   100,
			cursor:       50,
			height:       30,
			headerOffset: 10,
			expected:     VisibleRange{Start: 31, End: 51},
		},
		{
			name:         "zero header offset",
			totalItems:   100,
			cursor:       25,
			height:       20,
			headerOffset: 0,
			expected:     VisibleRange{Start: 6, End: 26},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateVisibleRange(tt.totalItems, tt.cursor, tt.height, tt.headerOffset)
			assert.Equal(t, tt.expected.Start, result.Start, "Start mismatch")
			assert.Equal(t, tt.expected.End, result.End, "End mismatch")
		})
	}
}

func TestVisibleRange_Invariants(t *testing.T) {
	// Test invariants that should always hold
	testCases := []struct {
		totalItems   int
		cursor       int
		height       int
		headerOffset int
	}{
		{100, 0, 20, 5},
		{100, 50, 20, 5},
		{100, 99, 20, 5},
		{10, 5, 20, 5},
		{1, 0, 20, 5},
		{0, 0, 20, 5},
	}

	for _, tc := range testCases {
		result := CalculateVisibleRange(tc.totalItems, tc.cursor, tc.height, tc.headerOffset)

		// Start should never be negative
		assert.GreaterOrEqual(t, result.Start, 0, "Start should never be negative")

		// End should never exceed total items
		assert.LessOrEqual(t, result.End, tc.totalItems, "End should not exceed total items")

		// Start should be less than or equal to End
		assert.LessOrEqual(t, result.Start, result.End, "Start should be <= End")

		// If there are items and cursor is valid, cursor should be in visible range
		if tc.totalItems > 0 && tc.cursor < tc.totalItems {
			assert.GreaterOrEqual(t, tc.cursor, result.Start, "Cursor should be >= Start")
			assert.Less(t, tc.cursor, result.End, "Cursor should be < End")
		}
	}
}
