// services/channel-svc/cmd/channel-svc/main.go

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/config"
	"github.com/ZoroNewbie00/kalakriti/pkg/logger"
	pkgkafka "github.com/ZoroNewbie00/kalakriti/pkg/kafka"
	pkgpostgres "github.com/ZoroNewbie00/kalakriti/pkg/postgres"
	pkgredis "github.com/ZoroNewbie00/kalakriti/pkg/redis"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/consumer"
	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/export"
	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/notification"
	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/repo"
)

const serviceName = "channel-svc"
const drainTimeout = 15 * time.Second

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("channel-svc exited", "error", err)
		os.Exit(1)
	}
}

type cfg struct {
	server    config.Server
	postgres  config.Postgres
	redis     config.Redis
	kafka     config.Kafka
	ondc      ONDCConfig
	whatsapp  WhatsAppConfig
	indiaPost IndiaPostConfig
	grpcAddr  string
	httpAddr  string
}

type ONDCConfig struct {
	PrivateKeyHex  string
	KeyID          string
	SubscriberID   string
	SubscriberURL  string
	DryRun         bool
}

type WhatsAppConfig struct {
	DryRun bool
}

type IndiaPostConfig struct {
	APIURL string
	APIKey string
	Stub   bool
}

func loadConfig() (cfg, error) {
	var c cfg
	var err error

	if c.server, err = config.Load[config.Server](); err != nil {
		return c, err
	}
	if c.postgres, err = config.Load[config.Postgres](); err != nil {
		return c, err
	}
	if c.redis, err = config.Load[config.Redis](); err != nil {
		return c, err
	}
	if c.kafka, err = config.Load[config.Kafka](); err != nil {
		return c, err
	}

	c.grpcAddr = os.Getenv("CHANNEL_SVC_GRPC_ADDR")
	if c.grpcAddr == "" {
		c.grpcAddr = ":50053"
	}
	c.httpAddr = os.Getenv("CHANNEL_SVC_HTTP_ADDR")
	if c.httpAddr == "" {
		c.httpAddr = ":8083"
	}

	c.ondc = ONDCConfig{
		PrivateKeyHex: os.Getenv("ONDC_PRIVATE_KEY"),
		KeyID:         os.Getenv("ONDC_KEY_ID"),
		SubscriberID:  os.Getenv("ONDC_SUBSCRIBER_ID"),
		SubscriberURL: os.Getenv("ONDC_SUBSCRIBER_URL"),
		DryRun:        true,
	}
	c.whatsapp = WhatsAppConfig{DryRun: true}
	c.indiaPost = IndiaPostConfig{
		APIURL: os.Getenv("INDIAPOST_API_URL"),
		APIKey: os.Getenv("INDIAPOST_API_KEY"),
		Stub:   true,
	}

	return c, nil
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

	// Database
	db, err := pkgpostgres.New(ctx, pkgpostgres.Config{
		DSN:      cfg.postgres.DSN,
		MaxConns: cfg.postgres.MaxConns,
		MinConns: cfg.postgres.MinConns,
	})
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer db.Close()

	// Redis
	redisClient, err := pkgredis.New(ctx, pkgredis.Config{
		Addr:     cfg.redis.Addr,
		Password: cfg.redis.Password,
		DB:       cfg.redis.DB,
	})
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	defer redisClient.Close()

	// Repos
	r := repo.New(db)

	// Notification service
	notifSvc := notification.NewService(r, log)

	// ONDC adapter, WhatsApp client, India Post client: none is wired to a
	// caller yet. ondc.Client.PublishOnSearch needs a hydrated []ondc.Listing
	// (title, price, media, GI status...) that the bare
	// catalog.listing.published event this service already consumes doesn't
	// carry, so publishing on that trigger means first giving channel-svc a
	// core-svc catalog client to fetch listing detail -- a real cross-service
	// wiring decision, not a one-line fix. WhatsApp/India Post are still
	// log-only stubs (see their package docs) with nothing to dispatch to
	// yet. Left uninstantiated rather than wired to a guessed-at consumer.
	//
	// ondcAdapter, err := ondc.NewAdapter(ondc.Config{
	//     PrivateKeyHex: cfg.ondc.PrivateKeyHex,
	//     KeyID:         cfg.ondc.KeyID,
	//     SubscriberID:  cfg.ondc.SubscriberID,
	//     SubscriberURL: cfg.ondc.SubscriberURL,
	//     DryRun:        cfg.ondc.DryRun,
	// })
	// ondcClient := ondc.NewClient(ondcAdapter, cfg.ondc.SubscriberURL, log)
	// waClient := whatsapp.NewClient(whatsapp.Config{DryRun: cfg.whatsapp.DryRun}, log)
	// ipClient := indiapost.NewClient(indiapost.Config{
	//     APIURL: cfg.indiaPost.APIURL,
	//     APIKey: cfg.indiaPost.APIKey,
	//     Stub:   cfg.indiaPost.Stub,
	// }, log)

	// Follow fanout consumer
	fanout := consumer.NewFollowFanout(notifSvc, r, log)

	// Kafka consumer for catalog.listing.published
	kafkaReader := pkgkafka.NewReader(pkgkafka.ReaderConfig{
		Brokers: cfg.kafka.Brokers,
		Topic:   topics.CatalogListingPublished,
		GroupID: serviceName + "-fanout",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer kafkaReader.Close()

	// Outbox relay - skipped for now as channel-svc doesn't have outbox tables yet
	// relay := outbox.NewRelay(
	//     repo.NewOutboxStore(db),
	//     pkgkafka.NewProducer(cfg.kafka.Brokers),
	//     outbox.RelayConfig{},
	//     func(err error) { log.Error("outbox relay", "error", err) },
	// )

	// HTTP server for health and export endpoints
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Export endpoint - placeholder
	httpMux.HandleFunc("/export/indiahandmade", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = export.WriteJSON(w, []export.IndiaHandmadeRecord{})
	})

	httpServer := &http.Server{
		Addr:    cfg.httpAddr,
		Handler: httpMux,
	}

	// Start background workers
	var wg sync.WaitGroup
	wg.Add(2)

	// 1. Fanout consumer
	go func() {
		defer wg.Done()
		fanout.Run(ctx, kafkaReader)
	}()

	// 2. HTTP server
	go func() {
		defer wg.Done()
		log.Info("http server listening", "addr", cfg.httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", "error", err)
		}
	}()

	// Wait for shutdown
	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()

	_ = httpServer.Shutdown(shutdownCtx)
	wg.Wait()

	log.Info("stopped")
	return nil
}

