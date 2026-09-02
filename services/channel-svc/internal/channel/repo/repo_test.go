// services/channel-svc/internal/channel/repo/repo_test.go
package repo

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/notification"
	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/sqlc"
)

func setupTestDB(t *testing.T) sqlc.DBTX {
	// Skip test if no postgres available - integration tests need real DB
	t.Skip("integration test requires postgres")
	return nil
}

func TestRepo_InsertNotification(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}

	r := New(db)
	ctx := context.Background()

	n := notification.Notification{
		ID:          uuid.Must(uuid.NewV7()),
		RecipientID: "user-1",
		Kind:        notification.LotOffered,
		Language:    notification.English,
		Title:       "Test",
		Body:        "Test body",
		Payload:     map[string]any{"key": "value"},
		CreatedAt:   time.Now().UTC(),
	}

	created, err := r.InsertNotification(ctx, n)
	require.NoError(t, err)
	assert.Equal(t, n.ID, created.ID)
	assert.Equal(t, "user-1", created.RecipientID)
}

// TestToNotificationDecodesPayload guards against the payload column being
// scanned but never unmarshalled back into the domain type — it used to be
// dropped silently, which meant every notification consumer (GetFeed
// included) lost listing_id/artisan_id on every read.
func TestToNotificationDecodesPayload(t *testing.T) {
	row := sqlc.Notification{
		ID:          uuid.Must(uuid.NewV7()),
		RecipientID: "user-1",
		Kind:        sqlc.NotificationKindARTISANFOLLOWED,
		Language:    sqlc.LanguageCodeENGLISH,
		Title:       "Title",
		Body:        "Body",
		Payload:     []byte(`{"listing_id":"lst-1","artisan_id":"art-1"}`),
		CreatedAt:   time.Now().UTC(),
	}

	n := toNotification(row)
	require.NotNil(t, n.Payload)
	assert.Equal(t, "lst-1", n.Payload["listing_id"])
	assert.Equal(t, "art-1", n.Payload["artisan_id"])
}

func TestRepo_GetFollowers(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}

	r := New(db)
	ctx := context.Background()

	// Test follows would need artisan setup
	followers, err := r.GetFollowers(ctx, uuid.Must(uuid.NewV7()))
	require.NoError(t, err)
	assert.NotNil(t, followers)
}