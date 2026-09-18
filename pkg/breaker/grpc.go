// pkg/breaker/grpc.go

package breaker

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// tripsBreaker reports whether err represents the downstream service being
// unreachable or overloaded, as opposed to an ordinary business error
// (NOT_FOUND, INVALID_ARGUMENT, PERMISSION_DENIED, ...). Only the former
// should open the circuit -- counting a business error would mean one
// caller sending a bad request degrades the connection for every other
// caller sharing it.
func tripsBreaker(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return true // not a gRPC status at all (dial/transport failure) -- treat as infra
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

// UnaryClientInterceptor wraps unary gRPC calls in a circuit breaker. Build
// one Breaker per downstream connection (grpcdial.Dial does this for every
// dial site) -- never share a single Breaker across calls to different
// services, or one service being down would also reject calls to a
// healthy one.
func UnaryClientInterceptor(b *Breaker) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var callErr error
		breakerErr := b.Call(func() error {
			callErr = invoker(ctx, method, req, reply, cc, opts...)
			if tripsBreaker(callErr) {
				return callErr
			}
			return nil // business error: let it through without counting against the breaker
		})
		if errors.Is(breakerErr, ErrCircuitOpen) {
			return status.Errorf(codes.Unavailable, "circuit breaker open for %s: too many recent failures", method)
		}
		return callErr
	}
}
