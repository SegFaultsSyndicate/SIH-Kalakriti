package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
)

// RateLimitConfig controls sliding window rate limits.
type RateLimitConfig struct {
	// PerIPLimit is requests per window per IP.
	PerIPLimit int
	// PerPrincipalLimit is requests per window per authenticated principal.
	PerPrincipalLimit int
	// Window is the sliding window duration.
	Window time.Duration
}

// RateLimit enforces Redis-backed sliding window rate limiting per IP and per
// principal (when authenticated). It checks both and rejects if either exceeds.
func RateLimit(rdb *redis.Client, cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			now := time.Now()

			// Per-IP limit.
			ip := extractIP(r)
			if ip != "" {
				key := fmt.Sprintf("rl:ip:%s", ip)
				if !allow(ctx, rdb, key, cfg.PerIPLimit, cfg.Window, now) {
					httpx.Error(w, domain.Unavailable("rate limit exceeded"))
					return
				}
			}

			// Per-principal limit if authenticated.
			if p, ok := auth.PrincipalFrom(ctx); ok {
				key := fmt.Sprintf("rl:principal:%s", p.Subject)
				if !allow(ctx, rdb, key, cfg.PerPrincipalLimit, cfg.Window, now) {
					httpx.Error(w, domain.Unavailable("rate limit exceeded"))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// allow implements the sliding window counter: ZADD the current timestamp,
// ZREMRANGEBYSCORE to expire old entries, ZCARD to count, EXPIRE to clean up.
func allow(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration, now time.Time) bool {
	pipe := rdb.Pipeline()
	nowScore := float64(now.UnixNano())
	cutoff := float64(now.Add(-window).UnixNano())

	pipe.ZAdd(ctx, key, redis.Z{Score: nowScore, Member: nowScore})
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%.0f", cutoff))
	count := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, window*2)

	_, err := pipe.Exec(ctx)
	if err != nil {
		// Redis failure doesn't block the request, it just means no rate limit.
		return true
	}

	return count.Val() <= int64(limit)
}

// extractIP returns the client IP from X-Forwarded-For (first entry), falling
// back to RemoteAddr. Returns empty string if unparseable.
func extractIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		for idx := 0; idx < len(xff); idx++ {
			if xff[idx] == ',' {
				xff = xff[:idx]
				break
			}
		}
		if ip := net.ParseIP(xff); ip != nil {
			return ip.String()
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return host
}

// RateLimitEndpoint applies a dedicated rate limit on specific sensitive routes (e.g. OTP request, order spam).
func RateLimitEndpoint(rdb *redis.Client, prefix string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}
			ip := extractIP(r)
			if ip == "" {
				ip = "unknown"
			}
			key := fmt.Sprintf("rl:%s:%s", prefix, ip)
			if !allow(r.Context(), rdb, key, limit, window, time.Now()) {
				httpx.Error(w, domain.Unavailable(fmt.Sprintf("%s rate limit exceeded, please retry later", prefix)))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CheckAccountLockout checks if an identifier (phone/email) is currently locked out due to failed attempts.
func CheckAccountLockout(ctx context.Context, rdb *redis.Client, identifier string) (bool, time.Duration) {
	if rdb == nil || identifier == "" {
		return false, 0
	}
	key := fmt.Sprintf("lockout:%s", identifier)
	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		return false, 0
	}
	return true, ttl
}

// RecordFailedLogin increments consecutive failed login count and locks account after 5 failed attempts.
func RecordFailedLogin(ctx context.Context, rdb *redis.Client, identifier string) bool {
	if rdb == nil || identifier == "" {
		return false
	}
	failedKey := fmt.Sprintf("failed_login:%s", identifier)
	count, err := rdb.Incr(ctx, failedKey).Result()
	if err != nil {
		return false
	}
	if count == 1 {
		rdb.Expire(ctx, failedKey, 15*time.Minute)
	}
	if count >= 5 {
		lockKey := fmt.Sprintf("lockout:%s", identifier)
		rdb.Set(ctx, lockKey, "locked", 15*time.Minute)
		rdb.Del(ctx, failedKey)
		return true
	}
	return false
}

// ClearFailedLogin clears failed login records on successful authentication.
func ClearFailedLogin(ctx context.Context, rdb *redis.Client, identifier string) {
	if rdb == nil || identifier == "" {
		return
	}
	rdb.Del(ctx, fmt.Sprintf("failed_login:%s", identifier))
	rdb.Del(ctx, fmt.Sprintf("lockout:%s", identifier))
}

