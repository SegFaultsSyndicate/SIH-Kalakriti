// services/core-svc/internal/core/service/pipeline_test.go
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
	"github.com/segfaultsyndicate/kalakriti/pkg/ids"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/domain"
)

// fakeInference is ml-svc: canned answers, a call counter, and a switch to make
// any one step fail.
type fakeInference struct {
	calls     map[string]int
	craftCode string
	failStep  string
	failWith  error
	// noteSeen is the last artisan_note handed to GenerateDescription, which is
	// how the do-not-translate masking is observed.
	noteSeen string
}

func newFakeInference(craftCode string) *fakeInference {
	return &fakeInference{calls: map[string]int{}, craftCode: craftCode}
}

func (f *fakeInference) fail(step string, err error) *fakeInference {
	f.failStep, f.failWith = step, err
	return f
}

func (f *fakeInference) EnhanceImage(_ context.Context, objectKey string) (string, error) {
	f.calls[stepEnhance]++
	if f.failStep == stepEnhance {
		return "", f.failWith
	}
	return "enhanced/" + objectKey, nil
}

func (f *fakeInference) ExtractAttributes(_ context.Context, _ []string, _, _ string) (domain.InferredAttributes, error) {
	f.calls[stepExtract]++
	if f.failStep == stepExtract {
		return domain.InferredAttributes{}, f.failWith
	}
	return domain.InferredAttributes{
		CraftCode:  f.craftCode,
		Material:   "cotton",
		Technique:  "hand-block-printing",
		Colours:    []string{"indigo"},
		Motifs:     []string{"buti"},
		Confidence: map[string]float32{"craft": 0.9, "material": 0.8},
	}, nil
}

func (f *fakeInference) GenerateDescription(_ context.Context, in domain.CopyRequest) (domain.GeneratedCopy, error) {
	f.calls[stepDescribe]++
	f.noteSeen = in.ArtisanNote
	if f.failStep == stepDescribe {
		return domain.GeneratedCopy{}, f.failWith
	}
	return domain.GeneratedCopy{
		Title:       "A stole in " + in.Language,
		Description: "Description in " + in.Language + ": " + in.ArtisanNote,
		Highlights:  []string{"Hand finished"},
	}, nil
}

// fakePipelineStore reads and writes the same maps the catalog and media fakes
// use, so the pipeline sees one consistent world.
type fakePipelineStore struct {
	catalog *fakeCatalogStore
	media   *fakeMediaStore
	artisan domain.Artisan
}

func (s *fakePipelineStore) GetArtisan(_ context.Context, id uuid.UUID) (domain.Artisan, error) {
	if id != s.artisan.ID {
		return domain.Artisan{}, fmt.Errorf("artisan not found: %w", pkgdomain.ErrNotFound)
	}
	return s.artisan, nil
}

func (s *fakePipelineStore) GetMedia(ctx context.Context, id uuid.UUID) (domain.Media, error) {
	return s.media.GetMedia(ctx, id)
}

// GetOrCreateProductForMedia models the unique index on product.source_media_id:
// the second caller for one media id gets the first caller's product.
func (s *fakePipelineStore) GetOrCreateProductForMedia(
	_ context.Context, in domain.CreateProductInput, mediaID uuid.UUID,
) (domain.Product, bool, error) {
	s.catalog.mu.Lock()
	defer s.catalog.mu.Unlock()

	for _, p := range s.catalog.products {
		if p.VoiceNoteMediaID != nil && *p.VoiceNoteMediaID == mediaID {
			return p, false, nil
		}
	}
	product := domain.Product{
		ID:               ids.New(),
		ArtisanID:        in.ArtisanID,
		CraftID:          in.CraftID,
		WorkingTitle:     in.WorkingTitle,
		VoiceNoteMediaID: &mediaID, // stands in for source_media_id in the fake
		CreatedBy:        in.CreatedBy,
	}
	s.catalog.products[product.ID] = product
	return product, true, nil
}

func (s *fakePipelineStore) GetListingByProduct(_ context.Context, productID uuid.UUID) (domain.Listing, error) {
	s.catalog.mu.Lock()
	defer s.catalog.mu.Unlock()
	for _, l := range s.catalog.listings {
		if l.ProductID == productID {
			return l, nil
		}
	}
	return domain.Listing{}, fmt.Errorf("listing not found: %w", pkgdomain.ErrNotFound)
}

