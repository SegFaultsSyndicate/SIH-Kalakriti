// services/collab-svc/cmd/collab-svc/main.go

// Command collab-svc serves Kalakriti's collective-fulfilment saga over gRPC.
// Alongside the server it runs the transactional outbox relay, the capacity
// reservation reaper, six Kafka consumers that fan order.lot.*,
// order.fulfilment.completed and order.bulk.cancelled events out to
// WatchOrder's live subscribers, and a seventh that drives the
// reoffer-after-decline saga step from the same order.lot.declined topic.
package main

import (
	"context"
	"encoding/json"
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

	segmentio "github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/segfaultsyndicate/kalakriti/pkg/config"
	pkgkafka "github.com/segfaultsyndicate/kalakriti/pkg/kafka"
	"github.com/segfaultsyndicate/kalakriti/pkg/logger"
	"github.com/segfaultsyndicate/kalakriti/pkg/outbox"
	fulfilmentv1 "github.com/segfaultsyndicate/kalakriti/pkg/pb/fulfilment/v1"
	pkgpostgres "github.com/segfaultsyndicate/kalakriti/pkg/postgres"
	"github.com/segfaultsyndicate/kalakriti/pkg/topics"

	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/handler"
	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/repo"
	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/service"
	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/wiring"
)

// drainTimeout bounds how long shutdown waits for in-flight work before
// forcing the server closed.
const drainTimeout = 15 * time.Second

