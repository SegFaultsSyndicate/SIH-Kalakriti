// pkg/config/config.go

// Package config loads typed configuration from environment variables via struct
// tags. Each concern gets its own struct so a service embeds only what it needs.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Server holds process-wide runtime settings.
type Server struct {
	// Env is the deployment environment, e.g. "local", "staging", "prod".
	Env string `env:"APP_ENV" envDefault:"local"`
	// LogLevel is parsed by pkg/logger.
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
	// ShutdownTimeout bounds how long graceful shutdown waits for in-flight work.
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

// Postgres holds the pgxpool connection settings.
type Postgres struct {
	// DSN is the full connection string; required because there is no safe default.
	DSN string `env:"POSTGRES_DSN,required"`
	// MaxConns caps the pool; 0 lets pkg/postgres pick its own default.
	MaxConns int32 `env:"POSTGRES_MAX_CONNS" envDefault:"20"`
	MinConns int32 `env:"POSTGRES_MIN_CONNS" envDefault:"2"`
}

// Redis holds the go-redis client settings.
type Redis struct {
	Addr     string `env:"REDIS_ADDR,required"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

// Kafka holds the segmentio/kafka-go producer and consumer settings.
type Kafka struct {
	// Brokers is comma-separated in the environment, e.g. "localhost:9092".
	Brokers []string `env:"KAFKA_BROKERS,required" envSeparator:","`
	// ConsumerGroupPrefix namespaces this service's consumer group id.
	ConsumerGroupPrefix string `env:"KAFKA_CONSUMER_GROUP_PREFIX" envDefault:"kalakriti"`
}

// S3 holds the MinIO / S3-compatible object storage settings.
type S3 struct {
	Endpoint  string `env:"S3_ENDPOINT,required"`
	AccessKey string `env:"S3_ACCESS_KEY,required"`
	SecretKey string `env:"S3_SECRET_KEY,required"`
	Bucket    string `env:"S3_BUCKET,required"`
	Region    string `env:"S3_REGION" envDefault:"ap-south-1"`
	UseSSL    bool   `env:"S3_USE_SSL" envDefault:"false"`
	// PublicURL is handed to clients for object downloads; may differ from Endpoint
	// behind a reverse proxy or CDN.
	PublicURL string `env:"S3_PUBLIC_URL"`
}

// Media holds the upload entry point's limits and lifetimes. The caps are here
// rather than in code because a cluster on a 2G link needs a longer upload
// window than a demo on a laptop, and neither should need a rebuild.
type Media struct {
	// MaxImageBytes defaults to 15 MB, MaxVideoBytes to 100 MB.
	MaxImageBytes int64 `env:"MEDIA_MAX_IMAGE_BYTES" envDefault:"15728640"`
	MaxVideoBytes int64 `env:"MEDIA_MAX_VIDEO_BYTES" envDefault:"104857600"`
	// UploadURLTTL is how long a presigned PUT stays valid.
	UploadURLTTL time.Duration `env:"MEDIA_UPLOAD_URL_TTL" envDefault:"15m"`
	// DownloadURLTTL is how long a presigned GET stays valid; the URL cache is
	// held just under this.
	DownloadURLTTL time.Duration `env:"MEDIA_DOWNLOAD_URL_TTL" envDefault:"1h"`
	// PendingTTL is how long a row may wait for its bytes before the reaper
	// deletes it along with any orphaned object.
	PendingTTL time.Duration `env:"MEDIA_PENDING_TTL" envDefault:"24h"`
	// ReaperInterval and ReaperBatchSize bound the sweep.
	ReaperInterval  time.Duration `env:"MEDIA_REAPER_INTERVAL" envDefault:"15m"`
	ReaperBatchSize int32         `env:"MEDIA_REAPER_BATCH_SIZE" envDefault:"200"`
}

// Pipeline holds the cataloguing pipeline's settings.
type Pipeline struct {
	// MLSvcAddr is ml-svc's gRPC address.
	MLSvcAddr string `env:"ML_SVC_ADDR" envDefault:"localhost:50055"`
	// BuyerLanguages are translated into once an artisan approves their listing.
	// Default is all 20 supported languages (the 22 scheduled languages minus
	// Manipuri and Santali, see migrations/030_drop_manipuri_santali.sql).
	BuyerLanguages []string `env:"PIPELINE_BUYER_LANGUAGES" envSeparator:"," envDefault:"ASSAMESE,BENGALI,BODO,DOGRI,GUJARATI,HINDI,KANNADA,KASHMIRI,KONKANI,MAITHILI,MALAYALAM,MARATHI,NEPALI,ODIA,PUNJABI,SANSKRIT,SINDHI,TAMIL,TELUGU,URDU,ENGLISH"`
	// StepTimeout bounds one model call; the whole chain gets four of these.
	StepTimeout time.Duration `env:"PIPELINE_STEP_TIMEOUT" envDefault:"90s"`
}

// Auth holds JWT issuance and verification settings.
type Auth struct {
	JWTSecret  string        `env:"JWT_SECRET,required"`
	AccessTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	RefreshTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"720h"`
	Issuer     string        `env:"JWT_ISSUER" envDefault:"kalakriti"`
}

// Load parses environment variables into a new T using its env struct tags. A
// missing required variable fails fast with a message naming it, courtesy of
// caarlos0/env's own error type.
func Load[T any]() (T, error) {
	cfg, err := env.ParseAs[T]()
	if err != nil {
		var zero T
		return zero, fmt.Errorf("loading %T from environment: %w", zero, err)
	}
	return cfg, nil
}
