# Request ID Propagation

**Status:** HTTP layer complete at the bff edge. gRPC propagation is still not
wired — this doc previously claimed the reason was that bff's gRPC clients
were TODO stubs; that's no longer true (they're real client connections, see
`services/bff/cmd/bff/main.go`), but nothing propagates a request ID across a
gRPC call regardless. `pkg/grpcx` (referenced below) doesn't exist yet — this
is a real gap, not a stale doc describing finished work.
**Last Updated:** 2026-09-15

---

## Current Implementation

### HTTP layer (bff)

Request IDs are generated and logged at the bff edge:

1. **Generation:** `pkg/httpx/middleware.go` — `httpx.RequestID` reads an
   inbound `X-Request-Id` header or generates a UUID.
2. **Context injection:** the same middleware attaches it via
   `logger.ContextWithRequestID`; `pkg/logger/logger.go`'s `Middleware` reads
   it back (falling back to a fresh UUID if that middleware isn't mounted).
3. **Logging:** every log line includes a `request_id` field via
   `logger.With(ctx, base)`.
4. **Response header:** `httpx.RequestID` echoes it back as `X-Request-Id`.

**What's covered:** every HTTP request to bff gets a unique request ID, every
bff log line includes it, and the response carries it back for client-side
correlation. A client can send its own `X-Request-ID` to have it preserved.

**What's not covered:** once bff calls core-svc/search-svc/collab-svc/
channel-svc/insight-svc over gRPC, the request ID is dropped. Each backend
service's own logs get a fresh, unrelated ID (or none, depending on whether
that service's own logging middleware generates one) — there is currently no
way to grep one request's logs across service boundaries.

---

## gRPC Propagation — Not Yet Wired (real gap, verified against current code)

Verified directly: `pkg/grpcx` does not exist anywhere in the repo, no service
attaches `x-request-id` (or any similar key) to outgoing gRPC metadata, and no
service's gRPC server reads incoming metadata for a request ID. bff's clients
(`services/bff/cmd/bff/main.go`) are plain `grpc.NewClient(...)` calls with no
interceptor chain at all today.

If you build this, here's the shape that would fit the existing HTTP-layer
pattern:

### Unary interceptor

```go
// pkg/grpcx/requestid.go — proposed, does not exist yet
package grpcx

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/ZoroNewbie00/kalakriti/pkg/logger"
)

// UnaryClientRequestID propagates request_id from context into gRPC metadata.
func UnaryClientRequestID() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if reqID := logger.RequestID(ctx); reqID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", reqID)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// UnaryServerRequestID extracts request_id from gRPC metadata into context.
func UnaryServerRequestID() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md.Get("x-request-id"); len(ids) > 0 {
				ctx = logger.ContextWithRequestID(ctx, ids[0])
			}
		}
		return handler(ctx, req)
	}
}
```

### Wiring at bff's clients

```go
// services/bff/cmd/bff/main.go — where the grpc.NewClient(...) calls already are
coreConn, err := grpc.NewClient(
	getEnv("CORE_SVC_ADDR", "localhost:50051"),
	grpc.WithTransportCredentials(insecure.NewCredentials()),
	grpc.WithChainUnaryInterceptor(grpcx.UnaryClientRequestID()),
)
```

### Wiring at each service's gRPC server

core-svc already chains interceptors for auth (`services/core-svc/internal/core/handler/identity.go`'s
`PublicMethods()` + `auth.UnaryServerInterceptor`); add the request-ID
interceptor to that same chain. search-svc/collab-svc/channel-svc/insight-svc
currently register bare or recovery-only interceptor chains (see
`docs/PORTS_AND_APIS.md` §3) — add it there too if you want request IDs to
survive the hop.

```go
grpcServer := grpc.NewServer(
	grpc.ChainUnaryInterceptor(
		grpcx.UnaryServerRequestID(),
		// existing interceptors (recovery, auth) go here too
	),
)
```

---

## Flow, once wired

```
1. Client → bff            X-Request-Id: (generated or passed through)
2. bff logs                {"request_id":"abc123",...}
3. bff → core-svc (gRPC)   metadata: x-request-id=abc123
4. core-svc logs           {"request_id":"abc123",...}
5. core-svc → ml-svc/search-svc (gRPC)   metadata: x-request-id=abc123
6. downstream service logs {"request_id":"abc123",...}
```

Every log line across all services would share the same `request_id`, making
distributed traces trivial to grep for even without the Jaeger/OTLP tracing
already running (`OTLP_ENDPOINT`, `jaeger:4317` — see `docs/PORTS_AND_APIS.md`
§1). Trace IDs from OTLP spans already give you cross-service correlation if
you're using Jaeger's UI at `:16686`; this doc is about being able to `grep`
plain-text/JSON logs by request ID without needing a tracing backend.

---

## Verification (once implemented)

```bash
curl -H "X-Request-ID: test-123" http://localhost:8000/api/v1/listings
docker compose logs | grep test-123
# Expect to see it in bff's logs today; core-svc/search-svc's logs only once
# the gRPC interceptors above are added.
```

---

## Summary

**Done:** HTTP request-ID generation, context threading, and logging at the
bff edge.
**Not done:** `pkg/grpcx` doesn't exist; no gRPC client or server anywhere
attaches/reads `x-request-id` metadata. This is real, actionable work — not a
few lines away as "propagation is mechanical once clients exist" (the clients
already exist; the interceptors don't).