// serviceName identifies this binary in logs, the gRPC health service and its
// Kafka consumer group ids.
const serviceName = "collab-svc"

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("collab-svc exited", "error", err)
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
	log.Info("starting", "grpc_addr", cfg.grpcAddr, "http_addr", cfg.httpAddr)

	// --- dependencies --------------------------------------------------------

	pool, err := pkgpostgres.New(ctx, pkgpostgres.Config{
		DSN:      cfg.postgres.DSN,
		MaxConns: cfg.postgres.MaxConns,
		MinConns: cfg.postgres.MinConns,
	})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	producer := pkgkafka.NewProducer(cfg.kafka.Brokers)
	defer func() {
		if err := producer.Close(); err != nil {
			log.Warn("closing kafka producer", "error", err)
		}
	}()

	// --- wiring ------------------------------------------------------------------

	repository := repo.New(pool)
	fulfilmentSvc := service.NewFulfilment(wiring.NewStore(repository), cfg.fulfilment.ReservationTTL, log)

	broker := handler.NewBroker()
	history := handler.NewHistoryStore(repository)
	fulfilmentHandler := handler.NewFulfilment(fulfilmentSvc, broker, history)

	// --- gRPC server ---------------------------------------------------------------

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(recoveryInterceptor(log)))
	fulfilmentv1.RegisterFulfilmentServiceServer(grpcServer, fulfilmentHandler)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)

	if cfg.server.Env != "production" {
		reflection.Register(grpcServer)
	}

	// --- HTTP health server ----------------------------------------------------------

	ready := &readyState{}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !ready.get() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if err := pool.Ping(context.Background()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	httpServer := &http.Server{
		Addr:              cfg.httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// --- run -----------------------------------------------------------------------

	bgCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()

	relay := outbox.NewRelay(
		repo.NewOutboxStore(repository),
		producer,
		outbox.RelayConfig{},
		func(err error) { log.Error("outbox relay", "error", err) },
	)

	// One consumer group per order.lot.* / order.fulfilment.completed topic,
	// the same one-group-id-per-topic shape core-svc uses — each feeds the
	// broker's fan-out so WatchOrder callers get a live stream without ever
	// polling Postgres.
	watchGroup := cfg.kafka.ConsumerGroupPrefix + "." + serviceName + ".watch-order"
	lotOfferedConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: cfg.kafka.Brokers, Topic: topics.OrderLotOffered, GroupID: watchGroup + ".offered",
	}, log)
	lotAcceptedConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: cfg.kafka.Brokers, Topic: topics.OrderLotAccepted, GroupID: watchGroup + ".accepted",
	}, log)
	lotDeclinedConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: cfg.kafka.Brokers, Topic: topics.OrderLotDeclined, GroupID: watchGroup + ".declined",
	}, log)
	lotProgressedConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: cfg.kafka.Brokers, Topic: topics.OrderLotProgressed, GroupID: watchGroup + ".progressed",
	}, log)
	orderConfirmedConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: cfg.kafka.Brokers, Topic: topics.OrderFulfilmentCompleted, GroupID: watchGroup + ".confirmed",
	}, log)
	orderCancelledConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: cfg.kafka.Brokers, Topic: topics.OrderBulkCancelled, GroupID: watchGroup + ".cancelled",
	}, log)

	// Separate consumer group (not the watch-order one above) on the same
	// topic: this one drives the actual reoffer-after-decline saga step, the
	// watch-order one only fans the event out to WatchOrder subscribers. Two
	// independent groups on one topic is standard Kafka fan-out — each sees
	// every message once, for its own purpose.
	reofferGroup := cfg.kafka.ConsumerGroupPrefix + "." + serviceName + ".reoffer-on-decline"
	reofferConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: cfg.kafka.Brokers, Topic: topics.OrderLotDeclined, GroupID: reofferGroup,
	}, log)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	backgroundLoops := []struct {
		name string
		run  func(ctx context.Context)
	}{
		{"outbox relay", relay.Run},
		{"capacity reservation reaper", func(ctx context.Context) {
			fulfilmentSvc.RunReservationReaper(ctx, cfg.fulfilment.ReaperInterval, cfg.fulfilment.ReaperBatchSize)
		}},
		{"lot offered watch consumer", consumerLoop(lotOfferedConsumer, broker.HandleLotOffered, log, topics.OrderLotOffered)},
		{"lot accepted watch consumer", consumerLoop(lotAcceptedConsumer, broker.HandleLotAccepted, log, topics.OrderLotAccepted)},
		{"lot declined watch consumer", consumerLoop(lotDeclinedConsumer, broker.HandleLotDeclined, log, topics.OrderLotDeclined)},
		{"lot progressed watch consumer", consumerLoop(lotProgressedConsumer, broker.HandleLotProgressed, log, topics.OrderLotProgressed)},
		{"order confirmed watch consumer", consumerLoop(orderConfirmedConsumer, broker.HandleOrderConfirmed, log, topics.OrderFulfilmentCompleted)},
		{"order cancelled watch consumer", consumerLoop(orderCancelledConsumer, broker.HandleOrderCancelled, log, topics.OrderBulkCancelled)},
		{"reoffer on decline consumer", consumerLoop(reofferConsumer, envelopeUnwrap(fulfilmentSvc.HandleLotDeclined), log, topics.OrderLotDeclined)},
	}
	for _, loop := range backgroundLoops {
		wg.Add(1)
		go func(name string, run func(ctx context.Context)) {
			defer wg.Done()
			log.Info(name + " started")
			run(bgCtx)
			log.Info(name + " stopped")
		}(loop.name, loop.run)
	}

	lis, err := net.Listen("tcp", cfg.grpcAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", cfg.grpcAddr, err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("grpc server listening", "addr", cfg.grpcAddr)
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("http health server listening", "addr", cfg.httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	ready.set(true)

	// --- shutdown --------------------------------------------------------------------

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received, draining", "timeout", drainTimeout)
	case err := <-errCh:
		log.Error("server failed, shutting down", "error", err)
	}

	ready.set(false)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	healthSrv.SetServingStatus(serviceName, healthpb.HealthCheckResponse_NOT_SERVING)

	drainCtx, cancelDrain := context.WithTimeout(context.Background(), drainTimeout)
	defer cancelDrain()

	if err := httpServer.Shutdown(drainCtx); err != nil {
		log.Warn("http server did not shut down cleanly", "error", err)
	}

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
		log.Info("grpc server stopped gracefully")
	case <-drainCtx.Done():
		log.Warn("drain deadline exceeded, forcing grpc server to stop")
		grpcServer.Stop()
	}

	stopBackground()
	wg.Wait()

	log.Info("shutdown complete")
	return nil
}

