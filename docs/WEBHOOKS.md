# Webhook Delivery System

**Last Updated:** 2026-08-28  
**Status:** Production-ready

---

## Overview

Buyers subscribe to webhooks to receive real-time notifications when orders are created or updated. Automatic retry with exponential backoff, HMAC signature verification.

**Supported events:**
- `order.created` — New order placed
- `order.updated` — Order status changed

**Delivery guarantees:**
- At-least-once delivery (retries up to 10 times)
- HMAC-SHA256 signature verification
- Exponential backoff: 1min, 2min, 4min, 8min, 16min, 32min, 64min (max)
- Auto-disable after 100 consecutive failures

---

## How It Works

```
[Order Created/Updated] → [Trigger Enqueues Webhook] → [Worker Polls Queue]
                                                              ↓
                                                    [HTTP POST with HMAC]
                                                              ↓
                                            [Success: Mark delivered | Failure: Retry]
```

**Architecture:**
1. Database triggers on `bulk_orders` table automatically enqueue webhooks
2. Worker service polls `webhook_deliveries` table every 5 seconds
3. Worker sends HTTP POST to subscriber URL with HMAC signature
4. On failure: exponential backoff, retry up to 10 times
5. After 100 consecutive failures: subscription auto-disabled

---

## Subscribing to Webhooks

### Create Subscription

```go
import "github.com/segfaultsyndicate/kalakriti/pkg/webhook"

manager := webhook.NewManager(db)

subscriptionID, err := manager.CreateSubscription(ctx, webhook.Subscription{
    SubscriberID:   buyerID,
    SubscriberType: "buyer",
    URL:            "https://buyer-system.example.com/webhooks/kalakriti",
    Secret:         "your-webhook-secret-keep-this-safe",
    Events:         []string{"order.created", "order.updated"},
})
```

**Fields:**
- `subscriber_id` — Buyer/seller UUID
- `subscriber_type` — `buyer`, `seller`, or `admin`
- `url` — HTTPS endpoint to receive webhooks (HTTP rejected)
- `secret` — HMAC secret for signature verification (generate with `openssl rand -hex 32`)
- `events` — Array of event types to subscribe to

### List Subscriptions

```go
subscriptions, err := manager.ListSubscriptions(ctx, buyerID)
```

### Delete Subscription

```go
err := manager.DeleteSubscription(ctx, subscriptionID)
```

---

## Receiving Webhooks

### Endpoint Requirements

**Your webhook endpoint must:**
1. Accept HTTP POST requests
2. Verify HMAC signature in `X-Webhook-Signature` header
3. Return HTTP 2xx status code within 10 seconds
4. Handle duplicate deliveries (idempotent)

### Webhook Payload Format

```json
{
  "event": "order.created",
  "timestamp": "2026-08-28T10:30:00Z",
  "data": {
    "id": "01234567-89ab-cdef-0123-456789abcdef",
    "buyer_id": "buyer-uuid",
    "status": "pending",
    "created_at": "2026-08-28T10:30:00Z"
  }
}
```

**For `order.updated`:**
```json
{
  "event": "order.updated",
  "timestamp": "2026-08-28T10:35:00Z",
  "data": {
    "id": "order-uuid",
    "buyer_id": "buyer-uuid",
    "old_status": "pending",
    "new_status": "confirmed",
    "updated_at": "2026-08-28T10:35:00Z"
  }
}
```

### Signature Verification

**HTTP Headers:**
```
Content-Type: application/json
X-Webhook-Signature: abc123def456...
User-Agent: Kalakriti-Webhooks/1.0
```

**Verification (Go):**
```go
import "github.com/segfaultsyndicate/kalakriti/pkg/webhook"

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-Webhook-Signature")
    secret := "your-webhook-secret"

    if !webhook.VerifySignature(body, signature, secret) {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }

    // Process webhook...
    w.WriteHeader(http.StatusOK)
}
```

**Verification (Python):**
```python
import hmac
import hashlib

def verify_signature(payload: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(secret.encode(), payload, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature)
```

