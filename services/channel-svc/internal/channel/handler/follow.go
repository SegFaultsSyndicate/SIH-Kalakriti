// services/channel-svc/internal/channel/handler/follow.go
package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	socialv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/social/v1"

	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/notification"
)

// FollowSvc is what the handler needs from the follow service.
type FollowSvc interface {
	FollowArtisan(ctx context.Context, followerID string, artisanID uuid.UUID) error
	UnfollowArtisan(ctx context.Context, followerID string, artisanID uuid.UUID) error
}

// FeedSvc is what the handler needs to page a follower's feed. GetFeed is
// backed by the same notification.Service every other notification read
// goes through — a follower's feed is exactly the notifications addressed
// to them (see fanout.go's ARTISAN_FOLLOWED writes).
type FeedSvc interface {
	ListForUser(ctx context.Context, recipientID string, limit, offset int) ([]notification.Notification, error)
}

// Follow implements social.v1.FollowService.
type Follow struct {
	socialv1.UnimplementedFollowServiceServer
	follow FollowSvc
	feed   FeedSvc
}

// NewFollow builds the follow handler.
func NewFollow(follow FollowSvc, feed FeedSvc) *Follow {
	return &Follow{follow: follow, feed: feed}
}

// FollowArtisan records a follow.
func (h *Follow) FollowArtisan(ctx context.Context, req *socialv1.FollowArtisanRequest) (*socialv1.FollowArtisanResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	if req.GetFollowerId() == "" {
		return nil, pkgdomain.GRPCError(fmt.Errorf("follower_id is required: %w", pkgdomain.ErrInvalidInput))
	}
	if err := h.follow.FollowArtisan(ctx, req.GetFollowerId(), artisanID); err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &socialv1.FollowArtisanResponse{}, nil
}

// UnfollowArtisan removes a follow.
func (h *Follow) UnfollowArtisan(ctx context.Context, req *socialv1.UnfollowArtisanRequest) (*socialv1.UnfollowArtisanResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	if req.GetFollowerId() == "" {
		return nil, pkgdomain.GRPCError(fmt.Errorf("follower_id is required: %w", pkgdomain.ErrInvalidInput))
	}
	if err := h.follow.UnfollowArtisan(ctx, req.GetFollowerId(), artisanID); err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &socialv1.UnfollowArtisanResponse{}, nil
}

// GetFeed pages a follower's feed of notifications about artisans they follow.
func (h *Follow) GetFeed(ctx context.Context, req *socialv1.GetFeedRequest) (*socialv1.GetFeedResponse, error) {
	if req.GetUserId() == "" {
		return nil, pkgdomain.GRPCError(fmt.Errorf("user_id is required: %w", pkgdomain.ErrInvalidInput))
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}

	notifications, err := h.feed.ListForUser(ctx, req.GetUserId(), limit, int(req.GetOffset()))
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	items := make([]*socialv1.FeedItem, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, feedItemToProto(n))
	}
	return &socialv1.GetFeedResponse{Items: items}, nil
}

func feedItemToProto(n notification.Notification) *socialv1.FeedItem {
	payload := make(map[string]string, len(n.Payload))
	for k, v := range n.Payload {
		if s, ok := v.(string); ok {
			payload[k] = s
		}
	}
	item := &socialv1.FeedItem{
		Id:        n.ID.String(),
		Kind:      string(n.Kind),
		Title:     n.Title,
		Body:      n.Body,
		Payload:   payload,
		CreatedAt: timestamppb.New(n.CreatedAt),
	}
	if n.ReadAt != nil {
		item.ReadAt = timestamppb.New(*n.ReadAt)
	}
	return item
}

// parseUUID converts a request field to a UUID, reporting a field-named
// ErrInvalidInput rather than a bare parse error.
func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s %q is not a valid uuid: %w", field, value, pkgdomain.ErrInvalidInput)
	}
	return id, nil
}
