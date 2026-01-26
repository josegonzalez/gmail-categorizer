package keychain

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}

func TestKeychain_RoundTrip(t *testing.T) {
	// Use a unique test username to avoid conflicts
	testUser := "gmail-categorizer-test-user@example.com"
	testPassword := "test-password-12345"

	// Clean up before test
	_ = Delete(testUser)

	// Initially should not exist
	exists, err := Exists(testUser)
	require.NoError(t, err)
	assert.False(t, exists)

	// Get should return empty string for non-existent entry
	password, err := Get(testUser)
	require.NoError(t, err)
	assert.Empty(t, password)

	// Set the password
	err = Set(testUser, testPassword)
	require.NoError(t, err)

	// Now it should exist
	exists, err = Exists(testUser)
	require.NoError(t, err)
	assert.True(t, exists)

	// Get should return the password
	password, err = Get(testUser)
	require.NoError(t, err)
	assert.Equal(t, testPassword, password)

	// Update the password
	newPassword := "new-password-67890"
	err = Set(testUser, newPassword)
	require.NoError(t, err)

	// Get should return the new password
	password, err = Get(testUser)
	require.NoError(t, err)
	assert.Equal(t, newPassword, password)

	// Delete the password
	err = Delete(testUser)
	require.NoError(t, err)

	// Should no longer exist
	exists, err = Exists(testUser)
	require.NoError(t, err)
	assert.False(t, exists)

	// Get should return empty string
	password, err = Get(testUser)
	require.NoError(t, err)
	assert.Empty(t, password)
}

func TestKeychain_DeleteNonExistent(t *testing.T) {
	// Deleting a non-existent entry should not error
	err := Delete("non-existent-user-12345@example.com")
	assert.NoError(t, err)
}

func TestKeychain_ServiceName(t *testing.T) {
	// Verify the service name is set correctly
	assert.Equal(t, "gmail-categorizer", ServiceName)
}
