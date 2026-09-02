// tests/integration/main_test.go
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	pgContainer    *postgres.PostgresContainer
	redisContainer *redis.RedisContainer
	kafkaContainer *kafka.KafkaContainer
	minioContainer testcontainers.Container

	pgConnStr   string
	redisAddr   string
	kafkaBroker string
	minioURL    string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error

	// Postgres with pgvector
	pgContainer, err = postgres.Run(ctx,
		"pgvector/pgvector:pg18",
		postgres.WithDatabase("kalakriti_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		panic(err)
	}
	defer pgContainer.Terminate(ctx)

	pgConnStr, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}
	os.Setenv("DATABASE_URL", pgConnStr)

	// Redis
	redisContainer, err = redis.Run(ctx, "redis:7-alpine")
	if err != nil {
		panic(err)
	}
	defer redisContainer.Terminate(ctx)

	redisAddr, err = redisContainer.ConnectionString(ctx)
	if err != nil {
		panic(err)
	}
	os.Setenv("REDIS_ADDR", redisAddr)

	// Kafka
	kafkaContainer, err = kafka.Run(ctx,
		"confluentinc/confluent-local:7.5.0",
		kafka.WithClusterID("test-cluster"),
	)
	if err != nil {
		panic(err)
	}
	defer kafkaContainer.Terminate(ctx)

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		panic(err)
	}
	kafkaBroker = brokers[0]
	os.Setenv("KAFKA_BROKERS", kafkaBroker)

	// MinIO
	minioContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "minio/minio:latest",
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER":     "minioadmin",
				"MINIO_ROOT_PASSWORD": "minioadmin",
			},
			Cmd:                  []string{"server", "/data"},
			WaitingFor:           wait.ForHTTP("/minio/health/live").WithPort("9000"),
		},
		Started: true,
	})
	if err != nil {
		panic(err)
	}
	defer minioContainer.Terminate(ctx)

	host, err := minioContainer.Host(ctx)
	if err != nil {
		panic(err)
	}
	port, err := minioContainer.MappedPort(ctx, "9000")
	if err != nil {
		panic(err)
	}
	minioURL = host + ":" + port.Port()
	os.Setenv("MINIO_ENDPOINT", minioURL)
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")

	code := m.Run()
	os.Exit(code)
}
