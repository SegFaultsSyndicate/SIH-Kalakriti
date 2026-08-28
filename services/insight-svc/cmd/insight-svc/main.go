// services/insight-svc/cmd/insight-svc/main.go
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/<org>/kalakriti/pkg/config"
	"github.com/<org>/kalakriti/pkg/logger"
	"github.com/<org>/kalakriti/pkg/postgres"
	"github.com/<org>/kalakriti/pkg/storage"
	"github.com/<org>/kalakriti/services/insight-svc/internal/insight/repo"
	"github.com/<org>/kalakriti/services/insight-svc/internal/insight/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Config struct {
	GRPCAddr      string `env:"GRPC_ADDR" envDefault:":8085"`
	SigningKey    string `env:"SIGNING_PRIVATE_KEY"`
	config.Postgres
	config.S3
}

func main() {
	ctx := context.Background()

	cfg := config.Load[Config]()
	logger.Init(os.Getenv("LOG_LEVEL"))

	pool := postgres.MustConnect(ctx, cfg.Postgres)
	defer pool.Close()

	s3Client := storage.NewClient(cfg.S3)

	privateKey := loadPrivateKey(cfg.SigningKey)

	repo := repo.New(pool)
	stmtSvc := service.NewStatementService(repo, s3Client, privateKey)

	grpcServer := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())

	// ponytail: No proto for insight-svc yet, manual handler registration would go here
	// For now, bff will call via HTTP REST endpoint

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	slog.Info("insight-svc starting", "addr", cfg.GRPCAddr)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	grpcServer.GracefulStop()

	_ = stmtSvc // Used via HTTP handler in bff for demo
}

func loadPrivateKey(keyHex string) ed25519.PrivateKey {
	if keyHex == "" {
		// Dev fallback: generate ephemeral key
		_, priv, _ := ed25519.GenerateKey(nil)
		return priv
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		log.Fatalf("decode signing key: %v", err)
	}
	return ed25519.PrivateKey(key)
}
