// services/core-svc/internal/core/service/pipeline.go
package service

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// Inference is the slice of ml-svc the pipeline calls. Declared here so the
// pipeline tests need neither a gRPC connection nor a model.
type Inference interface {
	AssessImageQuality(ctx context.Context, objectKey string) (domain.ImageQualityVerdict, error)
	EnhanceImage(ctx context.Context, objectKey string) (enhancedKey string, err error)
	ExtractAttributes(ctx context.Context, objectKeys []string, declaredCraftID, hint string) (domain.InferredAttributes, error)
	GenerateDescription(ctx context.Context, in domain.CopyRequest) (domain.GeneratedCopy, error)
	Translate(ctx context.Context, in domain.TranslateRequest) (domain.TranslatedCopy, error)
}

// PipelineStore is what the pipeline needs beyond the catalog and media
// services: reads for a listing that already exists (created by the artisan's
// own wizard, or by a cluster officer) and whatever media has been attached to
// it -- the pipeline enriches that listing, it no longer creates one.
type PipelineStore interface {
	GetArtisan(ctx context.Context, id uuid.UUID) (domain.Artisan, error)
	GetMedia(ctx context.Context, id uuid.UUID) (domain.Media, error)
	GetListing(ctx context.Context, id uuid.UUID) (domain.Listing, error)
	GetProduct(ctx context.Context, id uuid.UUID) (domain.Product, error)
	ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]domain.ListingMedia, error)
	SetListingNeedsDescription(ctx context.Context, listingID uuid.UUID, needs bool) error
	ListListingTranslations(ctx context.Context, listingID uuid.UUID) ([]domain.ListingTranslation, error)
}

// stepTiming publishes total wall-clock and call count per pipeline step on the
// health server's /debug/vars.
//
// ponytail: expvar because it is stdlib and the Go services have no metrics
// stack yet; swap the two Add calls for prometheus histograms when one lands.
var stepTiming = expvar.NewMap("pipeline_steps")

// step names, which are also the metric keys.
const (
	stepAssessQuality = "assess_quality"
	stepEnhance       = "enhance"
	stepExtract       = "extract"
	stepDescribe      = "describe"
	stepTranslate     = "translate"
)

// Pipeline enriches a listing an artisan (or a cluster officer) already
// created, once photos are attached to it: enhances each photo, extracts
// attributes, and drafts a description in the artisan's language. It used to
// create its own listing straight from a raw upload -- one product per photo,
// unique on source_media_id -- which raced the artisan wizard's own
// listing-creation flow and silently produced a second, orphaned listing per
// photo (see WIRING_AUDIT_PLAN.md). It no longer creates products or listings;
// it only enriches whichever one the media was attached to.
//
// Each step is replayable rather than ledgered: enhancement is a guarded state
// change per photo, attributes and copy are upserts keyed on the listing. A
// redelivery therefore converges on the same result instead of duplicating
// anything, and a process killed mid-chain resumes by re-running the steps
// that left no trace.
type Pipeline struct {
	store    PipelineStore
	catalog  *Catalog
	media    *Media
	crafts   CraftIndex
	inferrer Inference
	log      *slog.Logger

	// buyerLanguages are translated into on approval, on top of the artisan's own.
	buyerLanguages []string
	// modelVersion labels an enhancement when ml-svc does not report its own.
	modelVersion string
}

// NewPipeline builds the cataloguing pipeline.
func NewPipeline(
	store PipelineStore,
	catalog *Catalog,
	media *Media,
	crafts CraftIndex,
	inferrer Inference,
	buyerLanguages []string,
	log *slog.Logger,
) *Pipeline {
	return &Pipeline{
		store: store, catalog: catalog, media: media, crafts: crafts,
		inferrer: inferrer, buyerLanguages: buyerLanguages,
		modelVersion: "pipeline", log: log,
	}
}

// asArtisan gives the pipeline the artisan's own authority. It drafts on their
// behalf, so it should be held to exactly the permissions they have — a service
// account that could write to any listing would be a bigger hole than the
// convenience is worth.
func asArtisan(ctx context.Context, artisanID uuid.UUID) context.Context {
	return auth.ContextWithPrincipal(ctx, auth.Principal{
		Subject: artisanID.String(),
		Role:    auth.RoleArtisan,
	})
}

