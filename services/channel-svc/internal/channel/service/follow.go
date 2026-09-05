// services/channel-svc/internal/channel/service/follow.go

// Package service holds channel-svc's business logic above its repo layer.
package service

import (
	"context"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// defaultFollowSource matches follow.source's own DB default: every follow
// today comes from an artisan's profile or listing page, the only surface
// bff exposes a follow button on.
const defaultFollowSource = "listing-page"

// FollowRepo is what the Follow service needs from persistence.
type FollowRepo interface {
	InsertFollow(ctx context.Context, artisanID uuid.UUID, followerID, source string) error
	DeleteFollow(ctx context.Context, artisanID uuid.UUID, followerID string) error
	GetFollowers(ctx context.Context, artisanID uuid.UUID) ([]string, error)
}

// Follow manages buyer-follows-artisan relationships.
type Follow struct {
	repo FollowRepo
}

// NewFollow builds the follow service.
func NewFollow(repo FollowRepo) *Follow {
	return &Follow{repo: repo}
}

// FollowArtisan records a follow; idempotent.
func (s *Follow) FollowArtisan(ctx context.Context, followerID string, artisanID uuid.UUID) error {
	return s.repo.InsertFollow(ctx, artisanID, followerID, defaultFollowSource)
}

// UnfollowArtisan removes a follow; idempotent.
func (s *Follow) UnfollowArtisan(ctx context.Context, followerID string, artisanID uuid.UUID) error {
	return s.repo.DeleteFollow(ctx, artisanID, followerID)
}

// CountFollowers reports how many buyers currently follow one artisan.
// Framed on the artisan's own profile as recognition, not a vanity metric to
// optimise -- reuses the same GetFollowers row set the feed's fan-out already
// reads (see fanout.go), rather than a second, dedicated count query.
func (s *Follow) CountFollowers(ctx context.Context, artisanID uuid.UUID) (int, error) {
	followers, err := s.repo.GetFollowers(ctx, artisanID)
	if err != nil {
		return 0, err
	}
	return len(followers), nil
}

// NotificationOwnerRepo is what MarkFeedItemRead needs to check that the
// caller actually owns the notification they're marking read.
type NotificationOwnerRepo interface {
	RecipientOf(ctx context.Context, notificationID uuid.UUID) (string, error)
	MarkNotificationRead(ctx context.Context, notificationID uuid.UUID) error
}

// MarkFeedItemRead marks one feed item read, after confirming the caller is
// its own recipient -- MarkRead itself (notification.Service) takes no
// caller identity, so this is the one place that check happens.
func MarkFeedItemRead(ctx context.Context, repo NotificationOwnerRepo, notificationID uuid.UUID, userID string) error {
	recipientID, err := repo.RecipientOf(ctx, notificationID)
	if err != nil {
		return err
	}
	if recipientID != userID {
		return pkgdomain.Forbidden("notification does not belong to this user")
	}
	return repo.MarkNotificationRead(ctx, notificationID)
}
