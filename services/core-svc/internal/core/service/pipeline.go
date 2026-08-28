// services/core-svc/internal/core/service/pipeline.go
package service

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/segfaultsyndicate/kalakriti/pkg/auth"
	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/domain"
)

// Inference is the slice of ml-svc the pipeline calls. Declared here so the
// pipeline tests need neither a gRPC connection nor a model.
type Inference interface {
	EnhanceImage(ctx context.Context, objectKey string) (enhancedKey string, err error)
	ExtractAttributes(ctx context.Context, objectKeys []string, declaredCraftID, hint string) (domain.InferredAttributes, error)
	GenerateDescription(ctx context.Context, in domain.CopyRequest) (domain.GeneratedCopy, error)
}

// PipelineStore is what the pipeline needs beyond the catalog and media
// services: the two insert-if-absent reads that make its steps replayable, and
// the artisan's crafts, which are the allowlist the extractor is held to.
type PipelineStore interface {
	GetArtisan(ctx context.Context, id uuid.UUID) (domain.Artisan, error)
	GetMedia(ctx context.Context, id uuid.UUID) (domain.Media, error)
	GetOrCreateProductForMedia(ctx context.Context, in domain.CreateProductInput, mediaID uuid.UUID) (domain.Product, bool, error)
	GetListingByProduct(ctx context.Context, productID uuid.UUID) (domain.Listing, error)
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
	stepEnhance   = "enhance"
	stepExtract   = "extract"
	stepDescribe  = "describe"
	stepDraft     = "draft"
	stepTranslate = "translate"
)

// Pipeline turns an uploaded photograph into a listing waiting for its artisan.
//
// Each step is replayable rather than ledgered: enhancement is a guarded state
// change, the product insert is unique on its source media, attributes and copy
// are upserts. A redelivery therefore converges on the same listing instead of
// building a second one, and a process killed mid-chain resumes by re-running
// the steps that left no trace.
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

