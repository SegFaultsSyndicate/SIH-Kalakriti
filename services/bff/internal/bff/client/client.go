// services/bff/internal/bff/client/client.go

// Package client holds bff's outbound gRPC clients: thin adapters from the
// handler package's loose service interfaces onto the real backend RPCs on
// core-svc, search-svc and insight-svc.
package client

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// callTimeout bounds every outbound RPC. The handler interfaces these
// adapters satisfy take no context, so a bare context.Background() would let
// a wedged backend hang the bff handler indefinitely; this keeps the failure
// bounded even though it can't inherit the inbound request's deadline.
const callTimeout = 10 * time.Second

func withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), callTimeout)
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
