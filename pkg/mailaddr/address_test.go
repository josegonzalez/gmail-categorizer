package mailaddr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantName    string
		wantAddress string
		wantErr     bool
	}{
		{
			name:        "simple address",
			input:       "user@example.com",
			wantName:    "",
			wantAddress: "user@example.com",
			wantErr:     false,
		},
		{
			name:        "address with name",
			input:       "John Doe <john@example.com>",
			wantName:    "John Doe",
			wantAddress: "john@example.com",
			wantErr:     false,
		},
		{
			name:        "address with quoted name",
			input:       "\"John Doe\" <john@example.com>",
			wantName:    "John Doe",
			wantAddress: "john@example.com",
			wantErr:     false,
		},
		{
			name:        "uppercase address",
			input:       "USER@EXAMPLE.COM",
			wantName:    "",
			wantAddress: "user@example.com",
			wantErr:     false,
		},
		{
			name:        "plus addressing",
			input:       "user+tag@example.com",
			wantName:    "",
			wantAddress: "user@example.com",
			wantErr:     false,
		},
		{
			name:        "empty string",
			input:       "",
			wantName:    "",
			wantAddress: "",
			wantErr:     true,
		},
		{
			name:        "invalid format - fallback to normalized",
			input:       "just-a-local-part",
			wantName:    "",
			wantAddress: "just-a-local-part",
			wantErr:     false,
		},
		{
			name:        "invalid format - whitespace only",
			input:       "   ",
			wantName:    "",
			wantAddress: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := Parse(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, addr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, addr)
				assert.Equal(t, tt.wantName, addr.Name)
				assert.Equal(t, tt.wantAddress, addr.Address)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase conversion",
			input:    "USER@Example.COM",
			expected: "user@example.com",
		},
		{
			name:     "trim whitespace",
			input:    "  user@example.com  ",
			expected: "user@example.com",
		},
		{
			name:     "gmail plus addressing",
			input:    "user+tag@gmail.com",
			expected: "user@gmail.com",
		},
		{
			name:     "plus addressing with complex tag",
			input:    "user+newsletter+2024@example.com",
			expected: "user@example.com",
		},
		{
			name:     "already normalized",
			input:    "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no domain",
			input:    "user",
			expected: "user",
		},
		{
			name:     "plus at beginning (edge case)",
			input:    "+user@example.com",
			expected: "+user@example.com",
		},
		{
			name:     "multiple @ signs",
			input:    "user@foo@example.com",
			expected: "user@foo@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Normalize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		mailbox  string
		host     string
		expected string
	}{
		{
			name:     "basic format",
			mailbox:  "user",
			host:     "example.com",
			expected: "user@example.com",
		},
		{
			name:     "uppercase conversion",
			mailbox:  "USER",
			host:     "EXAMPLE.COM",
			expected: "user@example.com",
		},
		{
			name:     "empty mailbox",
			mailbox:  "",
			host:     "example.com",
			expected: "",
		},
		{
			name:     "empty host",
			mailbox:  "user",
			host:     "",
			expected: "",
		},
		{
			name:     "both empty",
			mailbox:  "",
			host:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Format(tt.mailbox, tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		prefix   string
		expected bool
	}{
		{
			name:     "matching prefix",
			addr:     "admin@example.com",
			prefix:   "admin@",
			expected: true,
		},
		{
			name:     "case insensitive match",
			addr:     "ADMIN@example.com",
			prefix:   "admin@",
			expected: true,
		},
		{
			name:     "no match",
			addr:     "user@example.com",
			prefix:   "admin@",
			expected: false,
		},
		{
			name:     "partial prefix",
			addr:     "admin@example.com",
			prefix:   "adm",
			expected: true,
		},
		{
			name:     "empty prefix",
			addr:     "admin@example.com",
			prefix:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPrefix(tt.addr, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLocalPart(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected string
	}{
		{
			name:     "basic extraction",
			addr:     "user@example.com",
			expected: "user",
		},
		{
			name:     "no @ sign",
			addr:     "user",
			expected: "user",
		},
		{
			name:     "multiple @ signs",
			addr:     "user@foo@example.com",
			expected: "user@foo",
		},
		{
			name:     "empty string",
			addr:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LocalPart(tt.addr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDomain(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected string
	}{
		{
			name:     "basic extraction",
			addr:     "user@example.com",
			expected: "example.com",
		},
		{
			name:     "no @ sign",
			addr:     "user",
			expected: "",
		},
		{
			name:     "multiple @ signs",
			addr:     "user@foo@example.com",
			expected: "example.com",
		},
		{
			name:     "empty domain",
			addr:     "user@",
			expected: "",
		},
		{
			name:     "empty string",
			addr:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Domain(tt.addr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
