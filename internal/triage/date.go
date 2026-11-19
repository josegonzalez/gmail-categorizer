package triage

import "time"

// ShortDate formats a message date the way mail clients list them: the time
// for messages from today, the month and day for this year, and the full date
// otherwise. A zero time returns an empty string.
func ShortDate(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	t = t.In(now.Location())
	switch {
	case t.Year() == now.Year() && t.YearDay() == now.YearDay():
		return t.Format("3:04 PM")
	case t.Year() == now.Year():
		return t.Format("Jan 2")
	default:
		return t.Format("Jan 2, 2006")
	}
}
