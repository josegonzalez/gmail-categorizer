package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortModeLabel(t *testing.T) {
	assert.Equal(t, "date ↓", sortModeLabel(SortDateDesc))
	assert.Equal(t, "date ↑", sortModeLabel(SortDateAsc))
	assert.Equal(t, "subject A→Z", sortModeLabel(SortSubjectAsc))
	assert.Equal(t, "subject Z→A", sortModeLabel(SortSubjectDesc))
}
