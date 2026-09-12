package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/middleware"
)

// Test that an unauthenticated call to a protected route returns 401.
func TestAuthMiddleware_Unauthenticated(t *testing.T) {
	issuer := mustIssuer(t)
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	srv, _ := bff.NewServer(bff.Config{
		Addr:                  ":0",
		BaseURL:               "http://test",
		WebDist:               "/tmp",
		Issuer:                issuer,
		Redis:                 rdb,
		IdempStore:            &mockIdempStore{},
		RateLimitPerIP:        1000,
		RateLimitPerPrincipal: 1000,
		RateLimitWindow:       time.Minute,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artisans/me", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "unauthenticated" {
		t.Errorf("expected error=unauthenticated, got %v", body["error"])
	}
}

// Test that replaying a POST with the same Idempotency-Key returns the original response.
func TestIdempotency_Replay(t *testing.T) {
	issuer := mustIssuer(t)
	store := &mockIdempStore{responses: make(map[string][]byte)}
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	srv, _ := bff.NewServer(bff.Config{
		Addr:                  ":0",
		BaseURL:               "http://test",
		WebDist:               "/tmp",
		Issuer:                issuer,
		Redis:                 rdb,
		IdempStore:            store,
		RateLimitPerIP:        1000,
		RateLimitPerPrincipal: 1000,
		RateLimitWindow:       time.Minute,
		// Mock service that returns a new ID each time.
		ArtisanSvc: &mockArtisanSvc{nextID: "art-1"},
	})

	token := mustToken(t, issuer, auth.Subject{ID: "user-1", Role: auth.RoleArtisan})

	// First call.
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/artisans", bytes.NewBufferString(`{"display_name":"Test"}`))
	req1.Header.Set("Authorization", "Bearer "+token)
	req1.Header.Set("Idempotency-Key", "idem-123")
	w1 := httptest.NewRecorder()
	srv.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("first call failed: %d", w1.Code)
	}

	var resp1 map[string]string
	json.NewDecoder(w1.Body).Decode(&resp1)
	firstID := resp1["artisan_id"]

	// Replay with same key → should return the same ID.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/artisans", bytes.NewBufferString(`{"display_name":"Test"}`))
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Idempotency-Key", "idem-123")
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)

	var resp2 map[string]string
	json.NewDecoder(w2.Body).Decode(&resp2)

	if resp2["artisan_id"] != firstID {
		t.Errorf("idempotency replay failed: first=%s, second=%s", firstID, resp2["artisan_id"])
	}
}

// Test SPA fallback: unknown paths serve index.html.
func TestSPAFallback(t *testing.T) {
	// TODO: requires a real dist directory with index.html. Skipped for now.
	t.Skip("SPA fallback requires a real web/dist")
}

// Test that error mapping produces consistent JSON bodies.
func TestErrorMapping(t *testing.T) {
	issuer := mustIssuer(t)
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	srv, _ := bff.NewServer(bff.Config{
		Addr:                  ":0",
		BaseURL:               "http://test",
		WebDist:               "/tmp",
		Issuer:                issuer,
		Redis:                 rdb,
		IdempStore:            &mockIdempStore{},
		RateLimitPerIP:        1000,
		RateLimitPerPrincipal: 1000,
		RateLimitWindow:       time.Minute,
		ListingSvc:            &mockListingSvc{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/listings/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "not_found" {
		t.Errorf("expected error=not_found, got %v", body)
	}
}

func mustIssuer(t *testing.T) *auth.Issuer {
	t.Helper()
	issuer, err := auth.NewIssuer(auth.Config{
		Secret:     "test-secret-at-least-32-bytes-long",
		Issuer:     "test",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	return issuer
}

func mustToken(t *testing.T, issuer *auth.Issuer, sub auth.Subject) string {
	t.Helper()
	pair, err := issuer.Issue(sub)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return pair.AccessToken
}

// mockIdempStore implements middleware.IdempotencyStore for testing.
type mockIdempStore struct {
	responses map[string][]byte
}

func (m *mockIdempStore) GetOrInsert(ctx context.Context, scope, key, requestHash string, expiresAt time.Time) (rec middleware.IdempotencyRecord, inserted bool, err error) {
	stored, ok := m.responses[scope+":"+key]
	if !ok {
		return middleware.IdempotencyRecord{RequestHash: requestHash}, true, nil
	}
	return middleware.IdempotencyRecord{RequestHash: requestHash, Response: stored}, false, nil
}

func (m *mockIdempStore) SaveResponse(ctx context.Context, scope, key string, response []byte) error {
	m.responses[scope+":"+key] = response
	return nil
}

// mockArtisanSvc returns sequential IDs for testing idempotency.
type mockArtisanSvc struct {
	nextID string
}

func (m *mockArtisanSvc) Register(ctx context.Context, phone, idempotencyKey string, fields map[string]any) (string, error) {
	id := m.nextID
	m.nextID = "art-" + id[4:] + "1" // increment for next call
	return id, nil
}

func (m *mockArtisanSvc) GetProfile(ctx context.Context, artisanID string) (map[string]any, error) {
	return map[string]any{"id": artisanID}, nil
}

func (m *mockArtisanSvc) UpdateProfile(ctx context.Context, artisanID string, updates map[string]any) error {
	return nil
}

// mockListingSvc returns not-found for every listing, for testing error mapping.
type mockListingSvc struct{}

func (m *mockListingSvc) CreateListing(ctx context.Context, artisanID, idempotencyKey string, listing map[string]any) (string, error) {
	return "", nil
}

func (m *mockListingSvc) UpdateListing(ctx context.Context, listingID, idempotencyKey string, updates map[string]any) error {
	return nil
}

func (m *mockListingSvc) SubmitForReview(ctx context.Context, listingID, idempotencyKey string) error {
	return nil
}

func (m *mockListingSvc) ApproveListing(ctx context.Context, listingID, reviewerID, idempotencyKey string, editedTranslations []map[string]any) error {
	return nil
}

func (m *mockListingSvc) GetListing(ctx context.Context, listingID string) (map[string]any, error) {
	return nil, pkgdomain.NotFound("listing not found")
}

func (m *mockListingSvc) GetListingSummary(ctx context.Context, listingID string) (map[string]any, error) {
	return nil, pkgdomain.NotFound("listing not found")
}

func (m *mockListingSvc) BatchGetListingSummaries(ctx context.Context, ids []string) ([]map[string]any, error) {
	return nil, nil
}

func (m *mockListingSvc) ListListings(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	return nil, nil
}

func (m *mockListingSvc) SealProvenance(ctx context.Context, listingID, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	return nil, pkgdomain.NotFound("listing not found")
}
