// services/core-svc/internal/core/handler/trends.go

package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	trendsv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/trends/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Trends implements trends.v1.TrendService.
type Trends struct {
	trendsv1.UnimplementedTrendServiceServer
	svc *service.Trends
}

// NewTrends builds the trends handler.
func NewTrends(svc *service.Trends) *Trends {
	return &Trends{svc: svc}
}

// CreateTrendLink creates a new curated trend link.
func (h *Trends) CreateTrendLink(ctx context.Context, req *trendsv1.CreateTrendLinkRequest) (*trendsv1.CreateTrendLinkResponse, error) {
	var craftID *uuid.UUID
	if req.CraftId != nil {
		if id, err := uuid.Parse(*req.CraftId); err == nil {
			craftID = &id
		}
	}

	link, err := h.svc.CreateTrendLink(ctx, domain.CreateTrendLinkInput{
		Title:          req.GetTitle(),
		Description:    req.GetDescription(),
		URL:            req.GetUrl(),
		SourceType:     domain.TrendSourceType(trendSourceTypeFromProto(req.GetSourceType())),
		CraftID:        craftID,
		ThumbnailURL:   req.ThumbnailUrl,
		AutoFetchEmbed: req.GetAutoFetchEmbed(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &trendsv1.CreateTrendLinkResponse{
		TrendLink: trendLinkToProto(link),
	}, nil
}

// ListTrendLinks lists curated market trend links.
func (h *Trends) ListTrendLinks(ctx context.Context, req *trendsv1.ListTrendLinksRequest) (*trendsv1.ListTrendLinksResponse, error) {
	var craftID *uuid.UUID
	if req.CraftId != nil {
		if id, err := uuid.Parse(*req.CraftId); err == nil {
			craftID = &id
		}
	}

	var sourceType *domain.TrendSourceType
	if req.SourceType != nil {
		st := domain.TrendSourceType(trendSourceTypeFromProto(req.GetSourceType()))
		sourceType = &st
	}

	page, err := pageFromProto(req.GetPage())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	links, err := h.svc.ListTrendLinks(ctx, domain.TrendFilter{
		CraftID:        craftID,
		SourceType:     sourceType,
		ExcludeExpired: req.GetExcludeExpired(),
		Page:           page,
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	pbLinks := make([]*trendsv1.TrendLink, len(links))
	for i, l := range links {
		pbLinks[i] = trendLinkToProto(l)
	}

	var lastID string
	if len(links) > 0 {
		lastID = links[len(links)-1].ID.String()
	}

	return &trendsv1.ListTrendLinksResponse{
		TrendLinks: pbLinks,
		Page:       nextPage(lastID, len(links), req.GetPage().GetPageSize()),
	}, nil
}

// DeleteTrendLink removes a trend link.
func (h *Trends) DeleteTrendLink(ctx context.Context, req *trendsv1.DeleteTrendLinkRequest) (*trendsv1.DeleteTrendLinkResponse, error) {
	id, err := uuid.Parse(req.GetTrendLinkId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	err = h.svc.DeleteTrendLink(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &trendsv1.DeleteTrendLinkResponse{}, nil
}

// PinTrendLink pins or unpins a trend link.
func (h *Trends) PinTrendLink(ctx context.Context, req *trendsv1.PinTrendLinkRequest) (*trendsv1.PinTrendLinkResponse, error) {
	id, err := uuid.Parse(req.GetTrendLinkId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	link, err := h.svc.PinTrendLink(ctx, id, req.GetPinned())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &trendsv1.PinTrendLinkResponse{
		TrendLink: trendLinkToProto(link),
	}, nil
}

func trendSourceTypeFromProto(t trendsv1.TrendSourceType) string {
	switch t {
	case trendsv1.TrendSourceType_TREND_SOURCE_TYPE_INSTAGRAM:
		return "INSTAGRAM"
	case trendsv1.TrendSourceType_TREND_SOURCE_TYPE_PINTEREST:
		return "PINTEREST"
	case trendsv1.TrendSourceType_TREND_SOURCE_TYPE_BLOG:
		return "BLOG"
	case trendsv1.TrendSourceType_TREND_SOURCE_TYPE_NEWS:
		return "NEWS"
	case trendsv1.TrendSourceType_TREND_SOURCE_TYPE_YOUTUBE:
		return "YOUTUBE"
	default:
		return "OTHER"
	}
}

func trendSourceTypeToProto(t domain.TrendSourceType) trendsv1.TrendSourceType {
	switch t {
	case domain.TrendSourceInstagram:
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_INSTAGRAM
	case domain.TrendSourcePinterest:
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_PINTEREST
	case domain.TrendSourceBlog:
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_BLOG
	case domain.TrendSourceNews:
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_NEWS
	case domain.TrendSourceYouTube:
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_YOUTUBE
	default:
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_OTHER
	}
}

func trendLinkToProto(l domain.TrendLink) *trendsv1.TrendLink {
	var craftID *string
	if l.CraftID != nil {
		s := l.CraftID.String()
		craftID = &s
	}

	var expiresAt *timestamppb.Timestamp
	if l.ExpiresAt != nil {
		expiresAt = timestamppb.New(*l.ExpiresAt)
	}

	return &trendsv1.TrendLink{
		Id:                l.ID.String(),
		Title:             l.Title,
		Description:       l.Description,
		Url:               l.URL,
		SourceType:        trendSourceTypeToProto(l.SourceType),
		CraftId:           craftID,
		CraftName:         l.CraftName,
		ThumbnailUrl:      l.ThumbnailURL,
		EmbedHtml:         l.EmbedHTML,
		EmbedMetadataJson: l.EmbedMetadataJSON,
		Pinned:            l.Pinned,
		CreatedBy:         l.CreatedBy,
		ExpiresAt:         expiresAt,
		CreatedAt:         timestamppb.New(l.CreatedAt),
		CuratorRole:       l.CuratorRole,
		CuratorName:       l.CuratorName,
	}
}
