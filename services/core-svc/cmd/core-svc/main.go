// services/core-svc/cmd/core-svc/main.go

// Command core-svc serves Kalakriti's identity, ontology, catalog, media and
// provenance domain over gRPC. Alongside the servers it runs three background
// loops: the transactional outbox relay that publishes its events, the
// media.enhanced consumer, and the reaper that clears abandoned uploads.
package main

import (
	"context"
	"errors"
	_ "expvar" // registers /debug/vars, where the pipeline publishes its step timings
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
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/config"
	"github.com/ZoroNewbie00/kalakriti/pkg/crypto"
	pkgkafka "github.com/ZoroNewbie00/kalakriti/pkg/kafka"
	"github.com/ZoroNewbie00/kalakriti/pkg/logger"
	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"
	pricingv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/pricing/v1"
	pkgpostgres "github.com/ZoroNewbie00/kalakriti/pkg/postgres"
	pkgredis "github.com/ZoroNewbie00/kalakriti/pkg/redis"
	"github.com/ZoroNewbie00/kalakriti/pkg/storage"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/handler"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/inference"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/ontology"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/wiring"
)

// drainTimeout bounds how long shutdown waits for in-flight work before forcing
// the servers closed.
const drainTimeout = 15 * time.Second

// serviceName identifies this binary in logs, in the gRPC health service and in
// its Kafka consumer group id.
const serviceName = "core-svc"

