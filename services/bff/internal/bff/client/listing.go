// services/bff/internal/bff/client/listing.go
package client

import (
	"context"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"
	inferencev1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/inference/v1"
)

// maxTranslations and maxMediaRefs bound array fields a client controls, so a
// malicious or buggy caller can't force an unbounded request downstream.
const (
	maxTranslations = 20
	maxMediaRefs    = 20
)

// Listing is bff's view of core-svc's product+listing lifecycle, satisfying
// handler.ListingService. core-svc splits "product" (the physical item) from
// "listing" (its sellable offer) across two RPCs — CreateProduct and
// UpsertListing — so CreateListing chains them in one call.
type Listing struct {
	catalog  catalogv1.CatalogServiceClient
	ontology catalogv1.OntologyServiceClient
	identity identityv1.IdentityServiceClient
	media    catalogv1.MediaServiceClient
}

// NewListing builds the listing client, sharing conn with core-svc's other services.
func NewListing(conn grpc.ClientConnInterface) *Listing {
	return &Listing{
		catalog:  catalogv1.NewCatalogServiceClient(conn),
		ontology: catalogv1.NewOntologyServiceClient(conn),
		identity: identityv1.NewIdentityServiceClient(conn),
		media:    catalogv1.NewMediaServiceClient(conn),
	}
}

// CreateListing registers the product and its sellable offer in one call.
// fields carries the POST /listings JSON body verbatim:
//
//	craft_id, working_title (product; media/voice_note_media_id are media ids
//	already confirmed via /media, dimensions is {length_mm?, width_mm?,
//	height_mm?, weight_g?})
//	type ("READY_STOCK"|"MADE_TO_ORDER"), price ({amount_paise, currency_code?}),
//	stock_quantity (required for READY_STOCK), min_order_quantity (defaults to
//	1), made_to_order_terms ({lead_time_days, capacity_per_month,
//	accepting_orders?, advance_pct?}), packaging ({fragile?, oversized?,
//	requires_custom_crating?, packed_dimensions?}), translations (array of
//	{language, title, description, highlights?})
//
// idempotencyKey (from X-Idempotency-Key or minted by the handler) is reused,
// suffixed, for both chained calls, so a retried HTTP call replays each RPC
// instead of creating a second product when only the second call failed.
//
// ponytail: CreateProduct and UpsertListing are not atomic — if UpsertListing
// fails, the product is orphaned (unreachable via any listing, but not
// deleted). Acceptable because idempotency_key makes a client retry safe
// (CreateProduct replays instead of creating a second product); add a
// compensating delete only if orphaned products turn out to matter.
func (l *Listing) CreateListing(ctx context.Context, artisanID, idempotencyKey string, fields map[string]any) (string, error) {
	craftID, _ := fields["craft_id"].(string)
	if craftID == "" {
		return "", domain.InvalidInput("craft_id: is required")
	}
	workingTitle, _ := fields["working_title"].(string)

	mediaIDs, err := stringSlice(fields, "media")
	if err != nil {
		return "", err
	}
	if len(mediaIDs) > maxMediaRefs {
		return "", domain.InvalidInput("media: too many")
	}
	media := make([]*commonv1.MediaRef, len(mediaIDs))
	for i, id := range mediaIDs {
		media[i] = &commonv1.MediaRef{Id: id}
	}

	var voiceNote *commonv1.MediaRef
	if id, ok := fields["voice_note_media_id"].(string); ok && id != "" {
		voiceNote = &commonv1.MediaRef{Id: id}
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	productResp, err := l.catalog.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		ArtisanId:      artisanID,
		CraftId:        craftID,
		WorkingTitle:   workingTitle,
		Media:          media,
		Dimensions:     dimensionsFromAny(fields["dimensions"]),
		VoiceNote:      voiceNote,
		IdempotencyKey: idempotencyKey + ":product",
	})
	if err != nil {
		return "", grpcErr(err)
	}
	productID := productResp.GetProduct().GetId()

	req, err := upsertListingRequest(fields, productID, nil, idempotencyKey+":listing")
	if err != nil {
		return "", err
	}
	listingResp, err := l.catalog.UpsertListing(ctx, req)
	if err != nil {
		return "", grpcErr(err)
	}
	return listingResp.GetListing().GetId(), nil
}

