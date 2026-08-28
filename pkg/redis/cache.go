// pkg/redis/cache.go
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss is returned by Cache.Get when the key is not present.
var ErrCacheMiss = errors.New("cache: miss")

// Cache is a typed, JSON-marshalling wrapper over a redis key namespace.
type Cache[T any] struct {
	client redis.Cmdable
	prefix string
}

// NewCache builds a Cache whose keys are prefixed with keyPrefix, so unrelated
// caches can share one Redis instance without colliding.
func NewCache[T any](client redis.Cmdable, keyPrefix string) *Cache[T] {
	return &Cache[T]{client: client, prefix: keyPrefix}
}

func (c *Cache[T]) key(id string) string { return c.prefix + ":" + id }

// Get returns the cached value for id, or ErrCacheMiss if absent.
func (c *Cache[T]) Get(ctx context.Context, id string) (T, error) {
	var value T
	raw, err := c.client.Get(ctx, c.key(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return value, ErrCacheMiss
	}
	if err != nil {
		return value, fmt.Errorf("getting cache key %s: %w", id, err)
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, fmt.Errorf("decoding cache key %s: %w", id, err)
	}
	return value, nil
}

// Set stores value for id with the given TTL. A zero TTL means "no expiry".
func (c *Cache[T]) Set(ctx context.Context, id string, value T, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encoding cache key %s: %w", id, err)
	}
	if err := c.client.Set(ctx, c.key(id), raw, ttl).Err(); err != nil {
		return fmt.Errorf("setting cache key %s: %w", id, err)
	}
	return nil
}

// Delete removes id from the cache. Deleting a missing key is not an error.
func (c *Cache[T]) Delete(ctx context.Context, id string) error {
	if err := c.client.Del(ctx, c.key(id)).Err(); err != nil {
		return fmt.Errorf("deleting cache key %s: %w", id, err)
	}
	return nil
}
