// pkg/canonical/canonical.go

// Package canonical provides deterministic JSON serialization for cryptographic signing.
package canonical

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// JSON serializes v to canonical JSON: sorted keys, no insignificant whitespace.
func JSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "")
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("encoding canonical JSON: %w", err)
	}
	// json.Encoder adds a trailing newline; strip it.
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}

// SHA256Hex returns the lowercase hex SHA-256 hash of data.
func SHA256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// HashJSON serializes v to canonical JSON and returns its SHA-256 hash as lowercase hex.
func HashJSON(v any) (string, error) {
	canonical, err := JSON(v)
	if err != nil {
		return "", err
	}
	return SHA256Hex(canonical), nil
}
