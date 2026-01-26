// Package keychain provides secure password storage using the OS keychain.
package keychain

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the identifier used in the keychain
	ServiceName = "gmail-categorizer"
)

// Get retrieves the password for the given username from the OS keychain.
// Returns an empty string and no error if the password is not found.
func Get(username string) (string, error) {
	password, err := keyring.Get(ServiceName, username)
	if err == keyring.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading from keychain: %w", err)
	}
	return password, nil
}

// Set stores the password for the given username in the OS keychain.
func Set(username, password string) error {
	if err := keyring.Set(ServiceName, username, password); err != nil {
		return fmt.Errorf("storing in keychain: %w", err)
	}
	return nil
}

// Delete removes the password for the given username from the OS keychain.
func Delete(username string) error {
	err := keyring.Delete(ServiceName, username)
	if err == keyring.ErrNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("deleting from keychain: %w", err)
	}
	return nil
}

// Exists checks if a password exists for the given username in the keychain.
func Exists(username string) (bool, error) {
	_, err := keyring.Get(ServiceName, username)
	if err == keyring.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking keychain: %w", err)
	}
	return true, nil
}
