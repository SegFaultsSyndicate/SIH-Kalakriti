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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
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
		httpClient: safeHTTPClient(),
	}
}

// isBlockedIP reports whether ip must never be dialed as a webhook
// destination: loopback, private (RFC1918/RFC4193), link-local, multicast or
// unspecified. A buyer- or artisan-supplied URL is otherwise free to resolve
// anywhere, including at our own internal services.
func isBlockedIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

// safeHTTPClient returns a client hardened against SSRF for webhook
// delivery: only https is accepted (subscribe time and every redirect
// target), and DialContext re-resolves the host and dials the resolved IP
// directly on every connection — including ones a redirect triggers — so a
// hostname that resolves safely at subscribe time but points at an internal
// IP at delivery time (DNS rebinding) is still caught, and there is no gap
// between "checked" and "dialed" for a redirect to exploit.
func safeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			var target netip.Addr
			found := false
			for _, ip := range ips {
				if isBlockedIP(ip) {
					continue
				}
				target = ip
				found = true
				break
			}
			if !found {
				return nil, fmt.Errorf("webhook: %s resolves only to disallowed addresses", host)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(target.String(), port))
		},
	}
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" {
				return fmt.Errorf("webhook: redirect to non-https URL %q rejected", req.URL)
			}
			return nil
		},
	}
}

// validateSubscriptionURL rejects an obviously-unsafe webhook URL at
// subscribe time: https makes signature interception non-trivial, and an IP
// literal pointed at a blocked range is refused outright rather than left to
// fail on the worker's next delivery attempt. A hostname is not re-resolved
// here — the worker's dial-time check is what actually protects delivery,
// since DNS can change between subscribe and delivery.
func validateSubscriptionURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid webhook url: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("webhook url must be https")
	}
	if u.Hostname() == "" {
		return fmt.Errorf("webhook url must have a host")
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && isBlockedIP(ip) {
		return fmt.Errorf("webhook url must not point at a private or loopback address")
	}
	return nil
}

// CreateSubscription registers a new webhook subscription.
func (m *Manager) CreateSubscription(ctx context.Context, sub Subscription) (uuid.UUID, error) {
	if err := validateSubscriptionURL(sub.URL); err != nil {
		return uuid.Nil, err
	}

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

// claimedDelivery is one row processBatch has claimed and is about to
// deliver, carrying everything deliverWebhook needs once the claiming
// transaction is closed.
type claimedDelivery struct {
	deliveryID, subscriptionID uuid.UUID
	url, secret                string
	payload                    map[string]any
	attempts, maxAttempts      int
}

// processBatch claims up to 100 pending deliveries and delivers them.
//
// Claiming happens in its own short transaction: SELECT ... FOR UPDATE OF wd
// SKIP LOCKED (the "OF wd" matters — without it, locking the joined
// webhook_subscriptions row too would make markFailed's own UPDATE against
// that subscription block, or a concurrent claim skip an unrelated delivery
// row for the same subscription) picks the batch, and attempts/next_retry_at
// are bumped to a short lease immediately, before commit. The transaction
// closes there — it does not stay open across delivery, since delivery is an
// outbound HTTP call with its own 10s timeout and 100 of those held open on
// one DB connection would starve the pool. The lease is what keeps a second
// poll (every 5s by default) from re-claiming and re-delivering the same row
// while this one is still in flight; if the worker crashes mid-delivery the
// lease simply expires and another poll picks the row back up, which is an
// acceptable at-least-once duplicate on crash, not on every batch the way an
// un-scoped SKIP LOCKED was.
func (w *Worker) processBatch(ctx context.Context) {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	rows, err := tx.QueryContext(ctx, `
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
		FOR UPDATE OF wd SKIP LOCKED
	`)
	if err != nil {
		return
	}

	var batch []claimedDelivery
	for rows.Next() {
		var d claimedDelivery
		var event string
		var payloadJSON []byte

		if err := rows.Scan(&d.deliveryID, &d.subscriptionID, &event, &payloadJSON, &d.attempts, &d.maxAttempts, &d.url, &d.secret); err != nil {
			continue
		}
		if err := json.Unmarshal(payloadJSON, &d.payload); err != nil {
			continue
		}
		batch = append(batch, d)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return
	}

	for i := range batch {
		batch[i].attempts++
		if _, err := tx.ExecContext(ctx, `
			UPDATE webhook_deliveries SET attempts = $1, next_retry_at = $2 WHERE id = $3
		`, batch[i].attempts, time.Now().Add(backoffFor(batch[i].attempts)), batch[i].deliveryID); err != nil {
			return
		}
	}

	if err := tx.Commit(); err != nil {
		return
	}
	committed = true

	for _, d := range batch {
		w.deliverWebhook(ctx, d.deliveryID, d.subscriptionID, d.url, d.secret, d.payload, d.attempts, d.maxAttempts)
	}
}

// backoffFor returns the delay before retrying a delivery that has now been
// attempted attempts times: 1min, 2min, 4min, ... capped at 64min.
func backoffFor(attempts int) time.Duration {
	backoffMinutes := 1 << (attempts - 1)
	if backoffMinutes > 64 {
		backoffMinutes = 64
	}
	return time.Duration(backoffMinutes) * time.Minute
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

// markFailed marks a delivery as failed and schedules retry with exponential
// backoff. attempts is already the post-claim count (processBatch bumped it
// before delivery started) — this does not increment it again.
func (w *Worker) markFailed(ctx context.Context, deliveryID, subscriptionID uuid.UUID, httpStatus int, responseBody string, attempts, maxAttempts int) {
	var status string
	var nextRetry time.Time

	if attempts >= maxAttempts {
		status = "failed"
		nextRetry = time.Now().Add(365 * 24 * time.Hour) // Far future (never retry)
	} else {
		status = "pending"
		nextRetry = time.Now().Add(backoffFor(attempts))
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