// readyState is a tiny concurrency-safe bool for the /readyz handler; a
// sync/atomic.Bool would do the same job with less code, kept as a struct
// only so get/set read as intent rather than raw atomic calls at each site.
type readyState struct {
	mu    sync.RWMutex
	ready bool
}

func (r *readyState) set(v bool) { r.mu.Lock(); r.ready = v; r.mu.Unlock() }
func (r *readyState) get() bool  { r.mu.RLock(); defer r.mu.RUnlock(); return r.ready }

// consumerLoop wraps a ConsumerGroup.Run call as the uniform (ctx) func the
// background-loop table above expects, logging a stop reason without
// treating a clean shutdown as an error.
func consumerLoop(cg *pkgkafka.ConsumerGroup, handle pkgkafka.HandlerFunc, log *slog.Logger, topic string) func(ctx context.Context) {
	return func(ctx context.Context) {
		if err := cg.Run(ctx, handle); err != nil {
			log.Error("consumer stopped", "topic", topic, "error", err)
		}
	}
}

// envelopeUnwrap adapts a (ctx, payload []byte) handler — the shape
// service.Fulfilment.HandleLotDeclined and any future same-package consumer
// entry points use, so the service package need not import kafka-go or know
// about the outbox envelope wrapper — to a pkgkafka.HandlerFunc by decoding
// the envelope's {header, payload} shape and passing through just the inner
// payload bytes.
func envelopeUnwrap(handle func(ctx context.Context, payload []byte) error) pkgkafka.HandlerFunc {
	type envelope struct {
		Payload json.RawMessage `json:"payload"`
	}
	return func(ctx context.Context, msg segmentio.Message) error {
		var env envelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			return fmt.Errorf("decoding envelope at offset %d: %w", msg.Offset, err)
		}
		return handle(ctx, env.Payload)
	}
}

// recoveryInterceptor converts a panic inside a gRPC handler into an
// Internal error instead of crashing the process, and logs it with the
// method name so it can be tracked down.
func recoveryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if p := recover(); p != nil {
				log.ErrorContext(ctx, "panic in grpc handler",
					"method", info.FullMethod, "panic", fmt.Sprintf("%v", p))
				err = fmt.Errorf("internal error")
			}
		}()
		return next(ctx, req)
	}
}

// appConfig is the assembled configuration for this binary.
type appConfig struct {
	server     config.Server
	postgres   config.Postgres
	kafka      config.Kafka
	fulfilment fulfilmentConfig
	grpcAddr   string
	httpAddr   string
}

// fulfilmentConfig controls the saga's reservation TTL and reaper cadence.
type fulfilmentConfig struct {
	// ReservationTTL is how long a capacity hold survives before the reaper
	// reclaims it. Default 48h per the batch spec.
	ReservationTTL time.Duration `env:"FULFILMENT_RESERVATION_TTL" envDefault:"48h"`
	// ReaperInterval is how often the reaper sweeps for expired reservations.
	ReaperInterval time.Duration `env:"FULFILMENT_REAPER_INTERVAL" envDefault:"5m"`
	// ReaperBatchSize caps how many reservations one sweep claims.
	ReaperBatchSize int32 `env:"FULFILMENT_REAPER_BATCH_SIZE" envDefault:"200"`
}

const (
	defaultGRPCAddr = ":50053"
	defaultHTTPAddr = ":8083"
)

func loadConfig() (appConfig, error) {
	var cfg appConfig
	var err error

	if cfg.server, err = config.Load[config.Server](); err != nil {
		return cfg, err
	}
	if cfg.postgres, err = config.Load[config.Postgres](); err != nil {
		return cfg, err
	}
	if cfg.kafka, err = config.Load[config.Kafka](); err != nil {
		return cfg, err
	}
	if cfg.fulfilment, err = config.Load[fulfilmentConfig](); err != nil {
		return cfg, err
	}

	cfg.grpcAddr = envOrDefault("COLLAB_GRPC_ADDR", defaultGRPCAddr)
	cfg.httpAddr = envOrDefault("COLLAB_HTTP_ADDR", defaultHTTPAddr)
	return cfg, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
