package views

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
)

func TestRenderMessage(t *testing.T) {
	msg := &imap.Message{
		Subject: "Build passed",
		From:    []imap.Address{{Name: "Build Service", Mailbox: "builds", Host: "service.example"}},
		To:      []imap.Address{{Mailbox: "notifications", Host: "example.com"}},
		Date:    time.Date(2026, 11, 19, 9, 42, 0, 0, time.UTC),
	}

	result := RenderMessage(msg, "the body", 0.5, 80)

	assert.Contains(t, result, "Build passed")
	assert.Contains(t, result, "Build Service <builds@service.example>")
	assert.Contains(t, result, "notifications@example.com")
	assert.Contains(t, result, "Thu, Nov 19 2026 9:42 AM")
	assert.Contains(t, result, "the body")
	assert.Contains(t, result, "50%")
	assert.Contains(t, result, "esc back")
}

func TestRenderMessage_NoSubjectOrMessage(t *testing.T) {
	assert.Contains(t, RenderMessage(&imap.Message{}, "", 0, 80), "(no subject)")
	assert.Contains(t, RenderMessage(nil, "", 0, 0), "(no subject)")
}
