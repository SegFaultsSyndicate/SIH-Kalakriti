// services/core-svc/internal/core/handler/consumer_test.go
package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	segmentio "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

type fakeBadgeTracker struct {
	trackedArtisanIDs []string
}

func (f *fakeBadgeTracker) TrackListingPublished(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	f.trackedArtisanIDs = append(f.trackedArtisanIDs, artisanID.String())
	return nil, nil
}
func (f *fakeBadgeTracker) TrackProvenanceSealed(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	return nil, nil
}
func (f *fakeBadgeTracker) TrackLotAccepted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	return nil, nil
}
func (f *fakeBadgeTracker) TrackLotCompleted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	return nil, nil
}

func TestCatalogListingPublishedBadgeHandler_TracksArtisan(t *testing.T) {
	artisanID := "018f1e2a-0000-7000-8000-000000000001"
	envelope := map[string]any{
		"header":  map[string]any{"aggregate_id": "018f1e2a-0000-7000-8000-000000000099", "event_id": "018f1e2a-0000-7000-8000-000000000099"},
		"payload": map[string]any{"listing_id": "018f1e2a-0000-7000-8000-000000000099", "artisan_id": artisanID, "craft_id": "018f1e2a-0000-7000-8000-000000000002"},
	}
	value, err := json.Marshal(envelope)
	require.NoError(t, err)

	tracker := &fakeBadgeTracker{}
	handle := CatalogListingPublishedBadgeHandler(tracker, nil)

	err = handle(context.Background(), segmentio.Message{Value: value})
	require.NoError(t, err)
	require.Equal(t, []string{artisanID}, tracker.trackedArtisanIDs)
}
