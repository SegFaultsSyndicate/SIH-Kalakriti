// pkg/config/config_test.go
package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadPostgresRequiresDSN(t *testing.T) {
	unsetenv(t, "POSTGRES_DSN")

	_, err := Load[Postgres]()
	if err == nil {
		t.Fatal("Load[Postgres]() with no DSN succeeded, want error")
	}
	if !strings.Contains(err.Error(), "POSTGRES_DSN") {
		t.Errorf("error %q does not name the missing variable POSTGRES_DSN", err.Error())
	}
}

func TestLoadPostgresAppliesDefaults(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://localhost/kalakriti")
	unsetenv(t, "POSTGRES_MAX_CONNS")
	unsetenv(t, "POSTGRES_MIN_CONNS")

	cfg, err := Load[Postgres]()
	if err != nil {
		t.Fatalf("Load[Postgres](): %v", err)
	}
	if cfg.DSN != "postgres://localhost/kalakriti" {
		t.Errorf("DSN = %q", cfg.DSN)
	}
	if cfg.MaxConns != 20 || cfg.MinConns != 2 {
		t.Errorf("MaxConns=%d MinConns=%d, want 20/2 defaults", cfg.MaxConns, cfg.MinConns)
	}
}

func TestLoadKafkaSplitsBrokers(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "a:9092,b:9092,c:9092")

	cfg, err := Load[Kafka]()
	if err != nil {
		t.Fatalf("Load[Kafka](): %v", err)
	}
	want := []string{"a:9092", "b:9092", "c:9092"}
	if len(cfg.Brokers) != len(want) {
		t.Fatalf("Brokers = %v, want %v", cfg.Brokers, want)
	}
	for i, b := range want {
		if cfg.Brokers[i] != b {
			t.Errorf("Brokers[%d] = %q, want %q", i, cfg.Brokers[i], b)
		}
	}
}

func TestLoadAuthDurationDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "dev-secret")

	cfg, err := Load[Auth]()
	if err != nil {
		t.Fatalf("Load[Auth](): %v", err)
	}
	if cfg.AccessTTL != 15*time.Minute {
		t.Errorf("AccessTTL = %v, want 15m", cfg.AccessTTL)
	}
	if cfg.Issuer != "kalakriti" {
		t.Errorf("Issuer = %q, want kalakriti", cfg.Issuer)
	}
}

// unsetenv clears an environment variable for the duration of the test, restoring
// whatever value (or absence) preceded it.
func unsetenv(t *testing.T, key string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unsetenv %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		}
	})
}