func (s *fakePipelineStore) SetListingNeedsDescription(_ context.Context, listingID uuid.UUID, needs bool) error {
	s.catalog.mu.Lock()
	defer s.catalog.mu.Unlock()
	l, ok := s.catalog.listings[listingID]
	if !ok {
		return fmt.Errorf("listing not found: %w", pkgdomain.ErrNotFound)
	}
	l.NeedsDescription = needs
	s.catalog.listings[listingID] = l
	return nil
}

func (s *fakePipelineStore) ListListingTranslations(_ context.Context, listingID uuid.UUID) ([]domain.ListingTranslation, error) {
	s.catalog.mu.Lock()
	defer s.catalog.mu.Unlock()
	return append([]domain.ListingTranslation(nil), s.catalog.listings[listingID].Translations...), nil
}

type pipelineFixture struct {
	pipeline *Pipeline
	store    *fakePipelineStore
	catalog  *fakeCatalogStore
	media    *fakeMediaStore
	inferrer *fakeInference
	mediaID  uuid.UUID
	artisan  uuid.UUID
	craftID  uuid.UUID
}

func newPipelineFixture(t *testing.T, inferrer *fakeInference) pipelineFixture {
	t.Helper()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	f := pipelineFixture{
		catalog:  newFakeCatalogStore(),
		media:    newFakeMediaStore(),
		inferrer: inferrer,
		mediaID:  ids.New(),
		artisan:  ids.New(),
		craftID:  ids.New(),
	}

	crafts := fakeCraftIndex{crafts: map[uuid.UUID]domain.Craft{
		f.craftID: {
			ID: f.craftID, Code: "ajrakh-block-printing", DisplayName: "Ajrakh Block Printing",
			Techniques: []string{"hand-block-printing"}, Materials: []string{"cotton"},
		},
	}}

	bio := "I have printed ajrakh for twenty years"
	f.store = &fakePipelineStore{
		catalog: f.catalog,
		media:   f.media,
		artisan: domain.Artisan{
			ID: f.artisan, DisplayName: "Rahim", CraftIDs: []uuid.UUID{f.craftID},
			Languages: []string{"GUJARATI"}, Bio: &bio,
		},
	}
	f.media.media[f.mediaID] = domain.Media{
		ID: f.mediaID, ArtisanID: f.artisan, Kind: domain.MediaImage,
		ObjectKey: "artisans/x/y.jpg", State: domain.MediaUploaded, SizeBytes: 4096,
	}

	catalogSvc := NewCatalog(f.catalog, crafts, log)
	mediaSvc := NewMedia(f.media, newFakeObjectStore(), nil, "bucket", testMediaLimits(), log)
	f.pipeline = NewPipeline(f.store, catalogSvc, mediaSvc, crafts, inferrer,
		[]string{"ENGLISH", "HINDI"}, log)
	return f
}

func (f pipelineFixture) listing(t *testing.T) domain.Listing {
	t.Helper()
	require.Len(t, f.catalog.listings, 1)
	for _, l := range f.catalog.listings {
		return l
	}
	return domain.Listing{}
}

// TestPipelineDraftsAListing is the happy path end to end.
func TestPipelineDraftsAListing(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))

	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))

	listing := f.listing(t)
	require.Equal(t, domain.StatePendingArtisanApproval, listing.State)
	require.False(t, listing.NeedsDescription)
	require.Equal(t, f.craftID, f.catalog.products[listing.ProductID].CraftID)

	// The copy is in the artisan's own language, not the buyer's.
	require.Len(t, listing.Translations, 1)
	require.Equal(t, "GUJARATI", listing.Translations[0].Language)
	require.True(t, listing.Translations[0].MachineGenerated)

	// The model's attributes are stored as MODEL rows, so the artisan can
	// overrule any of them later.
	attributes := f.catalog.attributes[listing.ID]
	require.NotEmpty(t, attributes)
	for _, a := range attributes {
		require.Equal(t, domain.SourceModel, a.Source)
	}

	// The enhanced rendition is on the media row and media.enhanced was emitted.
	stored, err := f.media.GetMedia(context.Background(), f.mediaID)
	require.NoError(t, err)
	require.Equal(t, domain.MediaReady, stored.State)
	require.NotNil(t, stored.EnhancedObjectKey)
}

