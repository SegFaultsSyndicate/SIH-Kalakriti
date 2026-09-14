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

	// collab-svc's own gRPC port (see its main.go default: defaultGRPCAddr
	// :50053); COLLAB_SVC_ADDR already existed in docker-compose.full.yml's
	// bff block pointing at collab-svc's HTTP port (8083) instead — same
	// stale-port shape as CORE_SVC_ADDR/SEARCH_SVC_ADDR were, never actually
	// read by any Go code until now.
	collabConn, err := grpc.NewClient(getEnv("COLLAB_SVC_ADDR", "localhost:50053"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling collab-svc: %w", err)
	}
	defer collabConn.Close()

	// channel-svc's gRPC port (see its main.go default: :9096, distinct from
	// collab-svc's :50053 to avoid a port collision) — CHANNEL_SVC_ADDR is
	// already used for channel-svc's HTTP port, so this is a separate var.
	channelConn, err := grpc.NewClient(getEnv("CHANNEL_SVC_GRPC_ADDR", "localhost:9096"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling channel-svc: %w", err)
	}
	defer channelConn.Close()

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
		// 1:1 with these interfaces.
		AuthSvc:    client.NewAuth(coreConn, rdb),
		ArtisanSvc: client.NewArtisan(coreConn),
		MediaSvc:   client.NewMedia(coreConn),
		ListingSvc: client.NewListing(coreConn),
		SearchSvc:  client.NewSearch(searchConn),
		PricingSvc: client.NewPricing(coreConn),
		OrderSvc:   client.NewOrder(collabConn),
		FollowSvc:  client.NewFollow(channelConn),
		StmtSvc:    insightClient,
		InsightSvc: insightClient,
		CatalogSvc: client.NewCatalog(coreConn),
		B2BSvc:     client.NewB2B(coreConn),
		TrendSvc:   client.NewTrends(coreConn),
		BadgeSvc:   client.NewBadges(coreConn),
		SchemeSvc:  client.NewSchemes(coreConn),
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
