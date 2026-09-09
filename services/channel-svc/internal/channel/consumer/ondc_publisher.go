// services/channel-svc/internal/channel/consumer/ondc_publisher.go
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	segmentio "github.com/segmentio/kafka-go"

	pkgkafka "github.com/ZoroNewbie00/kalakriti/pkg/kafka"

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

// HandlerFunc adapts Handle to pkg/kafka.HandlerFunc — see FollowFanout's
// identical note; a fetch/publish failure now actually gets retried and,
// on exhaustion, dead-lettered instead of silently vanishing.
func (p *ONDCPublisher) HandlerFunc() pkgkafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		return p.Handle(ctx, msg.Value)
	}
}
