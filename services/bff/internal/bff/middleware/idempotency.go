package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
	"github.com/ZoroNewbie00/kalakriti/pkg/idempotency"
)

// IdempotencyStore adapts the BFF's idempotency persistence (typically Postgres
// via sqlc) to the pkg/idempotency.Store interface.
type IdempotencyStore interface {
	GetOrInsert(ctx context.Context, scope, key, requestHash string, expiresAt time.Time) (rec idempotency.Record, inserted bool, err error)
	SaveResponse(ctx context.Context, scope, key string, response []byte) error
}

// Idempotency checks the X-Idempotency-Key header on mutating requests and
// replays the first response when the same key is retried. Routes mount this
// only on POST/PUT/PATCH/DELETE.
func Idempotency(store IdempotencyStore, scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Idempotency-Key")
			if key == "" {
				// No idempotency key → normal execution, no replay protection.
				next.ServeHTTP(w, r)
				return
			}

			// Read and hash the request body.
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpx.Error(w, domain.InvalidInput("cannot read request body"))
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			requestHash := idempotency.Hash256Hex(body)

			// Wrap the handler in a response recorder so we can capture and store it.
			rec := &responseRecorder{ResponseWriter: w, body: &bytes.Buffer{}}
			result, err := idempotency.Do(r.Context(), store, scope, key, requestHash, func(ctx context.Context) (capturedResponse, error) {
				next.ServeHTTP(rec, r.WithContext(ctx))
				if rec.status >= 500 {
					// rec already streamed this response straight through to
					// the real client (it wraps w) — nothing more to write.
					return capturedResponse{}, fmt.Errorf("upstream status %d", rec.status)
				}
				return capturedResponse{
					Status: rec.status,
					Header: rec.Header(),
					Body:   rec.body.Bytes(),
				}, nil
			})

			if err != nil {
				// rec.status is only set once next.ServeHTTP actually ran and
				// wrote through to the real client; every other error path
				// (key conflict, still-processing, store failure) never
				// reached the handler, so the client still needs a response.
				if rec.status == 0 {
					httpx.Error(w, err)
				}
				return
			}

			// Replay the stored response.
			for k, v := range result.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(result.Status)
			w.Write(result.Body)
		})
	}
}

type capturedResponse struct {
	Status int
	Header http.Header
	Body   []byte
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
