package middleware

import (
	"net/http"
	"strings"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
)

// Auth extracts and validates the JWT from the Authorization header, injecting
// the principal into the request context. Routes mount this selectively: public
// routes skip it, protected routes require it.
func Auth(issuer *auth.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := r.Header.Get("Authorization")
			if hdr == "" {
				httpx.Error(w, domain.Unauthenticated("missing Authorization header"))
				return
			}

			token := strings.TrimPrefix(hdr, "Bearer ")
			if token == hdr {
				httpx.Error(w, domain.Unauthenticated("Authorization must be Bearer <token>"))
				return
			}

			claims, err := issuer.Verify(token, auth.KindAccess)
			if err != nil {
				httpx.Error(w, err)
				return
			}

			principal, err := claims.Principal()
			if err != nil {
				httpx.Error(w, err)
				return
			}

			ctx := auth.ContextWithPrincipal(r.Context(), principal)
			ctx = auth.ContextWithToken(ctx, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth mirrors pkg/auth's gRPC interceptor for a "public methods"
// RPC ("still attach a principal when a token happens to be present, so a
// public RPC can tell an anonymous caller from a signed-in one") -- except
// nothing on the HTTP side did this at all before now. GET /listings/:id and
// GET /listings/:id/attributes are mounted on the plain (non-authed) api
// group, matching core-svc's own PublicMethods() list for
// CatalogService/GetListing and CurationService/GetListingAttributes, both
// of which read the caller's own identity via authoriseFor to show a DRAFT
// listing back to the artisan who owns it. Since this route group never ran
// Auth() at all, that principal never reached r.Context(), so it never
// reached the outbound gRPC call either (see client.go's per-call metadata
// forwarding) -- core-svc's authoriseFor always saw an anonymous caller and
// always rejected an unpublished listing, masked by GetListing's own
// generic "not found" on any authoriseFor failure. Confirmed live: a
// listing read back immediately after the artisan who owns it created it
// 404'd, every time, with the row correctly present and correctly owned in
// Postgres -- the token just never made the trip. A missing or malformed
// header here is not an error, unlike Auth() -- the caller may genuinely be
// anonymous, which is a valid case this route already handles.
func OptionalAuth(issuer *auth.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := r.Header.Get("Authorization")
			token := strings.TrimPrefix(hdr, "Bearer ")
			if hdr == "" || token == hdr {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := issuer.Verify(token, auth.KindAccess)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			principal, err := claims.Principal()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := auth.ContextWithPrincipal(r.Context(), principal)
			ctx = auth.ContextWithToken(ctx, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
