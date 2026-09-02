package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/postgres"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/client"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/repo"
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

	// Connect to Postgres, for the idempotency_key table this shares with
	// every other service — it is not bff's own schema, just the one table
	// bff also needs.
	pgPool, err := postgres.New(ctx, postgres.Config{DSN: mustEnv("POSTGRES_DSN")})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pgPool.Close()
	idempStore := repo.NewIdempotencyStore(repo.New(pgPool))

	// Dial backend services. Each is a single shared connection per backend;
	// grpc.NewClient doesn't connect until first use, so a backend that's down
	// at startup doesn't block bff from starting.
	coreConn, err := grpc.NewClient(getEnv("CORE_SVC_ADDR", "localhost:50051"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling core-svc: %w", err)
	}
	defer coreConn.Close()

	searchConn, err := grpc.NewClient(getEnv("SEARCH_SVC_ADDR", "localhost:50052"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling search-svc: %w", err)
	}
	defer searchConn.Close()

	insightConn, err := grpc.NewClient(getEnv("INSIGHT_SVC_ADDR", "localhost:8085"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling insight-svc: %w", err)
	}
	defer insightConn.Close()
	insightClient := client.NewInsight(insightConn) // satisfies both InsightSvc and StmtSvc

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
		IdempStore:             idempStore,
		RateLimitPerIP:         100,
		RateLimitPerPrincipal:  1000,
		RateLimitWindow:        time.Minute,
		// Service clients wired to real backends where the RPC shapes line up
		// 1:1 with these interfaces. ListingSvc, PricingSvc, OrderSvc and
		// FollowSvc stay nil: each needs either a bff API contract change or a
		// backend RPC that doesn't exist yet (see git history for the
		// per-service gaps found while scoping this).
		AuthSvc:    client.NewAuth(coreConn, rdb),
		ArtisanSvc: client.NewArtisan(coreConn),
		MediaSvc:   client.NewMedia(coreConn),
		ListingSvc: nil,
		SearchSvc:  client.NewSearch(searchConn),
		PricingSvc: nil,
		OrderSvc:   nil,
		FollowSvc:  nil,
		StmtSvc:    insightClient,
		InsightSvc: insightClient,
		CatalogSvc: client.NewCatalog(coreConn),
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
