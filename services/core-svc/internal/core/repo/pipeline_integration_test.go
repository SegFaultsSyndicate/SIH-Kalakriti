// services/core-svc/internal/core/repo/pipeline_integration_test.go

//go:build integration

// The cataloguing pipeline against a real PostgreSQL, with a fake ml-svc.
//
//	go test -tags=integration ./internal/core/repo/...
package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/segfaultsyndicate/kalakriti/pkg/ids"
	"github.com/segfaultsyndicate/kalakriti/pkg/topics"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/domain"
)

// seedPipelineFixture puts an artisan, a craft and one uploaded photograph in the
// database, which is the state a real media.uploaded event arrives in.
func seedPipelineFixture(ctx context.Context, t *testing.T, repo *Repo) (artisanID, craftID, mediaID uuid.UUID) {
	t.Helper()

	craftID, err := repo.UpsertCraft(ctx, domain.Craft{
		Code: "ajrakh-block-printing", DisplayName: "Ajrakh Block Printing",
		Techniques: []string{"hand-block-printing"}, Materials: []string{"cotton"},
	})
	require.NoError(t, err)

	artisanID = ids.New()
	bio := "twenty years of ajrakh"
	err = repo.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		_, err := tx.CreateArtisan(ctx, artisanID, domain.RegisterArtisanInput{
			DisplayName: "Rahim", PhoneE164: "+919876500011",
			CraftIDs: []uuid.UUID{craftID}, Languages: []string{"GUJARATI"},
			Region: domain.Region{StateCode: "IN-GJ"}, Bio: &bio, CreatedBy: "test",
		})
		return err
	})
	require.NoError(t, err)

	mediaID = ids.New()
	err = repo.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		_, err := tx.CreateMediaPending(ctx, domain.Media{
			ID: mediaID, ArtisanID: artisanID, Bucket: "kalakriti-media",
			ObjectKey: "artisans/" + artisanID.String() + "/photo.jpg",
			Kind:      domain.MediaImage, MimeType: "image/jpeg", SizeBytes: 4096,
			Source: "artisan-app",
		})
		if err != nil {
			return err
		}
		_, err = tx.TransitionMediaState(ctx, mediaID, domain.MediaPending, domain.MediaUploaded,
			domain.MediaTransition{})
		return err
	})
	require.NoError(t, err)

	return artisanID, craftID, mediaID
}

// countOutbox returns how many rows one topic has, which is how the test asserts
// "one event per topic" without a Kafka broker: the outbox is the event
// boundary, and the relay round trip is covered by TestRepoOutboxRelayRoundTrip.
func countOutbox(ctx context.Context, t *testing.T, repo *Repo, topic string) int {
	t.Helper()
	var count int
	err := repo.Pool().QueryRow(ctx, `SELECT count(*) FROM outbox WHERE topic = $1`, topic).Scan(&count)
	require.NoError(t, err)
	return count
}

// TestPipelineProductIsUniquePerMedia is the constraint the whole pipeline's
// idempotency rests on: two deliveries, one product.
func TestPipelineProductIsUniquePerMedia(t *testing.T) {
	ctx := context.Background()
	repo := New(startPostgres(ctx, t))
	artisanID, craftID, mediaID := seedPipelineFixture(ctx, t, repo)

	input := domain.CreateProductInput{
		ArtisanID: artisanID, CraftID: craftID,
		WorkingTitle: "indigo cotton ajrakh", CreatedBy: "pipeline",
	}

	var first, second domain.Product
	var createdFirst, createdSecond bool
	require.NoError(t, repo.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		var err error
		first, createdFirst, err = tx.GetOrCreateProductForMedia(ctx, input, mediaID)
		return err
	}))
	require.NoError(t, repo.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		var err error
		second, createdSecond, err = tx.GetOrCreateProductForMedia(ctx, input, mediaID)
		return err
	}))

	require.True(t, createdFirst)
	require.False(t, createdSecond, "the second delivery must not create a product")
	require.Equal(t, first.ID, second.ID)

	var products int
	require.NoError(t, repo.Pool().QueryRow(ctx,
		`SELECT count(*) FROM product WHERE source_media_id = $1`, mediaID).Scan(&products))
	require.Equal(t, 1, products)
}

