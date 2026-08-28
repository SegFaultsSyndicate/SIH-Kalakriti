// pkg/redis/lock.go
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ErrLockNotHeld is returned by Release when the lock has already expired or was
// never acquired by this token.
var ErrLockNotHeld = errors.New("redis: lock not held")

// releaseScript deletes the key only if it still holds our token, so a lock
// whose TTL has expired and been reacquired by someone else is never deleted out
// from under them.
const releaseScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
else
	return 0
end`

// Lock is a held distributed lock. It is not safe for concurrent use by more
// than one goroutine, matching the single critical section it guards.
type Lock struct {
	client redis.Cmdable
	key    string
	token  string
}

// TryLock attempts to acquire a lock named key for ttl using SET NX PX, and
// returns (nil, false, nil) if someone else already holds it — the caller
// decides whether to wait, skip, or retry, this function never blocks.
func TryLock(ctx context.Context, client redis.Cmdable, key string, ttl time.Duration) (*Lock, bool, error) {
	token := uuid.NewString()
	ok, err := client.SetNX(ctx, lockKey(key), token, ttl).Result()
	if err != nil {
		return nil, false, fmt.Errorf("acquiring lock %s: %w", key, err)
	}
	if !ok {
		return nil, false, nil
	}
	return &Lock{client: client, key: key, token: token}, true, nil
}

// Release gives up the lock, but only if it is still held by this token; a lock
// whose TTL already expired returns ErrLockNotHeld rather than silently no-oping,
// so a caller relying on mutual exclusion learns its critical section may have
// run unprotected.
func (l *Lock) Release(ctx context.Context) error {
	n, err := l.client.Eval(ctx, releaseScript, []string{lockKey(l.key)}, l.token).Int64()
	if err != nil {
		return fmt.Errorf("releasing lock %s: %w", l.key, err)
	}
	if n == 0 {
		return fmt.Errorf("releasing lock %s: %w", l.key, ErrLockNotHeld)
	}
	return nil
}

func lockKey(key string) string { return "lock:" + key }
