// services/collab-svc/internal/collab/repo/repo.go

// Package repo adapts collab-svc's sqlc-generated queries to the domain types
// the service layer works in. Nothing above this package sees a db.* type,
// and nothing in this package makes a business decision.
package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/collab-svc/internal/collab/repo/db"
)

// Repo owns the connection pool and hands out query sets bound either to the
// pool (autocommit) or to a transaction.
type Repo struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// New builds a Repo over an existing pool.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool, q: db.New(pool)}
}

// Pool exposes the underlying pool for health checks and pkg/postgres.RunInTx.
func (r *Repo) Pool() *pgxpool.Pool { return r.pool }

// Tx is the transactional view of the repository: the same methods, bound to
// one transaction. The saga's every write goes through a Tx so a state
// change, its audit row and its outbox row commit together or not at all.
type Tx struct {
	q *db.Queries
}

// InTx runs fn inside a single transaction, committing on success and rolling
// back on error or panic.
func (r *Repo) InTx(ctx context.Context, fn func(ctx context.Context, tx *Tx) error) (err error) {
	pgtx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = pgtx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			if rbErr := pgtx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, fmt.Errorf("rolling back: %w", rbErr))
			}
			return
		}
		if cErr := pgtx.Commit(ctx); cErr != nil {
			err = fmt.Errorf("committing transaction: %w", cErr)
		}
	}()

	return fn(ctx, &Tx{q: r.q.WithTx(pgtx)})
}

// --- error translation -------------------------------------------------------

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
)

// translate converts a pgx error into a domain sentinel so the service layer
// never inspects SQLSTATE codes itself.
func translate(err error, what string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s not found: %w", what, pkgdomain.ErrNotFound)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return fmt.Errorf("%s already exists (%s): %w", what, pgErr.ConstraintName, pkgdomain.ErrConflict)
		case pgForeignKeyViolation:
			return fmt.Errorf("%s references a row that does not exist (%s): %w",
				what, pgErr.ConstraintName, pkgdomain.ErrInvalidInput)
		case pgCheckViolation:
			return fmt.Errorf("%s violates constraint %s: %w", what, pgErr.ConstraintName, pkgdomain.ErrInvalidInput)
		}
	}
	return fmt.Errorf("%s: %w", what, err)
}
