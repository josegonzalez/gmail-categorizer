package imap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddress_String(t *testing.T) {
	tests := []struct {
		name     string
		address  Address
		expected string
	}{
		{
			name: "complete address",
			address: Address{
				Name:    "John Doe",
				Mailbox: "john",
				Host:    "example.com",
			},
			expected: "john@example.com",
		},
		{
			name: "address without name",
			address: Address{
				Mailbox: "user",
				Host:    "test.com",
			},
			expected: "user@test.com",
		},
		{
			name: "empty mailbox",
			address: Address{
				Name:    "John",
				Mailbox: "",
				Host:    "example.com",
			},
			expected: "",
		},
		{
			name: "empty host",
			address: Address{
				Name:    "John",
				Mailbox: "john",
				Host:    "",
			},
			expected: "",
		},
		{
			name: "both empty",
			address: Address{
				Name:    "John",
				Mailbox: "",
				Host:    "",
			},
			expected: "",
		},
		{
			name: "uppercase conversion",
			address: Address{
				Mailbox: "USER",
				Host:    "EXAMPLE.COM",
			},
			expected: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.address.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMailboxInfo(t *testing.T) {
	info := MailboxInfo{
		Name:        "INBOX",
		NumMessages: 100,
		NumUnseen:   25,
	}

	assert.Equal(t, "INBOX", info.Name)
	assert.Equal(t, uint32(100), info.NumMessages)
	assert.Equal(t, uint32(25), info.NumUnseen)
}

func TestMessage(t *testing.T) {
	msg := Message{
		UID:     12345,
		SeqNum:  42,
		Subject: "Test Subject",
		From: []Address{
			{Mailbox: "sender", Host: "example.com"},
		},
		To: []Address{
			{Mailbox: "recipient", Host: "example.com"},
		},
	}

	assert.Equal(t, uint32(12345), msg.UID)
	assert.Equal(t, uint32(42), msg.SeqNum)
	assert.Equal(t, "Test Subject", msg.Subject)
	assert.Equal(t, 1, len(msg.From))
	assert.Equal(t, 1, len(msg.To))
	assert.Equal(t, "sender@example.com", msg.From[0].String())
	assert.Equal(t, "recipient@example.com", msg.To[0].String())
}
