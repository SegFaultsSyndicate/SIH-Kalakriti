// services/search-svc/cmd/search-svc/main.go

// Command search-svc serves buyer-facing discovery over the pgvector index and
// keeps that index up to date from the catalogue's own event stream.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/ZoroNewbie00/kalakriti/pkg/config"
	"github.com/ZoroNewbie00/kalakriti/pkg/grpcdial"
	pkgkafka "github.com/ZoroNewbie00/kalakriti/pkg/kafka"
	"github.com/ZoroNewbie00/kalakriti/pkg/logger"
	searchv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/search/v1"
	pkgpostgres "github.com/ZoroNewbie00/kalakriti/pkg/postgres"
	pkgredis "github.com/ZoroNewbie00/kalakriti/pkg/redis"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/client"
	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/handler"
	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/repo"
	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/service"
)

const (
	serviceName  = "search-svc"
	drainTimeout = 15 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("search-svc exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server, err := config.Load[config.Server]()
	if err != nil {
		return err
	}
	postgres, err := config.Load[config.Postgres]()
	if err != nil {
		return err
	}
	redis, err := config.Load[config.Redis]()
	if err != nil {
		return err
	}
	kafkaCfg, err := config.Load[config.Kafka]()
	if err != nil {
		return err
	}
	s3, err := config.Load[config.S3]()
	if err != nil {
		return err
	}
	pipeline, err := config.Load[config.Pipeline]()
	if err != nil {
		return err
	}

	log := logger.New(server.LogLevel).With("service", serviceName, "env", server.Env)
	grpcAddr := envOr("SEARCH_SVC_GRPC_ADDR", ":50052")
	httpAddr := envOr("SEARCH_SVC_HTTP_ADDR", ":8082")
	coreAddr := envOr("CORE_SVC_ADDR", "localhost:50051")

	pool, err := pkgpostgres.New(ctx, pkgpostgres.Config{
		DSN: postgres.DSN, MaxConns: postgres.MaxConns, MinConns: postgres.MinConns,
	})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	rdb, err := pkgredis.New(ctx, pkgredis.Config{
		Addr: redis.Addr, Password: redis.Password, DB: redis.DB,
	})
	if err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Warn("closing redis", "error", err)
		}
	}()

	mlConn, err := grpcdial.Dial(pipeline.MLSvcAddr)
	if err != nil {
		return fmt.Errorf("dialling ml-svc: %w", err)
	}
	defer func() { _ = mlConn.Close() }()

	coreConn, err := grpcdial.Dial(coreAddr)
	if err != nil {
		return fmt.Errorf("dialling core-svc: %w", err)
	}
	defer func() { _ = coreConn.Close() }()

	repository := repo.New(pool)
	inference := client.NewInference(mlConn, s3.Bucket)
	ontology := client.NewOntology(coreConn)
	cache := service.NewRedisCache(rdb, "search")

	searchSvc := service.NewSearch(repository, inference, ontology,
		client.NewTransliterator(ontology, rdb), service.NewRuleUnderstander(), cache, log)
	indexer := service.NewIndexer(repository, inference, cache, log)

	grpcServer := grpc.NewServer()
	searchv1.RegisterSearchServiceServer(grpcServer, handler.NewSearch(searchSvc))

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)
	if server.Env != "production" {
		reflection.Register(grpcServer)
	}

	bgCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()

	indexConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: kafkaCfg.Brokers,
		Topic:   topics.CatalogListingPublished,
		GroupID: kafkaCfg.ConsumerGroupPrefix + "." + serviceName + ".index",
	}, log)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		// See WIRING_AUDIT_PLAN.md F-6: indexConsumer reconnects forever
		// after a disconnect rather than dying, which used to make it
		// invisible to readiness (it only ever pinged Postgres).
		if !indexConsumer.Healthy() {
			http.Error(w, "index consumer for "+indexConsumer.Topic()+" is reconnecting", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	httpServer := &http.Server{Addr: httpAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", grpcAddr, err)
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		log.Info("indexing consumer started", "topic", topics.CatalogListingPublished)
		if err := indexConsumer.Run(bgCtx, handler.ListingPublishedHandler(indexer, log)); err != nil {
			log.Error("indexing consumer stopped", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		log.Info("grpc server listening", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Error("grpc server", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		log.Info("http health server listening", "addr", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "error", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received, draining", "timeout", drainTimeout)
	healthSrv.SetServingStatus(serviceName, healthpb.HealthCheckResponse_NOT_SERVING)

	drainCtx, cancelDrain := context.WithTimeout(context.Background(), drainTimeout)
	defer cancelDrain()
	if err := httpServer.Shutdown(drainCtx); err != nil {
		log.Warn("http server did not shut down cleanly", "error", err)
	}
	grpcServer.GracefulStop()
	stopBackground()
	wg.Wait()

	log.Info("shutdown complete")
	return nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
