// services/channel-svc/internal/channel/service/follow_test.go
package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFollowRepo struct {
	inserted []struct {
		artisanID  uuid.UUID
		followerID string
		source     string
	}
	deleted []struct {
		artisanID  uuid.UUID
		followerID string
	}
}

func (f *fakeFollowRepo) InsertFollow(_ context.Context, artisanID uuid.UUID, followerID, source string) error {
	f.inserted = append(f.inserted, struct {
		artisanID  uuid.UUID
		followerID string
		source     string
	}{artisanID, followerID, source})
	return nil
}

func (f *fakeFollowRepo) DeleteFollow(_ context.Context, artisanID uuid.UUID, followerID string) error {
	f.deleted = append(f.deleted, struct {
		artisanID  uuid.UUID
		followerID string
	}{artisanID, followerID})
	return nil
}

func TestFollowArtisanUsesTheListingPageSource(t *testing.T) {
	repo := &fakeFollowRepo{}
	svc := NewFollow(repo)
	artisanID := uuid.Must(uuid.NewV7())

	err := svc.FollowArtisan(context.Background(), "buyer-1", artisanID)
	require.NoError(t, err)

	require.Len(t, repo.inserted, 1)
	assert.Equal(t, artisanID, repo.inserted[0].artisanID)
	assert.Equal(t, "buyer-1", repo.inserted[0].followerID)
	assert.Equal(t, "listing-page", repo.inserted[0].source)
}

func TestUnfollowArtisanDeletesTheFollow(t *testing.T) {
	repo := &fakeFollowRepo{}
	svc := NewFollow(repo)
	artisanID := uuid.Must(uuid.NewV7())

	err := svc.UnfollowArtisan(context.Background(), "buyer-1", artisanID)
	require.NoError(t, err)

	require.Len(t, repo.deleted, 1)
	assert.Equal(t, artisanID, repo.deleted[0].artisanID)
	assert.Equal(t, "buyer-1", repo.deleted[0].followerID)
}
