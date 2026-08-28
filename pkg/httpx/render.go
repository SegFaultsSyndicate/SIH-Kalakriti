// pkg/httpx/render.go
package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// JSON writes v as an application/json response with the given status code.
// Encoding failures are logged nowhere here (rendering happens after the
// status line is already committed in most call sites); callers that need to
// distinguish a marshal failure should marshal explicitly before calling.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes err as a JSON error body, mapping it to an HTTP status via
// pkg/domain's sentinel-error rules.
func Error(w http.ResponseWriter, err error) {
	domain.WriteHTTPError(w, err)
}

// NoContent writes an empty 204 response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
