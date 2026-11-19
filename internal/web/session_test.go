package web

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

func newTestSession(clients []*mockClient, validities []uint32, dialErr *error) (*Session, *int) {
	dials := 0
	s := NewSession(
		func(ctx context.Context) (imap.Client, *imap.MailboxInfo, error) {
			if dialErr != nil && *dialErr != nil {
				return nil, nil, *dialErr
			}
			i := min(dials, len(clients)-1)
			dials++
			return clients[i], &imap.MailboxInfo{Name: "INBOX", UIDValidity: validities[i]}, nil
		},
		func(c imap.Client) triage.Triager { return triage.NewTriager(c, nil) },
	)
	return s, &dials
}

func TestSession_DialsOnFirstUse(t *testing.T) {
	s, dials := newTestSession([]*mockClient{{}}, []uint32{1}, nil)

	invalidated, err := s.Ensure(context.Background())

	require.NoError(t, err)
	assert.False(t, invalidated)
	assert.Equal(t, 1, *dials)
	assert.NotNil(t, s.Triager())
}

func TestSession_ReusesLiveConnection(t *testing.T) {
	s, dials := newTestSession([]*mockClient{{}}, []uint32{1}, nil)

	_, err := s.Ensure(context.Background())
	require.NoError(t, err)
	invalidated, err := s.Ensure(context.Background())

	require.NoError(t, err)
	assert.False(t, invalidated)
	assert.Equal(t, 1, *dials)
}

func TestSession_RedialsWhenNoopFails(t *testing.T) {
	first := &mockClient{}
	second := &mockClient{}
	s, dials := newTestSession([]*mockClient{first, second}, []uint32{7, 7}, nil)

	_, err := s.Ensure(context.Background())
	require.NoError(t, err)

	first.noopErr = errors.New("connection closed")
	invalidated, err := s.Ensure(context.Background())

	require.NoError(t, err)
	assert.False(t, invalidated, "same UIDVALIDITY keeps cached UIDs valid")
	assert.Equal(t, 2, *dials)
	assert.True(t, first.closed, "the dead connection is closed")
}

func TestSession_ReportsUIDValidityChange(t *testing.T) {
	first := &mockClient{}
	s, _ := newTestSession([]*mockClient{first, {}}, []uint32{7, 8}, nil)

	_, err := s.Ensure(context.Background())
	require.NoError(t, err)

	first.noopErr = errors.New("connection closed")
	invalidated, err := s.Ensure(context.Background())

	require.NoError(t, err)
	assert.True(t, invalidated)
}

func TestSession_DialError(t *testing.T) {
	dialErr := errors.New("no route to host")
	s, _ := newTestSession([]*mockClient{{}}, []uint32{1}, &dialErr)

	_, err := s.Ensure(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, dialErr)
	assert.Nil(t, s.Triager())
}

func TestSession_Close(t *testing.T) {
	client := &mockClient{}
	s, _ := newTestSession([]*mockClient{client}, []uint32{1}, nil)
	_, err := s.Ensure(context.Background())
	require.NoError(t, err)

	s.Close()
	assert.True(t, client.closed)
	assert.Nil(t, s.Triager())

	// Closing again is a no-op.
	s.Close()
}

func TestServer_ReconnectReloadsWhenUIDsInvalidated(t *testing.T) {
	env := defaultEnv(t)
	require.Equal(t, 1, env.client.fetchCalls)

	// The connection drops; the redial reports a new UIDVALIDITY.
	replacement := &mockClient{messages: testInbox()[:1]}
	old := env.client
	env.client = replacement
	old.noopErr = errors.New("connection closed")
	env.server.session.dial = func(ctx context.Context) (imap.Client, *imap.MailboxInfo, error) {
		return replacement, &imap.MailboxInfo{Name: "INBOX", UIDValidity: 2}, nil
	}

	rec := env.get(t, "/")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, replacement.fetchCalls, "groupings reload after UIDVALIDITY changes")
	assert.NotContains(t, rec.Body.String(), "receipts@example.com")
}

func TestServer_ReconnectKeepsCacheWhenUIDsValid(t *testing.T) {
	env := defaultEnv(t)

	replacement := &mockClient{messages: testInbox()}
	old := env.client
	old.noopErr = errors.New("connection closed")
	env.server.session.dial = func(ctx context.Context) (imap.Client, *imap.MailboxInfo, error) {
		return replacement, &imap.MailboxInfo{Name: "INBOX", UIDValidity: 1}, nil
	}

	rec := env.get(t, "/")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, replacement.fetchCalls)
	assert.Contains(t, rec.Body.String(), "receipts@example.com")
}

func TestServer_DialErrorShowsErrorPage(t *testing.T) {
	env := defaultEnv(t)
	env.client.noopErr = errors.New("connection closed")
	env.server.session.dial = func(ctx context.Context) (imap.Client, *imap.MailboxInfo, error) {
		return nil, nil, errors.New("login failed")
	}

	rec := env.get(t, "/")

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Contains(t, rec.Body.String(), "login failed")
}
