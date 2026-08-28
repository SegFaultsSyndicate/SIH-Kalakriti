// services/core-svc/internal/core/handler/curation.go
package handler

import (
	"context"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
	catalogv1 "github.com/segfaultsyndicate/kalakriti/pkg/pb/catalog/v1"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/domain"
	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/service"
)

// Curation implements catalog.v1.CurationService.
type Curation struct {
	catalogv1.UnimplementedCurationServiceServer
	svc *service.Catalog
}

// NewCuration builds the curation handler.
func NewCuration(svc *service.Catalog) *Curation { return &Curation{svc: svc} }

// UpsertListingAttributes stores attribute rows from one source. Which source
// may overwrite which is decided in the service, not here.
func (h *Curation) UpsertListingAttributes(
	ctx context.Context,
	req *catalogv1.UpsertListingAttributesRequest,
) (*catalogv1.UpsertListingAttributesResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	attrs := make([]domain.ListingAttribute, 0, len(req.GetAttributes()))
	for _, a := range req.GetAttributes() {
		attrs = append(attrs, domain.ListingAttribute{
			Name:       a.GetName(),
			Value:      a.GetValue(),
			Confidence: a.GetConfidence(),
		})
	}

	stored, err := h.svc.UpsertListingAttributes(ctx, listingID, attrs,
		attributeSourceFromProto(req.GetSource()), req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.UpsertListingAttributesResponse{Attributes: attributesToProto(stored)}, nil
}

// UpsertListingTranslation stores buyer-facing copy for one language.
func (h *Curation) UpsertListingTranslation(
	ctx context.Context,
	req *catalogv1.UpsertListingTranslationRequest,
) (*catalogv1.UpsertListingTranslationResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	stored, err := h.svc.UpsertListingTranslation(ctx, listingID,
		translationFromProto(req.GetTranslation()), req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.UpsertListingTranslationResponse{Translation: translationToProto(stored)}, nil
}

// AttachListingMedia replaces a listing's media set.
func (h *Curation) AttachListingMedia(
	ctx context.Context,
	req *catalogv1.AttachListingMediaRequest,
) (*catalogv1.AttachListingMediaResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	items := make([]domain.ListingMedia, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		mediaID, err := parseUUID("items.media_id", item.GetMediaId())
		if err != nil {
			return nil, pkgdomain.GRPCError(err)
		}
		items = append(items, domain.ListingMedia{
			MediaID: mediaID,
			Ordinal: item.GetOrdinal(),
			Role:    mediaRoleFromProto(item.GetRole()),
		})
	}

	stored, err := h.svc.AttachListingMedia(ctx,
		domain.AttachMediaInput{ListingID: listingID, Items: items}, req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.AttachListingMediaResponse{Items: listingMediaSetToProto(stored)}, nil
}

// SuspendListing withdraws a published listing.
func (h *Curation) SuspendListing(ctx context.Context, req *catalogv1.SuspendListingRequest) (*catalogv1.SuspendListingResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	listing, err := h.svc.SuspendListing(ctx, listingID, req.GetReason(), req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.SuspendListingResponse{Listing: listingToProto(listing, "")}, nil
}

// ReinstateListing returns a suspended listing to PUBLISHED.
func (h *Curation) ReinstateListing(ctx context.Context, req *catalogv1.ReinstateListingRequest) (*catalogv1.ReinstateListingResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	listing, err := h.svc.ReinstateListing(ctx, listingID, req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.ReinstateListingResponse{Listing: listingToProto(listing, "")}, nil
}
