// pkg/idempotency/hash.go
package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
)

// sha256Hex is split out so Hash256Hex reads as a one-line public wrapper.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
