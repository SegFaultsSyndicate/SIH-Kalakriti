// pkg/breaker/grpc_test.go

package breaker

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func invokerReturning(err error) grpc.UnaryInvoker {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return err
	}
}

func TestTripsBreakerClassifiesGRPCCodes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil (success)", nil, false},
		{"NOT_FOUND", status.Error(codes.NotFound, "no such artisan"), false},
		{"INVALID_ARGUMENT", status.Error(codes.InvalidArgument, "bad phone"), false},
		{"PERMISSION_DENIED", status.Error(codes.PermissionDenied, "not yours"), false},
		{"ALREADY_EXISTS", status.Error(codes.AlreadyExists, "dup"), false},
		{"UNAVAILABLE", status.Error(codes.Unavailable, "connection refused"), true},
		{"DEADLINE_EXCEEDED", status.Error(codes.DeadlineExceeded, "timeout"), true},
		{"RESOURCE_EXHAUSTED", status.Error(codes.ResourceExhausted, "overloaded"), true},
		{"non-status error", context.Canceled, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tripsBreaker(tt.err); got != tt.want {
				t.Errorf("tripsBreaker(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestUnaryClientInterceptorNeverTripsOnBusinessErrors(t *testing.T) {
	b := New(3, time.Minute)
	interceptor := UnaryClientInterceptor(b)

	calls := 0
	for i := 0; i < 20; i++ {
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			calls++
			return status.Error(codes.NotFound, "no such listing")
		}
		err := interceptor(context.Background(), "/x.Service/Get", nil, nil, nil, invoker)
		if st, _ := status.FromError(err); st.Code() != codes.NotFound {
			t.Fatalf("call %d: got %v, want NOT_FOUND passed through untouched", i, err)
		}
	}
	if calls != 20 {
		t.Errorf("expected all 20 calls to reach the invoker (breaker must not open on business errors), got %d", calls)
	}
}

func TestUnaryClientInterceptorOpensOnInfraFailuresAndBlocksFurtherCalls(t *testing.T) {
	b := New(3, time.Hour) // long timeout: stay open for the rest of this test
	interceptor := UnaryClientInterceptor(b)

	calls := 0
	failing := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		calls++
		return status.Error(codes.Unavailable, "connection refused")
	}

	// Three consecutive UNAVAILABLE calls should reach the invoker and trip the breaker.
	for i := 0; i < 3; i++ {
		err := interceptor(context.Background(), "/x.Service/Get", nil, nil, nil, failing)
		if st, _ := status.FromError(err); st.Code() != codes.Unavailable {
			t.Fatalf("call %d: got %v, want UNAVAILABLE", i, err)
		}
	}
	if calls != 3 {
		t.Fatalf("expected 3 invoker calls before the circuit opens, got %d", calls)
	}

	// The 4th call must be rejected by the breaker itself, without touching the invoker.
	err := interceptor(context.Background(), "/x.Service/Get", nil, nil, nil, failing)
	if calls != 3 {
		t.Errorf("circuit should have short-circuited the 4th call, but invoker was called (calls=%d)", calls)
	}
	if st, _ := status.FromError(err); st.Code() != codes.Unavailable {
		t.Errorf("open-circuit error should still map to UNAVAILABLE for callers, got %v", err)
	}
}

func TestUnaryClientInterceptorHalfOpensAfterTimeout(t *testing.T) {
	b := New(2, 10*time.Millisecond)
	interceptor := UnaryClientInterceptor(b)

	failing := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return status.Error(codes.Unavailable, "down")
	}
	for i := 0; i < 2; i++ {
		_ = interceptor(context.Background(), "/x.Service/Get", nil, nil, nil, failing)
	}

	time.Sleep(20 * time.Millisecond)

	calls := 0
	succeeding := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		calls++
		return nil
	}
	// Half-open: the breaker must let this call reach the invoker rather than
	// short-circuiting it, even though the failure count is still at threshold.
	if err := interceptor(context.Background(), "/x.Service/Get", nil, nil, nil, succeeding); err != nil {
		t.Errorf("half-open trial call should have reached the invoker, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected the half-open trial to reach the invoker exactly once, got %d", calls)
	}
}
