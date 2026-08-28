// services/core-svc/cmd/seed-ontology/main.go

// Command seed-ontology loads the craft graph from CSV into Postgres. It is
// idempotent: crafts are upserted by code and aliases by (alias, script), so a
// re-run against a live database updates rows in place and never renumbers an
// id another table points at.
//
// When REDIS_ADDR is set it also rebuilds the cached index and announces the new
// version, so a running fleet picks the new aliases up without a restart.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/logger"
	pkgpostgres "github.com/ZoroNewbie00/kalakriti/pkg/postgres"
	pkgredis "github.com/ZoroNewbie00/kalakriti/pkg/redis"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/ontology"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo"
)

// seedTimeout bounds the whole load; a few hundred upserts is a matter of seconds.
const seedTimeout = 2 * time.Minute

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("seed-ontology failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		dsn       = flag.String("dsn", os.Getenv("POSTGRES_DSN"), "postgres connection string")
		craftsCSV = flag.String("crafts", "", "path to crafts.csv")
		aliasCSV  = flag.String("aliases", "", "path to aliases.csv")
		redisAddr = flag.String("redis", os.Getenv("REDIS_ADDR"), "redis address; refreshes the live index when set")
	)
	flag.Parse()

	if *dsn == "" {
		return errors.New("-dsn is required (or set POSTGRES_DSN)")
	}
	if *craftsCSV == "" || *aliasCSV == "" {
		return errors.New("-crafts and -aliases are required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, seedTimeout)
	defer cancel()

	log := logger.New(os.Getenv("LOG_LEVEL")).With("command", "seed-ontology")

	pool, err := pkgpostgres.New(ctx, pkgpostgres.Config{DSN: *dsn, MaxConns: 4, MinConns: 1})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	crafts, err := os.Open(*craftsCSV)
	if err != nil {
		return fmt.Errorf("opening %s: %w", *craftsCSV, err)
	}
	defer func() { _ = crafts.Close() }()

	aliases, err := os.Open(*aliasCSV)
	if err != nil {
		return fmt.Errorf("opening %s: %w", *aliasCSV, err)
	}
	defer func() { _ = aliases.Close() }()

	repository := repo.New(pool)
	result, err := ontology.Load(ctx, repository, crafts, aliases)
	if err != nil {
		return err
	}
	log.Info("ontology loaded", "crafts", result.Crafts, "aliases", result.Aliases)

	if *redisAddr == "" {
		log.Info("no redis address given, the running fleet will pick this up on its next refresh")
		return nil
	}

	rdb, err := pkgredis.New(ctx, pkgredis.Config{
		Addr:     *redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	if err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}
	defer func() { _ = rdb.Close() }()

	// Rebuilding here writes the new snapshot and publishes its version, which is
	// what every running replica's Watch loop is waiting for.
	registry := ontology.NewRegistry(repository, rdb, rdb, ontology.NoTransliteration(),
		ontology.RegistryConfig{}, log)
	stats, err := registry.Refresh(ctx)
	if err != nil {
		return fmt.Errorf("refreshing the cached index: %w", err)
	}
	log.Info("cached index refreshed and announced",
		"version", stats.Version, "crafts", stats.Crafts, "aliases", stats.Aliases)
	return nil
}
