// services/bff/internal/bff/middleware/auth_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/segfaultsyndicate/kalakriti/pkg/auth"
	"github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

func TestAuthReturns401ForMissingToken(t *testing.T) {
	issuer := mustIssuer(t)
	handler := Auth(issuer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing auth: got status %d, want 401", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Error("expected JSON error body")
	}
}

func TestAuthReturns401ForInvalidToken(t *testing.T) {
	issuer := mustIssuer(t)
	handler := Auth(issuer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid token: got status %d, want 401", rec.Code)
	}
}

func TestAuthReturns401ForMalformedBearer(t *testing.T) {
	issuer := mustIssuer(t)
	handler := Auth(issuer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "NotBearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("malformed bearer: got status %d, want 401", rec.Code)
	}
}

func TestAuthAllowsValidToken(t *testing.T) {
	issuer := mustIssuer(t)
	pair, _ := issuer.Issue(auth.Subject{ID: "artisan-1", Role: auth.RoleArtisan})

	var gotPrincipal auth.Principal
	handler := Auth(issuer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.PrincipalFrom(r.Context())
		if !ok {
			t.Fatal("principal not in context")
		}
		gotPrincipal = p
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("valid token: got status %d, want 200", rec.Code)
	}
	if gotPrincipal.Subject != "artisan-1" {
		t.Errorf("principal.Subject = %q, want artisan-1", gotPrincipal.Subject)
	}
}

func TestHTTPStatusMapsUnauthenticatedTo401(t *testing.T) {
	status := domain.HTTPStatus(domain.ErrUnauthenticated)
	if status != http.StatusUnauthorized {
		t.Errorf("ErrUnauthenticated mapped to %d, want 401", status)
	}

	status = domain.HTTPStatus(domain.Unauthenticated("test"))
	if status != http.StatusUnauthorized {
		t.Errorf("wrapped ErrUnauthenticated mapped to %d, want 401", status)
	}
}

func TestHTTPStatusMapsForbiddenTo403(t *testing.T) {
	status := domain.HTTPStatus(domain.ErrForbidden)
	if status != http.StatusForbidden {
		t.Errorf("ErrForbidden mapped to %d, want 403", status)
	}
}

func mustIssuer(t *testing.T) *auth.Issuer {
	t.Helper()
	iss, err := auth.NewIssuer(auth.Config{
		Secret:     "0123456789abcdef0123456789abcdef",
		Issuer:     "kalakriti-test",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	return iss
}
