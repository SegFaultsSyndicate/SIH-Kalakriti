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

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// fakeInference is ml-svc: canned answers, a call counter, and a switch to make
// any one step fail.
type fakeInference struct {
	calls          map[string]int
	craftCode      string
	failStep       string
	failWith       error
	qualityVerdict domain.ImageQualityVerdict
	// noteSeen is the last artisan_note handed to GenerateDescription, which is
	// how the do-not-translate masking is observed.
	noteSeen string
}

func newFakeInference(craftCode string) *fakeInference {
	return &fakeInference{calls: map[string]int{}, craftCode: craftCode, qualityVerdict: domain.ImageQualityVerdict{Passed: true}}
}

func (f *fakeInference) fail(step string, err error) *fakeInference {
	f.failStep, f.failWith = step, err
	return f
}

// failQuality makes AssessImageQuality return a normal (non-error) rejected
// verdict, as opposed to fail(stepAssessQuality, ...) which simulates a
// transport-level error.
func (f *fakeInference) failQuality(issues ...domain.ImageQualityIssue) *fakeInference {
	f.qualityVerdict = domain.ImageQualityVerdict{Passed: false, Issues: issues}
	return f
}

func (f *fakeInference) AssessImageQuality(_ context.Context, _ string) (domain.ImageQualityVerdict, error) {
	f.calls[stepAssessQuality]++
	if f.failStep == stepAssessQuality {
		return domain.ImageQualityVerdict{}, f.failWith
	}
	return f.qualityVerdict, nil
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

func (f *fakeInference) Translate(_ context.Context, in domain.TranslateRequest) (domain.TranslatedCopy, error) {
	f.calls[stepTranslate]++
	f.noteSeen = in.Description
	if f.failStep == stepTranslate {
		return domain.TranslatedCopy{}, f.failWith
	}
	return domain.TranslatedCopy{
		Title:       in.TargetLanguage + ": " + in.Title,
		Description: in.TargetLanguage + ": " + in.Description,
		Highlights:  in.Highlights,
	}, nil
}

// fakePipelineStore reads and writes the same maps the catalog and media fakes
// use, so the pipeline sees one consistent world. The listing/product/media-
// attachment reads delegate straight to the catalog fake, which already
// implements them for the service layer's own tests.
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

func (s *fakePipelineStore) GetListing(ctx context.Context, id uuid.UUID) (domain.Listing, error) {
	return s.catalog.GetListing(ctx, id)
}

func (s *fakePipelineStore) GetProduct(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return s.catalog.GetProduct(ctx, id)
}

func (s *fakePipelineStore) ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]domain.ListingMedia, error) {
	return s.catalog.ListListingMedia(ctx, listingID)
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
	pipeline  *Pipeline
	store     *fakePipelineStore
	catalog   *fakeCatalogStore
	media     *fakeMediaStore
	inferrer  *fakeInference
	mediaID   uuid.UUID
	artisan   uuid.UUID
	craftID   uuid.UUID
	productID uuid.UUID
	listingID uuid.UUID
}

// newPipelineFixture builds a product and a DRAFT listing exactly as the
// artisan wizard would (via CreateProduct + UpsertListing), with one photo
// already attached to the listing -- the state AttachListingMedia leaves
// behind and what triggers Pipeline.Run in production.
func newPipelineFixture(t *testing.T, inferrer *fakeInference) pipelineFixture {
	t.Helper()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	f := pipelineFixture{
		catalog:   newFakeCatalogStore(),
		media:     newFakeMediaStore(),
		inferrer:  inferrer,
		mediaID:   ids.New(),
		artisan:   ids.New(),
		craftID:   ids.New(),
		productID: ids.New(),
		listingID: ids.New(),
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

	// The wizard's own listing-creation flow: a product, and a DRAFT listing
	// offering it.
	f.catalog.products[f.productID] = domain.Product{
		ID: f.productID, ArtisanID: f.artisan, CraftID: f.craftID, WorkingTitle: "Ajrakh stole",
	}
	f.catalog.listings[f.listingID] = domain.Listing{
		ID: f.listingID, ProductID: f.productID, ArtisanID: f.artisan,
		Type: domain.ListingMadeToOrder, State: domain.StateDraft, MinOrderQuantity: 1,
	}
	// AttachListingMedia's effect: the photo is now on the listing.
	f.catalog.medias[f.listingID] = []domain.ListingMedia{
		{ListingID: f.listingID, MediaID: f.mediaID, Ordinal: 0, Role: domain.MediaRolePrimaryImage, Kind: domain.MediaImage},
	}

	catalogSvc := NewCatalog(f.catalog, crafts, log)
	mediaSvc := NewMedia(f.media, newFakeObjectStore(), nil, "bucket", testMediaLimits(), log)
	f.pipeline = NewPipeline(f.store, catalogSvc, mediaSvc, crafts, inferrer,
		[]string{"ENGLISH", "HINDI"}, log)
	return f
}

func (f pipelineFixture) listing(t *testing.T) domain.Listing {
	t.Helper()
	l, ok := f.catalog.listings[f.listingID]
	require.True(t, ok)
	return l
}

// TestPipelineEnrichesTheListing is the happy path end to end.
func TestPipelineEnrichesTheListing(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))

	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))

	listing := f.listing(t)
	require.False(t, listing.NeedsDescription)

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

