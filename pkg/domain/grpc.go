// pkg/domain/grpc.go
package domain

import (
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCStatus maps a domain error to a gRPC status. An unmapped error becomes
// Internal with a generic message -- unlike every other case here, this one
// deliberately does NOT carry err.Error() over the wire: an unmapped error is
// exactly the kind that can carry something never meant to leave the
// process (a raw pgx error can embed DSN fragments, a stack-adjacent path,
// etc.), see TestGRPCErrorUnmappedHidesDetail. Masking here without also
// logging is its own bug though -- it previously made every unmapped error
// completely undiagnosable, visible nowhere, not even in the originating
// service's own logs. So: log the real error right here, at the point
// closest to the actual failure, then mask before it goes over the wire.
func GRPCStatus(err error) *status.Status {
	switch {
	case err == nil:
		return status.New(codes.OK, "")
	case errors.Is(err, ErrNotFound):
		return status.New(codes.NotFound, err.Error())
	case errors.Is(err, ErrConflict):
		return status.New(codes.AlreadyExists, err.Error())
	case errors.Is(err, ErrInvalidInput):
		return status.New(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrUnauthenticated):
		return status.New(codes.Unauthenticated, err.Error())
	case errors.Is(err, ErrForbidden):
		return status.New(codes.PermissionDenied, err.Error())
	case errors.Is(err, ErrUnavailable):
		return status.New(codes.Unavailable, err.Error())
	default:
		slog.Default().Error("unmapped error rendered as grpc Internal", "error", err)
		return status.New(codes.Internal, "internal error")
	}
}

// GRPCError returns GRPCStatus(err).Err(), the form a gRPC handler actually returns.
func GRPCError(err error) error {
	if err == nil {
		return nil
	}
	return GRPCStatus(err).Err()
}
