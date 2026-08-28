// pkg/shortcode/shortcode_test.go
package shortcode

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratorCreatesUniqueCodes verifies that the generator produces codes.
func TestGeneratorCreatesUniqueCodes(t *testing.T) {
	codes := make(map[string]struct{})
	gen := NewGenerator(func(ctx context.Context, code string) (bool, error) {
		_, exists := codes[code]
		return exists, nil
	})

	for i := 0; i < 100; i++ {
		code, err := gen.Generate(context.Background())
		require.NoError(t, err)
		assert.Len(t, code, 10)
		assert.Regexp(t, "^[A-Z2-7]{10}$", code, "code should be valid base32 without padding")
		_, exists := codes[code]
		assert.False(t, exists, "code should be unique: %s", code)
		codes[code] = struct{}{}
	}
}

// TestGeneratorRespectsExistingCodes verifies collision checking works.
func TestGeneratorRespectsExistingCodes(t *testing.T) {
	existing := map[string]struct{}{
		"ABCDEFGHIJ": {},
	}
	gen := NewGenerator(func(ctx context.Context, code string) (bool, error) {
		_, exists := existing[code]
		return exists, nil
	})

	// Should eventually generate a different code.
	for i := 0; i < 10; i++ {
		code, err := gen.Generate(context.Background())
		require.NoError(t, err)
		assert.NotEqual(t, "ABCDEFGHIJ", code, "should not return existing code")
	}
}

// TestCodeFormat verifies codes are uppercase base32 without padding.
func TestCodeFormat(t *testing.T) {
	gen := NewGenerator(func(ctx context.Context, code string) (bool, error) {
		return false, nil
	})

	code, err := gen.Generate(context.Background())
	require.NoError(t, err)

	assert.Len(t, code, 10)
	for _, c := range code {
		assert.True(t, (c >= 'A' && c <= 'Z') || (c >= '2' && c <= '7'),
			"character %c should be valid base32", c)
	}
}