// Run enriches one listing: enhances every attached photo, extracts
// attributes across all of them, and drafts a description in the artisan's
// language. Triggered once photos are attached to an existing listing (see
// AttachListingMedia), not by the raw upload -- a listing (and its product)
// already exists by the time this runs.
//
// Every step below degrades gracefully -- logged and skipped, never failing
// the whole run -- because the listing the artisan is building already exists
// and is valid regardless of whether AI enrichment succeeds; only a genuine
// store/db error propagates for the consumer's retry. The one exception is a
// single photo's enhancement failing terminally, which is recorded on that
// photo's own row (RecordFailure) so the artisan sees a reason to retake it,
// without blocking enrichment from the artisan's other photos.
func (p *Pipeline) Run(ctx context.Context, listingID uuid.UUID) error {
	listing, err := p.store.GetListing(ctx, listingID)
	if err != nil {
		return err
	}
	ctx = asArtisan(ctx, listing.ArtisanID)

	artisan, err := p.store.GetArtisan(ctx, listing.ArtisanID)
	if err != nil {
		return err
	}
	product, err := p.store.GetProduct(ctx, listing.ProductID)
	if err != nil {
		return err
	}
	craft, _ := p.crafts.Craft(product.CraftID)

	items, err := p.store.ListListingMedia(ctx, listingID)
	if err != nil {
		return err
	}

	// 1. Enhance every attached photo, and record each enhanced rendition on
	//    its own media row.
	enhancedKeys := make([]string, 0, len(items))
	for _, item := range items {
		if item.Kind != domain.MediaImage {
			continue
		}
		media, err := p.store.GetMedia(ctx, item.MediaID)
		if err != nil {
			return err
		}
		if media.State == domain.MediaFailed {
			continue // already given up on, and the artisan has been told why
		}
		if media.EnhancedObjectKey != nil {
			enhancedKeys = append(enhancedKeys, *media.EnhancedObjectKey) // a previous delivery got this far
			continue
		}

		enhancedKey, err := timed(ctx, stepEnhance, func() (string, error) {
			return p.inferrer.EnhanceImage(ctx, media.ObjectKey)
		})
		if err != nil {
			if !terminal(err) {
				return fmt.Errorf("%s: %w", stepEnhance, err) // let the consumer retry
			}
			if failErr := p.RecordFailure(ctx, media.ID, fmt.Sprintf("%s: %v", stepEnhance, err)); failErr != nil {
				return failErr
			}
			continue
		}
		if _, err := p.media.ApplyEnhancement(ctx, EnhancementResult{
			MediaID:           media.ID,
			EnhancedObjectKey: enhancedKey,
			ModelVersion:      &p.modelVersion,
			IdempotencyKey:    "pipeline:" + stepEnhance + ":" + media.ID.String(),
		}); err != nil {
			return err
		}
		enhancedKeys = append(enhancedKeys, enhancedKey)
	}
	if len(enhancedKeys) == 0 {
		return nil // nothing usable to enrich from yet
	}

	// 2. Extract attributes across every photo together. The craft is already
	//    the artisan's own choice from listing creation, not the model's
	//    guess -- a mismatch is logged, not fatal, since nothing here still
	//    decides which craft the listing belongs to. A transient failure (e.g.
	//    ml-svc unreachable) is returned for the consumer's retry, same as
	//    enhance; a terminal one (bad input) is logged and skipped, since
	//    there is no longer a single photo to blame it on.
	attributes, err := timed(ctx, stepExtract, func() (domain.InferredAttributes, error) {
		return p.inferrer.ExtractAttributes(ctx, enhancedKeys, craft.Code, deref(artisan.Bio))
	})
	if err != nil && !terminal(err) {
		return fmt.Errorf("%s: %w", stepExtract, err) // let the consumer retry
	}
	if err != nil {
		p.log.WarnContext(ctx, "attribute extraction failed, leaving the listing without model attributes",
			"listing_id", listingID, "error", err)
	} else {
		if attributes.CraftCode != "" && attributes.CraftCode != craft.Code {
			p.log.WarnContext(ctx, "model attributes disagree with the listing's declared craft",
				"listing_id", listingID, "declared", craft.Code, "model", attributes.CraftCode)
		}
		if _, err := p.catalog.UpsertListingAttributes(ctx, listingID,
			attributes.ToListingAttributes(), domain.SourceModel,
			"pipeline:"+stepExtract+":"+listingID.String()); err != nil {
			return err
		}
	}

	// 3. Copy. A failure here is compensated, not rolled back: the listing
	//    stays usable and is flagged for the artisan to write the description
	//    themselves.
	language := firstOr(artisan.Languages, "ENGLISH")
	generated, err := timed(ctx, stepDescribe, func() (domain.GeneratedCopy, error) {
		return p.inferrer.GenerateDescription(ctx, domain.CopyRequest{
			Attributes:  attributes,
			CraftID:     product.CraftID,
			CraftCode:   craft.Code,
			Language:    language,
			ArtisanNote: "",
		})
	})
	if err != nil {
		p.log.WarnContext(ctx, "description generation failed, leaving the draft to the artisan",
			"listing_id", listingID, "error", err)
		return p.store.SetListingNeedsDescription(ctx, listingID, true)
	}

	if _, err := p.catalog.UpsertListingTranslation(ctx, listingID, domain.ListingTranslation{
		Language:         language,
		Title:            generated.Title,
		Description:      generated.Description,
		Highlights:       generated.Highlights,
		MachineGenerated: true,
	}, "pipeline:"+stepDescribe+":"+listingID.String()); err != nil {
		return err
	}
	if err := p.store.SetListingNeedsDescription(ctx, listingID, false); err != nil {
		return err
	}

	p.log.InfoContext(ctx, "listing enriched",
		"listing_id", listingID, "photos", len(enhancedKeys), "language", language)
	return nil
}