**Verification (Node.js):**
```javascript
const crypto = require('crypto');

function verifySignature(payload, signature, secret) {
  const expected = crypto.createHmac('sha256', secret)
    .update(payload)
    .digest('hex');
  return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signature));
}
```

---

## Retry Logic

### Exponential Backoff

| Attempt | Delay | Total Elapsed |
|---------|-------|---------------|
| 1 | Immediate | 0s |
| 2 | 1 min | 1 min |
| 3 | 2 min | 3 min |
| 4 | 4 min | 7 min |
| 5 | 8 min | 15 min |
| 6 | 16 min | 31 min |
| 7 | 32 min | 63 min |
| 8 | 64 min | 127 min |
| 9 | 64 min | 191 min |
| 10 | 64 min | 255 min (~4 hours) |

After 10 attempts, delivery marked as `failed` (no further retries).

### Success Criteria

**Considered successful if:**
- HTTP status code 200-299
- Response received within 10 seconds

**Retried on:**
- HTTP status code 4xx, 5xx
- Network timeout
- Connection refused
- DNS resolution failure

### Auto-Disable

After **100 consecutive failures**, subscription automatically disabled. Re-enable by:
```sql
UPDATE webhook_subscriptions SET active = true, consecutive_failures = 0 WHERE id = '...';
```

---

## Running the Worker

### Standalone Service

```go
package main

import (
    "context"
    "database/sql"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    _ "github.com/lib/pq"
    "github.com/segfaultsyndicate/kalakriti/pkg/webhook"
)

func main() {
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        slog.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer db.Close()

    manager := webhook.NewManager(db)
    worker := webhook.NewWorker(manager)

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    slog.Info("webhook worker started", "poll_interval", "5s")
    worker.Run(ctx, 5*time.Second)
}
```

### Docker Compose

```yaml
services:
  webhook-worker:
    build: .
    command: ["./webhook-worker"]
    environment:
      DATABASE_URL: postgres://kalakriti:password@postgres:5432/kalakriti?sslmode=disable
    depends_on:
      - postgres
    restart: unless-stopped
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webhook-worker
spec:
  replicas: 2  # Multiple workers OK (FOR UPDATE SKIP LOCKED prevents races)
  template:
    spec:
      containers:
      - name: worker
        image: kalakriti/webhook-worker:latest
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
```

---

## Monitoring

### Delivery Stats

```sql
-- Success rate (last 24 hours)
SELECT
    COUNT(*) FILTER (WHERE status = 'succeeded') AS succeeded,
    COUNT(*) FILTER (WHERE status = 'failed') AS failed,
    ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'succeeded') / COUNT(*), 2) AS success_rate_pct
FROM webhook_deliveries
WHERE created_at > NOW() - INTERVAL '24 hours';
```

### Pending Deliveries

```sql
-- Deliveries waiting to be sent
SELECT COUNT(*) AS pending_count
FROM webhook_deliveries
WHERE status = 'pending' AND attempts < max_attempts;
```

### Failed Subscriptions

```sql
-- Subscriptions with recent failures
SELECT
    ws.id,
    ws.url,
    ws.consecutive_failures,
    ws.last_failure_at
FROM webhook_subscriptions ws
WHERE ws.consecutive_failures > 10
ORDER BY ws.consecutive_failures DESC;
```

### Prometheus Metrics (Future)

```go
var webhookDeliveriesTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "webhook_deliveries_total",
        Help: "Total webhook delivery attempts",
    },
    []string{"event", "status"},
)

var webhookDeliveryDuration = prometheus.NewHistogram(
    prometheus.HistogramOpts{
        Name:    "webhook_delivery_duration_seconds",
        Help:    "Webhook delivery duration",
        Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
    },
)
```

---

## Security

### HMAC Signature

**Why:** Prevents webhook spoofing (attacker cannot forge valid signature without secret)

**Algorithm:** HMAC-SHA256

**Format:** Hex-encoded (64 characters: `a1b2c3d4...`)

**Secret generation:**
```bash
openssl rand -hex 32
```