// TestPipelineReplayProducesOneListing is the first acceptance criterion.
func TestPipelineReplayProducesOneListing(t *testing.T) {
	t.Parallel()
	inferrer := newFakeInference("ajrakh-block-printing")
	f := newPipelineFixture(t, inferrer)

	for i := 0; i < 3; i++ {
		require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID), "delivery %d", i)
	}

	require.Len(t, f.catalog.products, 1)
	require.Len(t, f.catalog.listings, 1)
	require.Equal(t, domain.StatePendingArtisanApproval, f.listing(t).State)

	// Enhancement is the expensive call and it ran once: the second delivery saw
	// the enhanced key already on the row.
	require.Equal(t, 1, inferrer.calls[stepEnhance])
	require.Len(t, f.listing(t).Translations, 1)
}

// TestPipelineResumesAfterACrashMidChain is the second acceptance criterion: the
// process dies after the attributes are written and before the copy lands.
func TestPipelineResumesAfterACrashMidChain(t *testing.T) {
	t.Parallel()
	inferrer := newFakeInference("ajrakh-block-printing").
		fail(stepDescribe, errors.New("ml-svc connection reset"))
	f := newPipelineFixture(t, inferrer)

	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))
	crashed := f.listing(t)
	require.True(t, crashed.NeedsDescription)
	require.NotEmpty(t, f.catalog.attributes[crashed.ID])

	// The consumer restarts and the message is redelivered.
	inferrer.failStep = ""
	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))

	require.Len(t, f.catalog.products, 1)
	resumed := f.listing(t)
	require.Equal(t, crashed.ID, resumed.ID)
	require.False(t, resumed.NeedsDescription)
	require.Equal(t, domain.StatePendingArtisanApproval, resumed.State)
	require.Equal(t, 1, inferrer.calls[stepEnhance])
}

// TestPipelineFailsClosedOnAnUnknownCraft is the third acceptance criterion.
func TestPipelineFailsClosedOnAnUnknownCraft(t *testing.T) {
	t.Parallel()
	// The model answers with a craft this artisan does not practise.
	f := newPipelineFixture(t, newFakeInference("banarasi-brocade-weaving"))

	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))

	stored, err := f.media.GetMedia(context.Background(), f.mediaID)
	require.NoError(t, err)
	require.Equal(t, domain.MediaFailed, stored.State)
	require.NotNil(t, stored.FailureReason)
	require.Contains(t, *stored.FailureReason, "banarasi-brocade-weaving")

	require.Empty(t, f.catalog.listings)
	require.Empty(t, f.catalog.products)
}

func TestPipelineFailsClosedOnATerminalExtractionError(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing").
		fail(stepExtract, fmt.Errorf("the image is unreadable: %w", pkgdomain.ErrInvalidInput)))

	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))

	stored, _ := f.media.GetMedia(context.Background(), f.mediaID)
	require.Equal(t, domain.MediaFailed, stored.State)
	require.Empty(t, f.catalog.listings)
}

// A transient failure is returned so the consumer retries it, and nothing is
// marked failed on the way past.
func TestPipelineRetriesATransientFailure(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing").
		fail(stepExtract, errors.New("ml-svc unavailable")))

	err := f.pipeline.Run(context.Background(), f.mediaID)
	require.Error(t, err)
	require.Contains(t, err.Error(), stepExtract)

	stored, _ := f.media.GetMedia(context.Background(), f.mediaID)
	require.NotEqual(t, domain.MediaFailed, stored.State)
}

// TestPipelineRecordFailureIsWhatTheDeadLetterHookCalls covers the path from a
// message that ran out of retries to a reason the artisan can read.
func TestPipelineRecordFailureIsIdempotent(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))

	require.NoError(t, f.pipeline.RecordFailure(context.Background(), f.mediaID, "extract: gave up after 3 attempts"))
	require.NoError(t, f.pipeline.RecordFailure(context.Background(), f.mediaID, "extract: gave up after 3 attempts"))

	stored, _ := f.media.GetMedia(context.Background(), f.mediaID)
	require.Equal(t, domain.MediaFailed, stored.State)
	require.Contains(t, *stored.FailureReason, "3 attempts")
}

