// services/bff/internal/bff/route_parity_test.go
//
// Regression coverage for WIRING_AUDIT_PLAN.md F-10: services/bff/openapi.json
// (the reverse direction). web/packages/api/src/route-parity.test.ts already
// asserts every spec path has a reference in operations.ts; nothing asserted
// the other direction, so a route added to server.go without a matching spec
// entry had no signal at all until a hand-rolled frontend fetch() got its
// shape wrong (the phone-change routes, see CLAUDE.md). This uses gin's own
// Routes() rather than parsing server.go, since a hand-rolled regex over Go
// source produced real false positives earlier in this audit (nested route
// groups, :id vs {id} normalization) -- Routes() is authoritative.
package bff

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	assets "github.com/ZoroNewbie00/kalakriti/services/bff"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/mosje"
	"github.com/redis/go-redis/v9"
)

// Routes intentionally outside the /api/v1 JSON contract (server-rendered
// SEO pages, sitemap/robots) or deliberately undocumented per
// WIRING_AUDIT_PLAN.md F-10 (the inbound HMAC-verified payment webhook, and
// the meta route that serves this very spec).
var routeParityExempt = map[string]bool{
	"GET /listing/:slug":            true,
	"GET /artisan/:slug":            true,
	"GET /v/:code":                  true,
	"GET /v/:code/verify.json":      true,
	"GET /export/indiahandmade":     true,
	"GET /sitemap.xml":              true,
	"GET /robots.txt":               true,
	"POST /api/v1/payments/webhook": true,
	"GET /api/v1/openapi.json":      true,
	"GET /healthz":                  true,
}

func TestOpenAPISpecHasEveryAPIRoute(t *testing.T) {
	issuer, err := auth.NewIssuer(auth.Config{
		Secret:     "dev-secret-change-in-prod-32bytes-minimum",
		Issuer:     "kalakriti",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})

	srv, err := NewServer(Config{
		Addr:    ":0",
		BaseURL: "http://localhost:8000",
		WebDist: t.TempDir(),
		Issuer:  issuer,
		Redis:   rdb,
		Mosje:   mosje.New(nil, nil), // mounts the tier-4 routes so they are checked too
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	var spec struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(assets.OpenAPIJSON, &spec); err != nil {
		t.Fatalf("parsing embedded openapi.json: %v", err)
	}

	var missing []string
	for _, rt := range srv.router.Routes() {
		key := rt.Method + " " + rt.Path
		if routeParityExempt[key] {
			continue
		}
		if !strings.HasPrefix(rt.Path, "/api/v1/") && rt.Path != "/api/v1" {
			// Outside the documented JSON contract and not explicitly
			// exempted above -- fail loudly rather than silently skip, so a
			// newly added non-API route can't hide from this check by
			// accident.
			t.Errorf("route %s is outside /api/v1 and not in routeParityExempt -- add it there if intentional", key)
			continue
		}

		specPath := ginPathToSpecPath(strings.TrimPrefix(rt.Path, "/api/v1"))
		methods, ok := spec.Paths[specPath]
		if !ok {
			missing = append(missing, key+" (spec has no path "+specPath+")")
			continue
		}
		if _, ok := methods[strings.ToLower(rt.Method)]; !ok {
			missing = append(missing, key+" (spec path "+specPath+" has no "+rt.Method+")")
		}
	}

	if len(missing) > 0 {
		t.Errorf("routes registered in server.go but missing from openapi.json:\n%s", strings.Join(missing, "\n"))
	}
}

// ginPathToSpecPath converts gin's ":param" route-parameter syntax to
// OpenAPI's "{param}" syntax, e.g. "/listings/:id/submit" -> "/listings/{id}/submit".
func ginPathToSpecPath(ginPath string) string {
	segments := strings.Split(ginPath, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			segments[i] = "{" + seg[1:] + "}"
		}
	}
	return strings.Join(segments, "/")
}
