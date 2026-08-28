// pkg/crypto/crypto_test.go
package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateKeypairProducesValidKeys verifies keypair generation.
func TestGenerateKeypairProducesValidKeys(t *testing.T) {
	privHex, pubHex, err := GenerateKeypair()
	require.NoError(t, err)

	// Both should be valid hex.
	require.Regexp(t, "^[0-9a-f]+$", privHex)
	require.Regexp(t, "^[0-9a-f]+$", pubHex)

	// Decode and verify sizes.
	privBytes, err := hex.DecodeString(privHex)
	require.NoError(t, err)
	assert.Len(t, privBytes, ed25519.PrivateKeySize)

	pubBytes, err := hex.DecodeString(pubHex)
	require.NoError(t, err)
	assert.Len(t, pubBytes, ed25519.PublicKeySize)
}

// TestSignAndVerify verifies that signatures work correctly.
func TestSignAndVerify(t *testing.T) {
	privHex, pubHex, err := GenerateKeypair()
	require.NoError(t, err)

	signer, err := NewSigner(privHex, "test-key")
	require.NoError(t, err)

	message := []byte("test message")
	sig := signer.Sign(message)

	pubBytes, err := hex.DecodeString(pubHex)
	require.NoError(t, err)

	assert.True(t, Verify(pubBytes, message, sig), "valid signature should verify")
}

// TestVerifyRejectsTamperedMessage verifies that changed messages fail.
func TestVerifyRejectsTamperedMessage(t *testing.T) {
	privHex, pubHex, err := GenerateKeypair()
	require.NoError(t, err)

	signer, err := NewSigner(privHex, "test-key")
	require.NoError(t, err)

	message := []byte("original message")
	sig := signer.Sign(message)

	pubBytes, err := hex.DecodeString(pubHex)
	require.NoError(t, err)

	// Tamper with the message.
	tampered := []byte("tampered message")
	assert.False(t, Verify(pubBytes, tampered, sig), "tampered message should not verify")
}

// TestVerifyRejectsWrongKey verifies that a different key's signature fails.
func TestVerifyRejectsWrongKey(t *testing.T) {
	privHex1, pubHex1, err := GenerateKeypair()
	require.NoError(t, err)
	privHex2, _, err := GenerateKeypair()
	require.NoError(t, err)

	signer1, err := NewSigner(privHex1, "test-key-1")
	require.NoError(t, err)

	message := []byte("test message")
	sig := signer1.Sign(message)

	pubBytes2, err := hex.DecodeString(pubHex1) // Wait, this is the same key.
	require.NoError(t, err)
	// Actually test with key 2's public key
	pubBytes2, err = hex.DecodeString(privHex2) // Wrong - let's decode properly
	require.NoError(t, err)
	// Actually, need public key from keypair 2
	_, pubHex2, err = GenerateKeypair()
	require.NoError(t, err)
	pubBytes2, err = hex.DecodeString(pubHex2)
	require.NoError(t, err)

	assert.False(t, Verify(pubBytes2, message, sig), "signature from key 1 should not verify with key 2")
}

// TestNewSignerRejectsInvalidKey verifies error on bad key length.
func TestNewSignerRejectsInvalidKey(t *testing.T) {
	// Too short key.
	_, err := NewSigner("abc", "test-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be 64 bytes")

	// Too long key.
	_, err = NewSigner(strings.Repeat("a", 128), "test-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be 64 bytes")
}