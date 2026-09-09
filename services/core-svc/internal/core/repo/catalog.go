// services/core-svc/internal/core/repo/catalog.go
package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// --- product -----------------------------------------------------------------

// CreateProduct inserts a product row.
func (t *Tx) CreateProduct(ctx context.Context, id uuid.UUID, in domain.CreateProductInput) (domain.Product, error) {
	row, err := t.q.CreateProduct(ctx, db.CreateProductParams{
		ID:               id,
		ArtisanID:        in.ArtisanID,
		CraftID:          in.CraftID,
		WorkingTitle:     in.WorkingTitle,
		LengthMm:         in.Dimensions.LengthMM,
		WidthMm:          in.Dimensions.WidthMM,
		HeightMm:         in.Dimensions.HeightMM,
		WeightG:          in.Dimensions.WeightG,
		Materials:        orEmpty(in.Materials),
		Techniques:       orEmpty(in.Techniques),
		Colours:          orEmpty(in.Colours),
		Motifs:           orEmpty(in.Motifs),
		VoiceNoteMediaID: in.VoiceNoteMediaID,
		CreatedBy:        in.CreatedBy,
	})
	if err != nil {
		return domain.Product{}, translate(err, "product")
	}
	return productFromRow(row), nil
}

// AttachProductMedia points already-uploaded media rows at a product. The
// artisan filter is the guard: a caller cannot pull another maker's photographs
// onto their own product by guessing an id, and a short rowcount says so.
func (t *Tx) AttachProductMedia(ctx context.Context, productID, artisanID uuid.UUID, mediaIDs []uuid.UUID) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	attached, err := t.q.AttachMediaToProduct(ctx, db.AttachMediaToProductParams{
		ProductID: productID,
		ArtisanID: artisanID,
		Ids:       mediaIDs,
	})
	if err != nil {
		return translate(err, "product media")
	}
	if int(attached) != len(mediaIDs) {
		return fmt.Errorf("%d of %d media ids do not exist or belong to another artisan: %w",
			len(mediaIDs)-int(attached), len(mediaIDs), pkgdomain.ErrInvalidInput)
	}
	return nil
}

// GetOrCreateProductForMedia is the pipeline's idempotency: the unique index on
// product.source_media_id means a redelivered media.uploaded inserts nothing and
// gets back the product the first delivery made. created says which happened.
func (t *Tx) GetOrCreateProductForMedia(
	ctx context.Context,
	in domain.CreateProductInput,
	mediaID uuid.UUID,
) (domain.Product, bool, error) {
	row, err := t.q.CreateProductForMedia(ctx, db.CreateProductForMediaParams{
		ID:            ids.New(),
		ArtisanID:     in.ArtisanID,
		CraftID:       in.CraftID,
		WorkingTitle:  in.WorkingTitle,
		SourceMediaID: &mediaID,
		CreatedBy:     in.CreatedBy,
	})
	if err == nil {
		return productFromRow(row), true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Product{}, false, translate(err, "product")
	}

	// DO NOTHING returned no row, so this media already has a product.
	existing, err := t.q.GetProductByMedia(ctx, &mediaID)
	if err != nil {
		return domain.Product{}, false, translate(err, "product for media")
	}
	return productFromRow(existing), false, nil
}

// GetProduct reads one product by id.
func (r *Repo) GetProduct(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	row, err := r.q.GetProduct(ctx, id)
	if err != nil {
		return domain.Product{}, translate(err, "product")
	}
	return productFromRow(row), nil
}

