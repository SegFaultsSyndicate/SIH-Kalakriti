# Webhook Delivery System

**Last Updated:** 2026-09-15
**Status:** Fully built, but currently unreachable from any live API path — see "Current Status" below before assuming this works end-to-end.

---

## Current Status (read this first)

`pkg/webhook.Manager`/`Worker`, migration `027_webhooks.sql`'s tables, and
`cmd/webhook-worker` form a complete, internally consistent subsystem. But:

1. **No REST endpoint exists to create/list/delete a subscription.** Nothing in
   `services/bff/internal/bff/server.go` mounts a route for it, and
   `Manager.CreateSubscription`/`ListSubscriptions`/`DeleteSubscription` are
   called from nowhere in `services/` or `web/`. The only way to populate
   `webhook_subscriptions` today is a direct SQL insert.
2. **`cmd/webhook-worker` is not in `docker-compose.yml`.** Even a manually
   inserted subscription won't get delivered unless you run the binary
   yourself alongside the rest of the stack.
3. The bff's only webhook-related route, `POST /api/v1/payments/webhook`, is
   an **unrelated, inbound** payment-gateway callback (Kalakriti *receiving* a
   signed webhook from a payment provider) — not part of this outbound system.
   It does reuse `pkg/webhook.VerifySignature`.

If you're wiring this up for real, you need: a BFF route that calls
`webhook.Manager`, and a `webhook-worker` entry in `docker-compose.yml`.
Everything else below (schema, retry math, signing) is accurate to the code as
it stands.

**Supported events (per the DB triggers that exist today):**
- `order.created` — new `bulk_order` inserted
- `order.updated` — `bulk_order` row updated

**Delivery guarantees, once wired:**
- At-least-once delivery (retries up to 10 times)
- HMAC-SHA256 signature verification
- Exponential backoff: 1min, 2min, 4min, 8min, 16min, 32min, 64min (capped)
- Auto-disable after 100 consecutive failures

---

## How It Works

```
[bulk_order INSERT/UPDATE] → [Postgres trigger enqueues a row in webhook_deliveries]
                                                              ↓
                                              [webhook-worker polls the queue]
                                                              ↓
                                                    [HTTP POST with HMAC]
                                                              ↓
                                            [Success: mark delivered | Failure: retry]
```

This is a **separate mechanism from the outbox/Kafka pattern** used everywhere
else in the backend — it's driven entirely by two Postgres trigger functions
(`webhook_enqueue_order_created`, `webhook_enqueue_order_updated` in migration
`027_webhooks.sql`) on `bulk_order`, matching subscribers by
`webhook_subscriptions.subscriber_id = bulk_order.buyer_id`. It does not go
through `pkg/outbox` or Kafka at all.

1. A trigger on `bulk_order` fires and enqueues a `webhook_deliveries` row for
   every active subscription whose `subscriber_id` matches the order's
   `buyer_id` and whose `events` array contains the fired event type.
2. `cmd/webhook-worker` polls `webhook_deliveries` (`WEBHOOK_POLL_INTERVAL`,
   default 5s) for rows `status='pending' AND next_retry_at <= NOW()`, using
   `FOR UPDATE SKIP LOCKED` so multiple worker replicas don't double-deliver.
3. Worker sends an HTTP POST to the subscriber's URL with an HMAC signature.
4. On failure: exponential backoff, up to 10 attempts.
5. After 100 consecutive failures on a subscription: it's auto-disabled
   (`active = false`).

---

## Managing Subscriptions (Go API — no REST wrapper exists yet)

```go
import "github.com/ZoroNewbie00/kalakriti/pkg/webhook"

manager := webhook.NewManager(db) // db is a *pgxpool.Pool via POSTGRES_DSN, not database/sql + DATABASE_URL

subscriptionID, err := manager.CreateSubscription(ctx, webhook.Subscription{
    SubscriberID:   buyerID,       // matched against bulk_order.buyer_id (opaque text — no `users` table)
    SubscriberType: "buyer",
    URL:            "https://buyer-system.example.com/webhooks/kalakriti",
    Secret:         "your-webhook-secret-keep-this-safe", // generate with `openssl rand -hex 32`
    Events:         []string{"order.created", "order.updated"},
})

subscriptions, err := manager.ListSubscriptions(ctx, buyerID)
err = manager.DeleteSubscription(ctx, subscriptionID)
```

If you're adding a real subscribe/unsubscribe flow, mount a new route in
`services/bff/internal/bff/server.go` that calls this `Manager` — there is
currently no handler doing so.

---

## Receiving Webhooks (subscriber side)

### Endpoint requirements

1. Accept HTTP POST requests.
2. Verify the HMAC signature in the `X-Webhook-Signature` header.
3. Return an HTTP 2xx status within 10 seconds.
4. Handle duplicate deliveries (at-least-once, not exactly-once).

### Payload format

```json
{
  "event": "order.created",
  "timestamp": "2026-09-15T10:30:00Z",
  "data": {
    "id": "01234567-89ab-cdef-0123-456789abcdef",
    "buyer_id": "buyer-opaque-id",
    "state": "ALLOCATING",
    "created_at": "2026-09-15T10:30:00Z"
  }
}
```

