// services/core-svc/internal/core/repo/repo.go

// Package repo adapts core-svc's sqlc-generated queries to the domain types the
// service layer works in. Nothing above this package sees a db.* type, and
// nothing in this package makes a business decision.
package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/repo/db"
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

// Pool exposes the underlying pool for health checks and for pkg/postgres.RunInTx.
func (r *Repo) Pool() *pgxpool.Pool { return r.pool }

// Tx is the transactional view of the repository: the same methods, bound to one
// transaction. A service that must write a business row and an outbox row
// atomically takes a Tx.
type Tx struct {
	q *db.Queries
}

// InTx runs fn inside a single transaction, committing on success and rolling
// back on error or panic. The callback receives a Tx bound to that transaction;
// every write it makes commits together or not at all.
func (r *Repo) InTx(ctx context.Context, fn func(ctx context.Context, tx *Tx) error) (err error) {
	pgtx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			// Roll back before re-panicking so the connection is not returned to
			// the pool holding an open transaction.
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

// PostgreSQL SQLSTATE codes translated into domain sentinels.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
)

// translate converts a pgx error into a domain sentinel so the service layer
// never inspects SQLSTATE codes itself. what names the thing being operated on,
// e.g. "artisan", and appears in the message the caller sees.
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
			// The constraint name is deliberately included: every constraint in
			// this schema is explicitly named, so it tells the caller exactly
			// which uniqueness rule they hit.
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
