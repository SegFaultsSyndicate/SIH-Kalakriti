// services/bff/internal/bff/handler/verification_test.go
package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

// fakeCatalogClient is a test double for CatalogService.
type fakeCatalogClient struct {
	provenance map[string]ProvenanceRecord
	listings   map[string]Listing
	artisans   map[string]Artisan
	crafts     map[string]Craft
}

func (f *fakeCatalogClient) GetProvenanceByShortCode(ctx context.Context, code string) (ProvenanceRecord, error) {
	prov, ok := f.provenance[code]
	if !ok {
		return ProvenanceRecord{}, pkgdomain.NotFound("provenance not found")
	}
	return prov, nil
}

func (f *fakeCatalogClient) GetListing(ctx context.Context, listingID string) (Listing, error) {
	listing, ok := f.listings[listingID]
	if !ok {
		return Listing{}, pkgdomain.NotFound("listing not found")
	}
	return listing, nil
}

func (f *fakeCatalogClient) GetArtisan(ctx context.Context, artisanID string) (Artisan, error) {
	artisan, ok := f.artisans[artisanID]
	if !ok {
		return Artisan{}, pkgdomain.NotFound("artisan not found")
	}
	return artisan, nil
}

func (f *fakeCatalogClient) GetCraft(ctx context.Context, craftID string) (Craft, error) {
	craft, ok := f.crafts[craftID]
	if !ok {
		return Craft{}, pkgdomain.NotFound("craft not found")
	}
	return craft, nil
}

func (f *fakeCatalogClient) GetListingBySlug(ctx context.Context, slug string) (*ListingDetail, error) {
	return nil, pkgdomain.NotFound("not implemented")
}

func (f *fakeCatalogClient) GetArtisanBySlug(ctx context.Context, slug string) (*ArtisanProfile, error) {
	return nil, pkgdomain.NotFound("not implemented")
}

func (f *fakeCatalogClient) ListPublishedListings(ctx context.Context, limit, offset int32) ([]ListingDetail, error) {
	return nil, nil
}

// TestVerificationPageRendersForValidCode verifies the HTML page renders correctly.
func TestVerificationPageRendersForValidCode(t *testing.T) {
	catalog := &fakeCatalogClient{
		provenance: map[string]ProvenanceRecord{
			"TESTCODE01": {
				ID:               "prov-123",
				ListingID:        "listing-456",
				ArtisanID:        "artisan-789",
				CraftID:          "craft-001",
				ContentHash:      "abcd1234",
				Signature:        []byte("fake-signature"),
				SignatureAlgo:    "ed25519",
				PublicKeyID:      "key-1",
				ShortCode:        "TESTCODE01",
				TechniqueMatched: true,
				MediaHashes:      []string{"hash1", "hash2"},
				SealedAt:         time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC),
			},
		},
		listings: map[string]Listing{
			"listing-456": {
				ID:        "listing-456",
				ProductID: "product-123",
				ArtisanID: "artisan-789",
				Title:     "Handwoven Saree",
				Price:     500000,
				Currency:  "INR",
			},
		},
		artisans: map[string]Artisan{
			"artisan-789": {
				ID:          "artisan-789",
				DisplayName: "Lakshmi Devi",
				ClusterID:   nil,
			},
		},
		crafts: map[string]Craft{
			"craft-001": {
				ID:               "craft-001",
				DisplayName:      "Kanchipuram Silk Weaving",
				GIRegistrationNo: strPtr("GI-123"),
			},
		},
	}

	cache := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	handler, err := NewVerificationHandler(catalog, cache, "https://example.com")
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Get("/v/{code}", handler.ServeHTTP)

	req := httptest.NewRequest(http.MethodGet, "/v/TESTCODE01", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "Lakshmi Devi")
	assert.Contains(t, w.Body.String(), "Kanchipuram Silk Weaving")
	assert.Contains(t, w.Body.String(), "TESTCODE01")
	assert.Contains(t, w.Body.String(), "Provenance Verified")
}

// TestVerificationPageReturns404ForUnknownCode verifies the styled 404 page.
func TestVerificationPageReturns404ForUnknownCode(t *testing.T) {
	catalog := &fakeCatalogClient{
		provenance: map[string]ProvenanceRecord{},
		listings:   map[string]Listing{},
		artisans:   map[string]Artisan{},
		crafts:     map[string]Craft{},
	}

	cache := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	handler, err := NewVerificationHandler(catalog, cache, "https://example.com")
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Get("/v/{code}", handler.ServeHTTP)

	req := httptest.NewRequest(http.MethodGet, "/v/INVALIDCODE", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "Cannot Verify This Tag")
	assert.Contains(t, w.Body.String(), "INVALIDCODE")
	assert.Contains(t, w.Body.String(), "not recognized")
}

// TestVerificationJSONEndpoint verifies the machine-readable JSON response.
func TestVerificationJSONEndpoint(t *testing.T) {
	catalog := &fakeCatalogClient{
		provenance: map[string]ProvenanceRecord{
			"JSONTEST01": {
				ID:               "prov-json-123",
				ListingID:        "listing-json-456",
				ArtisanID:        "artisan-json-789",
				CraftID:          "craft-json-001",
				ContentHash:      "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				Signature:        []byte{0x01, 0x02, 0x03},
				SignatureAlgo:    "ed25519",
				PublicKeyID:      "key-json-1",
				ShortCode:        "JSONTEST01",
				TechniqueMatched: true,
				MediaHashes:      []string{"hash1", "hash2"},
				SealedAt:         time.Date(2026, 8, 27, 22, 0, 0, 0, time.UTC),
			},
		},
		listings: map[string]Listing{},
		artisans: map[string]Artisan{},
		crafts:   map[string]Craft{},
	}

	cache := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	handler, err := NewVerificationHandler(catalog, cache, "https://example.com")
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Get("/v/{code}/verify.json", handler.ServeJSON)

	req := httptest.NewRequest(http.MethodGet, "/v/JSONTEST01/verify.json", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age=3600")

	body := w.Body.String()
	assert.Contains(t, body, "JSONTEST01")
	assert.Contains(t, body, "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	assert.Contains(t, body, "010203") // hex-encoded signature
	assert.Contains(t, body, "ed25519")
	assert.Contains(t, body, "key-json-1")
	assert.Contains(t, body, "true") // technique_matched
}

// TestVerificationJSONReturns404ForUnknownCode verifies JSON 404 handling.
func TestVerificationJSONReturns404ForUnknownCode(t *testing.T) {
	catalog := &fakeCatalogClient{
		provenance: map[string]ProvenanceRecord{},
		listings:   map[string]Listing{},
		artisans:   map[string]Artisan{},
		crafts:     map[string]Craft{},
	}

	cache := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	handler, err := NewVerificationHandler(catalog, cache, "https://example.com")
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Get("/v/{code}/verify.json", handler.ServeJSON)

	req := httptest.NewRequest(http.MethodGet, "/v/UNKNOWN99/verify.json", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func strPtr(s string) *string {
	return &s
}
