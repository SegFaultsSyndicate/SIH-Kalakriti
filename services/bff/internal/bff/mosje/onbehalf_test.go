package mosje

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
)

type fakeResolver struct {
	linked map[string]bool
	calls  int
}

func (f *fakeResolver) ResolveOnBehalf(_ context.Context, artisanID string) (bool, error) {
	f.calls++
	return f.linked[artisanID], nil
}

func TestOnBehalf(t *testing.T) {
	gin.SetMode(gin.TestMode)
	issuer, err := auth.NewIssuer(auth.Config{Secret: "test-secret-test-secret-test-secret!", Issuer: "kalakriti",
		AccessTTL: time.Minute, RefreshTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	agent := auth.Principal{Subject: uuid.NewString(), Role: auth.RoleFieldAgent}
	artisanP := auth.Principal{Subject: uuid.NewString(), Role: auth.RoleArtisan}
	linked, stranger := uuid.NewString(), uuid.NewString()

	cases := []struct {
		name       string
		caller     *auth.Principal
		method     string
		path       string
		target     string
		wantStatus int
		wantActing bool
	}{
		{"no header passes through", &agent, "POST", "/api/v1/listings", "", 200, false},
		{"allowed route, linked artisan", &agent, "POST", "/api/v1/listings", linked, 200, true},
		{"allowed route with a path param", &agent, "PATCH", "/api/v1/listings/abc", linked, 200, true},
		{"unlinked artisan", &agent, "POST", "/api/v1/listings", stranger, 403, false},
		{"denied: consent withdrawal", &agent, "DELETE", "/api/v1/finance/links/abc", linked, 403, false},
		{"denied: helper revoke", &agent, "DELETE", "/api/v1/helpers/abc", linked, 403, false},
		{"denied: orders", &agent, "POST", "/api/v1/orders/bulk", linked, 403, false},
		{"denied: phone change", &agent, "POST", "/api/v1/auth/phone/change/request", linked, 403, false},
		{"artisans cannot use the header", &artisanP, "POST", "/api/v1/listings", linked, 403, false},
		{"anonymous", nil, "POST", "/api/v1/listings", linked, 401, false},
		{"target must be a uuid", &agent, "POST", "/api/v1/listings", "not-a-uuid", 400, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := &fakeResolver{linked: map[string]bool{linked: true}}
			var seen auth.Principal
			var seenToken string
			r := gin.New()
			g := r.Group("/api/v1", func(c *gin.Context) {
				if tc.caller != nil {
					ctx := auth.ContextWithPrincipal(c.Request.Context(), *tc.caller)
					c.Request = c.Request.WithContext(auth.ContextWithToken(ctx, "agent-token"))
				}
				c.Next()
			}, OnBehalf(issuer, res))
			h := func(c *gin.Context) {
				seen, _ = auth.PrincipalFrom(c.Request.Context())
				seenToken, _ = auth.TokenFrom(c.Request.Context())
				c.Status(200)
			}
			for _, route := range []struct{ m, p string }{
				{"POST", "/listings"}, {"PATCH", "/listings/:id"}, {"DELETE", "/finance/links/:id"},
				{"DELETE", "/helpers/:id"}, {"POST", "/orders/bulk"}, {"POST", "/auth/phone/change/request"},
			} {
				g.Handle(route.m, route.p, h)
			}

			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.target != "" {
				req.Header.Set(HeaderOnBehalfOf, tc.target)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if !tc.wantActing {
				if tc.wantStatus == 200 && (seen.Subject != agent.Subject || seenToken != "agent-token") {
					t.Fatalf("principal must be untouched without the header: %+v", seen)
				}
				return
			}
			if seen.Subject != linked || seen.Role != auth.RoleArtisan || seen.Actor != agent.Subject || seen.PhoneE164 != "" {
				t.Fatalf("acting principal = %+v", seen)
			}
			if seenToken == "agent-token" || seenToken == "" {
				t.Fatal("the forwarded token must be the minted artisan token")
			}
		})
	}
}

// Mount panics on a conflicting route pattern; mounting beside the real
// neighbours it shares prefixes with proves it does not.
func TestMountRegistersWithoutConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	api.GET("/listings/:id", func(*gin.Context) {})
	api.GET("/listings/:id/attributes", func(*gin.Context) {})
	New(nil, nil).Mount(api, api.Group(""), func(h http.HandlerFunc) http.HandlerFunc { return h })
	for _, rt := range r.Routes() {
		if rt.Method != "GET" && rt.Method != "POST" && rt.Method != "PUT" && rt.Method != "PATCH" && rt.Method != "DELETE" {
			t.Fatalf("unexpected method %s", rt.Method)
		}
	}
}
