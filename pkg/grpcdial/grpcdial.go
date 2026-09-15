// pkg/grpcdial/grpcdial.go

// Package grpcdial dials a service-to-service gRPC connection with
// client-side round_robin load balancing, so scaling a backend to multiple
// pods behind a headless k8s Service actually spreads traffic across them.
package grpcdial

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// roundRobinServiceConfig is grpc-go's standard "loadBalancingConfig" JSON,
// documented at https://github.com/grpc/grpc/blob/master/doc/service_config.md.
const roundRobinServiceConfig = `{"loadBalancingConfig": [{"round_robin":{}}]}`

// Dial opens an insecure gRPC connection to addr with round_robin load
// balancing. The "dns:///" prefix is required: grpc-go's default resolver
// scheme is "passthrough", which treats addr as one opaque address and never
// re-resolves it, so round_robin would have nothing to balance across even
// with the policy set. The "dns" scheme re-resolves addr's A records, which
// is what lets it discover every pod IP behind a headless Service (or every
// address docker-compose's embedded DNS returns for a scaled container
// name) instead of pinning to whichever one it first resolved.
func Dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		"dns:///"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(roundRobinServiceConfig),
	)
}
