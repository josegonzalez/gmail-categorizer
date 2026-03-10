package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

func TestSortMessages_DateDesc(t *testing.T) {
	now := time.Now()
	messages := []*imap.Message{
		{Subject: "Old", Date: now.Add(-2 * time.Hour)},
		{Subject: "New", Date: now},
		{Subject: "Mid", Date: now.Add(-1 * time.Hour)},
	}

	sortMessages(messages, SortDateDesc)

	assert.Equal(t, "New", messages[0].Subject)
	assert.Equal(t, "Mid", messages[1].Subject)
	assert.Equal(t, "Old", messages[2].Subject)
}

func TestSortMessages_DateAsc(t *testing.T) {
	now := time.Now()
	messages := []*imap.Message{
		{Subject: "New", Date: now},
		{Subject: "Old", Date: now.Add(-2 * time.Hour)},
		{Subject: "Mid", Date: now.Add(-1 * time.Hour)},
	}

	sortMessages(messages, SortDateAsc)

	assert.Equal(t, "Old", messages[0].Subject)
	assert.Equal(t, "Mid", messages[1].Subject)
	assert.Equal(t, "New", messages[2].Subject)
}

func TestSortMessages_SubjectAsc(t *testing.T) {
	messages := []*imap.Message{
		{Subject: "Charlie"},
		{Subject: "alpha"},
		{Subject: "Bravo"},
	}

	sortMessages(messages, SortSubjectAsc)

	assert.Equal(t, "alpha", messages[0].Subject)
	assert.Equal(t, "Bravo", messages[1].Subject)
	assert.Equal(t, "Charlie", messages[2].Subject)
}

func TestSortMessages_SubjectDesc(t *testing.T) {
	messages := []*imap.Message{
		{Subject: "alpha"},
		{Subject: "Charlie"},
		{Subject: "Bravo"},
	}

	sortMessages(messages, SortSubjectDesc)

	assert.Equal(t, "Charlie", messages[0].Subject)
	assert.Equal(t, "Bravo", messages[1].Subject)
	assert.Equal(t, "alpha", messages[2].Subject)
}

func TestSortMessages_Empty(t *testing.T) {
	messages := []*imap.Message{}
	sortMessages(messages, SortDateDesc)
	assert.Empty(t, messages)
}

func TestSortMessages_Single(t *testing.T) {
	messages := []*imap.Message{
		{Subject: "Only"},
	}
	sortMessages(messages, SortSubjectAsc)
	assert.Equal(t, "Only", messages[0].Subject)
}

func TestSortModeLabel(t *testing.T) {
	assert.Equal(t, "date ↓", sortModeLabel(SortDateDesc))
	assert.Equal(t, "date ↑", sortModeLabel(SortDateAsc))
	assert.Equal(t, "subject A→Z", sortModeLabel(SortSubjectAsc))
	assert.Equal(t, "subject Z→A", sortModeLabel(SortSubjectDesc))
}
