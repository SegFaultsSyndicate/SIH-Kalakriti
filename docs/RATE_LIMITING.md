# Rate Limiting Configuration

Kalakriti BFF uses Redis-backed sliding window rate limiting to protect against abuse.

## How It Works

**Sliding Window Counter Algorithm:**
1. Each request adds a timestamp to a Redis sorted set
2. Old entries outside the window are removed
3. If count ≤ limit, request proceeds; otherwise 429 returned
4. Tracks both per-IP and per-authenticated-user separately

**Implementation:** `services/bff/internal/bff/middleware/ratelimit.go`

## Configuration

Set in `.env`:

```bash
# Global default (applies to all endpoints unless overridden)
RATE_LIMIT_RPS=100              # Requests per window per IP
RATE_LIMIT_BURST=200            # Burst allowance (not yet implemented)
RATE_LIMIT_WINDOW=1m            # Sliding window duration

# Per-endpoint overrides (future)
RATE_LIMIT_ENDPOINTS=/api/v1/auth/otp:5:10,/api/v1/search:50:100
```

**Current defaults in code:**
- **Per-IP limit:** 100 requests/minute
- **Per-authenticated-user limit:** 200 requests/minute  
- **Window:** 60 seconds (sliding)

## Limits by Endpoint Type

| Endpoint Type | Recommended Limit | Reason |
|---------------|-------------------|--------|
| `/auth/otp` | 5/min per IP | Prevent SMS abuse |
| `/auth/verify` | 10/min per IP | Prevent brute force |
| `/media/upload-url` | 20/min per user | Limit storage creation |
| `/search` | 50/min per IP | Expensive vector queries |
| `/listings` (read) | 100/min per IP | Cheap reads |
| `/orders` (write) | 10/min per user | Prevent duplicate orders |
| All others | 100/min per IP | Default protection |

## Behavior

**On rate limit exceeded:**
- HTTP 429 Too Many Requests
- Response body: `{"error": "rate limit exceeded"}`
- No retry-after header (yet)

**On Redis failure:**
- Request proceeds (fail-open, not fail-closed)
- Logged as warning
- Better than blocking all traffic if Redis is down

**IP extraction:**
1. First tries `X-Forwarded-For` header (load balancer/proxy)
2. Falls back to `RemoteAddr`
3. If behind Cloudflare/nginx, ensure X-Forwarded-For is set

## Implementation Details

```go
// RateLimitConfig in services/bff/cmd/bff/main.go
rlCfg := middleware.RateLimitConfig{
    PerIPLimit:        100,   // from RATE_LIMIT_RPS env
    PerPrincipalLimit: 200,   // 2x IP limit for authenticated users
    Window:            time.Minute,
}

// Applied in services/bff/internal/bff/server.go
r.Use(middleware.RateLimit(redisClient, rlCfg))
```

**Key format in Redis:**
- Per-IP: `rl:ip:<ip-address>`
- Per-user: `rl:principal:<user-id>`
- TTL: 2× window duration (auto-cleanup)

## Monitoring

**Redis keys to watch:**
```bash
# Check current rate limit state
redis-cli KEYS "rl:*"

# See how many requests an IP has made
redis-cli ZCARD "rl:ip:192.168.1.100"

# See timestamp details (last 10 requests)
redis-cli ZREVRANGE "rl:ip:192.168.1.100" 0 10 WITHSCORES
```

**Prometheus metrics** (if implemented):
- `http_rate_limit_exceeded_total{endpoint, ip}`
- `http_rate_limit_window_size{endpoint}`

## Testing Rate Limits

```bash
# Hammer an endpoint 150 times
for i in {1..150}; do
  curl -s -o /dev/null -w "%{http_code}\n" \
    http://localhost:8000/api/v1/listings
done

# Should see:
# 200 (x100 times)
# 429 (x50 times)
```

## Future Improvements

**Not yet implemented:**
1. **Per-endpoint limits** — currently all endpoints share global limit
2. **Burst allowance** — allow short bursts above sustained rate
3. **Retry-After header** — tell client when to retry
4. **Whitelist** — bypass rate limits for trusted IPs/API keys
5. **Distributed counting** — Redis cluster for multi-region
6. **Dynamic limits** — adjust based on system load
7. **User tier limits** — premium users get higher limits

**To add per-endpoint limits:**

Edit `services/bff/internal/bff/server.go`:
```go
// Apply stricter limits to specific routes
authRoutes := r.With(middleware.RateLimit(rdb, middleware.RateLimitConfig{
    PerIPLimit: 5,
    Window:     time.Minute,
}))
authRoutes.Post("/auth/otp", h.RequestOtp)
```

## Troubleshooting

**"Rate limit exceeded" but I just started testing:**
- Check if Redis has stale keys: `redis-cli FLUSHDB` to reset
- Ensure system clocks are synced (sliding window uses timestamps)

**Rate limits not working:**
- Check Redis connection: `redis-cli PING`
- Check logs for middleware errors
- Verify middleware is mounted in server.go

**False positives from load balancer:**
- If all requests come from load balancer IP, all users share one limit
- Ensure `X-Forwarded-For` header is preserved
- Or use authenticated user limits (`PerPrincipalLimit`)

## References

- Implementation: `services/bff/internal/bff/middleware/ratelimit.go`
- Configuration: `.env.example` (RATE_LIMIT_* vars)
- Sliding window algorithm: https://redis.io/glossary/rate-limiting/
