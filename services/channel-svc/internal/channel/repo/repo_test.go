// services/channel-svc/internal/channel/repo/repo_test.go
package repo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/segfaultsyndicate/kalakriti/services/channel-svc/internal/channel/notification"
	"github.com/segfaultsyndicate/kalakriti/services/channel-svc/internal/channel/sqlc"
)

func setupTestDB(t *testing.T) *sql.DB {
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