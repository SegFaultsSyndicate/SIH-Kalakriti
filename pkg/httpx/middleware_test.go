// pkg/httpx/middleware_test.go
package httpx

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testLogger is a real slog.Logger discarding output -- CSRFProtection logs
// through it on a rejection, and a nil *slog.Logger panics on Warn (as any
// nil *slog.Logger does), so real call sites (Mux) always pass one.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(discardWriter{}, nil))
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// Regression coverage for WIRING_AUDIT_PLAN.md F-13: CSRFProtection used to
// be nested inside the same "len(AllowedOrigins) > 0" guard as CORS, so the
// common same-origin deployment (AllowedOrigins left empty) ran with no CSRF
// protection at all. It's now gated only by DisableCSRF, and SelfOrigin
// keeps a same-origin request working once real cross-origin allow-listing
// is configured.
func TestCSRFProtectionRunsEvenWithNoAllowedOrigins(t *testing.T) {
	cfg := Config{SelfOrigin: "https://kalakriti.in"}
	mw := CSRFProtection(cfg, testLogger())
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artisans", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if called {
		t.Error("handler ran for a forged cross-site Origin with AllowedOrigins empty -- CSRF protection did not engage at all")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestCSRFProtectionAllowsSelfOrigin(t *testing.T) {
	cfg := Config{
		AllowedOrigins: []string{"https://app.kalakriti.in"}, // a separately-hosted frontend
		SelfOrigin:     "https://kalakriti.in",               // this server's own NGINX-served origin
	}
	mw := CSRFProtection(cfg, testLogger())
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artisans", nil)
	req.Header.Set("Origin", "https://kalakriti.in") // same-origin, NOT in AllowedOrigins
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Errorf("same-origin request (matching SelfOrigin, not AllowedOrigins) was rejected with status %d -- setting CORS_ALLOWED_ORIGINS to only a separate frontend would 403 this server's own reverse-proxied traffic", rec.Code)
	}
}

func TestCORSDefaultHeadersIncludeBothIdempotencyKeySpellings(t *testing.T) {
	// web/packages/api/src/transport.ts sends both spellings on every
	// idempotency-protected mutation; a CORS preflight allow-list missing
	// either one fails preflight for cross-origin deployments even though
	// the header the server middleware actually reads (bare
	// "Idempotency-Key") is present.
	cfg := Config{AllowedOrigins: []string{"https://app.kalakriti.in"}}
	mw := CORS(cfg)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/artisans", nil)
	req.Header.Set("Origin", "https://app.kalakriti.in")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "X-Idempotency-Key")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	got := rec.Header().Get("Access-Control-Allow-Headers")
	if !contains(got, "X-Idempotency-Key") {
		t.Errorf("Access-Control-Allow-Headers = %q, missing X-Idempotency-Key", got)
	}
	if !contains(got, "Idempotency-Key") {
		t.Errorf("Access-Control-Allow-Headers = %q, missing Idempotency-Key", got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
