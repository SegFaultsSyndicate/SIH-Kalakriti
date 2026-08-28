// pkg/domain/grpc.go
package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCStatus maps a domain error to a gRPC status. An unmapped error becomes
// Internal with a generic message rather than leaking its detail to a caller.
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
