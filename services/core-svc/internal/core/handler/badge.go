// services/core-svc/internal/core/handler/badge.go

package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	badgesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/badges/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Badges implements badges.v1.BadgeService.
type Badges struct {
	badgesv1.UnimplementedBadgeServiceServer
	svc *service.Badges
}

// NewBadges builds the badge handler.
func NewBadges(svc *service.Badges) *Badges {
	return &Badges{svc: svc}
}

func (h *Badges) ListBadgeCatalog(ctx context.Context, req *badgesv1.ListBadgeCatalogRequest) (*badgesv1.ListBadgeCatalogResponse, error) {
	catalog, err := h.svc.ListBadgeCatalog(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*badgesv1.Badge, len(catalog))
	for i, b := range catalog {
		out[i] = toProtoBadge(b)
	}
	return &badgesv1.ListBadgeCatalogResponse{Badges: out}, nil
}

func (h *Badges) ListArtisanBadges(ctx context.Context, req *badgesv1.ListArtisanBadgesRequest) (*badgesv1.ListArtisanBadgesResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	grants, err := h.svc.ListArtisanBadges(ctx, artisanID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*badgesv1.ArtisanBadge, len(grants))
	for i, g := range grants {
		out[i] = &badgesv1.ArtisanBadge{
			Badge: toProtoBadge(g.Badge), GrantedAt: timestamppb.New(g.GrantedAt), GrantedBy: g.GrantedBy,
		}
	}
	return &badgesv1.ListArtisanBadgesResponse{ArtisanBadges: out}, nil
}

func (h *Badges) GetBadgeProgress(ctx context.Context, req *badgesv1.GetBadgeProgressRequest) (*badgesv1.GetBadgeProgressResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	entries, err := h.svc.GetBadgeProgress(ctx, artisanID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*badgesv1.BadgeProgressEntry, len(entries))
	for i, e := range entries {
		out[i] = &badgesv1.BadgeProgressEntry{
			Metric: toProtoMetric(e.Metric), Value: e.Value, UpdatedAt: timestamppb.New(e.UpdatedAt),
		}
	}
	return &badgesv1.GetBadgeProgressResponse{Entries: out}, nil
}

func (h *Badges) GrantBadge(ctx context.Context, req *badgesv1.GrantBadgeRequest) (*badgesv1.GrantBadgeResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	badgeID, err := parseUUID("badge_id", req.GetBadgeId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	grant, err := h.svc.GrantBadge(ctx, domain.GrantBadgeInput{
		ArtisanID: artisanID, BadgeID: badgeID, GrantedBy: principalSubjectOrEmpty(ctx), Evidence: req.EvidenceJson,
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &badgesv1.GrantBadgeResponse{ArtisanBadge: &badgesv1.ArtisanBadge{
		Badge: toProtoBadge(grant.Badge), GrantedAt: timestamppb.New(grant.GrantedAt), GrantedBy: grant.GrantedBy,
	}}, nil
}

func (h *Badges) RevokeBadge(ctx context.Context, req *badgesv1.RevokeBadgeRequest) (*badgesv1.RevokeBadgeResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	badgeID, err := parseUUID("badge_id", req.GetBadgeId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	revoked, err := h.svc.RevokeBadge(ctx, domain.RevokeBadgeInput{
		ArtisanID: artisanID, BadgeID: badgeID, RevokedBy: principalSubjectOrEmpty(ctx), Reason: req.GetReason(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &badgesv1.RevokeBadgeResponse{Revoked: revoked}, nil
}

func toProtoBadge(b domain.Badge) *badgesv1.Badge {
	out := &badgesv1.Badge{
		Id: b.ID.String(), Code: b.Code, Kind: toProtoKind(b.Kind), IconName: b.IconName, SortOrder: b.SortOrder,
	}
	if b.Tier != nil {
		out.Tier = toProtoTier(*b.Tier)
	}
	if b.Metric != nil {
		out.Metric = toProtoMetric(*b.Metric)
	}
	out.Threshold = b.Threshold
	return out
}

func toProtoKind(k domain.BadgeKind) badgesv1.BadgeKind {
	if k == domain.BadgeKindConferred {
		return badgesv1.BadgeKind_BADGE_KIND_CONFERRED
	}
	return badgesv1.BadgeKind_BADGE_KIND_EARNED
}

func toProtoTier(t domain.BadgeTier) badgesv1.BadgeTier {
	switch t {
	case domain.BadgeTierBronze:
		return badgesv1.BadgeTier_BADGE_TIER_BRONZE
	case domain.BadgeTierSilver:
		return badgesv1.BadgeTier_BADGE_TIER_SILVER
	case domain.BadgeTierGold:
		return badgesv1.BadgeTier_BADGE_TIER_GOLD
	default:
		return badgesv1.BadgeTier_BADGE_TIER_UNSPECIFIED
	}
}

func toProtoMetric(m domain.BadgeMetric) badgesv1.BadgeMetric {
	switch m {
	case domain.MetricListingsPublished:
		return badgesv1.BadgeMetric_BADGE_METRIC_LISTINGS_PUBLISHED
	case domain.MetricProvenanceSealed:
		return badgesv1.BadgeMetric_BADGE_METRIC_PROVENANCE_SEALED
	case domain.MetricLotsAccepted:
		return badgesv1.BadgeMetric_BADGE_METRIC_LOTS_ACCEPTED
	case domain.MetricLotsCompleted:
		return badgesv1.BadgeMetric_BADGE_METRIC_LOTS_COMPLETED
	case domain.MetricLessonsCompleted:
		return badgesv1.BadgeMetric_BADGE_METRIC_LESSONS_COMPLETED
	default:
		return badgesv1.BadgeMetric_BADGE_METRIC_UNSPECIFIED
	}
}

func principalSubjectOrEmpty(ctx context.Context) string {
	if p, ok := auth.PrincipalFrom(ctx); ok {
		return p.Subject
	}
	return ""
}
