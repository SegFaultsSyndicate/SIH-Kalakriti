// pkg/crypto/crypto.go

// Package crypto handles Ed25519 signing for provenance records.
package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Signer signs provenance hashes with Ed25519.
type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	keyID      string
}

// NewSigner builds a Signer from a hex-encoded private key.
func NewSigner(privateKeyHex, keyID string) (*Signer, error) {
	keyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %w", err)
	}
	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key must be %d bytes, got %d", ed25519.PrivateKeySize, len(keyBytes))
	}
	privKey := ed25519.PrivateKey(keyBytes)
	pubKey := privKey.Public().(ed25519.PublicKey)
	return &Signer{
		privateKey: privKey,
		publicKey:  pubKey,
		keyID:      keyID,
	}, nil
}

// GenerateKeypair creates a fresh Ed25519 keypair for development.
func GenerateKeypair() (privateKeyHex, publicKeyHex string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generating keypair: %w", err)
	}
	return hex.EncodeToString(priv), hex.EncodeToString(pub), nil
}

// Sign returns the detached Ed25519 signature over message.
func (s *Signer) Sign(message []byte) []byte {
	return ed25519.Sign(s.privateKey, message)
}

// KeyID returns the configured key identifier.
func (s *Signer) KeyID() string {
	return s.keyID
}

// PublicKey returns the public key bytes.
func (s *Signer) PublicKey() []byte {
	return s.publicKey
}

// PublicKeyHex returns the hex-encoded public key.
func (s *Signer) PublicKeyHex() string {
	return hex.EncodeToString(s.publicKey)
}

// Verify checks a signature against a message and public key.
func Verify(publicKey, message, signature []byte) bool {
	if len(publicKey) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(publicKey, message, signature)
}
