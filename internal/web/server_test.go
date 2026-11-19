package web

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/internal/triage"
)

var testNow = time.Date(2026, 11, 19, 15, 0, 0, 0, time.UTC)

func addr(mailbox, host string) []imap.Address {
	return []imap.Address{{Mailbox: mailbox, Host: host}}
}

func testInbox() []*imap.Message {
	builds := []imap.Address{{Name: "Build Service", Mailbox: "builds", Host: "service.example"}}
	return []*imap.Message{
		{UID: 1, Subject: "Build #1 passed", Date: testNow.Add(-48 * time.Hour), From: builds, To: addr("notifications", "example.com")},
		{UID: 2, Subject: "Build #3 passed", Date: testNow.Add(-1 * time.Hour), From: builds, To: addr("notifications", "example.com")},
		{UID: 3, Subject: "Build #2 failed", Date: testNow.Add(-24 * time.Hour), From: builds, To: addr("notifications", "example.com")},
		{UID: 4, Subject: "Your receipt", Date: testNow.Add(-2 * time.Hour), From: addr("shop", "store.example"), To: addr("receipts", "example.com")},
		{UID: 5, Subject: "[Firing] latency", Date: testNow.Add(-3 * time.Hour), From: addr("alerts", "monitor.example"), To: addr("admin", "example.com")},
		{UID: 6, Subject: "[Resolved] latency", Date: testNow.Add(-30 * time.Minute), From: addr("alerts", "monitor.example"), To: addr("admin", "example.com")},
	}
}

type testEnv struct {
	server  *Server
	handler http.Handler
	client  *mockClient
	dials   int
}

func newTestEnv(t *testing.T, client *mockClient) *testEnv {
	t.Helper()
	env := &testEnv{client: client}
	session := NewSession(
		func(ctx context.Context) (imap.Client, *imap.MailboxInfo, error) {
			env.dials++
			info, _ := env.client.SelectMailbox("INBOX")
			info.UIDValidity = 1
			return env.client, info, nil
		},
		func(c imap.Client) triage.Triager { return triage.NewTriager(c, []string{"alerts@"}) },
	)
	server, err := NewServer(session, Options{
		Username: "me@example.com",
		Now:      func() time.Time { return testNow },
		Logger:   log.New(io.Discard, "", 0),
	})
	require.NoError(t, err)
	require.NoError(t, server.Load(context.Background()))
	env.server = server
	env.handler = server.Handler()
	return env
}

func defaultEnv(t *testing.T) *testEnv {
	return newTestEnv(t, &mockClient{messages: testInbox()})
}

func (e *testEnv) do(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, httptest.NewRequest(http.MethodGet, target, nil))
}

func (e *testEnv) post(t *testing.T, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	return e.do(t, req)
}

func TestGroupingsPage(t *testing.T) {
	env := defaultEnv(t)

	rec := env.get(t, "/")
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, body, "me@example.com")
	assert.Contains(t, body, "3 groupings, 6 messages")
	assert.Contains(t, body, "All 3")
	assert.Contains(t, body, "By sender 1")

	// Rows with folder, count and by-sender marker.
	assert.Contains(t, body, "notifications@example.com")
	assert.Contains(t, body, "automated/notifications")
	assert.Contains(t, body, "alerts@monitor.example")
	assert.Contains(t, body, "by sender")

	// Every subject is listed under its grouping, newest first, with dates
	// and links to the message pages.
	newest := strings.Index(body, "Build #3 passed")
	middle := strings.Index(body, "Build #2 failed")
	oldest := strings.Index(body, "Build #1 passed")
	require.True(t, newest > 0 && middle > 0 && oldest > 0)
	assert.Less(t, newest, middle)
	assert.Less(t, middle, oldest)
	assert.Contains(t, body, "2:00 PM")
	assert.Contains(t, body, "Nov 17")
	assert.Contains(t, body, `href="/groupings/notifications@example.com/messages/2"`)
}

func TestGroupingsPage_FilterBySender(t *testing.T) {
	env := defaultEnv(t)

	body := env.get(t, "/?filter=by-sender").Body.String()

	assert.Contains(t, body, "alerts@monitor.example")
	assert.NotContains(t, body, "receipts@example.com")
	assert.NotContains(t, body, "Build #3 passed")
	// Counts in the filter cover every grouping, not just the shown ones.
	assert.Contains(t, body, "All 3")
	assert.Contains(t, body, `<input type="hidden" name="filter" value="by-sender">`)
}

