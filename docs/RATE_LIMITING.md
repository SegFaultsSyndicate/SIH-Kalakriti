# Rate Limiting Configuration

Kalakriti BFF uses Redis-backed sliding window rate limiting to protect against abuse.

## How It Works

**Sliding Window Counter Algorithm:**
1. Each request adds a timestamp to a Redis sorted set
2. Old entries outside the window are removed
3. If count ≤ limit, request proceeds; otherwise the request is rejected as
   **`503 Service Unavailable`** -- not 429 (see "Behavior" below for why)
4. Tracks both per-IP and per-authenticated-user separately

**Implementation:** `services/bff/internal/bff/middleware/ratelimit.go`

## Configuration

**There is no env-based configuration today.** `RATE_LIMIT_RPS`/
`RATE_LIMIT_BURST`/`RATE_LIMIT_WINDOW`/`RATE_LIMIT_ENDPOINTS` (also listed in
`.env.example`) are not read by any code -- `grep -rn RATE_LIMIT --include=*.go`
turns up nothing outside this doc's own claims. The real limits are hardcoded
in `services/bff/cmd/bff/main.go`:

```go
RateLimitPerIP:        100,           // requests per window, per IP
RateLimitPerPrincipal: 1000,          // requests per window, per authenticated principal
RateLimitWindow:       time.Minute,
```

**Current values:**
- **Per-IP limit:** 100 requests/minute
- **Per-authenticated-principal limit:** 1000 requests/minute (10x the IP
  limit, not 2x -- both apply to an authenticated request; whichever is hit
  first rejects)
- **Window:** 60 seconds (sliding)

To change these, edit `main.go` and redeploy -- there's no live-tunable knob
yet. If you want the env vars above to actually work, that's unbuilt: wire
`RateLimitConfig`'s three fields to `os.Getenv`/`pkg/config` in `main.go`.

## Limits by Endpoint Type

Only one per-endpoint override actually exists today: `POST /auth/otp/request`
gets an additional, stricter check via `middleware.RateLimitEndpoint(redis,
"otp", 5, 10*time.Minute)` on top of the global limit above -- 5 requests per
10 minutes per IP (`services/bff/internal/bff/server.go`). Every other
route (`/listings`, `/search`, `/orders/bulk`, etc.) shares only the one
global per-IP/per-principal limit from the Configuration section; the table
below is a **recommendation for future per-endpoint tuning**, not current
behavior:

| Endpoint | Recommended Limit | Reason |
|---------------|-------------------|--------|
| `/auth/otp/request` | 5/10min per IP (**real, already enforced**) | Prevent SMS abuse |
| `/auth/otp/verify` | 10/min per IP (not yet enforced) | Prevent brute force |
| `/media/upload-url` | 20/min per user (not yet enforced) | Limit storage creation |
| `/search` | 50/min per IP (not yet enforced) | Expensive vector queries |
| `/listings` (read) | 100/min per IP (covered by the global default) | Cheap reads |
| `/orders/bulk` (write) | 10/min per user (not yet enforced) | Prevent duplicate orders |
| All others | 100/min per IP, 1000/min per principal (global default) | Default protection |

## Behavior

**On rate limit exceeded:**
- HTTP **503 Service Unavailable**, not 429 -- the middleware calls
  `domain.Unavailable("rate limit exceeded")`, and `pkg/domain`'s status
  mapping sends `ErrUnavailable` to 503. There is no 429 path in this
  codebase's rate limiter.
- Response body (the real two-field error envelope, see `docs/API.md`):
  `{"error": "unavailable", "message": "rate limit exceeded"}` (the
  OTP-specific limiter's message is `"otp rate limit exceeded, please retry later"`)
- No `Retry-After` header, no `X-RateLimit-*` response headers anywhere in
  the implementation

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
// RateLimitConfig in services/bff/cmd/bff/main.go -- literal values, not
// read from any env var despite the field names suggesting otherwise.
RateLimitPerIP:        100,
RateLimitPerPrincipal: 1000,  // 10x IP limit for authenticated principals
RateLimitWindow:       time.Minute,

// Applied in services/bff/internal/bff/server.go, scoped to the /api/v1
// group only -- SEO pages (/listing/:slug, /v/:code, /sitemap.xml, etc.)
// and the SPA fallback are NOT rate limited by this middleware.
api := r.Group("/api/v1")
api.Use(httpx.Wrap(middleware.RateLimit(cfg.Redis, middleware.RateLimitConfig{
    PerIPLimit:        cfg.RateLimitPerIP,
    PerPrincipalLimit: cfg.RateLimitPerPrincipal,
    Window:            cfg.RateLimitWindow,
})))
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
# 503 (x50 times)  -- not 429, see "Behavior" above
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
