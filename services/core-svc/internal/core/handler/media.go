// services/core-svc/internal/core/handler/media.go
package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Media implements catalog.v1.MediaService.
type Media struct {
	catalogv1.UnimplementedMediaServiceServer
	svc *service.Media
}

// NewMedia builds the media handler.
func NewMedia(svc *service.Media) *Media { return &Media{svc: svc} }

// RequestUpload mints a presigned PUT URL for a new asset.
func (h *Media) RequestUpload(ctx context.Context, req *catalogv1.RequestUploadRequest) (*catalogv1.RequestUploadResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	productID, err := parseOptionalUUID("product_id", req.ProductId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	ticket, err := h.svc.RequestUpload(ctx, domain.RequestUploadInput{
		ArtisanID:   artisanID,
		ContentType: req.GetContentType(),
		SizeBytes:   req.GetSizeBytes(),
		SHA256Hex:   req.Sha256Hex,
		ProductID:   productID,
		Source:      req.GetSource(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	resp := &catalogv1.RequestUploadResponse{
		MediaId:      ticket.MediaID.String(),
		ObjectKey:    ticket.ObjectKey,
		UploadUrl:    ticket.UploadURL,
		Deduplicated: ticket.Deduplicated,
	}
	if !ticket.ExpiresAt.IsZero() {
		resp.ExpiresAt = timestamppb.New(ticket.ExpiresAt)
	}
	return resp, nil
}

// ConfirmUpload verifies the object landed and moves the row to UPLOADED.
func (h *Media) ConfirmUpload(ctx context.Context, req *catalogv1.ConfirmUploadRequest) (*catalogv1.ConfirmUploadResponse, error) {
	mediaID, err := parseUUID("media_id", req.GetMediaId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	media, err := h.svc.ConfirmUpload(ctx, mediaID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.ConfirmUploadResponse{Media: mediaToProto(media)}, nil
}

// GetMedia fetches one media row.
func (h *Media) GetMedia(ctx context.Context, req *catalogv1.GetMediaRequest) (*catalogv1.GetMediaResponse, error) {
	mediaID, err := parseUUID("media_id", req.GetMediaId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	media, err := h.svc.GetMedia(ctx, mediaID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.GetMediaResponse{Media: mediaToProto(media)}, nil
}

// GetMediaURL returns a time-limited download URL.
func (h *Media) GetMediaURL(ctx context.Context, req *catalogv1.GetMediaURLRequest) (*catalogv1.GetMediaURLResponse, error) {
	mediaID, err := parseUUID("media_id", req.GetMediaId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	url, expiresAt, err := h.svc.GetMediaURL(ctx, mediaID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.GetMediaURLResponse{Url: url, ExpiresAt: timestamppb.New(expiresAt)}, nil
}

// mediaStateToProto maps the stored lifecycle state onto the wire enum.
func mediaStateToProto(s domain.MediaState) catalogv1.MediaState {
	if v, ok := catalogv1.MediaState_value["MEDIA_STATE_"+string(s)]; ok {
		return catalogv1.MediaState(v)
	}
	return catalogv1.MediaState_MEDIA_STATE_UNSPECIFIED
}

// mediaToProto converts a domain media row to the wire type.
func mediaToProto(m domain.Media) *catalogv1.Media {
	out := &catalogv1.Media{
		Id:                m.ID.String(),
		ArtisanId:         m.ArtisanID.String(),
		Bucket:            m.Bucket,
		ObjectKey:         m.ObjectKey,
		EnhancedObjectKey: m.EnhancedObjectKey,
		Kind:              mediaKindToProto(m.Kind),
		MimeType:          m.MimeType,
		SizeBytes:         m.SizeBytes,
		Sha256Hex:         m.SHA256Hex,
		State:             mediaStateToProto(m.State),
		FailureReason:     m.FailureReason,
		Audit: &commonv1.AuditMeta{
			CreatedAt: timestamppb.New(m.CreatedAt),
			UpdatedAt: timestamppb.New(m.UploadedAt),
			CreatedBy: m.Source,
		},
	}
	if m.ProductID != nil {
		productID := m.ProductID.String()
		out.ProductId = &productID
	}
	if m.ConfirmedAt != nil {
		out.ConfirmedAt = timestamppb.New(*m.ConfirmedAt)
	}
	return out
}