func TestGroupingsPage_UnknownFilterShowsAll(t *testing.T) {
	env := defaultEnv(t)

	body := env.get(t, "/?filter=bogus").Body.String()

	assert.Contains(t, body, "receipts@example.com")
	assert.Contains(t, body, "alerts@monitor.example")
}

func TestGroupingsPage_EmptyInbox(t *testing.T) {
	env := newTestEnv(t, &mockClient{})

	rec := env.get(t, "/")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "The inbox is empty.")
}

func TestGroupingPage_SortsWithoutChangingIndexOrder(t *testing.T) {
	env := defaultEnv(t)

	rec := env.get(t, "/groupings/notifications@example.com?sort=subject-asc")
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	first := strings.Index(body, "Build #1 passed")
	second := strings.Index(body, "Build #2 failed")
	third := strings.Index(body, "Build #3 passed")
	assert.Less(t, first, second)
	assert.Less(t, second, third)
	assert.Contains(t, body, `<option value="subject-asc" selected>`)
	assert.Contains(t, body, "Archive all 3")
	assert.Contains(t, body, "Grouped by recipient")
	assert.Contains(t, body, "Build Service")
	// Message links keep the sort so prev/next follow it.
	assert.Contains(t, body, `/groupings/notifications@example.com/messages/1?sort=subject-asc`)
	// s cycles to the next sort.
	assert.Contains(t, body, `href="/groupings/notifications@example.com?sort=subject-desc"`)

	// The cached order used by the index is still newest first.
	g := env.server.findGrouping("notifications@example.com")
	assert.Equal(t, uint32(2), g.Messages[0].UID)
}

func TestGroupingPage_GmailLink(t *testing.T) {
	env := defaultEnv(t)

	body := env.get(t, "/groupings/alerts@monitor.example").Body.String()

	assert.Contains(t, body, "Grouped by sender")
	assert.Contains(t, body, `href="https://mail.google.com/mail/?authuser=me%40example.com#search/from:alerts@monitor.example%20in:inbox"`)
}

func TestGroupingPage_NotFound(t *testing.T) {
	env := defaultEnv(t)

	rec := env.get(t, "/groupings/nobody@example.com")
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "not in the inbox")
}

func TestUnknownRoute(t *testing.T) {
	env := defaultEnv(t)

	rec := env.get(t, "/nope")
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "Page not found.")
}

func TestMessagePage_PlainText(t *testing.T) {
	client := &mockClient{
		messages: testInbox(),
		bodies:   map[uint32]*imap.MessageBody{3: {Text: "Build 2 failed.\n<b>not html</b>", HTML: "<p>ignored</p>"}},
	}
	env := newTestEnv(t, client)

	rec := env.get(t, "/groupings/notifications@example.com/messages/3")
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	assert.Contains(t, body, "Build #2 failed")
	assert.Contains(t, body, "Build Service &lt;builds@service.example&gt;")
	assert.Contains(t, body, "notifications@example.com")
	assert.Contains(t, body, "Wed, Nov 18 2026, 3:00 PM")
	assert.Contains(t, body, `<pre class="body-text">Build 2 failed.`)
	assert.Contains(t, body, "&lt;b&gt;not html&lt;/b&gt;", "plain text is escaped")
	assert.NotContains(t, body, "<iframe")
	assert.Equal(t, []uint32{3}, client.bodyCalls)

	// Newest first: 2, 3, 1. Message 3 is in the middle.
	assert.Contains(t, body, "2 of 3")
	assert.Contains(t, body, `href="/groupings/notifications@example.com/messages/2" aria-label="Previous message"`)
	assert.Contains(t, body, `href="/groupings/notifications@example.com/messages/1" aria-label="Next message"`)
}

func TestMessagePage_HTMLInSandboxedIframe(t *testing.T) {
	client := &mockClient{
		messages: testInbox(),
		bodies:   map[uint32]*imap.MessageBody{4: {HTML: `<p onclick="x()">Thanks</p><script>steal()</script><img src="https://tracker.example/p.gif">`}},
	}
	env := newTestEnv(t, client)

	rec := env.get(t, "/groupings/receipts@example.com/messages/4")
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	assert.Contains(t, body, `sandbox="allow-popups allow-popups-to-escape-sandbox"`)
	assert.Contains(t, body, `referrerpolicy="no-referrer"`)
	assert.Contains(t, body, "srcdoc=")
	assert.Contains(t, body, "Remote images are blocked.")
	// The message HTML is only ever present entity-escaped inside srcdoc.
	assert.NotContains(t, body, "<script>steal()")
	assert.NotContains(t, body, `<p onclick`)
	assert.Contains(t, body, "&lt;script&gt;steal()")
	assert.Contains(t, body, "&lt;base target=&#34;_blank&#34;&gt;")

	// Single message: no prev/next links.
	assert.Contains(t, body, "1 of 1")
	assert.NotContains(t, body, "data-prev")
	assert.NotContains(t, body, "data-next")
}