// Run drives the whole chain for one uploaded asset.
//
// The returned error is for the consumer's retry: a transient failure comes back
// so the message is redelivered, and a terminal one is recorded on the media row
// and swallowed, because retrying it three more times would only delay the
// artisan being told.
func (p *Pipeline) Run(ctx context.Context, mediaID uuid.UUID) error {
	media, err := p.store.GetMedia(ctx, mediaID)
	if err != nil {
		return err
	}
	if media.Kind != domain.MediaImage {
		// Video and audio have their own paths; nothing to catalogue here.
		return nil
	}
	if media.State == domain.MediaFailed {
		return nil // already given up on, and the artisan has been told why
	}

	ctx = asArtisan(ctx, media.ArtisanID)

	artisan, err := p.store.GetArtisan(ctx, media.ArtisanID)
	if err != nil {
		return err
	}

	// 1. Enhance, and record the enhanced rendition on the media row. The media
	//    service emits media.enhanced from inside that transaction.
	enhancedKey := ""
	if media.EnhancedObjectKey != nil {
		enhancedKey = *media.EnhancedObjectKey // a previous delivery got this far
	} else {
		enhancedKey, err = timed(ctx, stepEnhance, func() (string, error) {
			return p.inferrer.EnhanceImage(ctx, media.ObjectKey)
		})
		if err != nil {
			return p.fail(ctx, media, stepEnhance, err)
		}
		if media, err = p.media.ApplyEnhancement(ctx, EnhancementResult{
			MediaID:           mediaID,
			EnhancedObjectKey: enhancedKey,
			ModelVersion:      &p.modelVersion,
			IdempotencyKey:    "pipeline:" + stepEnhance + ":" + mediaID.String(),
		}); err != nil {
			return err
		}
	}

	// 2. Extract, constrained to the crafts this artisan actually practises.
	//    Anything outside that list is the model guessing, and a wrong craft
	//    poisons search and provenance alike.
	allowlist := p.craftCodes(artisan.CraftIDs)
	attributes, err := timed(ctx, stepExtract, func() (domain.InferredAttributes, error) {
		return p.inferrer.ExtractAttributes(ctx, []string{enhancedKey}, allowlist.declared, deref(artisan.Bio))
	})
	if err != nil {
		return p.fail(ctx, media, stepExtract, err)
	}
	craftID, ok := allowlist.resolve(attributes.CraftCode)
	if !ok {
		return p.fail(ctx, media, stepExtract, fmt.Errorf(
			"the model returned craft %q, which is not one of this artisan's crafts: %w",
			attributes.CraftCode, pkgdomain.ErrInvalidInput))
	}

	// 3. Product and listing, created once per uploaded photograph.
	product, created, err := p.store.GetOrCreateProductForMedia(ctx, domain.CreateProductInput{
		ArtisanID:    media.ArtisanID,
		CraftID:      craftID,
		WorkingTitle: workingTitle(attributes),
		Materials:    nonEmpty(attributes.Material),
		Techniques:   nonEmpty(attributes.Technique),
		Colours:      attributes.Colours,
		Motifs:       attributes.Motifs,
		MediaIDs:     []uuid.UUID{mediaID},
		CreatedBy:    "pipeline",
	}, mediaID)
	if err != nil {
		return err
	}
	p.log.InfoContext(ctx, "pipeline product",
		"media_id", mediaID, "product_id", product.ID, "created", created)

	listing, err := p.draftListing(ctx, product, attributes)
	if err != nil {
		return err
	}

	// The attributes are written before the copy is attempted, so a failure in
	// step 4 leaves the artisan a listing that already knows what it is.
	if _, err := p.catalog.UpsertListingAttributes(ctx, listing.ID,
		attributes.ToListingAttributes(), domain.SourceModel,
		"pipeline:"+stepExtract+":"+mediaID.String()); err != nil {
		return err
	}

	// 4. Copy. A failure here is compensated, not rolled back: everything above
	//    is worth keeping, so the listing stays a DRAFT flagged for the artisan
	//    to write the description themselves.
	language := firstOr(artisan.Languages, "ENGLISH")
	generated, err := timed(ctx, stepDescribe, func() (domain.GeneratedCopy, error) {
		return p.inferrer.GenerateDescription(ctx, domain.CopyRequest{
			Attributes:  attributes,
			CraftID:     craftID,
			CraftCode:   attributes.CraftCode,
			Language:    language,
			ArtisanNote: "",
		})
	})
	if err != nil {
		p.log.WarnContext(ctx, "description generation failed, leaving the draft to the artisan",
			"listing_id", listing.ID, "error", err)
		return p.store.SetListingNeedsDescription(ctx, listing.ID, true)
	}

	if _, err := p.catalog.UpsertListingTranslation(ctx, listing.ID, domain.ListingTranslation{
		Language:         language,
		Title:            generated.Title,
		Description:      generated.Description,
		Highlights:       generated.Highlights,
		MachineGenerated: true,
	}, "pipeline:"+stepDescribe+":"+mediaID.String()); err != nil {
		return err
	}
	if err := p.store.SetListingNeedsDescription(ctx, listing.ID, false); err != nil {
		return err
	}

	// 5. Hand it to the artisan. Submitting an already-submitted listing is an
	//    illegal transition, which on a redelivery is the correct no-op.
	if listing.State == domain.StateDraft {
		if _, err := p.catalog.SubmitForApproval(ctx, listing.ID,
			"pipeline:"+stepDraft+":"+mediaID.String()); err != nil && !errors.Is(err, pkgdomain.ErrInvalidInput) {
			return err
		}
	}

	p.log.InfoContext(ctx, "listing drafted",
		"media_id", mediaID, "listing_id", listing.ID, "craft_id", craftID, "language", language)
	return nil
}

