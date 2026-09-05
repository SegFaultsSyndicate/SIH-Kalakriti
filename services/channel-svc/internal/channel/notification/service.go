// services/channel-svc/internal/channel/notification/service.go

package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Service manages notifications.
type Service struct {
	repo Repo
	log  *slog.Logger
}

// NewService creates a notification service.
func NewService(repo Repo, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// Repo is the persistence interface.
type Repo interface {
	InsertNotification(ctx context.Context, n Notification) (Notification, error)
	GetNotification(ctx context.Context, id uuid.UUID) (Notification, error)
	ListNotificationsForUser(ctx context.Context, recipientID string, limit, offset int) ([]Notification, error)
	ListUnreadNotificationsForUser(ctx context.Context, recipientID string, limit int) ([]Notification, error)
	MarkNotificationRead(ctx context.Context, id uuid.UUID, readAt time.Time) error
	CountUnreadForUser(ctx context.Context, recipientID string) (int, error)
	InsertDelivery(ctx context.Context, d Delivery) (Delivery, error)
	UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status DeliveryStatus, sentAt *time.Time, lastError string) error
	GetDeliveriesForNotification(ctx context.Context, notificationID uuid.UUID) ([]Delivery, error)
}

// Notification is the domain model.
type Notification struct {
	ID          uuid.UUID
	RecipientID string
	Kind        NotificationKind
	Language    Language
	Title       string
	Body        string
	Payload     map[string]any
	ReadAt      *time.Time
	CreatedAt   time.Time
}

// Delivery tracks per-channel delivery.
type Delivery struct {
	ID             uuid.UUID
	NotificationID uuid.UUID
	Channel        DeliveryChannel
	Recipient      string
	Status         DeliveryStatus
	AttemptCount   int
	LastError      string
	SentAt         *time.Time
	CreatedAt      time.Time
}

// DeliveryChannel is the delivery method.
type DeliveryChannel string

const (
	ChannelWhatsApp DeliveryChannel = "whatsapp"
	ChannelEmail    DeliveryChannel = "email"
	ChannelPush     DeliveryChannel = "push"
	ChannelSMS      DeliveryChannel = "sms"
)

// DeliveryStatus tracks delivery state.
type DeliveryStatus string

const (
	DeliveryPending DeliveryStatus = "pending"
	DeliverySent    DeliveryStatus = "sent"
	DeliveryFailed  DeliveryStatus = "failed"
)

// CreateInput holds notification creation params.
type CreateInput struct {
	RecipientID string
	Kind        NotificationKind
	Language    Language
	Vars        map[string]string
	Payload     map[string]any
}

// Create renders and persists a notification.
func (s *Service) Create(ctx context.Context, in CreateInput) (Notification, error) {
	tpl, err := GetTemplate(in.Kind, in.Language)
	if err != nil {
		return Notification{}, fmt.Errorf("notification template: %w", err)
	}

	rendered := tpl.Render(in.Vars)

	n := Notification{
		ID:          uuid.Must(uuid.NewV7()),
		RecipientID: in.RecipientID,
		Kind:        in.Kind,
		Language:    in.Language,
		Title:       rendered.Title,
		Body:        rendered.Body,
		Payload:     in.Payload,
		CreatedAt:   time.Now().UTC(),
	}

	return s.repo.InsertNotification(ctx, n)
}

// MarkRead marks a notification as read.
func (s *Service) MarkRead(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkNotificationRead(ctx, id, time.Now().UTC())
}

// Get fetches one notification by id, so a caller can check who it belongs
// to before acting on it (see MarkFeedItemRead's ownership check).
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Notification, error) {
	return s.repo.GetNotification(ctx, id)
}

// ListForUser lists notifications for a user.
func (s *Service) ListForUser(ctx context.Context, recipientID string, limit, offset int) ([]Notification, error) {
	return s.repo.ListNotificationsForUser(ctx, recipientID, limit, offset)
}

// ListUnreadForUser lists unread notifications.
func (s *Service) ListUnreadForUser(ctx context.Context, recipientID string, limit int) ([]Notification, error) {
	return s.repo.ListUnreadNotificationsForUser(ctx, recipientID, limit)
}

// CountUnread counts unread notifications.
func (s *Service) CountUnread(ctx context.Context, recipientID string) (int, error) {
	return s.repo.CountUnreadForUser(ctx, recipientID)
}

// ScheduleDelivery creates a delivery attempt.
func (s *Service) ScheduleDelivery(ctx context.Context, notificationID uuid.UUID, channel DeliveryChannel, recipient string) (Delivery, error) {
	d := Delivery{
		ID:             uuid.Must(uuid.NewV7()),
		NotificationID: notificationID,
		Channel:        channel,
		Recipient:      recipient,
		Status:         DeliveryPending,
		CreatedAt:      time.Now().UTC(),
	}
	return s.repo.InsertDelivery(ctx, d)
}

// MarshalPayload is a helper for JSON payloads.
func MarshalPayload(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