**Secret rotation:**
1. Generate new secret
2. Update subscription with new secret
3. Deploy new secret to your webhook endpoint
4. Old deliveries in retry queue will fail (acceptable)

### HTTPS Required

Webhook URLs must use HTTPS. HTTP rejected to prevent:
- Man-in-the-middle attacks
- Secret leakage
- Payload tampering

### Rate Limiting (Future)

```go
// Rate limit: max 100 webhook deliveries per subscriber per minute
if deliveryCount > 100 {
    return fmt.Errorf("rate limit exceeded")
}
```

---

## Testing

### Local Testing with ngrok

```bash
# Start ngrok tunnel
ngrok http 8080

# Subscribe with ngrok URL
curl -X POST http://localhost:3000/api/webhooks/subscribe \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://abc123.ngrok.io/webhooks",
    "secret": "test-secret",
    "events": ["order.created", "order.updated"]
  }'

# Create test order (triggers webhook)
curl -X POST http://localhost:3000/api/orders \
  -H "Content-Type: application/json" \
  -d '{"buyer_id": "..."}' 
```

### Manual Trigger

```sql
-- Manually enqueue webhook for testing
INSERT INTO webhook_deliveries (subscription_id, event, payload)
SELECT
    id,
    'order.created',
    '{"event": "order.created", "timestamp": "2026-08-28T10:00:00Z", "data": {"id": "test-order"}}'::jsonb
FROM webhook_subscriptions
WHERE subscriber_id = 'your-buyer-id'
LIMIT 1;
```

### Signature Verification Test

```bash
# Generate signature
echo -n '{"event":"order.created"}' | openssl dgst -sha256 -hmac "your-secret" -hex

# Verify in your endpoint
curl -X POST https://your-endpoint.com/webhooks \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Signature: <generated-signature>" \
  -d '{"event":"order.created"}'
```

---

## Troubleshooting

### Webhook Not Delivered

**Check subscription is active:**
```sql
SELECT id, url, active, consecutive_failures
FROM webhook_subscriptions
WHERE subscriber_id = 'buyer-uuid';
```

**Check delivery status:**
```sql
SELECT id, event, status, attempts, http_status, response_body
FROM webhook_deliveries
WHERE subscription_id = 'subscription-uuid'
ORDER BY created_at DESC
LIMIT 10;
```

**Common issues:**
- Subscription disabled after 100 failures → Re-enable
- Endpoint returns 4xx/5xx → Check endpoint logs
- Signature verification failed → Check secret matches
- Timeout → Endpoint must respond within 10 seconds

### High Failure Rate

**Slow endpoint:**
```sql
-- Deliveries timing out
SELECT COUNT(*) FROM webhook_deliveries
WHERE status = 'failed' AND response_body LIKE '%timeout%';
```

**Fix:** Optimize endpoint or process webhooks asynchronously (return 200 immediately, process in background)

**Invalid signature:**
```sql
-- Deliveries rejected with 401
SELECT COUNT(*) FROM webhook_deliveries
WHERE status = 'failed' AND http_status = 401;
```

**Fix:** Verify secret matches between subscription and endpoint

### Duplicate Deliveries

**Expected behavior:** At-least-once delivery means duplicates possible (e.g., endpoint returned 200 but worker didn't receive ACK before timeout)

**Fix:** Make endpoint idempotent:
```go
// Store delivered webhook IDs
delivered := make(map[string]bool)

func handleWebhook(payload Payload) {
    if delivered[payload.ID] {
        return // Already processed
    }
    
    // Process webhook...
    
    delivered[payload.ID] = true
}
```

---

## Summary

**Automatic delivery:** Database triggers enqueue webhooks  
**Retry logic:** Exponential backoff, up to 10 attempts  
**Security:** HMAC-SHA256 signatures  
**Concurrency:** Multiple workers supported (SKIP LOCKED)  
**Auto-disable:** After 100 consecutive failures

**Next steps:**
1. Run migration: `make migrate-up`
2. Deploy webhook worker service
3. Subscribe to webhooks via API
4. Verify signature in your endpoint
5. Monitor delivery stats