// draftListing creates the offer for a product, or returns the one an earlier
// delivery created.
func (p *Pipeline) draftListing(
	ctx context.Context,
	product domain.Product,
	attributes domain.InferredAttributes,
) (domain.Listing, error) {
	existing, err := p.store.GetListingByProduct(ctx, product.ID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pkgdomain.ErrNotFound) {
		return domain.Listing{}, err
	}

	// Made to order with a conservative lead time: the artisan sets the real
	// price and terms at approval, and nothing goes on sale before they do.
	leadTime, capacity := int32(21), int32(4)
	return p.catalog.UpsertListing(ctx, domain.UpsertListingInput{
		ProductID:        product.ID,
		Type:             domain.ListingMadeToOrder,
		PricePaise:       0,
		MinOrderQuantity: 1,
		LeadTimeDays:     &leadTime,
		CapacityPerMonth: &capacity,
		AcceptingOrders:  false,
		CreatedBy:        "pipeline",
	}, "pipeline:"+stepDraft+":"+product.ID.String())
}

// fail records a terminal failure on the media row so the artisan sees a reason
// rather than a photograph that silently never became anything, and returns nil
// so the message is committed instead of retried into the same wall.
func (p *Pipeline) fail(ctx context.Context, media domain.Media, step string, cause error) error {
	if !terminal(cause) {
		return fmt.Errorf("%s: %w", step, cause) // let the consumer retry
	}
	return p.RecordFailure(ctx, media.ID, fmt.Sprintf("%s: %v", step, cause))
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

// Translate fans a published listing's copy out to the buyer languages. Craft
// terms are held out of the machine's reach and put back verbatim afterwards, so
// "Ajrakh" does not come back as "indigo cloth".
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

	for _, language := range p.targetLanguages(source.Language) {
		// An artisan's own words are never overwritten by a machine.
		if prior, seen := have[language]; seen && !prior.MachineGenerated {
			continue
		}

		generated, err := timed(ctx, stepTranslate, func() (domain.GeneratedCopy, error) {
			return p.inferrer.GenerateDescription(ctx, domain.CopyRequest{
				CraftID:     craftID,
				CraftCode:   craft.Code,
				Language:    language,
				ArtisanNote: mask(source.Description, protected),
			})
		})
		if err != nil {
			// One language failing must not cost the others.
			p.log.WarnContext(ctx, "translation failed",
				"listing_id", listingID, "language", language, "error", err)
			continue
		}

		if _, err := p.catalog.UpsertListingTranslation(ctx, listingID, domain.ListingTranslation{
			Language:         language,
			Title:            unmask(generated.Title, protected),
			Description:      unmask(generated.Description, protected),
			Highlights:       unmaskAll(generated.Highlights, protected),
			MachineGenerated: true,
		}, "pipeline:"+stepTranslate+":"+listingID.String()+":"+language); err != nil {
			return err
		}
	}
	return nil
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

// craftAllowlist is the set of crafts one artisan may have their work catalogued
// under, indexed by the slug the model answers with.
type craftAllowlist struct {
	byCode   map[string]uuid.UUID
	declared string
}

func (a craftAllowlist) resolve(code string) (uuid.UUID, bool) {
	id, ok := a.byCode[code]
	return id, ok
}

func (p *Pipeline) craftCodes(craftIDs []uuid.UUID) craftAllowlist {
	out := craftAllowlist{byCode: make(map[string]uuid.UUID, len(craftIDs))}
	for i, id := range craftIDs {
		craft, ok := p.crafts.Craft(id)
		if !ok {
			continue
		}
		out.byCode[craft.Code] = id
		if i == 0 {
			out.declared = craft.Code // the artisan's primary craft is the prior
		}
	}
	return out
}

func workingTitle(a domain.InferredAttributes) string {
	parts := make([]string, 0, 3)
	if len(a.Colours) > 0 {
		parts = append(parts, a.Colours[0])
	}
	if a.Material != "" {
		parts = append(parts, a.Material)
	}
	parts = append(parts, strings.ReplaceAll(a.CraftCode, "-", " "))
	return strings.TrimSpace(strings.Join(parts, " "))
}

func nonEmpty(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
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
