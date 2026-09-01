// pkg/webhook/webhook.go

// Package webhook provides webhook subscription management and delivery.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Subscription represents a webhook subscription.
type Subscription struct {
	ID                 uuid.UUID
	SubscriberID       uuid.UUID
	SubscriberType     string
	URL                string
	Secret             string
	Events             []string
	Active             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastSuccessAt      *time.Time
	LastFailureAt      *time.Time
	ConsecutiveFailures int
}

// Delivery represents a webhook delivery attempt.
type Delivery struct {
	ID             uuid.UUID
	SubscriptionID uuid.UUID
	Event          string
	Payload        map[string]any
	Attempts       int
	MaxAttempts    int
	NextRetryAt    time.Time
	Status         string
	HTTPStatus     *int
	ResponseBody   string
	CreatedAt      time.Time
	DeliveredAt    *time.Time
}

// Manager handles webhook subscriptions and deliveries.
type Manager struct {
	db         *sql.DB
	httpClient *http.Client
}

// NewManager creates a webhook manager.
func NewManager(db *sql.DB) *Manager {
	return &Manager{
		db:         db,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// CreateSubscription registers a new webhook subscription.
func (m *Manager) CreateSubscription(ctx context.Context, sub Subscription) (uuid.UUID, error) {
	id := uuid.New()

	_, err := m.db.ExecContext(ctx, `
		INSERT INTO webhook_subscriptions (
			id, subscriber_id, subscriber_type, url, secret, events
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, id, sub.SubscriberID, sub.SubscriberType, sub.URL, sub.Secret, pq.Array(sub.Events))

	return id, err
}

// ListSubscriptions returns all subscriptions for a subscriber.
func (m *Manager) ListSubscriptions(ctx context.Context, subscriberID uuid.UUID) ([]Subscription, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, subscriber_id, subscriber_type, url, secret, events, active,
		       created_at, updated_at, last_success_at, last_failure_at, consecutive_failures
		FROM webhook_subscriptions
		WHERE subscriber_id = $1
		ORDER BY created_at DESC
	`, subscriberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var s Subscription
		err := rows.Scan(
			&s.ID, &s.SubscriberID, &s.SubscriberType, &s.URL, &s.Secret, pq.Array(&s.Events),
			&s.Active, &s.CreatedAt, &s.UpdatedAt, &s.LastSuccessAt, &s.LastFailureAt,
			&s.ConsecutiveFailures,
		)
		if err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}

	return subs, rows.Err()
}

// DeleteSubscription removes a webhook subscription.
func (m *Manager) DeleteSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	_, err := m.db.ExecContext(ctx, `
		DELETE FROM webhook_subscriptions WHERE id = $1
	`, subscriptionID)
	return err
}

// Worker processes pending webhook deliveries.
type Worker struct {
	manager *Manager
	db      *sql.DB
}

// NewWorker creates a webhook delivery worker.
func NewWorker(manager *Manager) *Worker {
	return &Worker{
		manager: manager,
		db:      manager.db,
	}
}

// Run starts the webhook delivery worker (blocking).
func (w *Worker) Run(ctx context.Context, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

// processBatch fetches and delivers pending webhooks.
func (w *Worker) processBatch(ctx context.Context) {
	rows, err := w.db.QueryContext(ctx, `
		SELECT
			wd.id, wd.subscription_id, wd.event, wd.payload, wd.attempts, wd.max_attempts,
			ws.url, ws.secret
		FROM webhook_deliveries wd
		JOIN webhook_subscriptions ws ON wd.subscription_id = ws.id
		WHERE wd.status = 'pending'
		  AND wd.attempts < wd.max_attempts
		  AND wd.next_retry_at <= NOW()
		  AND ws.active = true
		ORDER BY wd.next_retry_at ASC
		LIMIT 100
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var deliveryID, subscriptionID uuid.UUID
		var event, url, secret string
		var payloadJSON []byte
		var attempts, maxAttempts int

		err := rows.Scan(&deliveryID, &subscriptionID, &event, &payloadJSON, &attempts, &maxAttempts, &url, &secret)
		if err != nil {
			continue
		}

		var payload map[string]any
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			continue
		}

		w.deliverWebhook(ctx, deliveryID, subscriptionID, url, secret, payload, attempts, maxAttempts)
	}
}

// deliverWebhook sends a webhook and updates delivery status.
func (w *Worker) deliverWebhook(ctx context.Context, deliveryID, subscriptionID uuid.UUID, url, secret string, payload map[string]any, attempts, maxAttempts int) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		w.markFailed(ctx, deliveryID, subscriptionID, 0, "marshal error", attempts, maxAttempts)
		return
	}

	// Create HMAC signature
	signature := computeSignature(payloadBytes, secret)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		w.markFailed(ctx, deliveryID, subscriptionID, 0, "request creation error", attempts, maxAttempts)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("User-Agent", "Kalakriti-Webhooks/1.0")

	resp, err := w.manager.httpClient.Do(req)
	if err != nil {
		w.markFailed(ctx, deliveryID, subscriptionID, 0, err.Error(), attempts, maxAttempts)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024)) // Read first 1KB of response

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.markSucceeded(ctx, deliveryID, subscriptionID, resp.StatusCode, string(body))
	} else {
		w.markFailed(ctx, deliveryID, subscriptionID, resp.StatusCode, string(body), attempts, maxAttempts)
	}
}

// markSucceeded marks a delivery as succeeded.
func (w *Worker) markSucceeded(ctx context.Context, deliveryID, subscriptionID uuid.UUID, httpStatus int, responseBody string) {
	_, _ = w.db.ExecContext(ctx, `
		UPDATE webhook_deliveries
		SET status = 'succeeded', http_status = $1, response_body = $2, delivered_at = NOW()
		WHERE id = $3
	`, httpStatus, responseBody, deliveryID)

	_, _ = w.db.ExecContext(ctx, `
		UPDATE webhook_subscriptions
		SET last_success_at = NOW(), consecutive_failures = 0
		WHERE id = $1
	`, subscriptionID)
}

// markFailed marks a delivery as failed and schedules retry with exponential backoff.
func (w *Worker) markFailed(ctx context.Context, deliveryID, subscriptionID uuid.UUID, httpStatus int, responseBody string, attempts, maxAttempts int) {
	attempts++

	var status string
	var nextRetry time.Time

	if attempts >= maxAttempts {
		status = "failed"
		nextRetry = time.Now().Add(365 * 24 * time.Hour) // Far future (never retry)
	} else {
		status = "pending"
		// Exponential backoff: 1min, 2min, 4min, 8min, 16min, 32min, 64min (max)
		backoffMinutes := 1 << (attempts - 1)
		if backoffMinutes > 64 {
			backoffMinutes = 64
		}
		nextRetry = time.Now().Add(time.Duration(backoffMinutes) * time.Minute)
	}

	_, _ = w.db.ExecContext(ctx, `
		UPDATE webhook_deliveries
		SET status = $1, attempts = $2, next_retry_at = $3, http_status = $4, response_body = $5
		WHERE id = $6
	`, status, attempts, nextRetry, httpStatus, responseBody, deliveryID)

	_, _ = w.db.ExecContext(ctx, `
		UPDATE webhook_subscriptions
		SET last_failure_at = NOW(), consecutive_failures = consecutive_failures + 1
		WHERE id = $1
	`, subscriptionID)

	// Auto-disable subscription after 100 consecutive failures
	_, _ = w.db.ExecContext(ctx, `
		UPDATE webhook_subscriptions
		SET active = false
		WHERE id = $1 AND consecutive_failures >= 100
	`, subscriptionID)
}

// computeSignature generates HMAC-SHA256 signature for webhook payload.
func computeSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies the webhook signature.
func VerifySignature(payload []byte, signature, secret string) bool {
	expected := computeSignature(payload, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}
