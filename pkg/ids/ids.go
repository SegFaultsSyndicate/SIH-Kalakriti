// pkg/ids/ids.go

// Package ids generates and parses the UUIDv7 identifiers used for every primary
// key in the platform. UUIDv7 embeds a millisecond timestamp in its high bits, so
// values sort by creation time without a separate created_at index.
package ids

import (
	"fmt"

	"github.com/google/uuid"
)

// Nil is the zero UUID.
var Nil = uuid.Nil

// New returns a fresh UUIDv7. It panics only if the OS entropy source is broken,
// a process-fatal condition every other call in the process would also hit; this
// is the one sanctioned panic in the codebase so call sites are not forced to
// handle an error that cannot occur in practice.
func New() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Errorf("generating uuidv7: %w", err))
	}
	return id
}

// Parse parses a canonical UUID string.
func Parse(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parsing uuid %q: %w", s, err)
	}
	return id, nil
}

// MustParse parses a canonical UUID string, panicking on a malformed value. For
// package-level constants and tests, never for request input.
func MustParse(s string) uuid.UUID { return uuid.MustParse(s) }

// IsNil reports whether id is the zero UUID.
func IsNil(id uuid.UUID) bool { return id == uuid.Nil }
