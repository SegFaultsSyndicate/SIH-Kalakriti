// services/core-svc/internal/core/handler/consumer.go
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	segmentio "github.com/segmentio/kafka-go"

	"github.com/segfaultsyndicate/kalakriti/pkg/kafka"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/service"
)

// envelope is the outbox envelope every Kalakriti event is wrapped in. Only the
// header fields a consumer needs are decoded here; the payload stays raw until
// the specific handler knows what shape it is.
type envelope struct {
	Header struct {
		EventID        string `json:"event_id"`
		AggregateID    string `json:"aggregate_id"`
		IdempotencyKey string `json:"idempotency_key"`
		SchemaVersion  int32  `json:"schema_version"`
		Producer       string `json:"producer"`
	} `json:"header"`
	Payload json.RawMessage `json:"payload"`
}

// mediaUploaded mirrors events.v1.MediaUploaded's payload fields; the pipeline
// only needs to know which asset landed.
type mediaUploaded struct {
	MediaID string `json:"media_id"`
}

// listingPublished mirrors the fields the translation fan-out needs.
type listingPublished struct {
	ListingID string `json:"listing_id"`
	ArtisanID string `json:"artisan_id"`
	CraftID   string `json:"craft_id"`
}

// mediaEnhanced mirrors events.v1.MediaEnhanced's payload fields.
type mediaEnhanced struct {
	MediaID           string  `json:"media_id"`
	EnhancedObjectKey string  `json:"enhanced_object_key"`
	ModelVersion      *string `json:"model_version,omitempty"`
	FailureReason     *string `json:"failure_reason,omitempty"`
}

// MediaEnhancedHandler adapts the media.enhanced topic to the media service. It
// is deliberately thin: decode, call, return. Retries, backoff and the
// dead-letter hop all belong to pkg/kafka's runner, and idempotency belongs to
// the service, which is what makes a redelivery safe.
func MediaEnhancedHandler(svc *service.Media, log *slog.Logger) kafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		var env envelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			// A message that cannot be decoded will never decode: returning an
			// error routes it to the dead-letter topic after the retries, which
			// is where a human can look at it.
			return fmt.Errorf("decoding the media.enhanced envelope at offset %d: %w", msg.Offset, err)
		}

		var payload mediaEnhanced
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return fmt.Errorf("decoding the media.enhanced payload for event %s: %w", env.Header.EventID, err)
		}

		mediaID, err := uuid.Parse(payload.MediaID)
		if err != nil {
			return fmt.Errorf("media.enhanced carries media_id %q, which is not a uuid: %w", payload.MediaID, err)
		}

		idempotencyKey := env.Header.IdempotencyKey
		if idempotencyKey == "" {
			// Older producers keyed only by event id; either is stable across a
			// redelivery, which is all the outbox needs.
			idempotencyKey = env.Header.EventID
		}

		result, err := svc.ApplyEnhancement(ctx, service.EnhancementResult{
			MediaID:           mediaID,
			EnhancedObjectKey: payload.EnhancedObjectKey,
			ModelVersion:      payload.ModelVersion,
			FailureReason:     payload.FailureReason,
			IdempotencyKey:    idempotencyKey,
		})
		if err != nil {
			return fmt.Errorf("applying enhancement to media %s: %w", mediaID, err)
		}

		log.DebugContext(ctx, "media.enhanced applied",
			"media_id", mediaID, "state", result.State, "event_id", env.Header.EventID)
		return nil
	}
}

// MediaUploadedHandler runs the cataloguing pipeline for one uploaded asset.
// Retries, backoff and the dead-letter hop are pkg/kafka's; idempotency is the
// pipeline's, so a redelivery here is safe by construction.
func MediaUploadedHandler(pipeline *service.Pipeline, log *slog.Logger) kafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		var payload mediaUploaded
		mediaID, err := decodeAggregate(msg, &payload)
		if err != nil {
			return err
		}
		if err := pipeline.Run(ctx, mediaID); err != nil {
			return fmt.Errorf("cataloguing media %s: %w", mediaID, err)
		}
		log.DebugContext(ctx, "pipeline run", "media_id", mediaID)
		return nil
	}
}

// MediaUploadedDeadLetter records why an asset was given up on, so a message
// that ran out of retries leaves the artisan a reason rather than silence.
func MediaUploadedDeadLetter(pipeline *service.Pipeline, log *slog.Logger) func(context.Context, segmentio.Message, error) {
	return func(ctx context.Context, msg segmentio.Message, cause error) {
		var payload mediaUploaded
		mediaID, err := decodeAggregate(msg, &payload)
		if err != nil {
			log.ErrorContext(ctx, "dead-lettered a message we cannot attribute", "error", err)
			return
		}
		if err := pipeline.RecordFailure(ctx, mediaID, cause.Error()); err != nil {
			log.ErrorContext(ctx, "recording a dead-lettered failure", "media_id", mediaID, "error", err)
		}
	}
}

// ListingPublishedHandler fans a newly published listing's copy out to the buyer
// languages.
func ListingPublishedHandler(pipeline *service.Pipeline, log *slog.Logger) kafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		var payload listingPublished
		if _, err := decodeAggregate(msg, &payload); err != nil {
			return err
		}

		listingID, err := uuid.Parse(payload.ListingID)
		if err != nil {
			return fmt.Errorf("listing_id %q is not a uuid: %w", payload.ListingID, err)
		}
		artisanID, err := uuid.Parse(payload.ArtisanID)
		if err != nil {
			return fmt.Errorf("artisan_id %q is not a uuid: %w", payload.ArtisanID, err)
		}
		craftID, err := uuid.Parse(payload.CraftID)
		if err != nil {
			return fmt.Errorf("craft_id %q is not a uuid: %w", payload.CraftID, err)
		}

		if err := pipeline.Translate(ctx, listingID, artisanID, craftID); err != nil {
			return fmt.Errorf("translating listing %s: %w", listingID, err)
		}
		log.DebugContext(ctx, "translation fan-out", "listing_id", listingID)
		return nil
	}
}

// decodeAggregate unwraps the envelope into payload and returns the aggregate id
// the event is keyed by.
func decodeAggregate(msg segmentio.Message, payload any) (uuid.UUID, error) {
	var env envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		return uuid.Nil, fmt.Errorf("decoding the envelope at offset %d: %w", msg.Offset, err)
	}
	if err := json.Unmarshal(env.Payload, payload); err != nil {
		return uuid.Nil, fmt.Errorf("decoding the payload of event %s: %w", env.Header.EventID, err)
	}
	// The envelope's aggregate id is authoritative and is what the topic is
	// partitioned by, so it is what we act on.
	id, err := uuid.Parse(env.Header.AggregateID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("aggregate_id %q is not a uuid: %w", env.Header.AggregateID, err)
	}
	return id, nil
}
