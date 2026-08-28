// services/search-svc/internal/search/handler/consumer.go
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	segmentio "github.com/segmentio/kafka-go"

	"github.com/ZoroNewbie00/kalakriti/pkg/kafka"

	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/service"
)

// envelope is the outbox envelope every Kalakriti event is wrapped in.
type envelope struct {
	Header struct {
		EventID     string `json:"event_id"`
		AggregateID string `json:"aggregate_id"`
	} `json:"header"`
	Payload json.RawMessage `json:"payload"`
}

// listingPublished mirrors the fields the indexer needs. Suspension arrives on
// the same topic as a publish with suspended set, because the index cares about
// one thing: is this listing still visible.
type listingPublished struct {
	ListingID string `json:"listing_id"`
	Suspended bool   `json:"suspended"`
}

// ListingPublishedHandler projects a published listing into listing_search, and
// drops it again when the listing is withdrawn.
func ListingPublishedHandler(indexer *service.Indexer, log *slog.Logger) kafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		var env envelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			return fmt.Errorf("decoding the envelope at offset %d: %w", msg.Offset, err)
		}
		var payload listingPublished
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return fmt.Errorf("decoding the payload of event %s: %w", env.Header.EventID, err)
		}

		listingID, err := uuid.Parse(env.Header.AggregateID)
		if err != nil {
			return fmt.Errorf("aggregate_id %q is not a uuid: %w", env.Header.AggregateID, err)
		}

		if payload.Suspended {
			return indexer.Remove(ctx, listingID)
		}
		if err := indexer.Index(ctx, listingID); err != nil {
			return fmt.Errorf("indexing listing %s: %w", listingID, err)
		}
		log.DebugContext(ctx, "listing indexed", "listing_id", listingID)
		return nil
	}
}