func TestPipelineIgnoresNonImageMedia(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))
	voice := f.media.media[f.mediaID]
	voice.Kind = domain.MediaAudio
	f.media.media[f.mediaID] = voice

	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))
	require.Empty(t, f.catalog.listings)
	require.Equal(t, 0, f.inferrer.calls[stepEnhance])
}

// --- translation ---------------------------------------------------------------

func TestTranslateFansOutToBuyerLanguages(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))
	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))

	listing := f.listing(t)
	require.NoError(t, f.pipeline.Translate(context.Background(), listing.ID, f.artisan, f.craftID))

	languages := map[string]bool{}
	for _, t := range f.listing(t).Translations {
		languages[t.Language] = true
	}
	require.True(t, languages["GUJARATI"], "the source copy survives")
	require.True(t, languages["ENGLISH"])
	require.True(t, languages["HINDI"])
}

func TestTranslateNeverOverwritesTheArtisansOwnWords(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))
	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))

	listing := f.listing(t)
	edited := "मैंने यह खुद लिखा है"
	editor := f.artisan.String()
	listing.Translations = append(listing.Translations, domain.ListingTranslation{
		ListingID: listing.ID, Language: "HINDI", Title: edited, Description: edited,
		MachineGenerated: false, EditedBy: &editor,
	})
	f.catalog.listings[listing.ID] = listing

	require.NoError(t, f.pipeline.Translate(context.Background(), listing.ID, f.artisan, f.craftID))

	for _, tr := range f.listing(t).Translations {
		if tr.Language == "HINDI" {
			require.Equal(t, edited, tr.Description)
			require.False(t, tr.MachineGenerated)
		}
	}
}

// TestTranslateProtectsCraftTerms is the do-not-translate rule: the ontology's
// own vocabulary is masked before the model sees it and restored verbatim after.
func TestTranslateProtectsCraftTerms(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))
	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))

	listing := f.listing(t)
	source := listing.Translations[0]
	source.Description = "A stole made by Ajrakh Block Printing on cotton."
	listing.Translations[0] = source
	f.catalog.listings[listing.ID] = listing

	require.NoError(t, f.pipeline.Translate(context.Background(), listing.ID, f.artisan, f.craftID))

	// The model never saw the craft name, only a token.
	require.NotContains(t, f.inferrer.noteSeen, "Ajrakh Block Printing")
	require.Contains(t, f.inferrer.noteSeen, "{{dnt")

	// And the reader gets it back spelled the way the ontology spells it.
	for _, tr := range f.listing(t).Translations {
		if tr.Language == "ENGLISH" {
			require.Contains(t, tr.Description, "Ajrakh Block Printing")
			require.NotContains(t, tr.Description, "{{dnt")
		}
	}
}

func TestDoNotTranslateMasksLongestTermsFirst(t *testing.T) {
	t.Parallel()
	craft := domain.Craft{
		Code: "ajrakh-block-printing", DisplayName: "Ajrakh Block Printing",
		Techniques: []string{"resist-dyeing"}, Materials: []string{"cotton"},
	}
	terms := doNotTranslate(craft)

	// "ajrakh block printing" must be masked before "cotton" cannot matter, but
	// before any shorter term that is a substring of it certainly does.
	require.Equal(t, "Ajrakh Block Printing", terms[0])

	text := "Ajrakh Block Printing on cotton, resist dyeing throughout"
	masked := mask(text, terms)
	require.NotContains(t, masked, "Ajrakh")
	require.NotContains(t, masked, "cotton")
	require.Equal(t, text, unmask(masked, terms))
}

func TestReplaceFoldIsCaseInsensitiveAndTotal(t *testing.T) {
	t.Parallel()
	require.Equal(t, "X and X", replaceFold("ajrakh and AJRAKH", "Ajrakh", "X"))
	require.Equal(t, "nothing", replaceFold("nothing", "absent", "X"))
	require.Equal(t, "unchanged", replaceFold("unchanged", "", "X"))
}
