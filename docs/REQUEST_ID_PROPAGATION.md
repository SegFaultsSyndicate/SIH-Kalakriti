# Request ID Propagation

**Status:** HTTP layer complete, gRPC layer ready for wiring  
**Last Updated:** 2026-08-28

---

## Current Implementation

### HTTP Layer (BFF)

Request IDs are **already generated and logged** at the BFF edge:

1. **Generation:** `pkg/httpx/middleware.go` line 51 — `chi/v5/middleware.RequestID` generates UUIDs
2. **Context injection:** `pkg/logger/logger.go` line 97-100 — extracts chi's request ID and attaches to context
3. **Logging:** Every log line includes `request_id` field via `logger.With(ctx, base)`
4. **Response header:** Chi's RequestID middleware sets `X-Request-ID` response header automatically

**What's covered:**
- Every HTTP request to BFF gets a unique request ID
- All BFF logs include `request_id`
- Response includes `X-Request-ID` header for client correlation
- Client can send `X-Request-ID` in request to preserve their own ID

---

## gRPC Propagation (Not Yet Wired)

The BFF's gRPC clients are currently TODO stubs (`services/bff/cmd/bff/main.go:46-57`). When wiring them, add this interceptor to propagate request IDs downstream:

### Unary Interceptor

```go
// pkg/grpcx/requestid.go
package grpcx

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	
	"github.com/segfaultsyndicate/kalakriti/pkg/logger"
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

### Wiring at BFF Client

```go
// services/bff/cmd/bff/main.go (when wiring gRPC clients)
import (
	"google.golang.org/grpc"
	"github.com/segfaultsyndicate/kalakriti/pkg/grpcx"
)

conn, err := grpc.NewClient(
	coreSvcAddr,
	grpc.WithUnaryInterceptor(grpcx.UnaryClientRequestID()),
)
```

### Wiring at Service Server

```go
// services/core-svc/cmd/core-svc/main.go
import (
	"google.golang.org/grpc"
	"github.com/segfaultsyndicate/kalakriti/pkg/grpcx"
)

grpcServer := grpc.NewServer(
	grpc.UnaryInterceptor(grpcx.UnaryServerRequestID()),
)
```

---

## Flow

```
1. Client → BFF
   Request-ID: (generated or passed through)
   
2. BFF logs with request_id
   {"level":"info","request_id":"abc123",...}
   
3. BFF → core-svc (gRPC)
   metadata: x-request-id=abc123
   
4. core-svc logs with request_id
   {"level":"info","request_id":"abc123",...}
   
5. core-svc → search-svc (gRPC)
   metadata: x-request-id=abc123
   
6. search-svc logs with request_id
   {"level":"info","request_id":"abc123",...}
```

Every log line across all services shares the same `request_id`, making distributed traces trivial to correlate.

---

## Verification

Once gRPC clients are wired:

```bash
# Make a request
curl -H "X-Request-ID: test-123" http://localhost:8000/api/v1/listings

# Grep all service logs for that ID
docker compose logs | grep test-123

# Should see:
# bff_1         | {"request_id":"test-123",...}
# core-svc_1    | {"request_id":"test-123",...}
# search-svc_1  | {"request_id":"test-123",...}
```

---

## Summary

**Done now:** HTTP request ID generation and logging at BFF edge  
**When gRPC clients wired:** Add 2 interceptors (10 lines each) to propagate through the stack  
**Cost:** ~20 lines of code, zero runtime overhead

The hard part (generating IDs, context threading, log injection) is already done. gRPC propagation is mechanical once clients exist.
