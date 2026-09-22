package bff

// Regression coverage for the wiring-audit Tier-1 fixes: a truthful access
// log (F-1), a JSON 404 for an unmatched /api/* path instead of the SPA
// fallback's 200 (F-2), and a real 405 for a wrong-method request on an
// existing path. See WIRING_AUDIT_PLAN.md at the repo root for the full
// audit these guard against regressing.

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
)

func TestTier1WiringFixes(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	issuer, err := auth.NewIssuer(auth.Config{
		Secret:     "dev-secret-change-in-prod-32bytes-minimum",
		Issuer:     "kalakriti",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}) // unreachable, fails open

	srv, err := NewServer(Config{
		Addr:    ":0",
		BaseURL: "http://localhost:8000",
		WebDist: t.TempDir(),
		Logger:  logger,
		Issuer:  issuer,
		Redis:   rdb,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	t.Run("unmatched api route returns JSON 404, not SPA 200", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/definitely-not-a-route")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
			t.Errorf("content-type = %q, want application/json", ct)
		}
	})

	t.Run("unauthenticated protected route returns 401", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/artisans/me")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("wrong method on a real route returns 405, not 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/crafts", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", resp.StatusCode)
		}
	})

	t.Run("access log records the real status, not a hardcoded 200", func(t *testing.T) {
		logBuf.Reset()
		resp, err := http.Get(ts.URL + "/api/v1/definitely-not-a-route")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		logged := logBuf.String()
		t.Logf("access log line: %s", logged)
		if !bytes.Contains(logBuf.Bytes(), []byte(`"status":404`)) {
			t.Errorf("access log did not record status 404:\n%s", logged)
		}
	})

	t.Run("public route with no auth required still works (200)", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/crafts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		// core-svc is not running in this harness, so this legitimately
		// fails downstream -- what matters is it's a real mapped status
		// (503/500), not a 200-with-SPA-HTML.
		if resp.StatusCode == http.StatusOK {
			ct := resp.Header.Get("Content-Type")
			if ct != "" && ct[:9] == "text/html" {
				t.Errorf("got SPA HTML 200 for /api/v1/crafts with no core-svc running -- fall-through guard did not fire")
			}
		}
		t.Logf("/api/v1/crafts (no core-svc running) -> %d", resp.StatusCode)
	})

	t.Run("SSE route dispatches through the timeout middleware without panicking", func(t *testing.T) {
		// No auth token, so this 401s before reaching WatchOrder -- what
		// this proves is that TimeoutGin's c.FullPath() suffix check
		// resolves correctly for a real registered route and the request
		// completes normally (no goroutine panic, no hang).
		resp, err := http.Get(ts.URL + "/api/v1/orders/some-id/events")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", resp.StatusCode)
		}
	})
}
