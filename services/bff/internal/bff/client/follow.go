// services/bff/internal/bff/client/follow.go
package client

import (
	"context"
	"time"

	"google.golang.org/grpc"

	socialv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/social/v1"
)

// Follow is bff's view of channel-svc's FollowService, satisfying
// handler.FollowService.
type Follow struct {
	follow socialv1.FollowServiceClient
}

// NewFollow builds the follow client.
func NewFollow(conn grpc.ClientConnInterface) *Follow {
	return &Follow{follow: socialv1.NewFollowServiceClient(conn)}
}

// FollowArtisan follows an artisan; idempotent.
func (f *Follow) FollowArtisan(ctx context.Context, followerID, artisanID string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	_, err := f.follow.FollowArtisan(ctx, &socialv1.FollowArtisanRequest{FollowerId: followerID, ArtisanId: artisanID})
	return grpcErr(err)
}

// UnfollowArtisan unfollows an artisan; idempotent.
func (f *Follow) UnfollowArtisan(ctx context.Context, followerID, artisanID string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	_, err := f.follow.UnfollowArtisan(ctx, &socialv1.UnfollowArtisanRequest{FollowerId: followerID, ArtisanId: artisanID})
	return grpcErr(err)
}

// GetFeed pages a follower's feed of notifications about artisans they follow.
func (f *Follow) GetFeed(ctx context.Context, userID string, limit, offset int32) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := f.follow.GetFeed(ctx, &socialv1.GetFeedRequest{UserId: userID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]map[string]any, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		out = append(out, feedItemToMap(item))
	}
	return out, nil
}

func feedItemToMap(item *socialv1.FeedItem) map[string]any {
	m := map[string]any{
		"id":         item.GetId(),
		"kind":       item.GetKind(),
		"title":      item.GetTitle(),
		"body":       item.GetBody(),
		"payload":    item.GetPayload(),
		"created_at": item.GetCreatedAt().AsTime().Format(time.RFC3339),
	}
	if item.ReadAt != nil {
		m["read_at"] = item.GetReadAt().AsTime().Format(time.RFC3339)
	}
	return m
}