// upsertListingRequest builds an UpsertListingRequest from the loose fields
// map. existing, when non-nil, backfills any field fields doesn't mention —
// UpsertListing replaces a listing's commercial fields wholesale, so
// UpdateListing must resend the current value of anything the caller didn't
// change, not just what the caller sent.
func upsertListingRequest(fields map[string]any, productID string, existing *catalogv1.Listing, idempotencyKey string) (*catalogv1.UpsertListingRequest, error) {
	req := &catalogv1.UpsertListingRequest{
		ProductId:      productID,
		IdempotencyKey: idempotencyKey,
	}

	if v, ok := fields["type"].(string); ok {
		req.Type = listingTypeFromString(v)
	} else if existing != nil {
		req.Type = existing.GetType()
	}

	if v, ok := fields["price"]; ok {
		money, err := moneyFromAny(v)
		if err != nil {
			return nil, err
		}
		req.Price = money
	} else if existing != nil {
		req.Price = existing.GetPrice()
	}

	// stock_quantity is present-but-null when a caller means to clear it (e.g.
	// switching a listing off READY_STOCK), which a map-value type assertion
	// alone can't tell apart from the key being absent — both decode to a nil
	// interface. Checked as key presence first so "null" reaches core-svc as
	// an explicit clear instead of silently reverting to the old value.
	if raw, present := fields["stock_quantity"]; present {
		if raw != nil {
			v, ok := raw.(float64)
			if !ok {
				return nil, domain.InvalidInput("stock_quantity: must be a number")
			}
			sq := int32(v)
			req.StockQuantity = &sq
		}
	} else if existing != nil {
		req.StockQuantity = existing.StockQuantity
	}

	if v, ok := fields["min_order_quantity"].(float64); ok {
		req.MinOrderQuantity = int32(v)
	} else if existing != nil {
		req.MinOrderQuantity = existing.GetMinOrderQuantity()
	}

	if v, ok := fields["made_to_order_terms"]; ok {
		terms, err := madeToOrderTermsFromAny(v)
		if err != nil {
			return nil, err
		}
		req.MadeToOrderTerms = terms
	} else if existing != nil {
		req.MadeToOrderTerms = existing.GetMadeToOrderTerms()
	}

	if v, ok := fields["packaging"]; ok {
		req.Packaging = packagingFromAny(v)
	} else if existing != nil {
		req.Packaging = existing.GetPackaging()
	} else {
		req.Packaging = &catalogv1.PackagingMeta{}
	}

	if v, ok := fields["translations"]; ok {
		translations, err := translationsFromAny(v)
		if err != nil {
			return nil, err
		}
		req.Translations = translations
	} else if existing != nil {
		req.Translations = existing.GetTranslations()
	}

	return req, nil
}

// UpdateListing patches an existing listing. UpsertListing replaces a
// listing's commercial fields wholesale, so this fetches the current listing
// first and only overrides the fields present in updates, matching
// UpdateProfile's own patch semantics elsewhere in this package.
func (l *Listing) UpdateListing(ctx context.Context, listingID, idempotencyKey string, updates map[string]any) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	current, err := l.catalog.GetListing(ctx, &catalogv1.GetListingRequest{ListingId: listingID})
	if err != nil {
		return grpcErr(err)
	}
	existing := current.GetListing()

	listingIDCopy := listingID
	req, err := upsertListingRequest(updates, existing.GetProductId(), existing, idempotencyKey)
	if err != nil {
		return err
	}
	req.ListingId = &listingIDCopy

	if _, err := l.catalog.UpsertListing(ctx, req); err != nil {
		return grpcErr(err)
	}
	return nil
}

// SubmitForReview hands a draft to the artisan for sign-off.
func (l *Listing) SubmitForReview(ctx context.Context, listingID, idempotencyKey string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	if _, err := l.catalog.SubmitForApproval(ctx, &catalogv1.SubmitForApprovalRequest{
		ListingId:      listingID,
		IdempotencyKey: idempotencyKey,
	}); err != nil {
		return grpcErr(err)
	}
	return nil
}

