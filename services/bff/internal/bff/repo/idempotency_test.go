// services/bff/internal/bff/repo/idempotency_test.go
package repo

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/repo/db"
)

// fakeQuerier embeds the real interface (left nil) and overrides only the
// methods a test needs, same pattern used for the gRPC adapters in
// services/bff/internal/bff/client.
type fakeQuerier struct {
	db.Querier
	getOrInsert func(ctx context.Context, arg db.GetOrInsertIdempotencyKeyParams) (db.GetOrInsertIdempotencyKeyRow, error)
	setResponse func(ctx context.Context, arg db.SetIdempotentResponseParams) (int64, error)
}

func (f *fakeQuerier) GetOrInsertIdempotencyKey(ctx context.Context, arg db.GetOrInsertIdempotencyKeyParams) (db.GetOrInsertIdempotencyKeyRow, error) {
	return f.getOrInsert(ctx, arg)
}

func (f *fakeQuerier) SetIdempotentResponse(ctx context.Context, arg db.SetIdempotentResponseParams) (int64, error) {
	return f.setResponse(ctx, arg)
}

func TestGetOrInsertReturnsTheWinningInsert(t *testing.T) {
	store := NewIdempotencyStore(&Repo{q: &fakeQuerier{
		getOrInsert: func(ctx context.Context, arg db.GetOrInsertIdempotencyKeyParams) (db.GetOrInsertIdempotencyKeyRow, error) {
			assert.Equal(t, "bff", arg.Scope)
			assert.Equal(t, "idem-123", arg.Key)
			assert.Equal(t, "hash-abc", arg.RequestHash)
			return db.GetOrInsertIdempotencyKeyRow{RequestHash: arg.RequestHash, IsNew: true}, nil
		},
	}})

	rec, inserted, err := store.GetOrInsert(context.Background(), "bff", "idem-123", "hash-abc", time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.True(t, inserted)
	assert.Equal(t, "hash-abc", rec.RequestHash)
}

func TestGetOrInsertReturnsTheExistingRowOnReplay(t *testing.T) {
	store := NewIdempotencyStore(&Repo{q: &fakeQuerier{
		getOrInsert: func(ctx context.Context, arg db.GetOrInsertIdempotencyKeyParams) (db.GetOrInsertIdempotencyKeyRow, error) {
			return db.GetOrInsertIdempotencyKeyRow{RequestHash: "hash-abc", Response: []byte(`{"ok":true}`), IsNew: false}, nil
		},
	}})

	rec, inserted, err := store.GetOrInsert(context.Background(), "bff", "idem-123", "hash-abc", time.Now())
	require.NoError(t, err)
	assert.False(t, inserted)
	assert.Equal(t, []byte(`{"ok":true}`), rec.Response)
}

func TestGetOrInsertTranslatesAUniqueViolation(t *testing.T) {
	store := NewIdempotencyStore(&Repo{q: &fakeQuerier{
		getOrInsert: func(ctx context.Context, arg db.GetOrInsertIdempotencyKeyParams) (db.GetOrInsertIdempotencyKeyRow, error) {
			return db.GetOrInsertIdempotencyKeyRow{}, &pgconn.PgError{Code: pgUniqueViolation, ConstraintName: "idempotency_key_scope_key_key"}
		},
	}})

	_, _, err := store.GetOrInsert(context.Background(), "bff", "idem-123", "hash-abc", time.Now())
	assert.True(t, domain.IsConflict(err))
}

func TestGetOrInsertTranslatesNoRows(t *testing.T) {
	store := NewIdempotencyStore(&Repo{q: &fakeQuerier{
		getOrInsert: func(ctx context.Context, arg db.GetOrInsertIdempotencyKeyParams) (db.GetOrInsertIdempotencyKeyRow, error) {
			return db.GetOrInsertIdempotencyKeyRow{}, pgx.ErrNoRows
		},
	}})

	_, _, err := store.GetOrInsert(context.Background(), "bff", "idem-123", "hash-abc", time.Now())
	assert.True(t, domain.IsNotFound(err))
}

func TestSaveResponsePassesFieldsThrough(t *testing.T) {
	store := NewIdempotencyStore(&Repo{q: &fakeQuerier{
		setResponse: func(ctx context.Context, arg db.SetIdempotentResponseParams) (int64, error) {
			assert.Equal(t, "bff", arg.Scope)
			assert.Equal(t, "idem-123", arg.Key)
			assert.Equal(t, []byte(`{"ok":true}`), arg.Response)
			return 1, nil
		},
	}})

	err := store.SaveResponse(context.Background(), "bff", "idem-123", []byte(`{"ok":true}`))
	assert.NoError(t, err)
}