func main() {
	if err := run(); err != nil {
		// The logger may not exist yet if config loading was what failed, so
		// this last-resort path writes to stderr directly.
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("core-svc exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Signals are trapped before any dependency is dialled, so a Ctrl-C during
	// slow startup still shuts things down cleanly.
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

	rdb, err := pkgredis.New(ctx, pkgredis.Config{
		Addr:     cfg.redis.Addr,
		Password: cfg.redis.Password,
		DB:       cfg.redis.DB,
	})
	if err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Warn("closing redis", "error", err)
		}
	}()

	objects, err := storage.New(ctx, storage.Config{
		Endpoint:  cfg.s3.Endpoint,
		AccessKey: cfg.s3.AccessKey,
		SecretKey: cfg.s3.SecretKey,
		Bucket:    cfg.s3.Bucket,
		Region:    cfg.s3.Region,
		UseSSL:    cfg.s3.UseSSL,
		PublicURL: cfg.s3.PublicURL,
	})
	if err != nil {
		return fmt.Errorf("connecting to object storage: %w", err)
	}

	// ml-svc is dialled lazily: grpc.NewClient does not connect until the first
	// RPC, so a cold ml-svc delays cataloguing rather than stopping core-svc
	// from serving the artisan app.
	mlConn, err := grpc.NewClient(cfg.pipeline.MLSvcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dialling ml-svc at %s: %w", cfg.pipeline.MLSvcAddr, err)
	}
	defer func() {
		if err := mlConn.Close(); err != nil {
			log.Warn("closing the ml-svc connection", "error", err)
		}
	}()

	producer := pkgkafka.NewProducer(cfg.kafka.Brokers)
	defer func() {
		if err := producer.Close(); err != nil {
			log.Warn("closing kafka producer", "error", err)
		}
	}()

	issuer, err := auth.NewIssuer(auth.Config{
		Secret:     cfg.auth.JWTSecret,
		Issuer:     cfg.auth.Issuer,
		AccessTTL:  cfg.auth.AccessTTL,
		RefreshTTL: cfg.auth.RefreshTTL,
	})
	if err != nil {
		return fmt.Errorf("configuring token issuer: %w", err)
	}

	// --- wiring --------------------------------------------------------------

	repository := repo.New(pool)

	if cfg.devOTP {
		log.Warn("development OTP is enabled: the well-known code is accepted for any login",
			"code", service.DevOTPCode)
	}
	otpSvc := service.NewOTPService(rdb,
		service.NewLoggingOTPSender(log, cfg.devOTP),
		service.OTPConfig{DevMode: cfg.devOTP},
		log)

	identitySvc := service.NewIdentity(wiring.NewStore(repository), issuer, otpSvc, log)

	// The ontology is loaded once at startup — from the Redis snapshot when a
	// sibling replica has already built it — and thereafter only changes through
	// RefreshCraftIndex, which announces itself on the invalidation channel.
	registry := ontology.NewRegistry(repository, rdb, rdb, ontology.NoTransliteration(),
		ontology.RegistryConfig{}, log)
	if err := registry.Start(ctx); err != nil {
		return err
	}
	stats := registry.Stats()
	log.Info("ontology loaded", "version", stats.Version, "crafts", stats.Crafts, "aliases", stats.Aliases)

	catalogSvc := service.NewCatalog(wiring.NewCatalogStore(repository), registry, log)
	ontologySvc := service.NewOntology(registry, log)
	mediaSvc := service.NewMedia(
		wiring.NewMediaStore(repository),
		wiring.NewObjectStore(objects),
		pkgredis.NewCache[string](rdb, "media:url"),
		cfg.s3.Bucket,
		domain.MediaLimits{
			MaxImageBytes:  cfg.media.MaxImageBytes,
			MaxVideoBytes:  cfg.media.MaxVideoBytes,
			UploadURLTTL:   cfg.media.UploadURLTTL,
			DownloadURLTTL: cfg.media.DownloadURLTTL,
		},
		log)

	pipelineSvc := service.NewPipeline(
		wiring.NewPipelineStore(repository),
		catalogSvc,
		mediaSvc,
		registry,
		inference.New(mlConn, cfg.s3.Bucket),
		cfg.pipeline.BuyerLanguages,
		log)

	pricingSvc := service.NewPricing(wiring.NewPricingStore(repository), log)

	signer, err := loadProvenanceSigner(cfg.provenanceSigningKey, cfg.provenanceKeyID, log)
	if err != nil {
		return fmt.Errorf("configuring provenance signer: %w", err)
	}
	provenanceSvc := service.NewProvenance(
		wiring.NewProvenanceStore(repository),
		wiring.NewCatalogStore(repository),
		wiring.NewMediaHasher(repository, objects),
		wiring.NewTechniqueVerifier(repository),
		signer,
	)

	identityHandler := handler.NewIdentity(identitySvc)
	catalogHandler := handler.NewCatalog(catalogSvc, provenanceSvc, cfg.baseURL)
	curationHandler := handler.NewCuration(catalogSvc)
	ontologyHandler := handler.NewOntology(ontologySvc)
	mediaHandler := handler.NewMedia(mediaSvc)
	pricingHandler := handler.NewPricing(pricingSvc)
	healthHandler := handler.NewHealth(pool, rdb)

	// --- gRPC server ---------------------------------------------------------

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(log),
			auth.UnaryServerInterceptor(issuer, handler.PublicMethods()),
		),
		grpc.ChainStreamInterceptor(
			auth.StreamServerInterceptor(issuer, handler.PublicMethods()),
		),
	)
	identityv1.RegisterIdentityServiceServer(grpcServer, identityHandler)
	catalogv1.RegisterCatalogServiceServer(grpcServer, catalogHandler)
	catalogv1.RegisterCurationServiceServer(grpcServer, curationHandler)
	catalogv1.RegisterOntologyServiceServer(grpcServer, ontologyHandler)
	catalogv1.RegisterMediaServiceServer(grpcServer, mediaHandler)
	pricingv1.RegisterPricingServiceServer(grpcServer, pricingHandler)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	// The empty service name is what a generic gRPC health probe checks.
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)

	if cfg.server.Env != "production" {
		// Reflection makes grpcurl work against a local stack; it is left off in
		// production so the service does not advertise its whole schema.
		reflection.Register(grpcServer)
	}

	// --- HTTP health server --------------------------------------------------

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler.LiveHandler())
	mux.HandleFunc("/readyz", healthHandler.ReadyHandler())
	// expvar's package init registers /debug/vars on http.DefaultServeMux; this
	// hands it to the private health server rather than exposing it publicly.
	mux.Handle("/debug/vars", http.DefaultServeMux)
	httpServer := &http.Server{
		Addr:              cfg.httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// --- run -----------------------------------------------------------------

	// bgCtx is cancelled at the start of shutdown so the background loops stop
	// claiming new work while in-flight RPCs finish.
	bgCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()

	relay := outbox.NewRelay(
		repo.NewOutboxStore(repository),
		producer,
		outbox.RelayConfig{},
		func(err error) { log.Error("outbox relay", "error", err) },
	)

	enhancedConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: cfg.kafka.Brokers,
		Topic:   topics.MediaEnhanced,
		GroupID: cfg.kafka.ConsumerGroupPrefix + "." + serviceName + ".media-enhanced",
	}, log)

	// The cataloguing pipeline. A message that exhausts its retries is recorded
	// against the media row on its way to the dead-letter topic, so the artisan
	// sees a reason rather than a photograph that quietly became nothing.
	pipelineConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers:      cfg.kafka.Brokers,
		Topic:        topics.MediaUploaded,
		GroupID:      cfg.kafka.ConsumerGroupPrefix + "." + serviceName + ".pipeline",
		OnDeadLetter: handler.MediaUploadedDeadLetter(pipelineSvc, log),
	}, log)

	translationConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
		Brokers: cfg.kafka.Brokers,
		Topic:   topics.CatalogListingPublished,
		GroupID: cfg.kafka.ConsumerGroupPrefix + "." + serviceName + ".translation",
	}, log)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("outbox relay started")
		relay.Run(bgCtx)
		log.Info("outbox relay stopped")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("media.enhanced consumer started", "topic", topics.MediaEnhanced)
		if err := enhancedConsumer.Run(bgCtx, handler.MediaEnhancedHandler(mediaSvc, log)); err != nil {
			log.Error("media.enhanced consumer stopped", "error", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("cataloguing pipeline started", "topic", topics.MediaUploaded)
		if err := pipelineConsumer.Run(bgCtx, handler.MediaUploadedHandler(pipelineSvc, log)); err != nil {
			log.Error("cataloguing pipeline stopped", "error", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("translation fan-out started", "topic", topics.CatalogListingPublished)
		if err := translationConsumer.Run(bgCtx, handler.ListingPublishedHandler(pipelineSvc, log)); err != nil {
			log.Error("translation fan-out stopped", "error", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("media reaper started",
			"interval", cfg.media.ReaperInterval, "pending_ttl", cfg.media.PendingTTL)
		mediaSvc.RunReaper(bgCtx, cfg.media.ReaperInterval, cfg.media.PendingTTL, cfg.media.ReaperBatchSize)
		log.Info("media reaper stopped")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		registry.Watch(bgCtx)
	}()

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

	healthHandler.SetReady(true)

	// --- shutdown ------------------------------------------------------------

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received, draining", "timeout", drainTimeout)
	case err := <-errCh:
		log.Error("server failed, shutting down", "error", err)
		defer func() { _ = err }()
	}

	// Fail readiness first so load balancers stop routing new work here while
	// the in-flight requests below are still being served.
	healthHandler.SetReady(false)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	healthSrv.SetServingStatus(serviceName, healthpb.HealthCheckResponse_NOT_SERVING)

	drainCtx, cancelDrain := context.WithTimeout(context.Background(), drainTimeout)
	defer cancelDrain()

	if err := httpServer.Shutdown(drainCtx); err != nil {
		log.Warn("http server did not shut down cleanly", "error", err)
	}

	// GracefulStop waits for in-flight RPCs, so it is raced against the drain
	// deadline rather than trusted to return promptly.
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

	// The background loops are stopped last: an event enqueued by a request that
	// only just finished still gets a chance to be published before we exit.
	stopBackground()
	wg.Wait()

	log.Info("shutdown complete")
	return nil
}

// appConfig is the assembled configuration for this binary. Each nested struct
// is loaded independently by pkg/config so a missing variable names itself.
type appConfig struct {
	server   config.Server
	postgres config.Postgres
	redis    config.Redis
	kafka    config.Kafka
	auth     config.Auth
	s3       config.S3
	media    config.Media
	pipeline config.Pipeline
	grpcAddr string
	httpAddr string
	devOTP   bool

	// provenanceSigningKey is the hex-encoded Ed25519 private key SealProvenance
	// signs with; provenanceKeyID names it in the signature (see scripts/keygen.go).
	provenanceSigningKey string
	provenanceKeyID      string
	// baseURL is the public origin embedded in a sealed record's QR/verify URL.
	baseURL string
}

// Defaults for the addresses this service listens on.
const (
	defaultGRPCAddr = ":50051"
	defaultHTTPAddr = ":8081"
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
	if cfg.redis, err = config.Load[config.Redis](); err != nil {
		return cfg, err
	}
	if cfg.kafka, err = config.Load[config.Kafka](); err != nil {
		return cfg, err
	}
	if cfg.auth, err = config.Load[config.Auth](); err != nil {
		return cfg, err
	}
	if cfg.s3, err = config.Load[config.S3](); err != nil {
		return cfg, err
	}
	if cfg.media, err = config.Load[config.Media](); err != nil {
		return cfg, err
	}
	if cfg.pipeline, err = config.Load[config.Pipeline](); err != nil {
		return cfg, err
	}

	cfg.grpcAddr = envOr("CORE_SVC_GRPC_ADDR", defaultGRPCAddr)
	cfg.httpAddr = envOr("CORE_SVC_HTTP_ADDR", defaultHTTPAddr)
	cfg.devOTP = envOr("AUTH_DEV_OTP_ENABLED", "false") == "true"
	cfg.provenanceSigningKey = os.Getenv("PROVENANCE_PRIVATE_KEY")
	cfg.provenanceKeyID = envOr("PROVENANCE_KEY_ID", "dev-key-1")
	cfg.baseURL = envOr("BASE_URL", "http://localhost:8000")

	// A development OTP bypass in production would make every account
	// trivially takeable, so it is refused rather than warned about.
	if cfg.devOTP && cfg.server.Env == "production" {
		return cfg, errors.New("AUTH_DEV_OTP_ENABLED must not be set in production")
	}
	return cfg, nil
}

// loadProvenanceSigner builds the Ed25519 signer SealProvenance uses. In
// production PROVENANCE_PRIVATE_KEY must be set; locally, an unset key
// generates an ephemeral one for the process lifetime (mirrors insight-svc's
// same dev fallback) so the service still starts without one configured.
func loadProvenanceSigner(privateKeyHex, keyID string, log *slog.Logger) (*crypto.Signer, error) {
	if privateKeyHex == "" {
		generatedHex, _, err := crypto.GenerateKeypair()
		if err != nil {
			return nil, fmt.Errorf("generating ephemeral provenance keypair: %w", err)
		}
		log.Warn("PROVENANCE_PRIVATE_KEY not set: using an ephemeral keypair for this process only",
			"key_id", keyID)
		privateKeyHex = generatedHex
	}
	return crypto.NewSigner(privateKeyHex, keyID)
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// recoveryInterceptor turns a panic in a handler into an Internal error rather
// than taking the whole server down with it.
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
