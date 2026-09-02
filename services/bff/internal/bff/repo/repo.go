// services/bff/internal/bff/repo/repo.go

// Package repo adapts bff's own sqlc-generated queries — currently just the
// shared idempotency_key table — to the domain types above it, mirroring
// services/core-svc/internal/core/repo's pattern.
package repo

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/repo/db"
)

// Repo owns the connection pool and hands out the query set bound to it. bff
// has no other repository need yet, so unlike core-svc's Repo this has no
// InTx: idempotency's two queries are each already atomic on their own.
type Repo struct {
	q db.Querier
}

// New builds a Repo over an existing pool.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{q: db.New(pool)}
}

// pgUniqueViolation is the SQLSTATE code for a uniqueness constraint hit.
const pgUniqueViolation = "23505"

// translate converts a pgx error into a domain sentinel so the caller never
// inspects SQLSTATE codes itself. what names the thing being operated on and
// appears in the message the caller sees.
func translate(err error, what string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s not found: %w", what, pkgdomain.ErrNotFound)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return fmt.Errorf("%s already exists (%s): %w", what, pgErr.ConstraintName, pkgdomain.ErrConflict)
	}
	return fmt.Errorf("%s: %w", what, err)
}
