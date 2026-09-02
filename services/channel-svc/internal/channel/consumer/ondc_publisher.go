// services/channel-svc/internal/channel/consumer/ondc_publisher.go
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pkgkafka "github.com/ZoroNewbie00/kalakriti/pkg/kafka"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/ondc"
)

// CatalogClient fetches one listing's full detail, hydrated for an outbound
// channel; satisfied by *client.Catalog.
type CatalogClient interface {
	GetListingForONDC(ctx context.Context, listingID string) (ondc.Listing, error)
}

// ONDCPublisher consumes catalog.listing.published and publishes the newly
// live listing to the ONDC network.
type ONDCPublisher struct {
	catalog CatalogClient
	ondc    *ondc.Client
	log     *slog.Logger
}

// NewONDCPublisher creates the publisher.
func NewONDCPublisher(catalog CatalogClient, ondcClient *ondc.Client, log *slog.Logger) *ONDCPublisher {
	if log == nil {
		log = slog.Default()
	}
	return &ONDCPublisher{catalog: catalog, ondc: ondcClient, log: log}
}

// listingPublishedEvent is the subset of catalog.listing.published this
// consumer needs — just enough to look the listing back up.
type listingPublishedEvent struct {
	Payload struct {
		ListingID string `json:"listing_id"`
	} `json:"payload"`
}

// Handle processes one catalog.listing.published event: fetch the listing's
// full detail and publish it to ONDC. A fetch or publish failure is returned
// so the caller can retry the delivery — this consumer does not swallow
// errors the way FollowFanout's per-recipient loop does, since there is only
// one listing per event, not many independent recipients.
func (p *ONDCPublisher) Handle(ctx context.Context, eventBytes []byte) error {
	var evt listingPublishedEvent
	if err := json.Unmarshal(eventBytes, &evt); err != nil {
		return fmt.Errorf("ondc publisher: unmarshal event: %w", err)
	}
	if evt.Payload.ListingID == "" {
		return fmt.Errorf("ondc publisher: event has no listing_id")
	}

	listing, err := p.catalog.GetListingForONDC(ctx, evt.Payload.ListingID)
	if err != nil {
		return fmt.Errorf("ondc publisher: fetching listing %s: %w", evt.Payload.ListingID, err)
	}

	if err := p.ondc.PublishOnSearch(ctx, []ondc.Listing{listing}); err != nil {
		return fmt.Errorf("ondc publisher: publishing listing %s: %w", evt.Payload.ListingID, err)
	}

	p.log.Info("ondc_published_listing", "listing_id", evt.Payload.ListingID)
	return nil
}

// Run starts the consumer loop, mirroring FollowFanout.Run — a second,
// independent consumer group on the same topic, so each subscriber gets
// every event regardless of the other's read position.
func (p *ONDCPublisher) Run(ctx context.Context, reader *pkgkafka.Reader) {
	p.log.Info("ondc publisher consumer started", "topic", topics.CatalogListingPublished)

	for {
		select {
		case <-ctx.Done():
			p.log.Info("ondc publisher consumer stopping")
			return
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				p.log.Error("ondc publisher read error", "error", err)
				continue
			}

			if err := p.Handle(ctx, msg.Value); err != nil {
				p.log.Error("ondc publisher handle error", "error", err)
				continue
			}
		}
	}
}
