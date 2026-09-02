// services/bff/internal/bff/client/follow_test.go
package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	socialv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/social/v1"
)

type fakeFollowService struct {
	socialv1.FollowServiceClient
	followArtisan   func(ctx context.Context, in *socialv1.FollowArtisanRequest, opts ...grpc.CallOption) (*socialv1.FollowArtisanResponse, error)
	unfollowArtisan func(ctx context.Context, in *socialv1.UnfollowArtisanRequest, opts ...grpc.CallOption) (*socialv1.UnfollowArtisanResponse, error)
	getFeed         func(ctx context.Context, in *socialv1.GetFeedRequest, opts ...grpc.CallOption) (*socialv1.GetFeedResponse, error)
}

func (f *fakeFollowService) FollowArtisan(ctx context.Context, in *socialv1.FollowArtisanRequest, opts ...grpc.CallOption) (*socialv1.FollowArtisanResponse, error) {
	return f.followArtisan(ctx, in, opts...)
}

func (f *fakeFollowService) UnfollowArtisan(ctx context.Context, in *socialv1.UnfollowArtisanRequest, opts ...grpc.CallOption) (*socialv1.UnfollowArtisanResponse, error) {
	return f.unfollowArtisan(ctx, in, opts...)
}

func (f *fakeFollowService) GetFeed(ctx context.Context, in *socialv1.GetFeedRequest, opts ...grpc.CallOption) (*socialv1.GetFeedResponse, error) {
	return f.getFeed(ctx, in, opts...)
}

func TestFollowArtisanSendsFollowerAndArtisanID(t *testing.T) {
	var sawReq *socialv1.FollowArtisanRequest
	f := &Follow{follow: &fakeFollowService{
		followArtisan: func(ctx context.Context, in *socialv1.FollowArtisanRequest, opts ...grpc.CallOption) (*socialv1.FollowArtisanResponse, error) {
			sawReq = in
			return &socialv1.FollowArtisanResponse{}, nil
		},
	}}

	err := f.FollowArtisan(context.Background(), "buyer-1", "artisan-1")
	require.NoError(t, err)
	require.NotNil(t, sawReq)
	assert.Equal(t, "buyer-1", sawReq.GetFollowerId())
	assert.Equal(t, "artisan-1", sawReq.GetArtisanId())
}

func TestUnfollowArtisanSendsFollowerAndArtisanID(t *testing.T) {
	var sawReq *socialv1.UnfollowArtisanRequest
	f := &Follow{follow: &fakeFollowService{
		unfollowArtisan: func(ctx context.Context, in *socialv1.UnfollowArtisanRequest, opts ...grpc.CallOption) (*socialv1.UnfollowArtisanResponse, error) {
			sawReq = in
			return &socialv1.UnfollowArtisanResponse{}, nil
		},
	}}

	err := f.UnfollowArtisan(context.Background(), "buyer-1", "artisan-1")
	require.NoError(t, err)
	require.NotNil(t, sawReq)
	assert.Equal(t, "buyer-1", sawReq.GetFollowerId())
	assert.Equal(t, "artisan-1", sawReq.GetArtisanId())
}

func TestGetFeedConvertsItemsToMaps(t *testing.T) {
	createdAt := timestamppb.Now()
	f := &Follow{follow: &fakeFollowService{
		getFeed: func(ctx context.Context, in *socialv1.GetFeedRequest, opts ...grpc.CallOption) (*socialv1.GetFeedResponse, error) {
			assert.Equal(t, "buyer-1", in.GetUserId())
			assert.Equal(t, int32(20), in.GetLimit())
			return &socialv1.GetFeedResponse{Items: []*socialv1.FeedItem{
				{
					Id: "notif-1", Kind: "ARTISAN_FOLLOWED", Title: "t", Body: "b",
					Payload:   map[string]string{"listing_id": "lst-1"},
					CreatedAt: createdAt,
				},
			}}, nil
		},
	}}

	feed, err := f.GetFeed(context.Background(), "buyer-1", 20, 0)
	require.NoError(t, err)
	require.Len(t, feed, 1)
	assert.Equal(t, "notif-1", feed[0]["id"])
	assert.Equal(t, "ARTISAN_FOLLOWED", feed[0]["kind"])
	assert.Nil(t, feed[0]["read_at"])
	payload, ok := feed[0]["payload"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "lst-1", payload["listing_id"])
}
