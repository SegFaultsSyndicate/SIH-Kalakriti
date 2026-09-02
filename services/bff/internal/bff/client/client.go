// services/bff/internal/bff/client/client.go

// Package client holds bff's outbound gRPC clients: thin adapters from the
// handler package's loose service interfaces onto the real backend RPCs on
// core-svc, search-svc and insight-svc.
package client

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// callTimeout bounds every outbound RPC on top of whatever deadline ctx
// already carries, so a wedged backend can't hang the bff handler indefinitely.
const callTimeout = 10 * time.Second

// withTimeout bounds ctx and, when it carries a bearer token (stashed by
// middleware.Auth on every authenticated HTTP request), forwards it as
// outgoing gRPC metadata — core-svc's own interceptor requires one on every
// RPC except OTP request/verify and refresh. A public route's ctx carries no
// token, so its calls go out unauthenticated, same as before.
func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if token, ok := auth.TokenFrom(ctx); ok && token != "" {
		ctx = metadata.NewOutgoingContext(ctx, auth.BearerMetadata(token))
	}
	return context.WithTimeout(ctx, callTimeout)
}

// grpcErr maps a gRPC status error from a backend call to the domain error
// httpx.Error expects, so the HTTP response carries the right status code
// instead of always falling through to 500 Internal. An unmapped code (or a
// non-status error) is returned unchanged, which httpx.Error already renders
// as 500 with a generic message.
func grpcErr(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	msg := st.Message()
	switch st.Code() {
	case codes.NotFound:
		return domain.NotFound(msg)
	case codes.AlreadyExists:
		return domain.Conflict(msg)
	case codes.InvalidArgument:
		return domain.InvalidInput(msg)
	case codes.Unauthenticated:
		return domain.Unauthenticated(msg)
	case codes.PermissionDenied:
		return domain.Forbidden(msg)
	case codes.Unavailable, codes.DeadlineExceeded:
		return domain.Unavailable(msg)
	default:
		return err
	}
}
