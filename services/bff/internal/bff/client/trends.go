// services/bff/internal/bff/client/trends.go
package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	trendsv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/trends/v1"
)

// Trends is bff's outbound client to core-svc's TrendService.
type Trends struct {
	trends trendsv1.TrendServiceClient
}

// NewTrends builds the trends client.
func NewTrends(conn grpc.ClientConnInterface) *Trends {
	return &Trends{trends: trendsv1.NewTrendServiceClient(conn)}
}

// CreateTrendLink creates a new curated trend link.
func (t *Trends) CreateTrendLink(ctx context.Context, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	title, _ := fields["title"].(string)
	if title == "" {
		return nil, domain.InvalidInput("title: is required")
	}
	urlStr, _ := fields["url"].(string)
	if urlStr == "" {
		return nil, domain.InvalidInput("url: is required")
	}
	stStr, _ := fields["source_type"].(string)
	desc, _ := fields["description"].(string)
	autoFetch, _ := fields["auto_fetch_embed"].(bool)

	req := &trendsv1.CreateTrendLinkRequest{
		Title:          title,
		Url:            urlStr,
		SourceType:     b2bTrendSourceType(stStr),
		Description:    desc,
		AutoFetchEmbed: autoFetch,
	}

	if cid, ok := fields["craft_id"].(string); ok && cid != "" {
		req.CraftId = &cid
	}
	if thumb, ok := fields["thumbnail_url"].(string); ok && thumb != "" {
		req.ThumbnailUrl = &thumb
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := t.trends.CreateTrendLink(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}
	return trendLinkToMap(resp.GetTrendLink()), nil
}

// ListTrendLinks lists curated market trend links.
func (t *Trends) ListTrendLinks(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	req := &trendsv1.ListTrendLinksRequest{}
	if cid, ok := filters["craft_id"].(string); ok && cid != "" {
		req.CraftId = &cid
	}
	if st, ok := filters["source_type"].(string); ok && st != "" {
		stVal := b2bTrendSourceType(st)
		req.SourceType = &stVal
	}
	if ee, ok := filters["exclude_expired"].(bool); ok {
		req.ExcludeExpired = ee
	}
	if ps, ok := filters["page_size"].(float64); ok {
		req.Page = &commonv1.PageRequest{PageSize: int32(ps)}
	}
	if pt, ok := filters["page_token"].(string); ok && pt != "" {
		if req.Page == nil {
			req.Page = &commonv1.PageRequest{PageSize: 20}
		}
		req.Page.PageToken = pt
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := t.trends.ListTrendLinks(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]map[string]any, len(resp.GetTrendLinks()))
	for i, l := range resp.GetTrendLinks() {
		out[i] = trendLinkToMap(l)
	}
	return out, nil
}

// DeleteTrendLink removes a trend link.
func (t *Trends) DeleteTrendLink(ctx context.Context, trendLinkID string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	_, err := t.trends.DeleteTrendLink(ctx, &trendsv1.DeleteTrendLinkRequest{TrendLinkId: trendLinkID})
	return grpcErr(err)
}

// PinTrendLink toggles pinned status.
func (t *Trends) PinTrendLink(ctx context.Context, trendLinkID string, pinned bool) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := t.trends.PinTrendLink(ctx, &trendsv1.PinTrendLinkRequest{
		TrendLinkId: trendLinkID,
		Pinned:      pinned,
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	return trendLinkToMap(resp.GetTrendLink()), nil
}

func b2bTrendSourceType(s string) trendsv1.TrendSourceType {
	switch s {
	case "INSTAGRAM":
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_INSTAGRAM
	case "PINTEREST":
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_PINTEREST
	case "BLOG":
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_BLOG
	case "NEWS":
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_NEWS
	case "YOUTUBE":
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_YOUTUBE
	default:
		return trendsv1.TrendSourceType_TREND_SOURCE_TYPE_OTHER
	}
}

func trendLinkToMap(l *trendsv1.TrendLink) map[string]any {
	if l == nil {
		return nil
	}
	m := map[string]any{
		"id":                  l.GetId(),
		"title":               l.GetTitle(),
		"description":         l.GetDescription(),
		"url":                 l.GetUrl(),
		"source_type":         l.GetSourceType().String(),
		"pinned":              l.GetPinned(),
		"created_by":          l.GetCreatedBy(),
		"created_at":          l.GetCreatedAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
		"curator_role":        l.GetCuratorRole(),
		"curator_name":        l.GetCuratorName(),
		"embed_html":          l.GetEmbedHtml(),
		"embed_metadata_json": l.GetEmbedMetadataJson(),
	}
	if l.CraftId != nil {
		m["craft_id"] = *l.CraftId
	}
	if l.CraftName != nil && *l.CraftName != "" {
		m["craft_name"] = *l.CraftName
	}
	if l.ThumbnailUrl != nil {
		m["thumbnail_url"] = *l.ThumbnailUrl
	}
	if l.ExpiresAt != nil {
		m["expires_at"] = l.ExpiresAt.AsTime().Format("2006-01-02T15:04:05Z07:00")
	}
	return m
}
