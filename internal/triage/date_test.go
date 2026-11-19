package triage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShortDate(t *testing.T) {
	now := time.Date(2026, 11, 19, 15, 0, 0, 0, time.UTC)

	assert.Equal(t, "9:42 AM", ShortDate(time.Date(2026, 11, 19, 9, 42, 0, 0, time.UTC), now))
	assert.Equal(t, "Nov 18", ShortDate(time.Date(2026, 11, 18, 23, 0, 0, 0, time.UTC), now))
	assert.Equal(t, "Dec 31, 2025", ShortDate(time.Date(2025, 12, 31, 8, 0, 0, 0, time.UTC), now))
	assert.Equal(t, "", ShortDate(time.Time{}, now))
}

func TestShortDate_UsesNowLocation(t *testing.T) {
	loc := time.FixedZone("UTC-8", -8*60*60)
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, loc)
	// 03:00 UTC on Nov 20 is 7 PM on Nov 19 in UTC-8, so it is "today".
	assert.Equal(t, "7:00 PM", ShortDate(time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC), now))
}
