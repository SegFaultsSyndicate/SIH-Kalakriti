// services/channel-svc/internal/channel/consumer/fanout.go

// Package consumer handles Kafka event consumption.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/ZoroNewbie00/kalakriti/services/channel-svc/internal/channel/notification"
)

// FollowFanout consumes catalog.listing.published and notifies followers.
type FollowFanout struct {
	notificationSvc Notifier
	followRepo      FollowRepo
	dedupeWindow    time.Duration
	log             *slog.Logger
	recent          map[string]time.Time // ponytail: in-memory dedupe, upgrade to Redis when cross-instance
}

// Notifier creates a notification; satisfied by *notification.Service, seamed
// as an interface so fanout logic is testable without a database.
type Notifier interface {
	Create(ctx context.Context, in notification.CreateInput) (notification.Notification, error)
}

// FollowRepo queries follows.
type FollowRepo interface {
	GetFollowers(ctx context.Context, artisanID uuid.UUID) ([]string, error)
}

// NewFollowFanout creates a fanout consumer.
func NewFollowFanout(notificationSvc Notifier, followRepo FollowRepo, log *slog.Logger) *FollowFanout {
	if log == nil {
		log = slog.Default()
	}
	return &FollowFanout{
		notificationSvc: notificationSvc,
		followRepo:      followRepo,
		dedupeWindow:    1 * time.Hour,
		log:             log,
		recent:          make(map[string]time.Time),
	}
}

// ListingPublishedEvent matches the event schema.
type ListingPublishedEvent struct {
	Header  EventHeader `json:"header"`
	Payload struct {
		ListingID string `json:"listing_id"`
		ArtisanID string `json:"artisan_id"`
		Title     string `json:"title"`
	} `json:"payload"`
}

// EventHeader is the standard envelope.
type EventHeader struct {
	EventID        string    `json:"event_id"`
	OccurredAt     time.Time `json:"occurred_at"`
	AggregateID    string    `json:"aggregate_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	SchemaVersion  int       `json:"schema_version"`
	Producer       string    `json:"producer"`
}

// Handle processes one catalog.listing.published event.
func (f *FollowFanout) Handle(ctx context.Context, eventBytes []byte) error {
	var evt ListingPublishedEvent
	if err := json.Unmarshal(eventBytes, &evt); err != nil {
		return fmt.Errorf("fanout: unmarshal event: %w", err)
	}

	artisanID, err := uuid.Parse(evt.Payload.ArtisanID)
	if err != nil {
		return fmt.Errorf("fanout: invalid artisan_id: %w", err)
	}

	// Get followers.
	followers, err := f.followRepo.GetFollowers(ctx, artisanID)
	if err != nil {
		return fmt.Errorf("fanout: get followers: %w", err)
	}

	if len(followers) == 0 {
		f.log.Info("fanout_no_followers", "artisan_id", evt.Payload.ArtisanID)
		return nil
	}

	// Enqueue notifications, deduped per recipient per artisan per hour so one
	// follower doesn't get flooded by the same artisan publishing repeatedly,
	// while every other follower still gets notified independently.
	for _, followerID := range followers {
		dedupeKey := followerID + ":" + evt.Payload.ArtisanID
		if last, ok := f.recent[dedupeKey]; ok && time.Since(last) < f.dedupeWindow {
			f.log.Info("fanout_dedupe", "artisan_id", evt.Payload.ArtisanID, "follower_id", followerID, "last", last)
			continue
		}
		f.recent[dedupeKey] = time.Now()

		_, err := f.notificationSvc.Create(ctx, notification.CreateInput{
			RecipientID: followerID,
			Kind:        notification.ArtisanFollowed,
			Language:    notification.English, // TODO: user preference
			Vars: map[string]string{
				"follower_name": evt.Payload.Title, // Listing title as proxy
			},
			Payload: map[string]any{
				"listing_id": evt.Payload.ListingID,
				"artisan_id": evt.Payload.ArtisanID,
			},
		})
		if err != nil {
			f.log.Error("fanout_create_notification", "error", err, "follower_id", followerID)
			continue
		}
	}

	f.log.Info("fanout_complete", "artisan_id", evt.Payload.ArtisanID, "follower_count", len(followers))
	return nil
}

// CompanyNotificationHandler consumes B2B company events and dispatches notifications.
type CompanyNotificationHandler struct {
	notificationSvc Notifier
	log             *slog.Logger
}

// NewCompanyNotificationHandler constructs the handler.
func NewCompanyNotificationHandler(notificationSvc Notifier, log *slog.Logger) *CompanyNotificationHandler {
	if log == nil {
		log = slog.Default()
	}
	return &CompanyNotificationHandler{
		notificationSvc: notificationSvc,
		log:             log,
	}
}

// HandleCompanyEvent dispatches notifications for company lifecycle events.
func (h *CompanyNotificationHandler) HandleCompanyEvent(ctx context.Context, topic string, eventBytes []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(eventBytes, &raw); err != nil {
		return fmt.Errorf("company_handler: unmarshal: %w", err)
	}

	payload, _ := raw["payload"].(map[string]any)
	if payload == nil {
		payload = raw
	}

	getString := func(key string) string {
		if v, ok := payload[key].(string); ok {
			return v
		}
		return ""
	}

	var kind notification.NotificationKind
	recipient := getString("contact_phone")
	if recipient == "" {
		recipient = getString("company_id")
	}

	vars := make(map[string]string)
	for k, v := range payload {
		vars[k] = fmt.Sprintf("%v", v)
	}

	switch topic {
	case "company.registered":
		kind = notification.CompanyRegistered
	case "company.verified":
		kind = notification.CompanyVerified
	case "company.rejected":
		kind = notification.CompanyRejected
	case "company.sale.settled":
		kind = notification.CompanySaleSettled
	case "company.interest.expressed":
		kind = notification.CompanyInterestReceived
		recipient = getString("artisan_id")
	case "company.interest.accepted":
		kind = notification.CompanyInterestAccepted
		recipient = getString("company_id")
	default:
		return nil
	}

	_, err := h.notificationSvc.Create(ctx, notification.CreateInput{
		RecipientID: recipient,
		Kind:        kind,
		Language:    notification.English,
		Vars:        vars,
		Payload:     payload,
	})
	if err != nil {
		h.log.Error("company_notification_failed", "topic", topic, "error", err)
		return err
	}

	h.log.Info("company_notification_sent", "topic", topic, "recipient", recipient, "kind", kind)
	return nil
}
