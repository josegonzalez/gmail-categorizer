// Package web serves a browser UI for the triage workflow.
package web

import (
	"context"
	"fmt"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

// DialFunc returns an IMAP client that is connected, logged in and has the
// inbox selected, along with the selected mailbox's info.
type DialFunc func(ctx context.Context) (imap.Client, *imap.MailboxInfo, error)

// TriagerFunc builds a Triager on top of a connected client.
type TriagerFunc func(client imap.Client) triage.Triager

// Session owns the IMAP connection behind the web UI and reconnects when the
// server drops it, which Gmail does to idle connections. It is not safe for
// concurrent use; the Server serializes access to it.
type Session struct {
	dial        DialFunc
	newTriager  TriagerFunc
	client      imap.Client
	triager     triage.Triager
	uidValidity uint32
}

// NewSession creates a session that connects with dial on first use.
func NewSession(dial DialFunc, newTriager TriagerFunc) *Session {
	return &Session{dial: dial, newTriager: newTriager}
}

// Ensure makes sure the session has a live connection, redialing if the
// connection has dropped. It reports whether cached UIDs are no longer valid,
// because the server's UIDVALIDITY changed across a reconnect.
func (s *Session) Ensure(ctx context.Context) (uidsInvalidated bool, err error) {
	if s.client != nil {
		if err := s.client.Noop(); err == nil {
			return false, nil
		}
		// The connection is gone; Logout would fail too.
		_ = s.client.Close()
		s.client = nil
		s.triager = nil
	}

	client, info, err := s.dial(ctx)
	if err != nil {
		return false, fmt.Errorf("connecting to IMAP server: %w", err)
	}

	previous := s.uidValidity
	s.client = client
	s.triager = s.newTriager(client)
	s.uidValidity = 0
	if info != nil {
		s.uidValidity = info.UIDValidity
	}

	return previous != 0 && previous != s.uidValidity, nil
}

// Triager returns the triager for the current connection. Call Ensure first.
func (s *Session) Triager() triage.Triager {
	return s.triager
}

// Close logs out and closes the connection, if there is one.
func (s *Session) Close() {
	if s.client == nil {
		return
	}
	_ = s.client.Logout()
	_ = s.client.Close()
	s.client = nil
	s.triager = nil
}
