// pkg/shortcode/shortcode.go

// Package shortcode generates collision-checked base32 codes for public verification URLs.
package shortcode

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

// Generator creates short codes with collision checking.
type Generator struct {
	exists func(ctx context.Context, code string) (bool, error)
	length int
}

// NewGenerator builds a Generator. exists checks whether a code is already taken.
func NewGenerator(exists func(ctx context.Context, code string) (bool, error)) *Generator {
	return &Generator{exists: exists, length: 10}
}

// Generate creates a random base32 code, retrying until collision-free.
func (g *Generator) Generate(ctx context.Context) (string, error) {
	const maxAttempts = 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		code, err := g.random()
		if err != nil {
			return "", fmt.Errorf("generating random code: %w", err)
		}
		taken, err := g.exists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("checking code collision: %w", err)
		}
		if !taken {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxAttempts)
}

// random generates one code without checking collisions.
func (g *Generator) random() (string, error) {
	// 10 base32 chars encode 50 bits. Collision probability is negligible
	// for the expected scale (thousands of seals per day).
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	// Take the first 10 characters, uppercase, no padding.
	return strings.ToUpper(encoded[:g.length]), nil
}
