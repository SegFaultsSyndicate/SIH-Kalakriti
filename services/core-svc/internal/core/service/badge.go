// services/core-svc/internal/core/service/badge.go

package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// BadgeStore is the persistence surface the badge service depends on.
type BadgeStore interface {
	ListBadgeCatalog(ctx context.Context) ([]domain.Badge, error)
	ListArtisanBadges(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanBadge, error)
	GetBadgeProgress(ctx context.Context, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error)
	RecomputeAndGrant(ctx context.Context, artisanID uuid.UUID, metric domain.BadgeMetric, count int64) ([]domain.Badge, error)
	CountPublishedListings(ctx context.Context, artisanID uuid.UUID) (int64, error)
	CountSealedProvenance(ctx context.Context, artisanID uuid.UUID) (int64, error)
	CountAcceptedLots(ctx context.Context, artisanID uuid.UUID) (int64, error)
	CountCompletedLots(ctx context.Context, artisanID uuid.UUID) (int64, error)
	GetBadgeByCode(ctx context.Context, code string, tier *domain.BadgeTier) (domain.Badge, error)
	GrantConferredBadge(ctx context.Context, in domain.GrantBadgeInput) (domain.ArtisanBadge, error)
	RevokeBadge(ctx context.Context, in domain.RevokeBadgeInput) (bool, error)
}

// Badges is the service for the artisan badge catalog, grants, and
// activity-based auto-grants.
type Badges struct {
	store BadgeStore
	log   *slog.Logger
}

// NewBadges builds the badge service.
func NewBadges(store BadgeStore, log *slog.Logger) *Badges {
	if log == nil {
		log = slog.Default()
	}
	return &Badges{store: store, log: log}
}

// ListBadgeCatalog is public reference data -- no principal required.
func (b *Badges) ListBadgeCatalog(ctx context.Context) ([]domain.Badge, error) {
	return b.store.ListBadgeCatalog(ctx)
}

// ListArtisanBadges is public -- it backs the buyer storefront's badge row,
// which an anonymous buyer must be able to see.
func (b *Badges) ListArtisanBadges(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanBadge, error) {
	return b.store.ListArtisanBadges(ctx, artisanID)
}

// GetBadgeProgress requires the caller to be the artisan whose progress is
// being read -- unlike grants, in-progress counters are not public.
func (b *Badges) GetBadgeProgress(ctx context.Context, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error) {
	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if principal.Subject != artisanID.String() {
		return nil, pkgdomain.Forbidden("cannot read another artisan's badge progress")
	}
	return b.store.GetBadgeProgress(ctx, artisanID)
}

// GrantBadge confers a CONFERRED badge. Only a MINISTRY principal may call
// this, and only against a badge whose catalog kind is CONFERRED -- EARNED
// badges are only ever written by the Track* methods below.
func (b *Badges) GrantBadge(ctx context.Context, in domain.GrantBadgeInput) (domain.ArtisanBadge, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return domain.ArtisanBadge{}, err
	}
	if err := in.Validate(); err != nil {
		return domain.ArtisanBadge{}, err
	}
	badge, err := b.badgeByID(ctx, in.BadgeID)
	if err != nil {
		return domain.ArtisanBadge{}, err
	}
	if badge.Kind != domain.BadgeKindConferred {
		return domain.ArtisanBadge{}, pkgdomain.InvalidInput("badge " + badge.Code + " is earned automatically and cannot be granted directly")
	}
	return b.store.GrantConferredBadge(ctx, in)
}

// RevokeBadge revokes a badge grant. MINISTRY only.
func (b *Badges) RevokeBadge(ctx context.Context, in domain.RevokeBadgeInput) (bool, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return false, err
	}
	if err := in.Validate(); err != nil {
		return false, err
	}
	return b.store.RevokeBadge(ctx, in)
}

// badgeByID is a small helper: the store has no GetBadgeByID, only
// GetBadgeByCode, so GrantBadge instead reads the full catalog and finds the
// row -- the catalog is 14 rows, so this is cheap, and it avoids adding a
// query used from exactly one call site.
func (b *Badges) badgeByID(ctx context.Context, id uuid.UUID) (domain.Badge, error) {
	catalog, err := b.store.ListBadgeCatalog(ctx)
	if err != nil {
		return domain.Badge{}, err
	}
	for _, badge := range catalog {
		if badge.ID == id {
			return badge, nil
		}
	}
	return domain.Badge{}, pkgdomain.NotFound("badge not found")
}

// TrackListingPublished recomputes LISTINGS_PUBLISHED from the real count of
// the artisan's published listings and grants any newly-crossed tier. Called
// only from the Kafka consumer (handler.CatalogListingPublishedHandler),
// never from a network-facing RPC, so it takes no principal.
func (b *Badges) TrackListingPublished(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	count, err := b.store.CountPublishedListings(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	return b.recomputeAndLog(ctx, artisanID, domain.MetricListingsPublished, count)
}

// TrackProvenanceSealed recomputes PROVENANCE_SEALED.
func (b *Badges) TrackProvenanceSealed(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	count, err := b.store.CountSealedProvenance(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	return b.recomputeAndLog(ctx, artisanID, domain.MetricProvenanceSealed, count)
}

// TrackLotAccepted recomputes LOTS_ACCEPTED. No badge in the current catalog
// is keyed on this metric yet -- it is tracked so a future badge can be
// seeded against it without a code change, matching how CountAcceptedLots
// already exists in the repo (see spec's LOTS_ACCEPTED enum value).
func (b *Badges) TrackLotAccepted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	count, err := b.store.CountAcceptedLots(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	return b.recomputeAndLog(ctx, artisanID, domain.MetricLotsAccepted, count)
}

// TrackLotCompleted recomputes LOTS_COMPLETED (the order_fulfiller family).
func (b *Badges) TrackLotCompleted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	count, err := b.store.CountCompletedLots(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	return b.recomputeAndLog(ctx, artisanID, domain.MetricLotsCompleted, count)
}

func (b *Badges) recomputeAndLog(ctx context.Context, artisanID uuid.UUID, metric domain.BadgeMetric, count int64) ([]domain.Badge, error) {
	granted, err := b.store.RecomputeAndGrant(ctx, artisanID, metric, count)
	if err != nil {
		return nil, err
	}
	for _, badge := range granted {
		b.log.InfoContext(ctx, "badge granted", "artisan_id", artisanID, "badge_code", badge.Code, "metric", metric, "count", count)
	}
	return granted, nil
}

// TrackLessonsCompleted recomputes LESSONS_COMPLETED (the digital_ready
// badge). The literacy service already holds the fresh count it just wrote,
// so it passes it in rather than this re-reading it.
func (b *Badges) TrackLessonsCompleted(ctx context.Context, artisanID uuid.UUID, count int64) ([]domain.Badge, error) {
	return b.recomputeAndLog(ctx, artisanID, domain.MetricLessonsCompleted, count)
}