// TestPipelineReplayConvergesOnTheSameListing is the first acceptance
// criterion, restated for a listing that already exists: a redelivery must
// not create anything new.
func TestPipelineReplayConvergesOnTheSameListing(t *testing.T) {
	t.Parallel()
	inferrer := newFakeInference("ajrakh-block-printing")
	f := newPipelineFixture(t, inferrer)

	for i := 0; i < 3; i++ {
		require.NoError(t, f.pipeline.Run(context.Background(), f.listingID), "delivery %d", i)
	}

	require.Len(t, f.catalog.products, 1)
	require.Len(t, f.catalog.listings, 1)

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

	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))
	crashed := f.listing(t)
	require.True(t, crashed.NeedsDescription)
	require.NotEmpty(t, f.catalog.attributes[crashed.ID])

	// The consumer restarts and the message is redelivered.
	inferrer.failStep = ""
	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))

	require.Len(t, f.catalog.products, 1)
	resumed := f.listing(t)
	require.Equal(t, crashed.ID, resumed.ID)
	require.False(t, resumed.NeedsDescription)
	require.Equal(t, 1, inferrer.calls[stepEnhance])
}

// TestPipelineToleratesACraftMismatch: the model no longer decides which craft
// a listing belongs to -- that was fixed by the artisan at listing creation.
// A model craft outside what it was told is logged, not fatal.
func TestPipelineToleratesACraftMismatch(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("banarasi-brocade-weaving"))

	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))

	stored, err := f.media.GetMedia(context.Background(), f.mediaID)
	require.NoError(t, err)
	require.Equal(t, domain.MediaReady, stored.State)
	require.Nil(t, stored.FailureReason)

	// Enrichment still happened -- attributes and copy stored as usual.
	require.NotEmpty(t, f.catalog.attributes[f.listingID])
	require.NotEmpty(t, f.listing(t).Translations)
}

// TestPipelineSkipsAttributesOnATerminalExtractionError: a terminal extraction
// error (bad input) is logged and skipped -- there is no longer a single photo
// to blame it on, and the listing itself is already valid regardless.
func TestPipelineSkipsAttributesOnATerminalExtractionError(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing").
		fail(stepExtract, fmt.Errorf("the image is unreadable: %w", pkgdomain.ErrInvalidInput)))

	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))

	require.Empty(t, f.catalog.attributes[f.listingID])
	// Describe still ran, off the zero-value attributes.
	require.NotEmpty(t, f.listing(t).Translations)
}

// A transient extraction failure is returned so the consumer retries it.
func TestPipelineRetriesATransientExtractionFailure(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing").
		fail(stepExtract, errors.New("ml-svc unavailable")))

	err := f.pipeline.Run(context.Background(), f.listingID)
	require.Error(t, err)
	require.Contains(t, err.Error(), stepExtract)

	require.Empty(t, f.catalog.attributes[f.listingID])
}

