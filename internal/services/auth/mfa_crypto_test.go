package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptSecretRoundTrip(t *testing.T) {
	enc, err := encryptMFASecret("base32secret", "32-byte-test-key-123456789012345")
	require.NoError(t, err)

	plain, err := decryptMFASecret(enc, "32-byte-test-key-123456789012345")
	require.NoError(t, err)
	require.Equal(t, "base32secret", plain)
}

func TestEncryptRejectsInvalidKeyLength(t *testing.T) {
	_, err := encryptMFASecret("base32secret", "invalid-key")
	require.Error(t, err)
}
