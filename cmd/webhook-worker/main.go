package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/ZoroNewbie00/kalakriti/pkg/webhook"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("POSTGRES_DSN")
	}
	if dbURL == "" {
		slog.Error("DATABASE_URL or POSTGRES_DSN not set")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	manager := webhook.NewManager(db)
	worker := webhook.NewWorker(manager)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pollInterval := 5 * time.Second
	if interval := os.Getenv("WEBHOOK_POLL_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			pollInterval = d
		}
	}

	slog.Info("webhook worker started", "poll_interval", pollInterval)
	worker.Run(ctx, pollInterval)
}
