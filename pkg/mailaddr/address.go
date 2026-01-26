// Package mailaddr provides email address parsing and normalization utilities.
package mailaddr

import (
	"fmt"
	"net/mail"
	"strings"
)

// Address represents a parsed email address.
type Address struct {
	Name    string
	Address string
}

// Parse extracts the email address from a header value like "Name <email@example.com>".
func Parse(s string) (*Address, error) {
	if s == "" {
		return nil, fmt.Errorf("empty address")
	}

	// Try to parse with net/mail
	addr, err := mail.ParseAddress(s)
	if err != nil {
		// Fall back to treating the whole string as an address
		normalized := Normalize(s)
		if normalized == "" {
			return nil, fmt.Errorf("invalid address: %s", s)
		}
		return &Address{Address: normalized}, nil
	}

	return &Address{
		Name:    addr.Name,
		Address: Normalize(addr.Address),
	}, nil
}

// Normalize converts an email address to a canonical form:
// - Lowercase
// - Trimmed whitespace
// - Gmail + addressing stripped (user+tag@gmail.com -> user@gmail.com)
func Normalize(addr string) string {
	addr = strings.ToLower(strings.TrimSpace(addr))
	if addr == "" {
		return ""
	}

	// Handle Gmail's + addressing
	atIdx := strings.LastIndex(addr, "@")
	if atIdx <= 0 {
		return addr
	}

	localPart := addr[:atIdx]
	domain := addr[atIdx:]

	// Strip + suffix from local part
	if plusIdx := strings.Index(localPart, "+"); plusIdx > 0 {
		localPart = localPart[:plusIdx]
	}

	return localPart + domain
}

// Format creates an email address string from mailbox and host parts.
func Format(mailbox, host string) string {
	if mailbox == "" || host == "" {
		return ""
	}
	return strings.ToLower(mailbox) + "@" + strings.ToLower(host)
}

// HasPrefix checks if an email address starts with the given prefix.
// The comparison is case-insensitive.
func HasPrefix(addr, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(addr), strings.ToLower(prefix))
}

// LocalPart extracts the local part (before @) from an email address.
func LocalPart(addr string) string {
	atIdx := strings.LastIndex(addr, "@")
	if atIdx <= 0 {
		return addr
	}
	return addr[:atIdx]
}

// Domain extracts the domain part (after @) from an email address.
func Domain(addr string) string {
	atIdx := strings.LastIndex(addr, "@")
	if atIdx < 0 || atIdx >= len(addr)-1 {
		return ""
	}
	return addr[atIdx+1:]
}
