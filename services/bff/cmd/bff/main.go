package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff"
)

const drainTimeout = 15 * time.Second

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("bff exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Build JWT issuer.
	issuer, err := auth.NewIssuer(auth.Config{
		Secret:     mustEnv("JWT_SECRET"),
		Issuer:     "kalakriti",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		return err
	}

	// Connect to Redis.
	rdb := redis.NewClient(&redis.Options{
		Addr: getEnv("REDIS_ADDR", "localhost:6379"),
	})
	defer func() {
		if err := rdb.Close(); err != nil {
			logger.Warn("closing redis", "error", err)
		}
	}()

	// Build the server (service clients are nil stubs for now).
	srv, err := bff.NewServer(bff.Config{
		Addr:                   getEnv("ADDR", ":8080"),
		BaseURL:                mustEnv("BASE_URL"),
		WebDist:                getEnv("WEB_DIST", "./web/dist"),
		ChannelSvcAddr:         getEnv("CHANNEL_SVC_ADDR", "http://localhost:8083"),
		AllowedOrigins:         splitEnv("CORS_ALLOWED_ORIGINS"),
		ProvenancePublicKeyHex: os.Getenv("PROVENANCE_PUBLIC_KEY"),
		Logger:                 logger,
		Issuer:                 issuer,
		Redis:                  rdb,
		IdempStore:             nil, // TODO: wire Postgres idempotency store
		RateLimitPerIP:         100,
		RateLimitPerPrincipal:  1000,
		RateLimitWindow:        time.Minute,
		// Service clients: TODO wire gRPC clients to backend services.
		AuthSvc:    nil,
		ArtisanSvc: nil,
		MediaSvc:   nil,
		ListingSvc: nil,
		SearchSvc:  nil,
		PricingSvc: nil,
		OrderSvc:   nil,
		FollowSvc:  nil,
		StmtSvc:    nil,
		InsightSvc: nil,
		CatalogSvc: nil,
	})
	if err != nil {
		return err
	}

	httpServer := srv.HTTPServer()
	errCh := make(chan error, 1)

	go func() {
		logger.Info("bff starting", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining", "timeout", drainTimeout)
	case err := <-errCh:
		logger.Error("server failed, shutting down", "error", err)
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http server did not shut down cleanly", "error", err)
	}

	logger.Info("shutdown complete")
	return nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("missing required environment variable", "key", key)
		os.Exit(1)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitEnv reads a comma-separated env var into a slice, or nil if unset.
func splitEnv(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
