// services/channel-svc/internal/channel/service/follow.go

// Package service holds channel-svc's business logic above its repo layer.
package service

import (
	"context"

	"github.com/google/uuid"
)

// defaultFollowSource matches follow.source's own DB default: every follow
// today comes from an artisan's profile or listing page, the only surface
// bff exposes a follow button on.
const defaultFollowSource = "listing-page"

// FollowRepo is what the Follow service needs from persistence.
type FollowRepo interface {
	InsertFollow(ctx context.Context, artisanID uuid.UUID, followerID, source string) error
	DeleteFollow(ctx context.Context, artisanID uuid.UUID, followerID string) error
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
