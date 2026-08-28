// pkg/redis/redis.go

// Package redis wraps go-redis with the two things every service actually
// reaches for: a typed cache and a distributed lock. Nothing here is
// Kalakriti-specific; it is generic infrastructure over one Redis instance.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config controls client construction.
type Config struct {
	Addr     string
	Password string
	DB       int
}

// New builds a go-redis client and pings it so a bad address fails at startup.
func New(ctx context.Context, cfg Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("pinging redis: %w", err)
	}
	return client, nil
}
