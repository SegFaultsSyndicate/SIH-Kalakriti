// pkg/i18n/middleware.go

package i18n

import (
	"net/http"
)

// Middleware extracts locale from Accept-Language header and stores in context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptLang := r.Header.Get("Accept-Language")
		locale := ParseAcceptLanguage(acceptLang)
		ctx := WithLocale(r.Context(), locale)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
