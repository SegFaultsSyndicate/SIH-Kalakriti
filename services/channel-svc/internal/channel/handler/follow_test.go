// services/channel-svc/internal/channel/handler/follow_test.go
package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	socialv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/social/v1"

	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/notification"
)

type fakeFollowSvc struct {
	followErr   error
	unfollowErr error
	sawFollower string
	sawArtisan  uuid.UUID
	unfollowed  bool
}

func (f *fakeFollowSvc) FollowArtisan(_ context.Context, followerID string, artisanID uuid.UUID) error {
	f.sawFollower, f.sawArtisan = followerID, artisanID
	return f.followErr
}

func (f *fakeFollowSvc) UnfollowArtisan(_ context.Context, followerID string, artisanID uuid.UUID) error {
	f.sawFollower, f.sawArtisan, f.unfollowed = followerID, artisanID, true
	return f.unfollowErr
}

func (f *fakeFollowSvc) CountFollowers(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

type fakeFeedSvc struct {
	items []notification.Notification
	err   error
}

func (f *fakeFeedSvc) ListForUser(_ context.Context, _ string, _, _ int) ([]notification.Notification, error) {
	return f.items, f.err
}

func TestFollowArtisanRejectsInvalidArtisanID(t *testing.T) {
	h := NewFollow(&fakeFollowSvc{}, &fakeFeedSvc{}, nil)
	_, err := h.FollowArtisan(context.Background(), &socialv1.FollowArtisanRequest{
		FollowerId: "buyer-1", ArtisanId: "not-a-uuid",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFollowArtisanRejectsMissingFollowerID(t *testing.T) {
	h := NewFollow(&fakeFollowSvc{}, &fakeFeedSvc{}, nil)
	_, err := h.FollowArtisan(context.Background(), &socialv1.FollowArtisanRequest{
		ArtisanId: uuid.Must(uuid.NewV7()).String(),
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestFollowArtisanCallsTheService(t *testing.T) {
	artisanID := uuid.Must(uuid.NewV7())
	svc := &fakeFollowSvc{}
	h := NewFollow(svc, &fakeFeedSvc{}, nil)

	_, err := h.FollowArtisan(context.Background(), &socialv1.FollowArtisanRequest{
		FollowerId: "buyer-1", ArtisanId: artisanID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, "buyer-1", svc.sawFollower)
	assert.Equal(t, artisanID, svc.sawArtisan)
}

func TestUnfollowArtisanCallsTheService(t *testing.T) {
	artisanID := uuid.Must(uuid.NewV7())
	svc := &fakeFollowSvc{}
	h := NewFollow(svc, &fakeFeedSvc{}, nil)

	_, err := h.UnfollowArtisan(context.Background(), &socialv1.UnfollowArtisanRequest{
		FollowerId: "buyer-1", ArtisanId: artisanID.String(),
	})
	require.NoError(t, err)
	assert.True(t, svc.unfollowed)
	assert.Equal(t, artisanID, svc.sawArtisan)
}

func TestGetFeedRejectsMissingUserID(t *testing.T) {
	h := NewFollow(&fakeFollowSvc{}, &fakeFeedSvc{}, nil)
	_, err := h.GetFeed(context.Background(), &socialv1.GetFeedRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetFeedConvertsNotificationsToFeedItems(t *testing.T) {
	readAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	feed := &fakeFeedSvc{items: []notification.Notification{
		{
			ID: uuid.Must(uuid.NewV7()), Kind: notification.ArtisanFollowed,
			Title: "New listing", Body: "An artisan you follow published something new",
			Payload:   map[string]any{"listing_id": "lst-1", "artisan_id": "art-1"},
			ReadAt:    &readAt,
			CreatedAt: createdAt,
		},
	}}
	h := NewFollow(&fakeFollowSvc{}, feed, nil)

	resp, err := h.GetFeed(context.Background(), &socialv1.GetFeedRequest{UserId: "buyer-1"})
	require.NoError(t, err)
	require.Len(t, resp.GetItems(), 1)

	item := resp.GetItems()[0]
	assert.Equal(t, "ARTISAN_FOLLOWED", item.GetKind())
	assert.Equal(t, "lst-1", item.GetPayload()["listing_id"])
	assert.Equal(t, "art-1", item.GetPayload()["artisan_id"])
	require.NotNil(t, item.ReadAt)
	assert.True(t, item.GetCreatedAt().AsTime().Equal(createdAt))
}
