// services/channel-svc/internal/channel/repo/repo.go

package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/notification"
	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/sqlc"
	"github.com/google/uuid"
)

// Repo implements the notification service repository.
type Repo struct {
	queries *sqlc.Queries
}

// New creates a Repo.
func New(db sqlc.DBTX) *Repo {
	return &Repo{
		queries: sqlc.New(db),
	}
}

// InsertNotification persists a notification.
func (r *Repo) InsertNotification(ctx context.Context, n notification.Notification) (notification.Notification, error) {
	payloadBytes := []byte("{}")
	if n.Payload != nil {
		var err error
		payloadBytes, err = json.Marshal(n.Payload)
		if err != nil {
			return notification.Notification{}, fmt.Errorf("marshal payload: %w", err)
		}
	}
	_, err := r.queries.InsertNotification(ctx, sqlc.InsertNotificationParams{
		ID:          n.ID,
		RecipientID: n.RecipientID,
		Kind:        sqlc.NotificationKind(n.Kind),
		Language:    sqlc.LanguageCode(n.Language),
		Title:       n.Title,
		Body:        n.Body,
		Payload:     payloadBytes,
		CreatedAt:   n.CreatedAt,
	})
	if err != nil {
		return notification.Notification{}, fmt.Errorf("insert notification: %w", err)
	}
	return n, nil
}

// GetNotification retrieves a notification by ID.
func (r *Repo) GetNotification(ctx context.Context, id uuid.UUID) (notification.Notification, error) {
	row, err := r.queries.GetNotification(ctx, id)
	if err != nil {
		return notification.Notification{}, fmt.Errorf("get notification: %w", err)
	}
	return toNotification(row), nil
}

// ListNotificationsForUser lists notifications for a user.
func (r *Repo) ListNotificationsForUser(ctx context.Context, recipientID string, limit, offset int) ([]notification.Notification, error) {
	rows, err := r.queries.ListNotificationsForUser(ctx, sqlc.ListNotificationsForUserParams{
		RecipientID: recipientID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	result := make([]notification.Notification, len(rows))
	for i, row := range rows {
		result[i] = toNotification(row)
	}
	return result, nil
}

// ListUnreadNotificationsForUser lists unread notifications.
func (r *Repo) ListUnreadNotificationsForUser(ctx context.Context, recipientID string, limit int) ([]notification.Notification, error) {
	rows, err := r.queries.ListUnreadNotificationsForUser(ctx, sqlc.ListUnreadNotificationsForUserParams{
		RecipientID: recipientID,
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list unread notifications: %w", err)
	}
	result := make([]notification.Notification, len(rows))
	for i, row := range rows {
		result[i] = toNotification(row)
	}
	return result, nil
}

// MarkNotificationRead marks a notification as read.
func (r *Repo) MarkNotificationRead(ctx context.Context, id uuid.UUID, readAt time.Time) error {
	return r.queries.MarkNotificationRead(ctx, sqlc.MarkNotificationReadParams{
		ID:     id,
		ReadAt: &readAt,
	})
}

// CountUnreadForUser counts unread notifications.
func (r *Repo) CountUnreadForUser(ctx context.Context, recipientID string) (int, error) {
	count, err := r.queries.CountUnreadForUser(ctx, recipientID)
	if err != nil {
		return 0, fmt.Errorf("count unread: %w", err)
	}
	return int(count), nil
}

// InsertDelivery persists a delivery record.
func (r *Repo) InsertDelivery(ctx context.Context, d notification.Delivery) (notification.Delivery, error) {
	_, err := r.queries.InsertNotificationDelivery(ctx, sqlc.InsertNotificationDeliveryParams{
		ID:             d.ID,
		NotificationID: d.NotificationID,
		Channel:        string(d.Channel),
		Recipient:      d.Recipient,
		Status:         string(d.Status),
		CreatedAt:      d.CreatedAt,
	})
	if err != nil {
		return notification.Delivery{}, fmt.Errorf("insert delivery: %w", err)
	}
	return d, nil
}

// UpdateDeliveryStatus updates delivery status.
func (r *Repo) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status notification.DeliveryStatus, sentAt *time.Time, lastError string) error {
	var lastErrorPtr *string
	if lastError != "" {
		lastErrorPtr = &lastError
	}
	return r.queries.UpdateDeliveryStatus(ctx, sqlc.UpdateDeliveryStatusParams{
		ID:        id,
		Status:    string(status),
		SentAt:    sentAt,
		LastError: lastErrorPtr,
	})
}

// GetDeliveriesForNotification retrieves all deliveries for a notification.
func (r *Repo) GetDeliveriesForNotification(ctx context.Context, notificationID uuid.UUID) ([]notification.Delivery, error) {
	rows, err := r.queries.GetDeliveriesForNotification(ctx, notificationID)
	if err != nil {
		return nil, fmt.Errorf("get deliveries: %w", err)
	}
	result := make([]notification.Delivery, len(rows))
	for i, row := range rows {
		result[i] = toDelivery(row)
	}
	return result, nil
}

// GetFollowers returns follower IDs for an artisan.
func (r *Repo) GetFollowers(ctx context.Context, artisanID uuid.UUID) ([]string, error) {
	rows, err := r.queries.GetFollowers(ctx, artisanID)
	if err != nil {
		return nil, fmt.Errorf("get followers: %w", err)
	}
	return rows, nil
}

// InsertFollow records a follow; a repeat follow of the same artisan by the
// same follower is a no-op (ON CONFLICT DO NOTHING on the query).
func (r *Repo) InsertFollow(ctx context.Context, artisanID uuid.UUID, followerID, source string) error {
	if err := r.queries.InsertFollow(ctx, sqlc.InsertFollowParams{
		ArtisanID:  artisanID,
		FollowerID: followerID,
		Source:     source,
		CreatedAt:  time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("insert follow: %w", err)
	}
	return nil
}

// DeleteFollow removes a follow; unfollowing a non-follow is a no-op.
func (r *Repo) DeleteFollow(ctx context.Context, artisanID uuid.UUID, followerID string) error {
	if err := r.queries.DeleteFollow(ctx, sqlc.DeleteFollowParams{
		ArtisanID:  artisanID,
		FollowerID: followerID,
	}); err != nil {
		return fmt.Errorf("delete follow: %w", err)
	}
	return nil
}

func toNotification(row sqlc.Notification) notification.Notification {
	var readAt *time.Time
	if row.ReadAt != nil {
		readAt = row.ReadAt
	}
	var payload map[string]any
	_ = json.Unmarshal(row.Payload, &payload)
	return notification.Notification{
		ID:          row.ID,
		RecipientID: row.RecipientID,
		Kind:        notification.NotificationKind(row.Kind),
		Language:    notification.Language(row.Language),
		Title:       row.Title,
		Body:        row.Body,
		Payload:     payload,
		ReadAt:      readAt,
		CreatedAt:   row.CreatedAt,
	}
}

func toDelivery(row sqlc.NotificationDelivery) notification.Delivery {
	lastError := ""
	if row.LastError != nil {
		lastError = *row.LastError
	}
	return notification.Delivery{
		ID:             row.ID,
		NotificationID: row.NotificationID,
		Channel:        notification.DeliveryChannel(row.Channel),
		Recipient:      row.Recipient,
		Status:         notification.DeliveryStatus(row.Status),
		AttemptCount:   int(row.AttemptCount),
		LastError:      lastError,
		SentAt:         row.SentAt,
		CreatedAt:      row.CreatedAt,
	}
}
