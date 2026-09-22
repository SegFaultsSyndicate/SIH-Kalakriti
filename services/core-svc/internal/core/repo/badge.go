// services/core-svc/internal/core/repo/badge.go

package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// ListBadgeCatalog returns every active badge sorted by sort_order.
func (r *Repo) ListBadgeCatalog(ctx context.Context) ([]domain.Badge, error) {
	rows, err := r.q.ListBadgeCatalog(ctx)
	if err != nil {
		return nil, translate(err, "badge catalog")
	}
	out := make([]domain.Badge, len(rows))
	for i, row := range rows {
		out[i] = badgeFromRow(row)
	}
	return out, nil
}

// GetBadgeByCode looks up one catalog entry by its code and tier (tier nil
// for an untiered badge).
func (r *Repo) GetBadgeByCode(ctx context.Context, code string, tier *domain.BadgeTier) (domain.Badge, error) {
	var dbTier *db.BadgeTier
	if tier != nil {
		t := db.BadgeTier(*tier)
		dbTier = &t
	}
	row, err := r.q.GetBadgeByCode(ctx, db.GetBadgeByCodeParams{Code: code, Tier: dbTier})
	if err != nil {
		return domain.Badge{}, translate(err, "badge")
	}
	return badgeFromRow(row), nil
}

// ListArtisanBadges lists an artisan's active (non-revoked) grants, joined
// with their catalog entries.
func (r *Repo) ListArtisanBadges(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanBadge, error) {
	rows, err := r.q.ListArtisanBadges(ctx, artisanID)
	if err != nil {
		return nil, translate(err, "artisan badges")
	}
	out := make([]domain.ArtisanBadge, len(rows))
	for i, row := range rows {
		var tier *domain.BadgeTier
		if row.Tier != nil {
			t := domain.BadgeTier(*row.Tier)
			tier = &t
		}
		var metric *domain.BadgeMetric
		if row.Metric != nil {
			m := domain.BadgeMetric(*row.Metric)
			metric = &m
		}
		var evidence *string
		if len(row.Evidence) > 0 {
			s := string(row.Evidence)
			evidence = &s
		}
		out[i] = domain.ArtisanBadge{
			Badge: domain.Badge{
				ID: row.BadgeID, Code: row.Code, Kind: domain.BadgeKind(row.Kind),
				Tier: tier, IconName: row.IconName, Metric: metric,
				Threshold: row.Threshold, SortOrder: row.SortOrder,
			},
			GrantedAt: row.GrantedAt, GrantedBy: row.GrantedBy, Evidence: evidence,
		}
	}
	return out, nil
}

// GetBadgeProgress reads an artisan's current counters, one row per metric
// they have any activity in (a metric never touched simply has no row).
func (r *Repo) GetBadgeProgress(ctx context.Context, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error) {
	rows, err := r.q.GetBadgeProgress(ctx, artisanID)
	if err != nil {
		return nil, translate(err, "badge progress")
	}
	out := make([]domain.BadgeProgressEntry, len(rows))
	for i, row := range rows {
		out[i] = domain.BadgeProgressEntry{
			Metric: domain.BadgeMetric(row.Metric), Value: row.Value, UpdatedAt: row.UpdatedAt,
		}
	}
	return out, nil
}

// RecomputeAndGrant sets an artisan's progress counter for metric to the
// given (freshly recomputed) count, then grants every EARNED badge on that
// metric whose threshold is now met and which the artisan does not already
// hold. Both statements are naturally idempotent: an UPSERT that sets an
// absolute value, and an INSERT ... SELECT ... WHERE NOT EXISTS ... ON
// CONFLICT DO NOTHING. A Kafka redelivery calling this twice with the same
// count is a no-op the second time -- no explicit transaction or dedupe
// ledger is needed.
func (r *Repo) RecomputeAndGrant(ctx context.Context, artisanID uuid.UUID, metric domain.BadgeMetric, count int64) ([]domain.Badge, error) {
	if _, err := r.q.UpsertBadgeProgress(ctx, db.UpsertBadgeProgressParams{
		ArtisanID: artisanID, Metric: db.BadgeMetric(metric), Value: count,
	}); err != nil {
		return nil, translate(err, "badge progress")
	}

	m := db.BadgeMetric(metric)
	c := count
	rows, err := r.q.GrantEligibleEarnedBadges(ctx, db.GrantEligibleEarnedBadgesParams{
		ArtisanID: artisanID, Metric: &m, Value: &c,
	})
	if err != nil {
		return nil, translate(err, "badge grant")
	}
	return r.hydrateGrantedBadges(ctx, rows)
}

