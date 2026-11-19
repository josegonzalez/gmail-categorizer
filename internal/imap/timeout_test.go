package imap

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
)

// silentListener accepts connections and never sends anything, like a server
// that has stopped responding.
func silentListener(t *testing.T) (host string, port int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { conn.Close() })
			go func() { _, _ = io.Copy(io.Discard, conn) }()
		}
	}()

	h, p, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err = strconv.Atoi(p)
	require.NoError(t, err)
	return h, port
}

// stalledClient returns a client whose server sends a greeting and then
// never answers a command.
func stalledClient(t *testing.T, timeout time.Duration) *gmailClient {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() {
		clientConn.Close()
		serverConn.Close()
	})

	go func() {
		_, _ = serverConn.Write([]byte("* OK IMAP4rev1 ready\r\n"))
		// Read commands so client writes do not block, and never reply.
		r := bufio.NewReader(serverConn)
		for {
			if _, err := r.ReadString('\n'); err != nil {
				return
			}
		}
	}()

	c := &gmailClient{
		config: &config.IMAPConfig{Server: "imap.test", Timeout: timeout},
		client: imapclient.New(clientConn, nil),
	}
	require.NoError(t, c.client.WaitGreeting())
	return c
}

// within fails the test if fn takes longer than limit.
func within(t *testing.T, limit time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("call did not return within %s", limit)
	}
}

func TestConnect_TimesOutWhenServerIsSilent(t *testing.T) {
	host, port := silentListener(t)
	client := NewClient(&config.IMAPConfig{Server: host, Port: port, Timeout: 200 * time.Millisecond})

	var err error
	within(t, 3*time.Second, func() { err = client.Connect(context.Background()) })

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "no response within 200ms")
}

func TestConnect_CancelledByContext(t *testing.T) {
	host, port := silentListener(t)
	client := NewClient(&config.IMAPConfig{Server: host, Port: port, Timeout: time.Minute})

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	var err error
	within(t, 3*time.Second, func() { err = client.Connect(ctx) })

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLogin_TimesOutWhenServerStalls(t *testing.T) {
	c := stalledClient(t, 200*time.Millisecond)

	var err error
	within(t, 3*time.Second, func() { err = c.Login(context.Background(), "user", "pass") })

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "authentication failed: authenticating: no response from imap.test within 200ms")

	// The abandoned connection is closed, so later calls fail fast too.
	within(t, time.Second, func() { assert.Error(t, c.Noop(context.Background())) })
}

func TestLogin_CancelledByContext(t *testing.T) {
	c := stalledClient(t, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	var err error
	within(t, 3*time.Second, func() { err = c.Login(ctx, "user", "pass") })

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, errors.Is(err, context.DeadlineExceeded))
}

func TestSelectMailbox_TimesOutWhenServerStalls(t *testing.T) {
	c := stalledClient(t, 200*time.Millisecond)

	var err error
	within(t, 3*time.Second, func() { _, err = c.SelectMailbox(context.Background(), "INBOX") })

	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestNoop_TimesOutWhenServerStalls(t *testing.T) {
	c := stalledClient(t, 200*time.Millisecond)

	var err error
	within(t, 3*time.Second, func() { err = c.Noop(context.Background()) })

	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestLogout_DoesNotHangWhenServerStalls(t *testing.T) {
	c := stalledClient(t, 200*time.Millisecond)

	var err error
	within(t, 3*time.Second, func() { err = c.Logout() })

	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestListMailboxes_CancelledByContextWithoutTimeout(t *testing.T) {
	// Long operations have no timeout but still stop when cancelled.
	c := stalledClient(t, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)

	start := time.Now()
	var err error
	within(t, 3*time.Second, func() { _, err = c.ListMailboxes(ctx) })

	assert.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, time.Since(start), 250*time.Millisecond, "the command timeout does not apply")
}

func TestCommandTimeout_Default(t *testing.T) {
	assert.Equal(t, defaultCommandTimeout, (&gmailClient{config: &config.IMAPConfig{}}).commandTimeout())
	assert.Equal(t, defaultCommandTimeout, (&gmailClient{}).commandTimeout())
	assert.Equal(t, 5*time.Second, (&gmailClient{config: &config.IMAPConfig{Timeout: 5 * time.Second}}).commandTimeout())
}
