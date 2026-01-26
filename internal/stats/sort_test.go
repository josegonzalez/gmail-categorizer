package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortByInt(t *testing.T) {
	type item struct {
		name  string
		value int
	}

	tests := []struct {
		name     string
		items    []item
		desc     bool
		expected []int
	}{
		{
			name: "ascending order",
			items: []item{
				{name: "c", value: 30},
				{name: "a", value: 10},
				{name: "b", value: 20},
			},
			desc:     false,
			expected: []int{10, 20, 30},
		},
		{
			name: "descending order",
			items: []item{
				{name: "a", value: 10},
				{name: "c", value: 30},
				{name: "b", value: 20},
			},
			desc:     true,
			expected: []int{30, 20, 10},
		},
		{
			name:     "empty slice",
			items:    []item{},
			desc:     false,
			expected: []int{},
		},
		{
			name: "single item",
			items: []item{
				{name: "a", value: 42},
			},
			desc:     true,
			expected: []int{42},
		},
		{
			name: "equal values",
			items: []item{
				{name: "a", value: 5},
				{name: "b", value: 5},
				{name: "c", value: 5},
			},
			desc:     false,
			expected: []int{5, 5, 5},
		},
		{
			name: "negative values descending",
			items: []item{
				{name: "a", value: -10},
				{name: "b", value: 0},
				{name: "c", value: 10},
			},
			desc:     true,
			expected: []int{10, 0, -10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortByInt(tt.items, func(i item) int { return i.value }, tt.desc)

			result := make([]int, len(tt.items))
			for i, item := range tt.items {
				result[i] = item.value
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortByString(t *testing.T) {
	type item struct {
		name  string
		value int
	}

	tests := []struct {
		name     string
		items    []item
		desc     bool
		expected []string
	}{
		{
			name: "ascending order",
			items: []item{
				{name: "charlie", value: 3},
				{name: "alice", value: 1},
				{name: "bob", value: 2},
			},
			desc:     false,
			expected: []string{"alice", "bob", "charlie"},
		},
		{
			name: "descending order",
			items: []item{
				{name: "alice", value: 1},
				{name: "charlie", value: 3},
				{name: "bob", value: 2},
			},
			desc:     true,
			expected: []string{"charlie", "bob", "alice"},
		},
		{
			name:     "empty slice",
			items:    []item{},
			desc:     false,
			expected: []string{},
		},
		{
			name: "single item",
			items: []item{
				{name: "solo", value: 1},
			},
			desc:     true,
			expected: []string{"solo"},
		},
		{
			name: "equal strings",
			items: []item{
				{name: "same", value: 1},
				{name: "same", value: 2},
				{name: "same", value: 3},
			},
			desc:     false,
			expected: []string{"same", "same", "same"},
		},
		{
			name: "case sensitive sorting",
			items: []item{
				{name: "Zebra", value: 1},
				{name: "apple", value: 2},
				{name: "Apple", value: 3},
			},
			desc:     false,
			expected: []string{"Apple", "Zebra", "apple"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortByString(tt.items, func(i item) string { return i.name }, tt.desc)

			result := make([]string, len(tt.items))
			for i, item := range tt.items {
				result[i] = item.name
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}