// ApproveListing records the artisan's sign-off and publishes.
func (l *Listing) ApproveListing(ctx context.Context, listingID, artisanID, idempotencyKey string, editedTranslations []map[string]any) error {
	var translations []*catalogv1.ListingTranslation
	if len(editedTranslations) > 0 {
		raw := make([]any, len(editedTranslations))
		for i, t := range editedTranslations {
			raw[i] = t
		}
		var err error
		translations, err = translationsFromAny(raw)
		if err != nil {
			return err
		}
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	if _, err := l.catalog.ApproveListing(ctx, &catalogv1.ApproveListingRequest{
		ListingId:          listingID,
		ArtisanId:          artisanID,
		EditedTranslations: translations,
		IdempotencyKey:     idempotencyKey,
	}); err != nil {
		return grpcErr(err)
	}
	return nil
}

// GetListing fetches one listing.
func (l *Listing) GetListing(ctx context.Context, listingID string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := l.catalog.GetListing(ctx, &catalogv1.GetListingRequest{ListingId: listingID})
	if err != nil {
		return nil, grpcErr(err)
	}
	return listingToMap(resp.GetListing()), nil
}

// GetListingSummary is GetListing plus every field the buyer-facing product
// page (and search/home cards, which only use a subset) needs: craft name
// and GI registration, materials, colours, image/video URLs, made-to-order
// terms, the artisan's story (bio + district + voice note) and -- when
// sealed -- the provenance verdicts. One listing page, one call: everything
// below is a join this handler already had the clients for.
func (l *Listing) GetListingSummary(ctx context.Context, listingID string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := l.catalog.GetListing(ctx, &catalogv1.GetListingRequest{
		ListingId:         listingID,
		IncludeProduct:    true,
		IncludeProvenance: true,
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	m := listingToMap(resp.GetListing())
	if terms := resp.GetListing().GetMadeToOrderTerms(); terms != nil {
		m["made_to_order_terms"] = madeToOrderTermsToMap(terms)
	}
	if prov := resp.GetProvenance(); prov != nil {
		m["provenance"] = provenanceRecordToMap(prov)
	}

	product := resp.GetProduct()
	if product == nil {
		return m, nil
	}
	m["materials"] = product.GetMaterials()
	m["colours"] = product.GetColours()
	if vn := product.GetVoiceNote(); vn.GetId() != "" {
		if url, err := l.media.GetMediaURL(ctx, &catalogv1.GetMediaURLRequest{MediaId: vn.GetId()}); err == nil {
			m["story_audio_url"] = url.GetUrl()
		}
	}

	if craft, err := l.ontology.GetCraft(ctx, &catalogv1.GetCraftRequest{CraftId: product.GetCraftId()}); err == nil {
		m["craft_id"] = craft.GetCraft().GetId()
		m["craft_name"] = craft.GetCraft().GetDisplayName()
		m["craft_slug"] = craft.GetCraft().GetCode()
		if reg := craft.GetCraft().GetGiRegistrationNo(); reg != "" {
			m["craft_gi_registration_no"] = reg
		}
	}
	if art, err := l.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: resp.GetListing().GetArtisanId()}); err == nil {
		a := art.GetArtisan()
		m["artisan_name"] = a.GetDisplayName()
		m["artisan_verified"] = a.GetVerified()
		if a.Bio != nil {
			m["artisan_bio"] = *a.Bio
		}
		if d := a.GetRegion().GetDistrict(); d != "" {
			m["artisan_district"] = d
		}
		m["artisan_state_code"] = a.GetRegion().GetStateCode()
	}

	media := product.GetMedia()
	if len(media) > maxMediaRefs {
		media = media[:maxMediaRefs]
	}
	items := make([]map[string]any, len(media))
	group, groupCtx := errgroup.WithContext(ctx)
	for i, ref := range media {
		i, ref := i, ref
		group.Go(func() error {
			url, err := l.media.GetMediaURL(groupCtx, &catalogv1.GetMediaURLRequest{MediaId: ref.GetId()})
			items[i] = map[string]any{
				"kind": trimEnumPrefix(ref.GetKind().String(), "MEDIA_KIND_"),
				"url":  url.GetUrl(),
			}
			if err != nil {
				items[i]["url"] = ""
			}
			return nil
		})
	}
	_ = group.Wait()
	m["media"] = items
	for _, item := range items {
		if item["kind"] == "IMAGE" && item["url"] != "" {
			m["image_url"] = item["url"]
			break
		}
	}
	return m, nil
}

// maxBatchSummaryIDs bounds a client-supplied id list, same reasoning as
// maxMediaRefs/maxTranslations above.
const maxBatchSummaryIDs = 50

// BatchGetListingSummaries fetches GetListingSummary for many listings in
// one round trip from the caller's point of view. Collapses what used to be
// N sequential HTTP calls from the buyer storefront page (one per listing)
// into a single request; each summary is still assembled the same way
// GetListingSummary does it, just fanned out concurrently.
func (l *Listing) BatchGetListingSummaries(ctx context.Context, ids []string) ([]map[string]any, error) {
	if len(ids) > maxBatchSummaryIDs {
		ids = ids[:maxBatchSummaryIDs]
	}

	out := make([]map[string]any, len(ids))
	group, groupCtx := errgroup.WithContext(ctx)
	for i, id := range ids {
		i, id := i, id
		group.Go(func() error {
			summary, err := l.GetListingSummary(groupCtx, id)
			if err != nil {
				return nil // skip listings that failed to resolve, don't fail the whole batch
			}
			out[i] = summary
			return nil
		})
	}
	_ = group.Wait()

	result := make([]map[string]any, 0, len(out))
	for _, s := range out {
		if s != nil {
			result = append(result, s)
		}
	}
	return result, nil
}

// ListListings pages through listings under the usual filters.
func (l *Listing) ListListings(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	req := &catalogv1.ListListingsRequest{
		Page: &commonv1.PageRequest{PageSize: defaultPageSize},
	}
	if v, ok := filters["craft_id"].(string); ok && v != "" {
		req.CraftId = &v
	}
	if v, ok := filters["artisan_id"].(string); ok && v != "" {
		req.ArtisanId = &v
	}
	if v, ok := filters["state"].(string); ok && v != "" {
		req.State = listingStateFromString(v)
	}
	if v, ok := filters["type"].(string); ok && v != "" {
		req.Type = listingTypeFromString(v)
	}

	resp, err := l.catalog.ListListings(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]map[string]any, 0, len(resp.GetListings()))
	for _, listing := range resp.GetListings() {
		out = append(out, listingToMap(listing))
	}
	return out, nil
}

