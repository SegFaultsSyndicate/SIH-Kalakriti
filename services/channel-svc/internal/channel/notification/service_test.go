// services/channel-svc/internal/channel/notification/service_test.go
package notification

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	notifications map[uuid.UUID]Notification
	deliveries    map[uuid.UUID][]Delivery
}

func (f *fakeRepo) InsertNotification(ctx context.Context, n Notification) (Notification, error) {
	if f.notifications == nil {
		f.notifications = make(map[uuid.UUID]Notification)
	}
	f.notifications[n.ID] = n
	return n, nil
}

func (f *fakeRepo) GetNotification(ctx context.Context, id uuid.UUID) (Notification, error) {
	n, ok := f.notifications[id]
	if !ok {
		return Notification{}, assert.AnError
	}
	return n, nil
}

func (f *fakeRepo) ListNotificationsForUser(ctx context.Context, recipientID string, limit, offset int) ([]Notification, error) {
	var result []Notification
	for _, n := range f.notifications {
		if n.RecipientID == recipientID {
			result = append(result, n)
		}
	}
	return result, nil
}

func (f *fakeRepo) ListUnreadNotificationsForUser(ctx context.Context, recipientID string, limit int) ([]Notification, error) {
	var result []Notification
	for _, n := range f.notifications {
		if n.RecipientID == recipientID && n.ReadAt == nil {
			result = append(result, n)
		}
	}
	return result, nil
}

func (f *fakeRepo) MarkNotificationRead(ctx context.Context, id uuid.UUID, readAt time.Time) error {
	n, ok := f.notifications[id]
	if !ok {
		return assert.AnError
	}
	n.ReadAt = &readAt
	f.notifications[id] = n
	return nil
}

func (f *fakeRepo) CountUnreadForUser(ctx context.Context, recipientID string) (int, error) {
	count := 0
	for _, n := range f.notifications {
		if n.RecipientID == recipientID && n.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepo) InsertDelivery(ctx context.Context, d Delivery) (Delivery, error) {
	if f.deliveries == nil {
		f.deliveries = make(map[uuid.UUID][]Delivery)
	}
	f.deliveries[d.NotificationID] = append(f.deliveries[d.NotificationID], d)
	return d, nil
}

func (f *fakeRepo) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status DeliveryStatus, sentAt *time.Time, lastError string) error {
	for notifID, deliveries := range f.deliveries {
		for i, d := range deliveries {
			if d.ID == id {
				d.Status = status
				d.SentAt = sentAt
				d.LastError = lastError
				d.AttemptCount++
				f.deliveries[notifID][i] = d
				return nil
			}
		}
	}
	return assert.AnError
}

func (f *fakeRepo) GetDeliveriesForNotification(ctx context.Context, notificationID uuid.UUID) ([]Delivery, error) {
	return f.deliveries[notificationID], nil
}

func TestCreateNotification(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)

	n, err := svc.Create(context.Background(), CreateInput{
		RecipientID: "user-1",
		Kind:        LotOffered,
		Language:    English,
		Vars: map[string]string{
			"product_name": "Saree",
			"deadline":     "2026-08-30",
		},
		Payload: map[string]any{"listing_id": "listing-1"},
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, n.ID)
	assert.Equal(t, "user-1", n.RecipientID)
	assert.Equal(t, LotOffered, n.Kind)
	assert.Contains(t, n.Title, "New Lot Offer")
	assert.Contains(t, n.Body, "Saree")
	assert.Contains(t, n.Body, "2026-08-30")
}

func TestMarkRead(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)

	n, err := svc.Create(context.Background(), CreateInput{
		RecipientID: "user-1",
		Kind:        LotOffered,
		Language:    English,
		Vars:        map[string]string{"product_name": "Saree"},
	})
	require.NoError(t, err)

	assert.Nil(t, n.ReadAt)

	err = svc.MarkRead(context.Background(), n.ID)
	require.NoError(t, err)

	fetched, err := repo.GetNotification(context.Background(), n.ID)
	require.NoError(t, err)
	assert.NotNil(t, fetched.ReadAt)
}

func TestCountUnread(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)

	_, err := svc.Create(context.Background(), CreateInput{
		RecipientID: "user-1",
		Kind:        LotOffered,
		Language:    English,
		Vars:        map[string]string{"product_name": "Saree"},
	})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), CreateInput{
		RecipientID: "user-1",
		Kind:        PaymentSettled,
		Language:    English,
		Vars:        map[string]string{"amount": "5000", "order_id": "123"},
	})
	require.NoError(t, err)

	count, err := svc.CountUnread(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestScheduleDelivery(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)

	n, err := svc.Create(context.Background(), CreateInput{
		RecipientID: "user-1",
		Kind:        LotOffered,
		Language:    English,
		Vars:        map[string]string{"product_name": "Saree"},
	})
	require.NoError(t, err)

	d, err := svc.ScheduleDelivery(context.Background(), n.ID, ChannelWhatsApp, "+919876543210")
	require.NoError(t, err)
	assert.Equal(t, ChannelWhatsApp, d.Channel)
	assert.Equal(t, "+919876543210", d.Recipient)
	assert.Equal(t, DeliveryPending, d.Status)
}