// services/channel-svc/internal/channel/consumer/consumer.go

package consumer

import (
	"context"

	segmentio "github.com/segmentio/kafka-go"

	pkgkafka "github.com/ZoroNewbie00/kalakriti/pkg/kafka"
)

// HandlerFunc adapts Handle to pkg/kafka.HandlerFunc, so retries, backoff
// and the dead-letter hop are pkg/kafka.ConsumerGroup's job, not this
// package's — the same division of labour core-svc's consumers already use.
func (f *FollowFanout) HandlerFunc() pkgkafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		return f.Handle(ctx, msg.Value)
	}
}
