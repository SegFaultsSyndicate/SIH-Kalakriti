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
