// services/search-svc/internal/search/service/cache.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache is the rendered-page cache plus the version counter that retires it.
//
// Invalidation is a counter, not a key scan: the version is part of every cache
// key, so one INCR on publish makes every existing page unreachable and lets it
// expire on its own. Redis has no safe wildcard delete under load, and this
// needs none.
type RedisCache struct {
	client redis.Cmdable
	prefix string
}

// NewRedisCache builds the search page cache.
func NewRedisCache(client redis.Cmdable, prefix string) *RedisCache {
	if prefix == "" {
		prefix = "search"
	}
	return &RedisCache{client: client, prefix: prefix}
}

func (c *RedisCache) pageKey(key string) string { return c.prefix + ":page:" + key }

func (c *RedisCache) versionKey() string { return c.prefix + ":version" }

// Get returns a cached page, or nil on a miss.
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	raw, err := c.client.Get(ctx, c.pageKey(key)).Bytes()
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// Set stores a rendered page.
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := c.client.Set(ctx, c.pageKey(key), value, ttl).Err(); err != nil {
		return fmt.Errorf("caching search page %s: %w", key, err)
	}
	return nil
}

// Version reads the current index generation. A missing counter is generation
// zero, which is correct for a cold Redis.
func (c *RedisCache) Version(ctx context.Context) (int64, error) {
	version, err := c.client.Get(ctx, c.versionKey()).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("reading the search index version: %w", err)
	}
	return version, nil
}

// BumpVersion retires every cached page.
func (c *RedisCache) BumpVersion(ctx context.Context) error {
	if err := c.client.Incr(ctx, c.versionKey()).Err(); err != nil {
		return fmt.Errorf("bumping the search index version: %w", err)
	}
	return nil
}
