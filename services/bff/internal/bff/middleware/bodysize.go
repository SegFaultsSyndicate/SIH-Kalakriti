package middleware

import (
	"net/http"
)

// MaxBodySize wraps every request body in http.MaxBytesReader to enforce a size
// cap. A body exceeding maxBytes is rejected before the handler sees it.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
