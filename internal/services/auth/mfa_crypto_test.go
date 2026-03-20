package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptSecretRoundTrip(t *testing.T) {
	key := "12345678901234567890123456789012"

	enc, err := encryptMFASecret("base32secret", key)
	require.NoError(t, err)
	require.NotEmpty(t, enc)

	plain, err := decryptMFASecret(enc, key)
	require.NoError(t, err)
	require.Equal(t, "base32secret", plain)
}

func TestEncryptRejectsInvalidKeyLength(t *testing.T) {
	_, err := encryptMFASecret("base32secret", "short-key")
	require.Error(t, err)
}

func TestDecryptRejectsInvalidKeyLength(t *testing.T) {
	_, err := decryptMFASecret("ciphertext", "short-key")
	require.Error(t, err)
}

func TestDecryptRejectsMalformedCiphertext(t *testing.T) {
	_, err := decryptMFASecret("not-base64", "12345678901234567890123456789012")
	require.Error(t, err)
}