// hydrateGrantedBadges looks up each newly-granted row's catalog entry by id
// so the caller (and ultimately the consumer's log line / any future
// notification) has the badge's code, not just its id. GrantEligibleEarnedBadges
// returns artisan_badge rows, which do not carry the catalog columns.
func (r *Repo) hydrateGrantedBadges(ctx context.Context, rows []db.ArtisanBadge) ([]domain.Badge, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	catalog, err := r.q.ListBadgeCatalog(ctx)
	if err != nil {
		return nil, translate(err, "badge catalog")
	}
	byID := make(map[uuid.UUID]db.Badge, len(catalog))
	for _, b := range catalog {
		byID[b.ID] = b
	}
	out := make([]domain.Badge, 0, len(rows))
	for _, row := range rows {
		if b, ok := byID[row.BadgeID]; ok {
			out = append(out, badgeFromRow(b))
		}
	}
	return out, nil
}

// CountPublishedListings, CountSealedProvenance, CountAcceptedLots, and
// CountCompletedLots are the source-of-truth counts RecomputeAndGrant's
// caller (service.Badges, see Task 6) passes in as count.

func (r *Repo) CountPublishedListings(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	n, err := r.q.CountPublishedListings(ctx, artisanID)
	return n, translate(err, "published listing count")
}

func (r *Repo) CountSealedProvenance(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	n, err := r.q.CountSealedProvenance(ctx, artisanID)
	return n, translate(err, "sealed provenance count")
}

func (r *Repo) CountAcceptedLots(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	n, err := r.q.CountAcceptedLots(ctx, artisanID)
	return n, translate(err, "accepted lot count")
}

func (r *Repo) CountCompletedLots(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	n, err := r.q.CountCompletedLots(ctx, artisanID)
	return n, translate(err, "completed lot count")
}

// GrantConferredBadge writes (or re-writes, clearing any prior revoke) an
// admin-conferred grant.
func (r *Repo) GrantConferredBadge(ctx context.Context, in domain.GrantBadgeInput) (domain.ArtisanBadge, error) {
	var evidence []byte
	if in.Evidence != nil {
		evidence = []byte(*in.Evidence)
	}
	row, err := r.q.InsertConferredGrant(ctx, db.InsertConferredGrantParams{
		ArtisanID: in.ArtisanID, BadgeID: in.BadgeID, GrantedBy: in.GrantedBy, Evidence: evidence,
	})
	if err != nil {
		return domain.ArtisanBadge{}, translate(err, "badge grant")
	}
	badge, err := r.hydrateGrantedBadges(ctx, []db.ArtisanBadge{row})
	if err != nil || len(badge) == 0 {
		return domain.ArtisanBadge{}, translate(err, "badge")
	}
	return domain.ArtisanBadge{Badge: badge[0], GrantedAt: row.GrantedAt, GrantedBy: row.GrantedBy}, nil
}

// RevokeBadge marks a grant revoked. Returns false if there was no active
// grant to revoke (already revoked, or never granted).
func (r *Repo) RevokeBadge(ctx context.Context, in domain.RevokeBadgeInput) (bool, error) {
	n, err := r.q.RevokeGrant(ctx, db.RevokeGrantParams{
		ArtisanID: in.ArtisanID, BadgeID: in.BadgeID, RevokedBy: &in.RevokedBy, RevokeReason: &in.Reason,
	})
	if err != nil {
		return false, translate(err, "badge revoke")
	}
	return n > 0, nil
}

func badgeFromRow(row db.Badge) domain.Badge {
	var tier *domain.BadgeTier
	if row.Tier != nil {
		t := domain.BadgeTier(*row.Tier)
		tier = &t
	}
	var metric *domain.BadgeMetric
	if row.Metric != nil {
		m := domain.BadgeMetric(*row.Metric)
		metric = &m
	}
	return domain.Badge{
		ID: row.ID, Code: row.Code, Kind: domain.BadgeKind(row.Kind), Tier: tier,
		IconName: row.IconName, Metric: metric, Threshold: row.Threshold, SortOrder: row.SortOrder,
	}
}