// RecordFailure marks an asset terminally failed with the reason on the row. It
// is also what the consumer's dead-letter hook calls, so a message that ran out
// of retries leaves the same visible trail as one that failed outright.
func (p *Pipeline) RecordFailure(ctx context.Context, mediaID uuid.UUID, reason string) error {
	media, err := p.store.GetMedia(ctx, mediaID)
	if err != nil {
		return err
	}
	if media.State == domain.MediaFailed {
		return nil
	}
	if _, err := p.media.ApplyEnhancement(asArtisan(ctx, media.ArtisanID), EnhancementResult{
		MediaID:        mediaID,
		FailureReason:  &reason,
		IdempotencyKey: "pipeline:failed:" + mediaID.String(),
	}); err != nil {
		return err
	}
	p.log.ErrorContext(ctx, "pipeline failed", "media_id", mediaID, "reason", reason)
	return nil
}

// --- translation ---------------------------------------------------------------

// translateConcurrency bounds how many languages are in flight against ml-svc
// at once. 20 buyer languages fired sequentially turned "listing published" into
// a 20x-latency step once each call is a real IndicTrans2 request instead of the
// near-instant fake; this caps the fan-out instead of removing it.
//
// ponytail: fixed cap, not adaptive to ml-svc's actual capacity. Raise or make
// configurable if a future language-count bump makes this the bottleneck.
const translateConcurrency = 8

