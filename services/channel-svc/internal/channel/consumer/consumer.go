// services/channel-svc/internal/channel/consumer/consumer.go

package consumer

import (
	"context"

	pkgkafka "github.com/ZoroNewbie00/kalakriti/pkg/kafka"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"
)

// Run starts the fanout consumer loop.
func (f *FollowFanout) Run(ctx context.Context, reader *pkgkafka.Reader) {
	f.log.Info("fanout consumer started", "topic", topics.CatalogListingPublished)

	for {
		select {
		case <-ctx.Done():
			f.log.Info("fanout consumer stopping")
			return
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				f.log.Error("fanout read error", "error", err)
				continue
			}

			if err := f.Handle(ctx, msg.Value); err != nil {
				f.log.Error("fanout handle error", "error", err)
				continue
			}
		}
	}
}