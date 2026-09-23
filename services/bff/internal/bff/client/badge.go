// services/bff/internal/bff/client/badge.go
package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	badgesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/badges/v1"
)

// Badges is bff's outbound client to core-svc's BadgeService.
type Badges struct {
	badges badgesv1.BadgeServiceClient
}

// NewBadges builds the badges client.
func NewBadges(conn grpc.ClientConnInterface) *Badges {
	return &Badges{badges: badgesv1.NewBadgeServiceClient(conn)}
}

// ListBadgeCatalog lists the active badge catalog.
func (b *Badges) ListBadgeCatalog(ctx context.Context) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	resp, err := b.badges.ListBadgeCatalog(ctx, &badgesv1.ListBadgeCatalogRequest{})
	if err != nil {
		return nil, grpcErr(err)
	}
	out := make([]map[string]any, len(resp.Badges))
	for i, badge := range resp.Badges {
		out[i] = badgeToMap(badge)
	}
	return out, nil
}

// ListArtisanBadges lists one artisan's active grants.
func (b *Badges) ListArtisanBadges(ctx context.Context, artisanID string) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	resp, err := b.badges.ListArtisanBadges(ctx, &badgesv1.ListArtisanBadgesRequest{ArtisanId: artisanID})
	if err != nil {
		return nil, grpcErr(err)
	}
	out := make([]map[string]any, len(resp.ArtisanBadges))
	for i, ab := range resp.ArtisanBadges {
		out[i] = map[string]any{
			"badge":      badgeToMap(ab.Badge),
			"granted_at": ab.GetGrantedAt().AsTime(),
			"granted_by": ab.GrantedBy,
		}
	}
	return out, nil
}

// GetBadgeProgress returns the caller's own progress entries.
func (b *Badges) GetBadgeProgress(ctx context.Context, artisanID string) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	resp, err := b.badges.GetBadgeProgress(ctx, &badgesv1.GetBadgeProgressRequest{ArtisanId: artisanID})
	if err != nil {
		return nil, grpcErr(err)
	}
	out := make([]map[string]any, len(resp.Entries))
	for i, e := range resp.Entries {
		out[i] = map[string]any{
			"metric": trimEnumPrefix(e.Metric.String(), "BADGE_METRIC_"), "value": e.Value, "updated_at": e.GetUpdatedAt().AsTime(),
		}
	}
	return out, nil
}

// GrantBadge confers a CONFERRED badge on an artisan.
func (b *Badges) GrantBadge(ctx context.Context, artisanID string, fields map[string]any) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	badgeID, _ := fields["badge_id"].(string)
	if badgeID == "" {
		return nil, domain.InvalidInput("badge_id: is required")
	}
	req := &badgesv1.GrantBadgeRequest{ArtisanId: artisanID, BadgeId: badgeID}
	if evidence, ok := fields["evidence_json"].(string); ok && evidence != "" {
		req.EvidenceJson = &evidence
	}
	resp, err := b.badges.GrantBadge(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}
	return map[string]any{
		"badge":      badgeToMap(resp.ArtisanBadge.Badge),
		"granted_at": resp.ArtisanBadge.GetGrantedAt().AsTime(),
		"granted_by": resp.ArtisanBadge.GrantedBy,
	}, nil
}

// RevokeBadge revokes a badge grant.
func (b *Badges) RevokeBadge(ctx context.Context, artisanID, badgeID string, fields map[string]any) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	reason, _ := fields["reason"].(string)
	if reason == "" {
		return domain.InvalidInput("reason: is required")
	}
	_, err := b.badges.RevokeBadge(ctx, &badgesv1.RevokeBadgeRequest{ArtisanId: artisanID, BadgeId: badgeID, Reason: reason})
	return grpcErr(err)
}

func badgeToMap(b *badgesv1.Badge) map[string]any {
	out := map[string]any{
		"id": b.Id, "code": b.Code, "kind": trimEnumPrefix(b.Kind.String(), "BADGE_KIND_"), "icon_name": b.IconName, "sort_order": b.SortOrder,
	}
	if b.Tier != badgesv1.BadgeTier_BADGE_TIER_UNSPECIFIED {
		out["tier"] = trimEnumPrefix(b.Tier.String(), "BADGE_TIER_")
	}
	if b.Metric != badgesv1.BadgeMetric_BADGE_METRIC_UNSPECIFIED {
		out["metric"] = trimEnumPrefix(b.Metric.String(), "BADGE_METRIC_")
	}
	if b.Threshold != nil {
		out["threshold"] = *b.Threshold
	}
	return out
}

// GetBadgeByCode looks up one catalog entry by its code.
func (b *Badges) GetBadgeByCode(ctx context.Context, code string) (map[string]any, error) {
	catalog, err := b.ListBadgeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	for _, badge := range catalog {
		if badge["code"] == code {
			return badge, nil
		}
	}
	return nil, domain.NotFound("badge not found: " + code)
}
