// pkg/httpx/error.go

package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

// Translator translates message keys to localized strings.
type Translator func(ctx context.Context, key string) string

// Error writes a domain error as JSON with proper status code and optional translation.
func Error(w http.ResponseWriter, err error) {
	ErrorWithTranslator(w, nil, err)
}

// ErrorWithTranslator writes a domain error with translation support.
func ErrorWithTranslator(w http.ResponseWriter, translator Translator, err error) {
	status := 500
	message := err.Error()

	// Translate if translator provided and error has translation key
	if translator != nil {
		if key, ok := domain.GetTranslationKey(err); ok {
			ctx := w.(*responseWriter).ctx // ponytail: assumes wrapped responseWriter, works for BFF
			if ctx != nil {
				message = translator(ctx, key)
			}
		}
	}

	// Map domain errors to HTTP status codes
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthenticated):
		status = http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, domain.ErrUnavailable):
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": message,
	})
}

// responseWriter wraps http.ResponseWriter to store context.
type responseWriter struct {
	http.ResponseWriter
	ctx context.Context
}