func TestMessagePage_PrevNextFollowSort(t *testing.T) {
	env := defaultEnv(t)

	// Subject A to Z: 1, 3, 2.
	body := env.get(t, "/groupings/notifications@example.com/messages/3?sort=subject-asc").Body.String()

	assert.Contains(t, body, `href="/groupings/notifications@example.com/messages/1?sort=subject-asc" aria-label="Previous message"`)
	assert.Contains(t, body, `href="/groupings/notifications@example.com/messages/2?sort=subject-asc" aria-label="Next message"`)
	assert.Contains(t, body, `href="/groupings/notifications@example.com?sort=subject-asc" aria-label="Back to notifications@example.com"`)
}

func TestMessagePage_NotFound(t *testing.T) {
	env := defaultEnv(t)

	for _, target := range []string{
		"/groupings/notifications@example.com/messages/4", // UID from another grouping
		"/groupings/notifications@example.com/messages/999",
		"/groupings/notifications@example.com/messages/abc",
		"/groupings/nobody@example.com/messages/1",
	} {
		rec := env.get(t, target)
		assert.Equal(t, http.StatusNotFound, rec.Code, target)
	}
	assert.Empty(t, env.client.bodyCalls, "no body is fetched for an unknown message")
}

func TestMessagePage_BodyError(t *testing.T) {
	env := newTestEnv(t, &mockClient{messages: testInbox(), bodyErr: errors.New("server said no")})

	rec := env.get(t, "/groupings/receipts@example.com/messages/4")
	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Contains(t, rec.Body.String(), "server said no")
}