// TestPipelineDraftAndRedeliveryEmitsOneEventPerTopic walks the writes the
// pipeline makes, twice, and asserts nothing doubles.
func TestPipelineDraftAndRedeliveryEmitsOneEventPerTopic(t *testing.T) {
	ctx := context.Background()
	repo := New(startPostgres(ctx, t))
	artisanID, craftID, mediaID := seedPipelineFixture(ctx, t, repo)

	draft := func() domain.Listing {
		var listing domain.Listing
		require.NoError(t, repo.InTx(ctx, func(ctx context.Context, tx *Tx) error {
			product, _, err := tx.GetOrCreateProductForMedia(ctx, domain.CreateProductInput{
				ArtisanID: artisanID, CraftID: craftID,
				WorkingTitle: "indigo cotton ajrakh", CreatedBy: "pipeline",
			}, mediaID)
			if err != nil {
				return err
			}

			// Enhancement, exactly as the media service applies it.
			enhanced := "enhanced/" + mediaID.String() + ".webp"
			if _, err := tx.TransitionMediaState(ctx, mediaID, domain.MediaUploaded,
				domain.MediaProcessing, domain.MediaTransition{}); err == nil {
				if _, err := tx.SetMediaEnhanced(ctx, mediaID, domain.MediaProcessing, enhanced, nil); err != nil {
					return err
				}
				if err := outboxOnce(ctx, tx, mediaID, topics.MediaEnhanced); err != nil {
					return err
				}
			}

			existing, err := repo.GetListingByProduct(ctx, product.ID)
			if err == nil {
				listing = existing
				return nil
			}

			leadTime, capacity := int32(21), int32(4)
			listing, err = tx.CreateListing(ctx, ids.New(), artisanID, domain.UpsertListingInput{
				ProductID: product.ID, Type: domain.ListingMadeToOrder, PricePaise: 0,
				MinOrderQuantity: 1, LeadTimeDays: &leadTime, CapacityPerMonth: &capacity,
				CreatedBy: "pipeline",
			})
			if err != nil {
				return err
			}
			return outboxOnce(ctx, tx, listing.ID, topics.CatalogListingDrafted)
		}))
		return listing
	}

	first := draft()
	second := draft()
	require.Equal(t, first.ID, second.ID, "a redelivery must reuse the listing")

	// Attributes are upserts, so writing them twice leaves one row per value.
	for i := 0; i < 2; i++ {
		require.NoError(t, repo.InTx(ctx, func(ctx context.Context, tx *Tx) error {
			for _, a := range (domain.InferredAttributes{
				CraftCode: "ajrakh-block-printing", Material: "cotton",
				Technique: "hand-block-printing", Colours: []string{"indigo"},
			}).ToListingAttributes() {
				a.ID, a.ListingID, a.Source = ids.New(), first.ID, domain.SourceModel
				if err := tx.UpsertListingAttribute(ctx, a); err != nil {
					return err
				}
			}
			return nil
		}))
	}

	attributes, err := repo.GetListingDetail(ctx, first.ID)
	require.NoError(t, err)
	require.Len(t, attributes.Attributes, 4, "one row per attribute value, not two")

	var listings int
	require.NoError(t, repo.Pool().QueryRow(ctx, `SELECT count(*) FROM listing`).Scan(&listings))
	require.Equal(t, 1, listings)

	require.Equal(t, 1, countOutbox(ctx, t, repo, topics.MediaEnhanced))
	require.Equal(t, 1, countOutbox(ctx, t, repo, topics.CatalogListingDrafted))
}

// TestPipelineFailureIsRecordedOnTheMediaRow is the third acceptance criterion's
// database half: a failed run leaves a reason and no listing.
func TestPipelineFailureIsRecordedOnTheMediaRow(t *testing.T) {
	ctx := context.Background()
	repo := New(startPostgres(ctx, t))
	_, _, mediaID := seedPipelineFixture(ctx, t, repo)

	reason := "extract: the model returned a craft this artisan does not practise"
	require.NoError(t, repo.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		_, err := tx.TransitionMediaState(ctx, mediaID, domain.MediaUploaded, domain.MediaFailed,
			domain.MediaTransition{FailureReason: &reason})
		return err
	}))

	media, err := repo.GetMedia(ctx, mediaID)
	require.NoError(t, err)
	require.Equal(t, domain.MediaFailed, media.State)
	require.NotNil(t, media.FailureReason)
	require.Contains(t, *media.FailureReason, "does not practise")

	var listings int
	require.NoError(t, repo.Pool().QueryRow(ctx, `SELECT count(*) FROM listing`).Scan(&listings))
	require.Zero(t, listings)
}

// outboxOnce enqueues one event keyed by its aggregate, which is what makes a
// redelivered step a no-op at the event boundary as well as in the tables.
func outboxOnce(ctx context.Context, tx *Tx, aggregateID uuid.UUID, topic string) error {
	return tx.InsertOutbox(ctx, ids.New().String(), aggregateID.String(), topic,
		"pipeline:"+topic+":"+aggregateID.String(), []byte(`{"header":{},"payload":{}}`))
}
