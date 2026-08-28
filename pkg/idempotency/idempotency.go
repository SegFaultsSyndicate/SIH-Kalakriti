// pkg/idempotency/idempotency.go

// Package idempotency guards mutating RPCs against replay: the same (scope, key)
// pair runs its side effects at most once and returns the first result to every
// retry. It has no SQL and no service imports — a service adapts its own
// sqlc-generated idempotency_key queries (see migrations/queries/idempotency.sql)
// to the Store interface below.
package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// TTL is how long a key is honoured before it may be reused for a new request.
const TTL = 7 * 24 * time.Hour

// Record is the persisted state of one idempotency key.
type Record struct {
	// RequestHash is a content hash of the original request, used to detect a
	// key reused for a materially different request.
	RequestHash string
	// Response is nil until the first call to Do for this key has succeeded.
	Response []byte
}

// Store is the persistence Do needs. GetOrInsert must be atomic (get-or-insert in
// one round trip, e.g. an INSERT ... ON CONFLICT DO NOTHING unioned with a
// SELECT) so two concurrent callers with the same key cannot both proceed.
type Store interface {
	// GetOrInsert inserts (scope, key) with requestHash and expiresAt if no row
	// exists yet, and returns the row either way. inserted is true only for the
	// caller that performed the insert.
	GetOrInsert(ctx context.Context, scope, key, requestHash string, expiresAt time.Time) (rec Record, inserted bool, err error)
	// SaveResponse records the outcome of the first successful call.
	SaveResponse(ctx context.Context, scope, key string, response []byte) error
}

// Do runs fn at most once per (scope, key):
//   - First call: runs fn, stores its JSON-encoded result, returns it.
//   - Replay with the same requestHash: returns the stored result without
//     calling fn again.
//   - Replay with a different requestHash: returns domain.ErrConflict — the
//     caller reused a key for a different request, which is a client bug.
//   - Replay while the first call is still in flight (inserted response not yet
//     saved): returns domain.ErrUnavailable so the caller retries rather than
//     racing the in-flight side effects.
func Do[T any](ctx context.Context, store Store, scope, key, requestHash string, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	rec, inserted, err := store.GetOrInsert(ctx, scope, key, requestHash, time.Now().UTC().Add(TTL))
	if err != nil {
		return zero, fmt.Errorf("checking idempotency key %s/%s: %w", scope, key, err)
	}

	if !inserted {
		if rec.RequestHash != requestHash {
			return zero, fmt.Errorf("idempotency key %s/%s reused with a different request: %w", scope, key, domain.ErrConflict)
		}
		if rec.Response == nil {
			return zero, fmt.Errorf("idempotency key %s/%s is still being processed: %w", scope, key, domain.ErrUnavailable)
		}
		if err := json.Unmarshal(rec.Response, &zero); err != nil {
			return zero, fmt.Errorf("decoding stored response for %s/%s: %w", scope, key, err)
		}
		return zero, nil
	}

	result, err := fn(ctx)
	if err != nil {
		return zero, err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return zero, fmt.Errorf("encoding response for %s/%s: %w", scope, key, err)
	}
	if err := store.SaveResponse(ctx, scope, key, payload); err != nil {
		return zero, fmt.Errorf("saving response for %s/%s: %w", scope, key, err)
	}
	return result, nil
}

// Hash256Hex is a convenience for computing the request_hash column: the
// caller's canonical request bytes in, a lowercase hex SHA-256 out.
func Hash256Hex(requestBytes []byte) string { return sha256Hex(requestBytes) }
