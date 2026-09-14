// services/core-svc/internal/core/service/badge_test.go
package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

type fakeBadgeStore struct {
	catalog            []domain.Badge
	countPublished     int64
	recomputeCalls     []domain.BadgeMetric
	grantedOnRecompute []domain.Badge
}

func (f *fakeBadgeStore) ListBadgeCatalog(ctx context.Context) ([]domain.Badge, error) {
	return f.catalog, nil
}
func (f *fakeBadgeStore) ListArtisanBadges(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanBadge, error) {
	return nil, nil
}
func (f *fakeBadgeStore) GetBadgeProgress(ctx context.Context, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error) {
	return nil, nil
}
func (f *fakeBadgeStore) RecomputeAndGrant(ctx context.Context, artisanID uuid.UUID, metric domain.BadgeMetric, count int64) ([]domain.Badge, error) {
	f.recomputeCalls = append(f.recomputeCalls, metric)
	return f.grantedOnRecompute, nil
}
func (f *fakeBadgeStore) CountPublishedListings(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	return f.countPublished, nil
}
func (f *fakeBadgeStore) CountSealedProvenance(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *fakeBadgeStore) CountAcceptedLots(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *fakeBadgeStore) CountCompletedLots(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *fakeBadgeStore) GetBadgeByCode(ctx context.Context, code string, tier *domain.BadgeTier) (domain.Badge, error) {
	return domain.Badge{}, nil
}
func (f *fakeBadgeStore) GrantConferredBadge(ctx context.Context, in domain.GrantBadgeInput) (domain.ArtisanBadge, error) {
	return domain.ArtisanBadge{}, nil
}
func (f *fakeBadgeStore) RevokeBadge(ctx context.Context, in domain.RevokeBadgeInput) (bool, error) {
	return false, nil
}

func TestBadges_TrackListingPublished_RecomputesFromRealCount(t *testing.T) {
	store := &fakeBadgeStore{countPublished: 5}
	svc := NewBadges(store, nil)

	_, err := svc.TrackListingPublished(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, []domain.BadgeMetric{domain.MetricListingsPublished}, store.recomputeCalls)
}

func TestBadges_GrantBadge_RejectsEarnedBadge(t *testing.T) {
	earnedBadge := domain.Badge{ID: uuid.New(), Code: "first_listing", Kind: domain.BadgeKindEarned}
	store := &fakeBadgeStore{catalog: []domain.Badge{earnedBadge}}
	svc := NewBadges(store, nil)

	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: "admin-1", Role: auth.RoleMinistry})

	_, err := svc.GrantBadge(ctx, domain.GrantBadgeInput{ArtisanID: uuid.New(), BadgeID: earnedBadge.ID, GrantedBy: "admin-1"})
	require.Error(t, err)
}