// Translate fans a published listing's copy out to the buyer languages. Craft
// terms are held out of the machine's reach and put back verbatim afterwards, so
// "Ajrakh" does not come back as "indigo cloth". Languages translate concurrently
// (bounded by translateConcurrency) since each is an independent ml-svc call and
// an independent stored row.
func (p *Pipeline) Translate(ctx context.Context, listingID, artisanID uuid.UUID, craftID uuid.UUID) error {
	existing, err := p.store.ListListingTranslations(ctx, listingID)
	if err != nil {
		return err
	}

	have := make(map[string]domain.ListingTranslation, len(existing))
	for _, t := range existing {
		have[t.Language] = t
	}
	source, ok := sourceCopy(existing)
	if !ok {
		return nil // nothing to translate from yet
	}

	craft, _ := p.crafts.Craft(craftID)
	protected := doNotTranslate(craft)
	ctx = asArtisan(ctx, artisanID)

	sem := make(chan struct{}, translateConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, language := range p.targetLanguages(source.Language) {
		// An artisan's own words are never overwritten by a machine.
		if prior, seen := have[language]; seen && !prior.MachineGenerated {
			continue
		}

		language := language
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			translated, err := timed(ctx, stepTranslate, func() (domain.TranslatedCopy, error) {
				return p.inferrer.Translate(ctx, domain.TranslateRequest{
					Title:          mask(source.Title, protected),
					Description:    mask(source.Description, protected),
					Highlights:     maskAll(source.Highlights, protected),
					SourceLanguage: source.Language,
					TargetLanguage: language,
				})
			})
			if err != nil {
				// One language failing must not cost the others.
				p.log.WarnContext(ctx, "translation failed",
					"listing_id", listingID, "language", language, "error", err)
				return
			}

			if _, err := p.catalog.UpsertListingTranslation(ctx, listingID, domain.ListingTranslation{
				Language:         language,
				Title:            unmask(translated.Title, protected),
				Description:      unmask(translated.Description, protected),
				Highlights:       unmaskAll(translated.Highlights, protected),
				MachineGenerated: true,
			}, "pipeline:"+stepTranslate+":"+listingID.String()+":"+language); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

// targetLanguages is the configured buyer languages, minus the one the copy is
// already written in. The artisan's own language is the source, so it is already
// covered.
func (p *Pipeline) targetLanguages(sourceLanguage string) []string {
	seen := map[string]struct{}{sourceLanguage: {}}
	out := make([]string, 0, len(p.buyerLanguages))
	for _, language := range p.buyerLanguages {
		if _, dup := seen[language]; dup || language == "" {
			continue
		}
		seen[language] = struct{}{}
		out = append(out, language)
	}
	return out
}

// --- do-not-translate ----------------------------------------------------------

// dntToken is what a protected term is swapped for while the model works. The
// braces survive every NMT engine tested and no Indic script uses them.
const dntToken = "{{dnt%d}}"

// doNotTranslate is the craft's own vocabulary: its display name, its slug read
// as words, and its techniques and materials. These are proper nouns of the
// trade and a translation of them is always wrong.
func doNotTranslate(craft domain.Craft) []string {
	terms := []string{craft.DisplayName, strings.ReplaceAll(craft.Code, "-", " ")}
	terms = append(terms, craft.Techniques...)
	terms = append(terms, craft.Materials...)

	seen := map[string]struct{}{}
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(strings.ReplaceAll(term, "-", " "))
		if term == "" {
			continue
		}
		if _, dup := seen[strings.ToLower(term)]; dup {
			continue
		}
		seen[strings.ToLower(term)] = struct{}{}
		out = append(out, term)
	}
	// Longest first, so "ajrakh block printing" is masked before "ajrakh".
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && len(out[j]) > len(out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// mask replaces every protected term with its token, case-insensitively.
func mask(text string, terms []string) string {
	for i, term := range terms {
		text = replaceFold(text, term, fmt.Sprintf(dntToken, i))
	}
	return text
}

// unmask puts the protected terms back exactly as the ontology spells them.
func unmask(text string, terms []string) string {
	for i, term := range terms {
		text = strings.ReplaceAll(text, fmt.Sprintf(dntToken, i), term)
	}
	return text
}

func maskAll(texts []string, terms []string) []string {
	out := make([]string, 0, len(texts))
	for _, text := range texts {
		out = append(out, mask(text, terms))
	}
	return out
}

func unmaskAll(texts []string, terms []string) []string {
	out := make([]string, 0, len(texts))
	for _, text := range texts {
		out = append(out, unmask(text, terms))
	}
	return out
}

// replaceFold is a case-insensitive ReplaceAll. strings has no such function and
// a regexp would need every term quoted, so this walks the string once per term.
func replaceFold(haystack, needle, replacement string) string {
	if needle == "" {
		return haystack
	}
	var b strings.Builder
	lowerHay, lowerNeedle := strings.ToLower(haystack), strings.ToLower(needle)
	for {
		i := strings.Index(lowerHay, lowerNeedle)
		if i < 0 {
			b.WriteString(haystack)
			return b.String()
		}
		b.WriteString(haystack[:i])
		b.WriteString(replacement)
		haystack, lowerHay = haystack[i+len(needle):], lowerHay[i+len(needle):]
	}
}

// --- helpers -------------------------------------------------------------------

// timed runs one step and records its wall clock against the step's name.
func timed[T any](_ context.Context, step string, fn func() (T, error)) (T, error) {
	started := time.Now()
	out, err := fn()
	stepTiming.Add(step+".calls", 1)
	stepTiming.Add(step+".ms", time.Since(started).Milliseconds())
	if err != nil {
		stepTiming.Add(step+".errors", 1)
	}
	return out, err
}

// terminal reports whether retrying could plausibly help. A bad request or a
// craft outside the allowlist will fail identically three more times.
func terminal(err error) bool {
	return errors.Is(err, pkgdomain.ErrInvalidInput) ||
		errors.Is(err, pkgdomain.ErrForbidden) ||
		errors.Is(err, pkgdomain.ErrNotFound)
}

func firstOr(values []string, fallback string) string {
	if len(values) == 0 || values[0] == "" {
		return fallback
	}
	return values[0]
}

// sourceCopy is the translation source: the artisan's own edit if there is one,
// and otherwise whatever the pipeline generated.
func sourceCopy(translations []domain.ListingTranslation) (domain.ListingTranslation, bool) {
	for _, t := range translations {
		if !t.MachineGenerated {
			return t, true
		}
	}
	if len(translations) > 0 {
		return translations[0], true
	}
	return domain.ListingTranslation{}, false
}

// deref reads an optional string, which the artisan's bio is.
func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// issueSummary joins a failed quality verdict's issues into one artisan-facing
// sentence, e.g. "BLURRY: image is too blurred to be usable (sharpness score 42.1)".
func issueSummary(issues []domain.ImageQualityIssue) string {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Code, issue.Message))
	}
	return strings.Join(parts, "; ")
}
