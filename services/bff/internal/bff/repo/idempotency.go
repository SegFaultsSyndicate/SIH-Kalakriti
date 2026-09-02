// services/bff/internal/bff/repo/idempotency.go
package repo

import (
	"context"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/idempotency"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"

	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/repo/db"
)

// IdempotencyStore adapts bff's idempotency_key queries to
// pkg/idempotency.Store (and, via that identical shape, to
// middleware.IdempotencyStore).
type IdempotencyStore struct {
	repo *Repo
}

// NewIdempotencyStore builds the store backing idempotency.Do.
func NewIdempotencyStore(r *Repo) *IdempotencyStore { return &IdempotencyStore{repo: r} }

var _ idempotency.Store = (*IdempotencyStore)(nil)

// GetOrInsert atomically claims a key or returns the existing record. The CTE
// in GetOrInsertIdempotencyKey is what makes the claim atomic; inserted is
// true only for the caller that won the race and must therefore do the work.
func (s *IdempotencyStore) GetOrInsert(
	ctx context.Context,
	scope, key, requestHash string,
	expiresAt time.Time,
) (idempotency.Record, bool, error) {
	row, err := s.repo.q.GetOrInsertIdempotencyKey(ctx, db.GetOrInsertIdempotencyKeyParams{
		ID:          ids.New(),
		Scope:       scope,
		Key:         key,
		RequestHash: requestHash,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return idempotency.Record{}, false, translate(err, "idempotency key")
	}
	return idempotency.Record{
		RequestHash: row.RequestHash,
		Response:    row.Response,
	}, row.IsNew, nil
}

// SaveResponse stores the encoded response for a completed first call. The
// query only updates a row whose response is still NULL, so a late writer
// can never overwrite a response another caller already recorded.
func (s *IdempotencyStore) SaveResponse(ctx context.Context, scope, key string, response []byte) error {
	if _, err := s.repo.q.SetIdempotentResponse(ctx, db.SetIdempotentResponseParams{
		Scope:    scope,
		Key:      key,
		Response: response,
	}); err != nil {
		return translate(err, "idempotency response")
	}
	return nil
}
