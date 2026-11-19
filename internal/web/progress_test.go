package web

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

func (e *testEnv) postProgress(t *testing.T, form url.Values) (*httptest.ResponseRecorder, []progressEvent) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/archive", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Accept", progressContentType)
	rec := e.do(t, req)

	var events []progressEvent
	scanner := bufio.NewScanner(strings.NewReader(rec.Body.String()))
	for scanner.Scan() {
		var ev progressEvent
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &ev), scanner.Text())
		events = append(events, ev)
	}
	return rec, events
}

// bigInbox has n messages to one address, so archiving it takes several
// chunks.
func bigInbox(n int) []*imap.Message {
	msgs := make([]*imap.Message, 0, n)
	for i := 1; i <= n; i++ {
		msgs = append(msgs, &imap.Message{
			UID:     uint32(i),
			Subject: fmt.Sprintf("Update %d", i),
			Date:    testNow.Add(-time.Duration(i) * time.Minute),
			To:      addr("updates", "example.com"),
		})
	}
	return msgs
}

func TestArchive_ProgressStream(t *testing.T) {
	client := &mockClient{messages: append(bigInbox(60), testInbox()...)}
	env := newTestEnv(t, client)

	rec, events := env.postProgress(t, url.Values{
		"address": {"updates@example.com", "receipts@example.com"},
		"filter":  {"by-sender"},
	})

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, progressContentType, rec.Header().Get("Content-Type"))
	require.GreaterOrEqual(t, len(events), 4)

	first := events[0]
	assert.Equal(t, "progress", first.Type)
	assert.Equal(t, 2, first.Groupings)
	assert.Equal(t, 61, first.Messages)
	assert.Equal(t, 0, first.DoneMessages)

	// 60 messages move in chunks of 25, so the bar moves within the grouping.
	var seen []int
	for _, ev := range events {
		if ev.Type == "progress" && ev.Current == "updates@example.com" {
			seen = append(seen, ev.DoneMessages)
		}
	}
	assert.Contains(t, seen, 25)
	assert.Contains(t, seen, 50)
	for i := 1; i < len(events); i++ {
		assert.GreaterOrEqual(t, events[i].DoneMessages, events[i-1].DoneMessages, "progress never goes backwards")
	}

	last := events[len(events)-1]
	assert.Equal(t, "done", last.Type)
	assert.Equal(t, "/?filter=by-sender", last.Redirect)
	assert.Equal(t, 61, last.DoneMessages)
	assert.Equal(t, 2, last.DoneGroupings)

	assert.Len(t, client.archived["automated/updates"], 60)
	assert.Nil(t, env.server.findGrouping("updates@example.com"))
	assert.Contains(t, env.get(t, "/").Body.String(), "Archived 2 groupings (61 messages) to automated/updates, automated/receipts")
}

func TestArchive_ProgressStream_PartialFailureKeepsRemainder(t *testing.T) {
	client := &mockClient{messages: bigInbox(60), failMoveCall: 2}
	env := newTestEnv(t, client)

	rec, events := env.postProgress(t, url.Values{"address": {"updates@example.com"}})

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "done", events[len(events)-1].Type)

	// The first chunk of 25 moved; the other 35 are still in the inbox.
	g := env.server.findGrouping("updates@example.com")
	require.NotNil(t, g)
	assert.Equal(t, 35, g.Count)
	assert.Len(t, g.Messages, 35)

	body := env.get(t, "/").Body.String()
	assert.Contains(t, body, "Could not archive updates@example.com")
	assert.Contains(t, body, "25 messages were moved to automated/updates before the error")
	assert.Contains(t, body, "updates@example.com")
	assert.Contains(t, body, "1 groupings, 35 messages")
}

func TestArchive_ProgressStream_NothingToArchive(t *testing.T) {
	env := defaultEnv(t)

	rec, events := env.postProgress(t, url.Values{"address": {"gone@example.com"}})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, events, 1)
	assert.Equal(t, "done", events[0].Type)
	assert.Equal(t, "/", events[0].Redirect)
	assert.Contains(t, env.get(t, "/").Body.String(), "gone@example.com is no longer in the inbox.")
}

func TestArchive_ProgressStream_ErrorBeforeArchivingIsAPage(t *testing.T) {
	// The page falls back to a normal form post when it gets anything other
	// than a progress stream, so errors before archiving starts stay HTML.
	env := defaultEnv(t)
	env.client.noopErr = errors.New("connection closed")
	env.server.session.dial = func(ctx context.Context) (imap.Client, *imap.MailboxInfo, error) {
		return nil, nil, errors.New("login failed")
	}

	rec, _ := env.postProgressRaw(t, url.Values{"address": {"receipts@example.com"}})

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Empty(t, env.client.archived)
}

func (e *testEnv) postProgressRaw(t *testing.T, form url.Values) (*httptest.ResponseRecorder, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/archive", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Accept", progressContentType)
	rec := e.do(t, req)
	return rec, rec.Body.String()
}

func TestArchive_WithoutProgressStillRedirects(t *testing.T) {
	env := defaultEnv(t)

	rec := env.post(t, "/archive", url.Values{"address": {"receipts@example.com"}})

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/", rec.Header().Get("Location"))
	assert.NotEqual(t, progressContentType, rec.Header().Get("Content-Type"))
}

func TestProgressMarkupInLayout(t *testing.T) {
	env := defaultEnv(t)

	for _, target := range []string{"/", "/groupings/notifications@example.com"} {
		body := env.get(t, target).Body.String()
		assert.Contains(t, body, `role="progressbar"`, target)
		assert.Contains(t, body, `data-progress-status`, target)
	}
}
