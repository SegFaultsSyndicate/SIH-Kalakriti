// pkg/grpcdial/grpcdial_test.go
package grpcdial

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
)

// countingHealth answers a gRPC health check and counts how many times this
// particular server instance was hit, so the test can tell whether requests
// actually spread across backends or all pinned to one.
type countingHealth struct {
	grpc_health_v1.UnimplementedHealthServer
	hits *int64
}

func (h *countingHealth) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	atomic.AddInt64(h.hits, 1)
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

// TestRoundRobinDistributesAcrossBackends proves roundRobinServiceConfig (the
// same string Dial uses) actually spreads RPCs across multiple resolved
// addresses, rather than pinning to the first one -- the failure mode this
// package exists to avoid. It stands in for real DNS/headless-Service
// resolution with grpc's manual resolver, since a unit test can't spin up
// CoreDNS or multiple pods.
func TestRoundRobinDistributesAcrossBackends(t *testing.T) {
	const numBackends = 3
	const numCalls = 300 // 100 per backend on a perfectly even split

	hitCounts := make([]int64, numBackends)
	var addrs []resolver.Address

	for i := 0; i < numBackends; i++ {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listening for fake backend %d: %v", i, err)
		}
		srv := grpc.NewServer()
		grpc_health_v1.RegisterHealthServer(srv, &countingHealth{hits: &hitCounts[i]})
		go func() { _ = srv.Serve(lis) }()
		defer srv.Stop()

		addrs = append(addrs, resolver.Address{Addr: lis.Addr().String()})
	}

	res := manual.NewBuilderWithScheme("grpcdialtest")
	res.InitialState(resolver.State{Addresses: addrs})

	conn, err := grpc.NewClient(
		res.Scheme()+":///ignored",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(roundRobinServiceConfig),
		grpc.WithResolvers(res),
	)
	if err != nil {
		t.Fatalf("dialing: %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	for i := 0; i < numCalls; i++ {
		if _, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{}); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}

	for i, hits := range hitCounts {
		if hits == 0 {
			t.Errorf("backend %d received zero requests out of %d -- round_robin is not distributing across backends", i, numCalls)
		}
	}
}
