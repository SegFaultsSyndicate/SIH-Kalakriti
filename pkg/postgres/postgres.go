// pkg/postgres/postgres.go

// Package postgres builds the pgxpool.Pool every service shares and provides the
// one place transactions are opened, committed and rolled back correctly.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

// Config controls pool sizing. Zero values fall back to sane defaults, so a
// service only sets what it needs to override.
type Config struct {
	// DSN is the full connection string; the only field with no default.
	DSN string
	// MaxConns and MinConns bound the pool. 0 means "use the default".
	MaxConns int32
	MinConns int32
	// MaxConnLifetime and MaxConnIdleTime recycle connections so a long-lived
	// pool does not accumulate ones the database has quietly dropped.
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

const (
	defaultMaxConns        = 20
	defaultMinConns        = 2
	defaultMaxConnLifetime = 30 * time.Minute
	defaultMaxConnIdleTime = 5 * time.Minute
	pingTimeout            = 5 * time.Second
)

// New builds a pgxpool.Pool: pgvector's Go type is registered on every new
// connection via AfterConnect, and the pool is pinged before returning so a bad
// DSN or an unreachable database fails at startup, not on the first query.
func New(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if cfg.DSN == "" {
		return nil, errors.New("postgres: DSN is required")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parsing postgres dsn: %w", err)
	}

	poolCfg.MaxConns = orDefaultI32(cfg.MaxConns, defaultMaxConns)
	poolCfg.MinConns = orDefaultI32(cfg.MinConns, defaultMinConns)
	poolCfg.MaxConnLifetime = orDefaultD(cfg.MaxConnLifetime, defaultMaxConnLifetime)
	poolCfg.MaxConnIdleTime = orDefaultD(cfg.MaxConnIdleTime, defaultMaxConnIdleTime)
	poolCfg.HealthCheckPeriod = time.Minute

	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if err := pgxvec.RegisterTypes(ctx, conn); err != nil {
			return fmt.Errorf("registering pgvector types: %w", err)
		}
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating postgres pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}
	return pool, nil
}

func orDefaultI32(v, def int32) int32 {
	if v > 0 {
		return v
	}
	return def
}

func orDefaultD(v, def time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return def
}

// RunInTx runs fn inside a transaction: commits on success, rolls back on error,
// and rolls back then re-panics on a panic so the pool never leaks a hanging
// transaction and the caller's stack trace still surfaces.
func RunInTx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context, tx pgx.Tx) error) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = fmt.Errorf("%w (rollback also failed: %v)", err, rbErr)
			}
			return
		}
		if cErr := tx.Commit(ctx); cErr != nil {
			err = fmt.Errorf("committing transaction: %w", cErr)
		}
	}()

	err = fn(ctx, tx)
	return err
}
