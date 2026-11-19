package triage

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

	SortMessages(messages, SortDateDesc)

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

	SortMessages(messages, SortDateAsc)

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

	SortMessages(messages, SortSubjectAsc)

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

	SortMessages(messages, SortSubjectDesc)

	assert.Equal(t, "Charlie", messages[0].Subject)
	assert.Equal(t, "Bravo", messages[1].Subject)
	assert.Equal(t, "alpha", messages[2].Subject)
}

func TestSortMessages_Empty(t *testing.T) {
	messages := []*imap.Message{}
	SortMessages(messages, SortDateDesc)
	assert.Empty(t, messages)
}

func TestSortMessages_Single(t *testing.T) {
	messages := []*imap.Message{
		{Subject: "Only"},
	}
	SortMessages(messages, SortSubjectAsc)
	assert.Equal(t, "Only", messages[0].Subject)
}

func TestSortMode_KeyRoundTrip(t *testing.T) {
	modes := []SortMode{SortDateDesc, SortDateAsc, SortSubjectAsc, SortSubjectDesc}
	keys := []string{"date-desc", "date-asc", "subject-asc", "subject-desc"}
	for i, mode := range modes {
		assert.Equal(t, keys[i], mode.Key())
		assert.Equal(t, mode, ParseSortMode(mode.Key()))
	}
}

func TestParseSortMode_UnknownDefaultsToDateDesc(t *testing.T) {
	assert.Equal(t, SortDateDesc, ParseSortMode(""))
	assert.Equal(t, SortDateDesc, ParseSortMode("bogus"))
}

func TestSortMode_Next(t *testing.T) {
	assert.Equal(t, SortDateAsc, SortDateDesc.Next())
	assert.Equal(t, SortSubjectAsc, SortDateAsc.Next())
	assert.Equal(t, SortSubjectDesc, SortSubjectAsc.Next())
	assert.Equal(t, SortDateDesc, SortSubjectDesc.Next())
}