Note: the payload's order field is `state` (the real `bulk_order_state` enum
column), not `status` — see `docs/BACKEND_FLOW.md` §4 for the full lifecycle.

### Signature verification

**Headers sent:**
```
Content-Type: application/json
X-Webhook-Signature: <hex hmac-sha256(payload, secret)>
User-Agent: Kalakriti-Webhooks/1.0
```

**Go:**
```go
import "github.com/ZoroNewbie00/kalakriti/pkg/webhook"

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-Webhook-Signature")

    if !webhook.VerifySignature(body, signature, secret) {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }
    w.WriteHeader(http.StatusOK)
}
```

**Python:**
```python
import hmac, hashlib

def verify_signature(payload: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(secret.encode(), payload, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature)
```

**Node.js:**
```javascript
const crypto = require('crypto');
function verifySignature(payload, signature, secret) {
  const expected = crypto.createHmac('sha256', secret).update(payload).digest('hex');
  return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signature));
}
```

---

## Retry Logic

| Attempt | Delay | Total elapsed |
|---------|-------|---------------|
| 1 | Immediate | 0s |
| 2 | 1 min | 1 min |
| 3 | 2 min | 3 min |
| 4 | 4 min | 7 min |
| 5 | 8 min | 15 min |
| 6 | 16 min | 31 min |
| 7 | 32 min | 63 min |
| 8–10 | 64 min (capped) | up to ~255 min (~4 hours) |

After 10 attempts, delivery is marked `failed` (no further retries for that
delivery). Retried on 4xx/5xx, timeout, connection refused, DNS failure.
Considered successful on HTTP 200–299 within 10 seconds.

After **100 consecutive failures**, a subscription is auto-disabled. Re-enable:
```sql
UPDATE webhook_subscriptions SET active = true, consecutive_failures = 0 WHERE id = '...';
```

---

## Running the Worker (once you decide to deploy it)

`cmd/webhook-worker/main.go` reads `POSTGRES_DSN` (not `DATABASE_URL`) and runs
`Worker.Run` on `WEBHOOK_POLL_INTERVAL` (default 5s). It is **not** currently a
service in `docker-compose.yml` — add one if you need this running:

```yaml
services:
  webhook-worker:
    build:
      context: .
      dockerfile: Dockerfile.webhook-worker  # doesn't exist yet — write one, or reuse a Go build stage
    command: ["./webhook-worker"]
    environment:
      POSTGRES_DSN: postgres://kalakriti:kalakriti@postgres:5432/kalakriti?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped
```

Multiple replicas are safe — `FOR UPDATE SKIP LOCKED` prevents double delivery.

---

## Monitoring

```sql
-- Success rate (last 24 hours)
SELECT
    COUNT(*) FILTER (WHERE status = 'succeeded') AS succeeded,
    COUNT(*) FILTER (WHERE status = 'failed') AS failed,
    ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'succeeded') / NULLIF(COUNT(*), 0), 2) AS success_rate_pct
FROM webhook_deliveries
WHERE created_at > NOW() - INTERVAL '24 hours';

-- Pending deliveries
SELECT COUNT(*) FROM webhook_deliveries WHERE status = 'pending' AND attempts < max_attempts;

-- Subscriptions with recent failures
SELECT id, url, consecutive_failures, last_failure_at
FROM webhook_subscriptions
WHERE consecutive_failures > 10
ORDER BY consecutive_failures DESC;
```

---

## Security

- **HMAC-SHA256**, hex-encoded (64 chars). Generate secrets with
  `openssl rand -hex 32`.
- **HTTPS required** — the subscription URL should be rejected if not HTTPS
  (verify this is still enforced in `Manager.CreateSubscription` if you're
  wiring a public-facing subscribe endpoint).
- **Secret rotation:** generate a new secret, update the subscription, deploy
  the new secret to the subscriber's endpoint. Deliveries already in the retry
  queue with the old secret will fail — acceptable.

---

## Testing (manual, since there's no subscribe endpoint yet)

```sql
-- Manually create a subscription
INSERT INTO webhook_subscriptions (id, subscriber_id, subscriber_type, url, secret, events, active)
VALUES (gen_random_uuid(), 'your-buyer-id', 'buyer', 'https://abc123.ngrok.io/webhooks', 'test-secret', ARRAY['order.created','order.updated'], true);

-- Manually enqueue a delivery for testing (bypasses the trigger)
INSERT INTO webhook_deliveries (subscription_id, event, payload)
SELECT id, 'order.created', '{"event":"order.created","timestamp":"2026-09-15T10:00:00Z","data":{"id":"test-order"}}'::jsonb
FROM webhook_subscriptions
WHERE subscriber_id = 'your-buyer-id'
LIMIT 1;
```

```bash
# Generate a signature to test your endpoint's verification
echo -n '{"event":"order.created"}' | openssl dgst -sha256 -hmac "your-secret" -hex
```

---

## Summary

**Mechanism:** complete and correct — Postgres triggers enqueue, a Go worker
delivers with HMAC signing and exponential backoff.
**Gap:** no REST endpoint to subscribe, and `cmd/webhook-worker` isn't
deployed anywhere. Treat this as a library you'd finish wiring, not a feature
a buyer can use today.