// A terminal enhancement failure is recorded on that one photo's row and does
// not stop the run (it just has nothing left to enrich from, here, since it's
// the only photo).
func TestPipelineRecordsATerminalEnhancementFailureOnThePhoto(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing").
		fail(stepEnhance, fmt.Errorf("corrupt image: %w", pkgdomain.ErrInvalidInput)))

	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))

	stored, err := f.media.GetMedia(context.Background(), f.mediaID)
	require.NoError(t, err)
	require.Equal(t, domain.MediaFailed, stored.State)
	require.NotNil(t, stored.FailureReason)

	// Nothing left to enrich from -- extract/describe never ran.
	require.Empty(t, f.catalog.attributes[f.listingID])
	require.Equal(t, 0, f.inferrer.calls[stepExtract])
}

<<<<<<< Updated upstream
// A transient enhancement failure is returned so the consumer retries it, and
// nothing is marked failed on the way past.
func TestPipelineRetriesATransientEnhancementFailure(t *testing.T) {
=======
// TestPipelineFailsClosedOnPoorImageQuality is the quality gate: a genuine
// quality failure (blur, here) fails closed before EnhanceImage or
// ExtractAttributes ever run.
func TestPipelineFailsClosedOnPoorImageQuality(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing").
		failQuality(domain.ImageQualityIssue{Code: "BLURRY", Message: "image is too blurred to be usable"}))

	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))

	stored, err := f.media.GetMedia(context.Background(), f.mediaID)
	require.NoError(t, err)
	require.Equal(t, domain.MediaFailed, stored.State)
	require.NotNil(t, stored.FailureReason)
	require.Contains(t, *stored.FailureReason, "BLURRY")
	require.Contains(t, *stored.FailureReason, "too blurred")

	require.Empty(t, f.catalog.listings)
	require.Empty(t, f.catalog.products)
	require.Equal(t, 0, f.inferrer.calls[stepEnhance], "a rejected image must never reach EnhanceImage")
	require.Equal(t, 0, f.inferrer.calls[stepExtract], "a rejected image must never reach ExtractAttributes")
}

// TestPipelineOnlyRejectsGenuineQualityFailures confirms a passing quality
// verdict (the only kind mock/real ml-svc ever returns for a subject outside
// the craft ontology) never blocks the pipeline -- recognition failure is
// handled downstream by ExtractAttributes abstaining, not by this gate.
func TestPipelineOnlyRejectsGenuineQualityFailures(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))

	require.NoError(t, f.pipeline.Run(context.Background(), f.mediaID))

	stored, err := f.media.GetMedia(context.Background(), f.mediaID)
	require.NoError(t, err)
	require.NotEqual(t, domain.MediaFailed, stored.State)
	require.Equal(t, 1, f.inferrer.calls[stepAssessQuality])
}

func TestPipelineFailsClosedOnATerminalExtractionError(t *testing.T) {
>>>>>>> Stashed changes
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing").
		fail(stepEnhance, errors.New("ml-svc unavailable")))

	err := f.pipeline.Run(context.Background(), f.listingID)
	require.Error(t, err)
	require.Contains(t, err.Error(), stepEnhance)

	stored, _ := f.media.GetMedia(context.Background(), f.mediaID)
	require.NotEqual(t, domain.MediaFailed, stored.State)
}

// TestPipelineRecordFailureIsIdempotent covers the path from a photo whose
// enhancement ran out of retries to a reason the artisan can read.
func TestPipelineRecordFailureIsIdempotent(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))

	require.NoError(t, f.pipeline.RecordFailure(context.Background(), f.mediaID, "enhance: gave up after 3 attempts"))
	require.NoError(t, f.pipeline.RecordFailure(context.Background(), f.mediaID, "enhance: gave up after 3 attempts"))

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
	f.catalog.medias[f.listingID][0].Kind = domain.MediaAudio

	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))
	require.Empty(t, f.catalog.attributes[f.listingID])
	require.Equal(t, 0, f.inferrer.calls[stepEnhance])
}

// --- translation ---------------------------------------------------------------

func TestTranslateFansOutToBuyerLanguages(t *testing.T) {
	t.Parallel()
	f := newPipelineFixture(t, newFakeInference("ajrakh-block-printing"))
	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))

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
	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))

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
	require.NoError(t, f.pipeline.Run(context.Background(), f.listingID))

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
