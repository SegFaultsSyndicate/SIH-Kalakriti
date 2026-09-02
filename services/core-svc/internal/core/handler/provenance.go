// services/core-svc/internal/core/handler/provenance.go
package handler

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	inferencev1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/inference/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// SealProvenance freezes provenance evidence for a listing and mints its
// verification certificate.
func (h *Catalog) SealProvenance(ctx context.Context, req *catalogv1.SealProvenanceRequest) (*catalogv1.SealProvenanceResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	mediaIDs, err := mediaRefIDs(req.GetMedia())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	in := domain.SealProvenanceInput{
		ListingID:        listingID,
		MediaIDs:         mediaIDs,
		ClaimedTechnique: req.GetClaimedTechnique(),
		CreatedBy:        principal.Subject,
	}

	stored, err := h.provenance.SealProvenance(ctx, in, req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	// product_id isn't on the stored record (the DB schema doesn't carry it),
	// so it's fetched fresh from the listing that was just sealed.
	listing, err := h.svc.GetListing(ctx, stored.ListingID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	observed, confidence, err := h.provenance.ObserveTechnique(ctx, stored.ListingID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	verdict := &inferencev1.TechniqueVerdict{
		Claimed:    req.GetClaimedTechnique(),
		Observed:   observed,
		Matches:    stored.TechniqueMatched,
		Confidence: confidence,
	}

	return &catalogv1.SealProvenanceResponse{
		Provenance: provenanceRecordToProto(stored, listing.ProductID.String(), req.GetMedia(), verdict, h.verifyBaseURL),
	}, nil
}

// GetProvenanceByShortCode looks up a sealed record by its public short
// code, e.g. the code printed on a verification tag. Returns what was
// actually frozen at seal time, not a recomputed inference verdict.
func (h *Catalog) GetProvenanceByShortCode(ctx context.Context, req *catalogv1.GetProvenanceByShortCodeRequest) (*catalogv1.GetProvenanceByShortCodeResponse, error) {
	rec, err := h.provenance.GetProvenanceByShortCode(ctx, req.GetShortCode())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.GetProvenanceByShortCodeResponse{Record: sealedProvenanceToProto(rec)}, nil
}

func sealedProvenanceToProto(rec domain.ProvenanceRecord) *catalogv1.SealedProvenance {
	return &catalogv1.SealedProvenance{
		Id:                 rec.ID.String(),
		ListingId:          rec.ListingID.String(),
		ArtisanId:          rec.ArtisanID.String(),
		CraftId:            rec.CraftID.String(),
		ContentHash:        rec.ContentHash,
		PreviousHash:       rec.PreviousHash,
		Signature:          rec.Signature,
		SignatureAlgorithm: rec.SignatureAlgo,
		PublicKeyId:        rec.PublicKeyID,
		ShortCode:          rec.ShortCode,
		TechniqueMatched:   rec.TechniqueMatched,
		MediaHashes:        rec.MediaHashes,
		SealedAt:           timestamppb.New(rec.SealedAt),
	}
}

// provenanceRecordToProto converts a sealed record to the wire type. media is
// the client's own declared evidence list, passed through as-is: the
// cryptographic authority is content_hash + signature, not this echo.
func provenanceRecordToProto(
	rec domain.ProvenanceRecord,
	productID string,
	media []*commonv1.MediaRef,
	verdict *inferencev1.TechniqueVerdict,
	verifyBaseURL string,
) *catalogv1.ProvenanceRecord {
	return &catalogv1.ProvenanceRecord{
		Id:                 rec.ID.String(),
		ListingId:          rec.ListingID.String(),
		ProductId:          productID,
		Media:              media,
		TechniqueVerdict:   verdict,
		ContentHash:        rec.ContentHash,
		PreviousHash:       rec.PreviousHash,
		Signature:          rec.Signature,
		SignatureAlgorithm: rec.SignatureAlgo,
		QrCode:             fmt.Sprintf("%s/v/%s", verifyBaseURL, rec.ShortCode),
		SealedAt:           timestamppb.New(rec.SealedAt),
		Audit: &commonv1.AuditMeta{
			CreatedAt: timestamppb.New(rec.CreatedAt),
			UpdatedAt: timestamppb.New(rec.CreatedAt),
			CreatedBy: rec.ArtisanID.String(),
		},
	}
}
