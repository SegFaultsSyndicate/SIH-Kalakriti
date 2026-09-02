// services/core-svc/internal/core/handler/catalog.go
package handler

import (
	"context"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Catalog implements catalog.v1.CatalogService. verifyBaseURL is the public
// origin the QR/verification URL in a sealed record's response is built from
// (e.g. "https://kalakriti.in", matching bff's own BASE_URL).
type Catalog struct {
	catalogv1.UnimplementedCatalogServiceServer
	svc           *service.Catalog
	provenance    *service.Provenance
	verifyBaseURL string
}

// NewCatalog builds the catalog handler.
func NewCatalog(svc *service.Catalog, provenance *service.Provenance, verifyBaseURL string) *Catalog {
	return &Catalog{svc: svc, provenance: provenance, verifyBaseURL: verifyBaseURL}
}

// CreateProduct registers a physical item against an artisan.
func (h *Catalog) CreateProduct(ctx context.Context, req *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	craftID, err := parseUUID("craft_id", req.GetCraftId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	mediaIDs, err := mediaRefIDs(req.GetMedia())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	in := domain.CreateProductInput{
		ArtisanID:    artisanID,
		CraftID:      craftID,
		WorkingTitle: req.GetWorkingTitle(),
		Dimensions:   dimensionsFromProto(req.GetDimensions()),
		MediaIDs:     mediaIDs,
	}
	if voice := req.GetVoiceNote(); voice != nil {
		voiceID, err := parseUUID("voice_note.id", voice.GetId())
		if err != nil {
			return nil, pkgdomain.GRPCError(err)
		}
		in.VoiceNoteMediaID = &voiceID
	}

	product, err := h.svc.CreateProduct(ctx, in, req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.CreateProductResponse{Product: productToProto(product)}, nil
}

// UpsertListing creates or replaces the sellable offer for a product.
func (h *Catalog) UpsertListing(ctx context.Context, req *catalogv1.UpsertListingRequest) (*catalogv1.UpsertListingResponse, error) {
	listingID, err := parseOptionalUUID("listing_id", req.ListingId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	productID, err := parseUUID("product_id", req.GetProductId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	in := domain.UpsertListingInput{
		ListingID:        listingID,
		ProductID:        productID,
		Type:             listingTypeFromProto(req.GetType()),
		PricePaise:       req.GetPrice().GetAmountPaise(),
		CurrencyCode:     req.GetPrice().GetCurrencyCode(),
		StockQuantity:    req.StockQuantity,
		MinOrderQuantity: req.GetMinOrderQuantity(),
		// A listing with no made-to-order terms is still open for orders; the
		// terms message is what turns that off.
		AcceptingOrders: true,
		Packaging:       packagingFromProto(req.GetPackaging()),
		Translations:    translationsFromProto(req.GetTranslations()),
	}
	// An unset minimum means one unit, not "no orders at all".
	if in.MinOrderQuantity == 0 {
		in.MinOrderQuantity = 1
	}
	if terms := req.GetMadeToOrderTerms(); terms != nil {
		lead, capacity, advance := terms.GetLeadTimeDays(), terms.GetCapacityPerMonth(), terms.GetAdvancePct()
		in.LeadTimeDays, in.CapacityPerMonth, in.AdvancePct = &lead, &capacity, &advance
		in.AcceptingOrders = terms.GetAcceptingOrders()
	}

	listing, err := h.svc.UpsertListing(ctx, in, req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.UpsertListingResponse{Listing: listingToProto(listing, "")}, nil
}

// SubmitForApproval hands a draft to the artisan for sign-off. review_languages
// is accepted for forward compatibility; the artisan reviews whatever copy the
// listing carries, and batch 9 turns the field into a notification fan-out.
func (h *Catalog) SubmitForApproval(ctx context.Context, req *catalogv1.SubmitForApprovalRequest) (*catalogv1.SubmitForApprovalResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	listing, err := h.svc.SubmitForApproval(ctx, listingID, req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.SubmitForApprovalResponse{Listing: listingToProto(listing, "")}, nil
}

// ApproveListing records the artisan's sign-off and publishes.
func (h *Catalog) ApproveListing(ctx context.Context, req *catalogv1.ApproveListingRequest) (*catalogv1.ApproveListingResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	// The artisan id is optional on the wire: the service reads the owner off
	// the listing, and only checks the field when the caller supplied one.
	artisanID := uuid.Nil
	if req.GetArtisanId() != "" {
		if artisanID, err = parseUUID("artisan_id", req.GetArtisanId()); err != nil {
			return nil, pkgdomain.GRPCError(err)
		}
	}

	listing, err := h.svc.ApproveListing(ctx, listingID, artisanID,
		translationsFromProto(req.GetEditedTranslations()), req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.ApproveListingResponse{Listing: listingToProto(listing, "")}, nil
}

// GetListing fetches one listing, optionally with the product behind it.
func (h *Catalog) GetListing(ctx context.Context, req *catalogv1.GetListingRequest) (*catalogv1.GetListingResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	listing, err := h.svc.GetListing(ctx, listingID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	resp := &catalogv1.GetListingResponse{
		Listing: listingToProto(listing, languageNameFromProto(req.GetLanguage())),
	}
	if req.GetIncludeProduct() {
		product, err := h.svc.GetProduct(ctx, listing.ProductID)
		if err != nil {
			return nil, pkgdomain.GRPCError(err)
		}
		resp.Product = productToProto(product)
	}
	if req.GetIncludeProvenance() && listing.ProvenanceID != nil {
		resp.Provenance = h.hydrateProvenance(ctx, listing.ID, listing.ProductID)
	}
	return resp, nil
}

// ListListings pages through listings under the usual filters.
func (h *Catalog) ListListings(ctx context.Context, req *catalogv1.ListListingsRequest) (*catalogv1.ListListingsResponse, error) {
	artisanID, err := parseOptionalUUID("artisan_id", req.ArtisanId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	clusterID, err := parseOptionalUUID("cluster_id", req.ClusterId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	craftID, err := parseOptionalUUID("craft_id", req.CraftId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	page, err := pageFromProto(req.GetPage())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	filter := domain.ListingFilter{ArtisanID: artisanID, ClusterID: clusterID, CraftID: craftID}
	if state := listingStateFromProto(req.GetState()); state != "" {
		filter.State = &state
	}
	if listingType := listingTypeFromProto(req.GetType()); listingType != "" {
		filter.Type = &listingType
	}

	listings, err := h.svc.ListListings(ctx, filter, page)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	out := make([]*catalogv1.Listing, 0, len(listings))
	lastID := ""
	for _, l := range listings {
		out = append(out, listingToProto(l, ""))
		lastID = l.ID.String()
	}
	return &catalogv1.ListListingsResponse{
		Listings: out,
		Page:     nextPage(lastID, len(listings), page.Normalise().Size),
	}, nil
}
