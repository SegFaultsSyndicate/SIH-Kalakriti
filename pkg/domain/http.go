// pkg/domain/http.go
package domain

import (
	"encoding/json"
	"errors"
	"net/http"
)

// HTTPErrorBody is the JSON shape written for every mapped error.
type HTTPErrorBody struct {
	// Error is a stable machine-readable code, e.g. "not_found".
	Error string `json:"error"`
	// Message is safe to show a user; an unmapped error never leaks its detail here.
	Message string `json:"message"`
}

// HTTPStatus maps a domain error to the HTTP status code it should produce.
// An unmapped error becomes 500 rather than guessing.
func HTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthenticated):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// httpErrorCode returns the stable machine-readable code for a mapped status.
func httpErrorCode(status int) string {
	switch status {
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusBadRequest:
		return "invalid_input"
	case http.StatusUnauthorized:
		return "unauthenticated"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusServiceUnavailable:
		return "unavailable"
	default:
		return "internal_error"
	}
}

// WriteHTTPError writes the mapped status and a JSON error body. The message on
// an unmapped (500) error is replaced with a generic one so internals never
// reach a client.
func WriteHTTPError(w http.ResponseWriter, err error) {
	status := HTTPStatus(err)
	message := err.Error()
	if status == http.StatusInternalServerError {
		message = "internal error"
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(HTTPErrorBody{Error: httpErrorCode(status), Message: message})
}
