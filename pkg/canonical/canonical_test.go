// pkg/canonical/canonical_test.go
package canonical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCanonicalJSONStability verifies that JSON serialization is deterministic
// across map iteration order.
func TestCanonicalJSONStability(t *testing.T) {
	// Run 100 times to catch any non-determinism from map iteration.
	var prev []byte
	for i := 0; i < 100; i++ {
		data := map[string]any{
			"listing_id":        "listing-123",
			"artisan_id":        "artisan-456",
			"craft_id":          "craft-789",
			"technique_matched": true,
			"media_hashes":      []string{"hash1", "hash2", "hash3"},
			"created_at":        "2026-08-27T22:00:00Z",
		}
		result, err := JSON(data)
		require.NoError(t, err)

		if i > 0 {
			assert.Equal(t, prev, result, "canonical JSON must be stable across iterations")
		}
		prev = result
	}
}

// TestHashJSONDeterministic verifies that the same input produces the same hash.
func TestHashJSONDeterministic(t *testing.T) {
	data := map[string]any{
		"listing_id": "test-listing",
		"artisan_id": "test-artisan",
		"craft_id":   "test-craft",
	}

	hash1, err := HashJSON(data)
	require.NoError(t, err)

	hash2, err := HashJSON(data)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "same input must produce same hash")
	assert.Len(t, hash1, 64, "SHA-256 hash should be 64 hex chars")
}

// TestHashJSONChangeDetection verifies that any change produces a different hash.
func TestHashJSONChangeDetection(t *testing.T) {
	base := map[string]any{
		"listing_id": "listing-123",
		"artisan_id": "artisan-456",
	}

	baseHash, err := HashJSON(base)
	require.NoError(t, err)

	// Change a value.
	modified := map[string]any{
		"listing_id": "listing-123",
		"artisan_id": "artisan-999", // changed
	}
	modifiedHash, err := HashJSON(modified)
	require.NoError(t, err)

	assert.NotEqual(t, baseHash, modifiedHash, "changed field must produce different hash")
}

// TestSHA256HexFormat verifies the hash output format.
func TestSHA256HexFormat(t *testing.T) {
	data := []byte("test data")
	hash := SHA256Hex(data)

	assert.Len(t, hash, 64, "SHA-256 hash should be 64 hex characters")
	assert.Regexp(t, "^[0-9a-f]{64}$", hash, "hash should be lowercase hex")
}
