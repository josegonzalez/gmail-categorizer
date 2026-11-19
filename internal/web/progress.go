package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

// progressContentType is the media type of an archive progress stream: one
// JSON event per line.
const progressContentType = "application/x-ndjson"

// wantsProgress reports whether the request asked for a progress stream.
func wantsProgress(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), progressContentType)
}

// progressEvent is one line of an archive progress stream.
type progressEvent struct {
	// Type is "progress" while archiving and "done" once finished.
	Type          string `json:"type"`
	Current       string `json:"current,omitempty"`
	Groupings     int    `json:"groupings"`
	DoneGroupings int    `json:"doneGroupings"`
	Messages      int    `json:"messages"`
	DoneMessages  int    `json:"doneMessages"`
	// Redirect is where the page should go once done.
	Redirect string `json:"redirect,omitempty"`
}

// progressStream writes archive progress to the response as it happens.
// Write errors are ignored: if the browser goes away the archive carries on
// and the result is shown on the next page load.
type progressStream struct {
	enc  *json.Encoder
	rc   *http.ResponseController
	last progressEvent
}

func newProgressStream(w http.ResponseWriter) *progressStream {
	w.Header().Set("Content-Type", progressContentType)
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	s := &progressStream{enc: json.NewEncoder(w), rc: http.NewResponseController(w)}
	return s
}

func (s *progressStream) send(e progressEvent) {
	_ = s.enc.Encode(e)
	_ = s.rc.Flush()
}

func (s *progressStream) progress(p triage.BatchProgress) {
	s.last = progressEvent{
		Type:          "progress",
		Current:       p.Current,
		Groupings:     p.Groupings,
		DoneGroupings: p.DoneGroupings,
		Messages:      p.Messages,
		DoneMessages:  p.DoneMessages,
	}
	s.send(s.last)
}

func (s *progressStream) done(redirect string) {
	e := s.last
	e.Type = "done"
	e.Current = ""
	e.DoneGroupings = e.Groupings
	e.DoneMessages = e.Messages
	e.Redirect = redirect
	s.send(e)
}