func TestArchive_Single(t *testing.T) {
	env := defaultEnv(t)

	rec := env.post(t, "/archive", url.Values{"address": {"receipts@example.com"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/", rec.Header().Get("Location"))

	assert.Equal(t, []uint32{4}, env.client.archived["automated/receipts"])
	assert.Nil(t, env.server.findGrouping("receipts@example.com"), "archived grouping leaves the cache")
	assert.Equal(t, 1, env.client.fetchCalls, "archiving does not refetch the inbox")

	body := env.get(t, "/").Body.String()
	assert.Contains(t, body, "Archived 1 grouping (1 message) to automated/receipts")
	assert.NotContains(t, body, "Your receipt")

	// The flash is shown once.
	assert.NotContains(t, env.get(t, "/").Body.String(), "Archived 1 grouping")
}

func TestArchive_BatchWithPartialFailure(t *testing.T) {
	client := &mockClient{
		messages:   testInbox(),
		archiveErr: map[string]error{"automated/alerts": errors.New("quota exceeded")},
	}
	env := newTestEnv(t, client)

	rec := env.post(t, "/archive", url.Values{
		"address": {"notifications@example.com", "alerts@monitor.example", "notifications@example.com", "gone@example.com"},
		"filter":  {"by-sender"},
	})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/?filter=by-sender", rec.Header().Get("Location"))

	assert.ElementsMatch(t, []uint32{1, 2, 3}, client.archived["automated/notifications"], "duplicates are archived once")
	assert.Nil(t, env.server.findGrouping("notifications@example.com"))
	assert.NotNil(t, env.server.findGrouping("alerts@monitor.example"), "a failed grouping stays")

	body := env.get(t, "/").Body.String()
	assert.Contains(t, body, "Archived 1 grouping (3 messages) to automated/notifications")
	assert.Contains(t, body, "Could not archive alerts@monitor.example: ")
	assert.Contains(t, body, "quota exceeded")
	assert.Contains(t, body, "gone@example.com is no longer in the inbox.")
}

func TestArchive_NothingSelected(t *testing.T) {
	env := defaultEnv(t)

	rec := env.post(t, "/archive", url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Empty(t, env.client.archived)
	assert.NotContains(t, env.get(t, "/").Body.String(), `data-flash`)
}

func TestArchive_CrossOriginRejected(t *testing.T) {
	env := defaultEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/archive", strings.NewReader("address=receipts%40example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := env.do(t, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Empty(t, env.client.archived)
	assert.NotNil(t, env.server.findGrouping("receipts@example.com"))
}

func TestRefresh_ReloadsGroupings(t *testing.T) {
	env := defaultEnv(t)
	env.client.messages = append(env.client.messages, &imap.Message{
		UID: 7, Subject: "New one", Date: testNow, To: addr("jobs", "example.com"),
	})

	rec := env.post(t, "/refresh", url.Values{"filter": {"by-sender"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/?filter=by-sender", rec.Header().Get("Location"))
	assert.Equal(t, 2, env.client.fetchCalls)
	assert.Contains(t, env.get(t, "/").Body.String(), "jobs@example.com")
}

func TestRefresh_Error(t *testing.T) {
	env := defaultEnv(t)
	env.client.fetchErr = errors.New("connection reset")

	rec := env.post(t, "/refresh", url.Values{})
	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Contains(t, rec.Body.String(), "connection reset")
}

func TestRefresh_GetNotAllowed(t *testing.T) {
	env := defaultEnv(t)

	for _, target := range []string{"/refresh", "/archive"} {
		rec := env.get(t, target)
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code, target)
		assert.Equal(t, http.MethodPost, rec.Header().Get("Allow"), target)
	}
}

func TestSecurityHeaders(t *testing.T) {
	env := defaultEnv(t)

	for _, target := range []string{"/", "/static/app.css", "/nope"} {
		rec := env.get(t, target)
		csp := rec.Header().Get("Content-Security-Policy")
		assert.Contains(t, csp, "script-src 'self'", target)
		assert.Contains(t, csp, "img-src 'self' data:", target)
		assert.Contains(t, csp, "frame-ancestors 'none'", target)
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"), target)
		assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"), target)
	}
}

func TestStaticAssets(t *testing.T) {
	env := defaultEnv(t)

	css := env.get(t, "/static/app.css")
	require.Equal(t, http.StatusOK, css.Code)
	assert.Contains(t, css.Header().Get("Content-Type"), "text/css")
	assert.Contains(t, css.Body.String(), "@media (max-width: 640px)")

	js := env.get(t, "/static/app.js")
	require.Equal(t, http.StatusOK, js.Code)
	assert.Contains(t, js.Header().Get("Content-Type"), "javascript")

	assert.Equal(t, http.StatusNotFound, env.get(t, "/static/missing.js").Code)
}

func TestPagesHaveNoInlineScriptsOrExternalResources(t *testing.T) {
	client := &mockClient{
		messages: testInbox(),
		bodies:   map[uint32]*imap.MessageBody{4: {HTML: "<p>hi</p>"}},
	}
	env := newTestEnv(t, client)

	for _, target := range []string{"/", "/groupings/notifications@example.com", "/groupings/receipts@example.com/messages/4", "/nope"} {
		body := env.get(t, target).Body.String()
		assert.Contains(t, body, `<script src="/static/app.js" defer></script>`, target)
		assert.Equal(t, 1, strings.Count(body, "<script"), "only the app script on %s", target)
		assert.NotContains(t, body, "fonts.googleapis.com", target)
		assert.NotContains(t, body, " onclick=", target)
	}
}

func TestRequestsAreLogged(t *testing.T) {
	env := defaultEnv(t)
	var buf strings.Builder
	env.server.opts.Logger = log.New(&buf, "", 0)
	env.handler = env.server.Handler()

	env.get(t, "/?filter=by-sender")
	env.get(t, "/groupings/nobody@example.com")
	env.post(t, "/archive", url.Values{"address": {"receipts@example.com"}})

	req := httptest.NewRequest(http.MethodPost, "/archive", strings.NewReader("address=notifications%40example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	env.do(t, req)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 4, buf.String())
	assert.Regexp(t, `^GET /\?filter=by-sender 200 \d+B \S+$`, lines[0])
	assert.Regexp(t, `^GET /groupings/nobody@example.com 404 \d+B \S+$`, lines[1])
	assert.Regexp(t, `^POST /archive 303 \d+B \S+$`, lines[2])
	assert.Regexp(t, `^POST /archive 403 \d+B \S+$`, lines[3], "rejected cross-origin requests are logged")
	assert.NotContains(t, buf.String(), "address=", "form bodies are not logged")
}

func TestStatusRecorder_DefaultsToOK(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	_, err := rec.Write([]byte("hi"))
	require.NoError(t, err)
	rec.WriteHeader(http.StatusTeapot) // ignored after the first write
	assert.Equal(t, http.StatusOK, rec.status)
	assert.Equal(t, 2, rec.bytes)
}