func productFromRow(row db.Product) domain.Product {
	return domain.Product{
		ID:           row.ID,
		ArtisanID:    row.ArtisanID,
		CraftID:      row.CraftID,
		WorkingTitle: row.WorkingTitle,
		Dimensions: domain.Dimensions{
			LengthMM: row.LengthMm,
			WidthMM:  row.WidthMm,
			HeightMM: row.HeightMm,
			WeightG:  row.WeightG,
		},
		Materials:        row.Materials,
		Techniques:       row.Techniques,
		Colours:          row.Colours,
		Motifs:           row.Motifs,
		VoiceNoteMediaID: row.VoiceNoteMediaID,
		CreatedBy:        row.CreatedBy,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

// --- listing -----------------------------------------------------------------

// CreateListing inserts a listing row, which the schema defaults to DRAFT.
func (t *Tx) CreateListing(ctx context.Context, id, artisanID uuid.UUID, in domain.UpsertListingInput) (domain.Listing, error) {
	row, err := t.q.CreateListing(ctx, db.CreateListingParams{
		ID:                             id,
		ProductID:                      in.ProductID,
		ArtisanID:                      artisanID,
		Type:                           db.ListingType(in.Type),
		PricePaise:                     in.PricePaise,
		StockQuantity:                  in.StockQuantity,
		MinOrderQuantity:               in.MinOrderQuantity,
		LeadTimeDays:                   in.LeadTimeDays,
		CapacityPerMonth:               in.CapacityPerMonth,
		AcceptingOrders:                in.AcceptingOrders,
		AdvancePct:                     in.AdvancePct,
		PackagingFragile:               in.Packaging.Fragile,
		PackagingOversized:             in.Packaging.Oversized,
		PackagingRequiresCustomCrating: in.Packaging.RequiresCustomCrating,
		PackedLengthMm:                 in.Packaging.Packed.LengthMM,
		PackedWidthMm:                  in.Packaging.Packed.WidthMM,
		PackedHeightMm:                 in.Packaging.Packed.HeightMM,
		PackedWeightG:                  in.Packaging.Packed.WeightG,
		GiCertified:                    in.GICertified,
		CreatedBy:                      in.CreatedBy,
	})
	if err != nil {
		return domain.Listing{}, translate(err, "listing")
	}
	return listingFromRow(row), nil
}

// UpdateListing patches the commercial fields of an existing listing.
func (t *Tx) UpdateListing(ctx context.Context, in domain.UpsertListingInput) (domain.Listing, error) {
	if in.ListingID == nil {
		return domain.Listing{}, fmt.Errorf("listing_id is required to update: %w", pkgdomain.ErrInvalidInput)
	}
	row, err := t.q.UpdateListing(ctx, db.UpdateListingParams{
		ID:                             *in.ListingID,
		PricePaise:                     &in.PricePaise,
		StockQuantity:                  in.StockQuantity,
		MinOrderQuantity:               &in.MinOrderQuantity,
		LeadTimeDays:                   in.LeadTimeDays,
		CapacityPerMonth:               in.CapacityPerMonth,
		AcceptingOrders:                &in.AcceptingOrders,
		AdvancePct:                     in.AdvancePct,
		PackagingFragile:               &in.Packaging.Fragile,
		PackagingOversized:             &in.Packaging.Oversized,
		PackagingRequiresCustomCrating: &in.Packaging.RequiresCustomCrating,
		PackedLengthMm:                 in.Packaging.Packed.LengthMM,
		PackedWidthMm:                  in.Packaging.Packed.WidthMM,
		PackedHeightMm:                 in.Packaging.Packed.HeightMM,
		PackedWeightG:                  in.Packaging.Packed.WeightG,
		GiCertified:                    &in.GICertified,
	})
	if err != nil {
		return domain.Listing{}, translate(err, "listing")
	}
	return listingFromRow(row), nil
}

// TransitionListingState performs the guarded state change. A zero-row update
// means the listing moved under the caller between the read and the write, which
// is a conflict rather than a missing row.
func (t *Tx) TransitionListingState(
	ctx context.Context,
	listingID uuid.UUID,
	from, to domain.ListingState,
	reason *string,
) (domain.Listing, error) {
	row, err := t.q.TransitionListingState(ctx, db.TransitionListingStateParams{
		ID:               listingID,
		NextState:        db.ListingState(to),
		ExpectedState:    db.ListingState(from),
		SuspensionReason: reason,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Listing{}, fmt.Errorf(
				"listing %s is no longer in %s, so it cannot move to %s: %w",
				listingID, from, to, pkgdomain.ErrConflict)
		}
		return domain.Listing{}, translate(err, "listing state")
	}
	return listingFromRow(row), nil
}

// GetListing reads one listing without its copy, attributes or media.
func (r *Repo) GetListing(ctx context.Context, id uuid.UUID) (domain.Listing, error) {
	row, err := r.q.GetListing(ctx, id)
	if err != nil {
		return domain.Listing{}, translate(err, "listing")
	}
	return listingFromRow(row), nil
}

// GetListingDetail reads one listing with its translations and attributes.
func (r *Repo) GetListingDetail(ctx context.Context, id uuid.UUID) (domain.Listing, error) {
	listing, err := r.GetListing(ctx, id)
	if err != nil {
		return domain.Listing{}, err
	}

	translations, err := r.q.GetListingTranslations(ctx, id)
	if err != nil {
		return domain.Listing{}, translate(err, "listing translations")
	}
	listing.Translations = make([]domain.ListingTranslation, 0, len(translations))
	for _, row := range translations {
		listing.Translations = append(listing.Translations, translationFromRow(row))
	}

	attributes, err := r.q.GetListingAttributes(ctx, id)
	if err != nil {
		return domain.Listing{}, translate(err, "listing attributes")
	}
	listing.Attributes = make([]domain.ListingAttribute, 0, len(attributes))
	for _, row := range attributes {
		listing.Attributes = append(listing.Attributes, attributeFromRow(row))
	}
	return listing, nil
}

// ListListings pages listings under the catalogue filters, by keyset on the
// UUIDv7 id.
func (r *Repo) ListListings(ctx context.Context, filter domain.ListingFilter, page domain.Page) ([]domain.Listing, error) {
	page = page.Normalise()

	params := db.ListListingsParams{
		ArtisanID: filter.ArtisanID,
		ClusterID: filter.ClusterID,
		CraftID:   filter.CraftID,
		After:     page.Cursor,
		PageSize:  page.Size,
	}
	if filter.State != nil {
		params.State = db.NullListingState{
			ListingState: db.ListingState(*filter.State),
			Valid:        true,
		}
	}
	if filter.Type != nil {
		params.Type = db.NullListingType{
			ListingType: db.ListingType(*filter.Type),
			Valid:       true,
		}
	}

	rows, err := r.q.ListListings(ctx, params)
	if err != nil {
		return nil, translate(err, "listings")
	}
	out := make([]domain.Listing, 0, len(rows))
	for _, row := range rows {
		out = append(out, listingFromRow(row))
	}
	return out, nil
}

// GetListingByProduct finds the listing already drafted for a product, which is
// how a redelivered pipeline run avoids drafting a second one.
func (r *Repo) GetListingByProduct(ctx context.Context, productID uuid.UUID) (domain.Listing, error) {
	row, err := r.q.GetListingByProduct(ctx, productID)
	if err != nil {
		return domain.Listing{}, translate(err, "listing for product")
	}
	return listingFromRow(row), nil
}

// SetListingNeedsDescription flags a draft as waiting on the artisan's own words.
func (r *Repo) SetListingNeedsDescription(ctx context.Context, listingID uuid.UUID, needs bool) error {
	_, err := r.q.SetListingNeedsDescription(ctx, db.SetListingNeedsDescriptionParams{
		ID:               listingID,
		NeedsDescription: needs,
	})
	return translate(err, "listing")
}

// ListListingTranslations reads a listing's copy in every language it has.
func (r *Repo) ListListingTranslations(ctx context.Context, listingID uuid.UUID) ([]domain.ListingTranslation, error) {
	rows, err := r.q.GetListingTranslations(ctx, listingID)
	if err != nil {
		return nil, translate(err, "listing translations")
	}
	out := make([]domain.ListingTranslation, 0, len(rows))
	for _, row := range rows {
		out = append(out, translationFromRow(row))
	}
	return out, nil
}

func listingFromRow(row db.Listing) domain.Listing {
	return domain.Listing{
		ID:               row.ID,
		ProductID:        row.ProductID,
		ArtisanID:        row.ArtisanID,
		Type:             domain.ListingType(row.Type),
		State:            domain.ListingState(row.State),
		PricePaise:       row.PricePaise,
		CurrencyCode:     row.CurrencyCode,
		StockQuantity:    row.StockQuantity,
		MinOrderQuantity: row.MinOrderQuantity,
		LeadTimeDays:     row.LeadTimeDays,
		CapacityPerMonth: row.CapacityPerMonth,
		AcceptingOrders:  row.AcceptingOrders,
		AdvancePct:       row.AdvancePct,
		Packaging: domain.Packaging{
			Fragile:               row.PackagingFragile,
			Oversized:             row.PackagingOversized,
			RequiresCustomCrating: row.PackagingRequiresCustomCrating,
			Packed: domain.Dimensions{
				LengthMM: row.PackedLengthMm,
				WidthMM:  row.PackedWidthMm,
				HeightMM: row.PackedHeightMm,
				WeightG:  row.PackedWeightG,
			},
		},
		ProvenanceID:     row.ProvenanceID,
		GICertified:      row.GiCertified,
		NeedsDescription: row.NeedsDescription,
		PublishedAt:      row.PublishedAt,
		SuspensionReason: row.SuspensionReason,
		CreatedBy:        row.CreatedBy,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

// --- translations ------------------------------------------------------------

// UpsertListingTranslation stores copy for one listing in one language.
func (t *Tx) UpsertListingTranslation(ctx context.Context, tr domain.ListingTranslation) (domain.ListingTranslation, error) {
	row, err := t.q.UpsertListingTranslation(ctx, db.UpsertListingTranslationParams{
		ListingID:        tr.ListingID,
		Language:         db.LanguageCode(tr.Language),
		Title:            tr.Title,
		Description:      tr.Description,
		Highlights:       orEmpty(tr.Highlights),
		MachineGenerated: tr.MachineGenerated,
		EditedBy:         tr.EditedBy,
	})
	if err != nil {
		return domain.ListingTranslation{}, translate(err, "listing translation")
	}
	return translationFromRow(row), nil
}

func translationFromRow(row db.ListingTranslation) domain.ListingTranslation {
	return domain.ListingTranslation{
		ListingID:        row.ListingID,
		Language:         string(row.Language),
		Title:            row.Title,
		Description:      row.Description,
		Highlights:       row.Highlights,
		MachineGenerated: row.MachineGenerated,
		EditedBy:         row.EditedBy,
		UpdatedAt:        row.UpdatedAt,
	}
}

// --- attributes --------------------------------------------------------------

// GetListingAttributes reads a listing's attributes outside any transaction,
// for read-only callers (e.g. provenance sealing) that don't need one.
func (r *Repo) GetListingAttributes(ctx context.Context, listingID uuid.UUID) ([]domain.ListingAttribute, error) {
	rows, err := r.q.GetListingAttributes(ctx, listingID)
	if err != nil {
		return nil, translate(err, "listing attributes")
	}
	out := make([]domain.ListingAttribute, 0, len(rows))
	for _, row := range rows {
		out = append(out, attributeFromRow(row))
	}
	return out, nil
}

// ListListingAttributes reads a listing's attributes inside the caller's
// transaction, which is how the service reads the current precedence before
// deciding what a model write may touch.
func (t *Tx) ListListingAttributes(ctx context.Context, listingID uuid.UUID) ([]domain.ListingAttribute, error) {
	rows, err := t.q.GetListingAttributes(ctx, listingID)
	if err != nil {
		return nil, translate(err, "listing attributes")
	}
	out := make([]domain.ListingAttribute, 0, len(rows))
	for _, row := range rows {
		out = append(out, attributeFromRow(row))
	}
	return out, nil
}

// UpsertListingAttribute writes one attribute row through the query that matches
// its source. The model's write is guarded in SQL as well as in the service: the
// INSERT selects nothing when the name already carries an artisan or curator
// value, so a concurrent model write cannot slip past the service's check.
func (t *Tx) UpsertListingAttribute(ctx context.Context, a domain.ListingAttribute) error {
	if a.Source == domain.SourceModel {
		if _, err := t.q.UpsertModelAttribute(ctx, db.UpsertModelAttributeParams{
			ID:         a.ID,
			ListingID:  a.ListingID,
			Name:       a.Name,
			Value:      a.Value,
			Confidence: a.Confidence,
		}); err != nil {
			return translate(err, "listing attribute")
		}
		return nil
	}

	err := t.q.UpsertAuthoritativeAttribute(ctx, db.UpsertAuthoritativeAttributeParams{
		ID:         a.ID,
		ListingID:  a.ListingID,
		Name:       a.Name,
		Value:      a.Value,
		Confidence: a.Confidence,
		Source:     db.AttributeSource(a.Source),
	})
	return translate(err, "listing attribute")
}

// DeleteListingAttributesByName removes the model's values for one attribute
// name, leaving artisan and curator values untouched.
func (t *Tx) DeleteListingAttributesByName(ctx context.Context, listingID uuid.UUID, name string) (int64, error) {
	n, err := t.q.DeleteListingAttributesByName(ctx, db.DeleteListingAttributesByNameParams{
		ListingID: listingID,
		Name:      name,
	})
	if err != nil {
		return 0, translate(err, "listing attributes")
	}
	return n, nil
}

func attributeFromRow(row db.ListingAttribute) domain.ListingAttribute {
	return domain.ListingAttribute{
		ID:         row.ID,
		ListingID:  row.ListingID,
		Name:       row.Name,
		Value:      row.Value,
		Confidence: row.Confidence,
		Source:     domain.AttributeSource(row.Source),
		CreatedAt:  row.CreatedAt,
	}
}

// --- media -------------------------------------------------------------------

// ReplaceListingMedia swaps a listing's whole media set. Delete-then-insert
// inside one transaction is what lets ordinals be reshuffled without tripping
// the unique index on (listing_id, ordinal).
func (t *Tx) ReplaceListingMedia(ctx context.Context, listingID uuid.UUID, items []domain.ListingMedia) error {
	if _, err := t.q.DeleteListingMedia(ctx, listingID); err != nil {
		return translate(err, "listing media")
	}
	for _, item := range items {
		if err := t.q.InsertListingMedia(ctx, db.InsertListingMediaParams{
			ListingID: listingID,
			MediaID:   item.MediaID,
			Ordinal:   item.Ordinal,
			Role:      db.ListingMediaRole(item.Role),
		}); err != nil {
			return translate(err, fmt.Sprintf("listing media %s", item.MediaID))
		}
	}
	return nil
}

// ListListingMedia reads a listing's media in display order.
func (r *Repo) ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]domain.ListingMedia, error) {
	rows, err := r.q.ListListingMedia(ctx, listingID)
	if err != nil {
		return nil, translate(err, "listing media")
	}
	out := make([]domain.ListingMedia, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ListingMedia{
			ListingID: row.ListingID,
			MediaID:   row.MediaID,
			Ordinal:   row.Ordinal,
			Role:      domain.ListingMediaRole(row.Role),
			Kind:      domain.MediaKind(row.Kind),
		})
	}
	return out, nil
}

// ListMediaOwnership reads who owns a set of media rows and what kind each is.
// Ids that do not exist are simply absent from the result; the service turns
// that into a not-found naming the id.
func (r *Repo) ListMediaOwnership(ctx context.Context, mediaIDs []uuid.UUID) ([]domain.MediaOwnership, error) {
	if len(mediaIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListMediaByIDs(ctx, mediaIDs)
	if err != nil {
		return nil, translate(err, "media")
	}
	out := make([]domain.MediaOwnership, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.MediaOwnership{
			MediaID:   row.ID,
			ArtisanID: row.ArtisanID,
			Kind:      domain.MediaKind(row.Kind),
		})
	}
	return out, nil
}

// GetSHGForArtisan reads the self-help group an artisan belongs to, which is how
// the catalog decides whether a group signatory may act for them.
func (r *Repo) GetSHGForArtisan(ctx context.Context, artisanID uuid.UUID) (domain.SelfHelpGroup, error) {
	row, err := r.q.GetShgForArtisan(ctx, artisanID)
	if err != nil {
		return domain.SelfHelpGroup{}, translate(err, "self-help group")
	}
	return shgFromRow(row), nil
}

// orEmpty replaces a nil slice with an empty one, because a NOT NULL text[]
// column rejects NULL but is happy with '{}'.
func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