// SealProvenance freezes process evidence for a PUBLISHED listing: the media
// hashed into the record, the artisan's declared technique (checked against
// the model), and -- for textile crafts, unless skip_loom_check -- a loom
// verdict. fields carries media (array of confirmed media ids, required),
// claimed_technique (string, required) and skip_loom_check (bool, optional).
func (l *Listing) SealProvenance(ctx context.Context, listingID, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	mediaIDs, err := stringSlice(fields, "media")
	if err != nil {
		return nil, err
	}
	if len(mediaIDs) > maxMediaRefs {
		return nil, domain.InvalidInput("media: too many")
	}
	media := make([]*commonv1.MediaRef, len(mediaIDs))
	for i, id := range mediaIDs {
		media[i] = &commonv1.MediaRef{Id: id}
	}

	claimedTechnique, _ := fields["claimed_technique"].(string)
	if claimedTechnique == "" {
		return nil, domain.InvalidInput("claimed_technique: is required")
	}
	skipLoomCheck, _ := fields["skip_loom_check"].(bool)

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := l.catalog.SealProvenance(ctx, &catalogv1.SealProvenanceRequest{
		ListingId:        listingID,
		Media:            media,
		ClaimedTechnique: claimedTechnique,
		SkipLoomCheck:    skipLoomCheck,
		IdempotencyKey:   idempotencyKey,
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	return provenanceRecordToMap(resp.GetProvenance()), nil
}

// provenanceRecordToMap maps SealProvenance's response -- the richer,
// request-time verdict shape a fresh seal has on hand. Distinct from the
// leaner SealedProvenance a later short-code lookup returns (see
// GetProvenanceByShortCode), which only has what was frozen at seal time.
func provenanceRecordToMap(rec *catalogv1.ProvenanceRecord) map[string]any {
	if rec == nil {
		return nil
	}
	out := map[string]any{
		"id":                rec.GetId(),
		"listing_id":        rec.GetListingId(),
		"product_id":        rec.GetProductId(),
		"content_hash":      rec.GetContentHash(),
		"signature_algo":    rec.GetSignatureAlgorithm(),
		"qr_code":           rec.GetQrCode(),
		"sealed_at":         rec.GetSealedAt().AsTime().Format(time.RFC3339),
		"technique_verdict": techniqueVerdictToMap(rec.GetTechniqueVerdict()),
	}
	if rec.PreviousHash != nil {
		out["previous_hash"] = *rec.PreviousHash
	}
	if lv := rec.GetLoomVerdict(); lv != nil {
		out["loom_verdict"] = loomVerdictToMap(lv)
	}
	if cert := rec.GetCertificate(); cert != nil {
		out["certificate_media_id"] = cert.GetId()
	}
	return out
}

// madeToOrderTermsToMap presents made-to-order as the feature the buyer is
// paying for, not a stock shortage: lead time, monthly capacity, whether the
// artisan is currently taking new orders, the advance split, and whatever
// the buyer may customise.
func madeToOrderTermsToMap(t *catalogv1.MadeToOrderTerms) map[string]any {
	opts := make([]map[string]any, 0, len(t.GetCustomisationOptions()))
	for _, o := range t.GetCustomisationOptions() {
		opt := map[string]any{"name": o.GetName(), "values": o.GetValues()}
		if o.Surcharge != nil {
			opt["surcharge"] = moneyMap(o.GetSurcharge())
		}
		if o.ExtraLeadTimeDays != nil {
			opt["extra_lead_time_days"] = *o.ExtraLeadTimeDays
		}
		opts = append(opts, opt)
	}
	return map[string]any{
		"lead_time_days":        t.GetLeadTimeDays(),
		"capacity_per_month":    t.GetCapacityPerMonth(),
		"accepting_orders":      t.GetAcceptingOrders(),
		"advance_pct":           t.GetAdvancePct(),
		"customisation_options": opts,
	}
}

func techniqueVerdictToMap(v *inferencev1.TechniqueVerdict) map[string]any {
	if v == nil {
		return nil
	}
	out := map[string]any{
		"claimed":       v.GetClaimed(),
		"observed":      v.GetObserved(),
		"matches":       v.GetMatches(),
		"confidence":    v.GetConfidence(),
		"model_version": v.GetModelVersion(),
	}
	if v.Explanation != nil {
		out["explanation"] = *v.Explanation
	}
	return out
}

func loomVerdictToMap(v *inferencev1.LoomVerdict) map[string]any {
	return map[string]any{
		"is_handloom":    v.GetIsHandloom(),
		"confidence":     v.GetConfidence(),
		"fft_peak_ratio": v.GetFftPeakRatio(),
		"explanation":    v.GetExplanation(),
		"model_version":  v.GetModelVersion(),
	}
}

func listingToMap(l *catalogv1.Listing) map[string]any {
	m := map[string]any{
		"id":                 l.GetId(),
		"product_id":         l.GetProductId(),
		"artisan_id":         l.GetArtisanId(),
		"type":               trimEnumPrefix(l.GetType().String(), "LISTING_TYPE_"),
		"state":              trimEnumPrefix(l.GetState().String(), "LISTING_STATE_"),
		"price":              moneyMap(l.GetPrice()),
		"min_order_quantity": l.GetMinOrderQuantity(),
		"gi_certified":       l.GetGiCertified(),
		"translations":       translationsToAny(l.GetTranslations()),
	}
	if l.StockQuantity != nil {
		m["stock_quantity"] = *l.StockQuantity
	}
	if l.ProvenanceId != nil {
		m["provenance_id"] = *l.ProvenanceId
	}
	return m
}

func translationsToAny(translations []*catalogv1.ListingTranslation) []map[string]any {
	out := make([]map[string]any, 0, len(translations))
	for _, t := range translations {
		out = append(out, map[string]any{
			"language":          languageToString(t.GetLanguage()),
			"title":             t.GetTitle(),
			"description":       t.GetDescription(),
			"highlights":        t.GetHighlights(),
			"machine_generated": t.GetMachineGenerated(),
		})
	}
	return out
}

// listingTypeFromString and listingStateFromString mirror languageToProto's
// convention (client/search.go): a compact name looked up against the
// generated enum's own _value map, defaulting to UNSPECIFIED.
func listingTypeFromString(name string) catalogv1.ListingType {
	if v, ok := catalogv1.ListingType_value["LISTING_TYPE_"+name]; ok {
		return catalogv1.ListingType(v)
	}
	return catalogv1.ListingType_LISTING_TYPE_UNSPECIFIED
}

func listingStateFromString(name string) catalogv1.ListingState {
	if v, ok := catalogv1.ListingState_value["LISTING_STATE_"+name]; ok {
		return catalogv1.ListingState(v)
	}
	return catalogv1.ListingState_LISTING_STATE_UNSPECIFIED
}

func trimEnumPrefix(s, prefix string) string { return strings.TrimPrefix(s, prefix) }

func languageToString(l commonv1.Language) string {
	return trimEnumPrefix(l.String(), "LANGUAGE_")
}

// moneyFromAny converts the JSON "price" object ({amount_paise, currency_code?})
// into a Money. currency_code defaults to INR, the only currency core-svc
// accepts today.
func moneyFromAny(raw any) (*commonv1.Money, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, domain.InvalidInput("price: must be an object")
	}
	amount, ok := m["amount_paise"].(float64)
	if !ok {
		return nil, domain.InvalidInput("price.amount_paise: is required")
	}
	currency, _ := m["currency_code"].(string)
	if currency == "" {
		currency = "INR"
	}
	return &commonv1.Money{AmountPaise: int64(amount), CurrencyCode: currency}, nil
}

// dimensionsFromAny converts the optional JSON "dimensions" object. Every
// field is optional at the proto level, so a missing or malformed object
// yields nil rather than an error — dimensions are informational, not a
// business rule core-svc enforces.
func dimensionsFromAny(raw any) *commonv1.Dimensions {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	d := &commonv1.Dimensions{}
	if v, ok := m["length_mm"].(float64); ok {
		lv := int32(v)
		d.LengthMm = &lv
	}
	if v, ok := m["width_mm"].(float64); ok {
		wv := int32(v)
		d.WidthMm = &wv
	}
	if v, ok := m["height_mm"].(float64); ok {
		hv := int32(v)
		d.HeightMm = &hv
	}
	if v, ok := m["weight_g"].(float64); ok {
		gv := int32(v)
		d.WeightG = &gv
	}
	return d
}

// madeToOrderTermsFromAny converts the optional JSON "made_to_order_terms"
// object. core-svc's own Validate() enforces lead_time_days/
// capacity_per_month being positive for a MADE_TO_ORDER listing; this only
// maps the shape, it doesn't duplicate that business rule.
func madeToOrderTermsFromAny(raw any) (*catalogv1.MadeToOrderTerms, error) {
	if raw == nil {
		return nil, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, domain.InvalidInput("made_to_order_terms: must be an object")
	}
	terms := &catalogv1.MadeToOrderTerms{AcceptingOrders: true}
	if v, ok := m["lead_time_days"].(float64); ok {
		terms.LeadTimeDays = int32(v)
	}
	if v, ok := m["capacity_per_month"].(float64); ok {
		terms.CapacityPerMonth = int32(v)
	}
	if v, ok := m["accepting_orders"].(bool); ok {
		terms.AcceptingOrders = v
	}
	if v, ok := m["advance_pct"].(float64); ok {
		terms.AdvancePct = int32(v)
	}
	return terms, nil
}

// packagingFromAny converts the optional JSON "packaging" object. Packaging
// isn't itself optional on the wire — a listing with no special handling
// needs is the all-false zero value, not a nil message.
func packagingFromAny(raw any) *catalogv1.PackagingMeta {
	m, ok := raw.(map[string]any)
	if !ok {
		return &catalogv1.PackagingMeta{}
	}
	p := &catalogv1.PackagingMeta{}
	if v, ok := m["fragile"].(bool); ok {
		p.Fragile = v
	}
	if v, ok := m["oversized"].(bool); ok {
		p.Oversized = v
	}
	if v, ok := m["requires_custom_crating"].(bool); ok {
		p.RequiresCustomCrating = v
	}
	if v := dimensionsFromAny(m["packed_dimensions"]); v != nil {
		p.PackedDimensions = v
	}
	return p
}

// translationsFromAny converts the JSON "translations" array. Each entry
// must have a recognised language and a title; description and highlights
// are optional.
func translationsFromAny(raw any) ([]*catalogv1.ListingTranslation, error) {
	list, ok := raw.([]any)
	if !ok {
		return nil, domain.InvalidInput("translations: must be an array")
	}
	if len(list) > maxTranslations {
		return nil, domain.InvalidInput("translations: too many")
	}
	out := make([]*catalogv1.ListingTranslation, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, domain.InvalidInput("translations: each entry must be an object")
		}
		lang, _ := m["language"].(string)
		title, _ := m["title"].(string)
		if lang == "" || title == "" {
			return nil, domain.InvalidInput("translations: language and title are required")
		}
		t := &catalogv1.ListingTranslation{
			Language:    languageToProto(lang),
			Title:       title,
			Description: mustString(m["description"]),
		}
		if highlights, ok := m["highlights"].([]any); ok {
			for _, h := range highlights {
				if s, ok := h.(string); ok {
					t.Highlights = append(t.Highlights, s)
				}
			}
		}
		if v, ok := m["machine_generated"].(bool); ok {
			t.MachineGenerated = v
		}
		out = append(out, t)
	}
	return out, nil
}

func mustString(v any) string {
	s, _ := v.(string)
	return s
}
