// pkg/grpcdial/grpcdial.go

// Package grpcdial dials a service-to-service gRPC connection with
// client-side round_robin load balancing, so scaling a backend to multiple
// pods behind a headless k8s Service actually spreads traffic across them.
package grpcdial

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ZoroNewbie00/kalakriti/pkg/breaker"
)

// roundRobinServiceConfig is grpc-go's standard "loadBalancingConfig" JSON,
// documented at https://github.com/grpc/grpc/blob/master/doc/service_config.md.
const roundRobinServiceConfig = `{"loadBalancingConfig": [{"round_robin":{}}]}`

// Breaker defaults: open after 5 consecutive infrastructure failures
// (UNAVAILABLE/DEADLINE_EXCEEDED/RESOURCE_EXHAUSTED -- see
// breaker.UnaryClientInterceptor), stay open 30s before a half-open trial.
// A bare timeout alone let a single unresponsive downstream hold every
// caller of it at the full RPC timeout indefinitely (see
// WIRING_AUDIT_PLAN.md F-11 -- a 10s follower-count call in a stale log,
// full timeout's worth of latency on a route the artisan dashboard blocks
// on); once open, callers fail fast instead.
// On a cold `docker compose up`, services start in parallel, so the first
// handful of calls to a not-yet-listening downstream are 5 back-to-back
// UNAVAILABLE dials -- enough to open the breaker and self-inflict up to
// breakerOpenTimeout of failing fast on a backend that would otherwise have
// been reachable within a second or two of binding its port. Bounded (it
// half-opens and recovers on its own), but worth knowing if `make demo-up`
// looks slow to come up rather than broken.
const (
	breakerFailureThreshold = 5
	breakerOpenTimeout      = 30 * time.Second
)

// Dial opens an insecure gRPC connection to addr with round_robin load
// balancing and a circuit breaker. The "dns:///" prefix is required:
// grpc-go's default resolver scheme is "passthrough", which treats addr as
// one opaque address and never re-resolves it, so round_robin would have
// nothing to balance across even with the policy set. The "dns" scheme
// re-resolves addr's A records, which is what lets it discover every pod IP
// behind a headless Service (or every address docker-compose's embedded DNS
// returns for a scaled container name) instead of pinning to whichever one
// it first resolved.
//
// Each call to Dial gets its own Breaker instance, scoped to this one
// connection -- never share a Breaker across Dial calls to different
// services, or one downstream being down would also reject calls to a
// healthy one.
func Dial(addr string) (*grpc.ClientConn, error) {
	b := breaker.New(breakerFailureThreshold, breakerOpenTimeout)
	return grpc.NewClient(
		"dns:///"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(roundRobinServiceConfig),
		grpc.WithChainUnaryInterceptor(breaker.UnaryClientInterceptor(b)),
	)
}
