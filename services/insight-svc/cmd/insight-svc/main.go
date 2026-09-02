// services/insight-svc/cmd/insight-svc/main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/ZoroNewbie00/kalakriti/pkg/config"
	"github.com/ZoroNewbie00/kalakriti/pkg/crypto"
	"github.com/ZoroNewbie00/kalakriti/pkg/logger"
	insightv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/insight/v1"
	pkgpostgres "github.com/ZoroNewbie00/kalakriti/pkg/postgres"
	"github.com/ZoroNewbie00/kalakriti/pkg/qrcode"
	"github.com/ZoroNewbie00/kalakriti/pkg/shortcode"
	"github.com/ZoroNewbie00/kalakriti/pkg/storage"

	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/handler"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/repo"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/service"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/wiring"
)

const serviceName = "insight-svc"

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("insight-svc exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	log := logger.New(cfg.server.LogLevel).With("service", serviceName, "env", cfg.server.Env)

	pool, err := pkgpostgres.New(ctx, pkgpostgres.Config{
		DSN: cfg.postgres.DSN, MaxConns: cfg.postgres.MaxConns, MinConns: cfg.postgres.MinConns,
	})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	objects, err := storage.New(ctx, storage.Config{
		Endpoint: cfg.s3.Endpoint, AccessKey: cfg.s3.AccessKey, SecretKey: cfg.s3.SecretKey,
		Bucket: cfg.s3.Bucket, Region: cfg.s3.Region, UseSSL: cfg.s3.UseSSL, PublicURL: cfg.s3.PublicURL,
	})
	if err != nil {
		return fmt.Errorf("connecting to object storage: %w", err)
	}

	signer, err := loadSigner(cfg.signingKey, cfg.keyID, log)
	if err != nil {
		return fmt.Errorf("configuring signer: %w", err)
	}

	repository := repo.New(pool)
	store := wiring.NewStore(repository)
	svc := service.New(service.Config{
		Store:         store,
		Signer:        signer,
		QRGenerator:   qrcode.NewGenerator(cfg.baseURL),
		CodeGenerator: shortcode.NewGenerator(repository.ShortCodeExists),
		S3Client:      objects,
		VerifyBaseURL: cfg.baseURL,
		MinBucketSize: cfg.minBucketSize,
		PresignExpiry: 24 * time.Hour,
	})
	h := handler.New(svc, cfg.baseURL)

	grpcServer := grpc.NewServer()
	insightv1.RegisterInsightServiceServer(grpcServer, h)
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", cfg.grpcAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", cfg.grpcAddr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("grpc server listening", "addr", cfg.grpcAddr)
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		log.Error("server failed", "error", err)
	}

	grpcServer.GracefulStop()
	log.Info("shutdown complete")
	return nil
}

// loadSigner builds the Ed25519 signer income statements are signed with. An
// unset key generates an ephemeral one for the process lifetime, so the
// service still starts in dev without one configured.
func loadSigner(privateKeyHex, keyID string, log *slog.Logger) (*crypto.Signer, error) {
	if privateKeyHex == "" {
		generatedHex, _, err := crypto.GenerateKeypair()
		if err != nil {
			return nil, fmt.Errorf("generating ephemeral keypair: %w", err)
		}
		log.Warn("SIGNING_PRIVATE_KEY not set: using an ephemeral keypair for this process only", "key_id", keyID)
		privateKeyHex = generatedHex
	}
	return crypto.NewSigner(privateKeyHex, keyID)
}

type appConfig struct {
	server        config.Server
	postgres      config.Postgres
	s3            config.S3
	signingKey    string
	keyID         string
	baseURL       string
	minBucketSize int32
	grpcAddr      string
}

func loadConfig() (appConfig, error) {
	var cfg appConfig
	var err error
	if cfg.server, err = config.Load[config.Server](); err != nil {
		return cfg, err
	}
	if cfg.postgres, err = config.Load[config.Postgres](); err != nil {
		return cfg, err
	}
	if cfg.s3, err = config.Load[config.S3](); err != nil {
		return cfg, err
	}
	cfg.signingKey = os.Getenv("SIGNING_PRIVATE_KEY")
	cfg.keyID = envOr("SIGNING_KEY_ID", "dev-key-1")
	cfg.baseURL = envOr("BASE_URL", "http://localhost:8000")
	cfg.minBucketSize = 5
	cfg.grpcAddr = envOr("GRPC_ADDR", ":8085")
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
