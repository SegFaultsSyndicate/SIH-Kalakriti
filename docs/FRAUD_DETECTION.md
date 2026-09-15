# Fraud Detection

**Last Updated:** 2026-09-15
**Status:** Detection is live (Postgres triggers). Manual review workflow is built in Go but wired into no service — see "Current Status".

---

## Current Status (read this first)

Detection genuinely runs today: migration `026_fraud_detection.sql`'s four
trigger functions fire on real writes and populate `fraud_flags` correctly,
matching the rules below exactly. **However**, `pkg/fraud.Detector`
(`CreateFlag`/`ListPending`/`Resolve`/`HasActiveFraudFlags`) — the Go API this
doc's "Admin Dashboard" and "Application Usage" sections describe — is
imported by no service in `services/`. Flags get created; there is currently
no wired admin API or handler to list/resolve them, and no service calls
`HasActiveFraudFlags` before processing a payment. The SQL queries below
against `fraud_flags` work fine run by hand (`psql`); the Go snippets describe
an API that exists in `pkg/fraud` but isn't reachable from any route yet.

There is also **no `users` table** anywhere in this schema — buyer identity is
external and opaque. The real order table is `bulk_order` (singular), and its
status column is `state`, not `status`. Every SQL example in this doc has been
corrected to match `migrations/007_orders.sql`'s actual schema.

**Detection rules:**
1. **New buyer, large order** — a new `buyer_id` (first seen <7 days ago per
   `fraud_flags`/audit history) places an order >₹50,000
2. **Phone reuse** — same phone number used for >5 artisan accounts
3. **Rapid listing creation** — >20 listings created by one artisan in <1 hour
4. **Rapid cancellation** — >3 orders cancelled by one buyer within 1 hour of placement

**Severity levels:** Critical (future: auto-block) / High (review within 24h)
/ Medium (72h) / Low (when convenient).

---

## How It Works

### Automatic detection (database triggers)

```sql
CREATE TRIGGER trigger_fraud_check_new_buyer_large_order
    AFTER INSERT ON bulk_order
    FOR EACH ROW
    EXECUTE FUNCTION fraud_check_new_buyer_large_order();
```

When a trigger detects a pattern, it inserts a row into `fraud_flags`. All
four triggers and their exact thresholds live in `migrations/026_fraud_detection.sql`:
`fraud_check_new_buyer_large_order` (on `bulk_order` insert, >₹50k),
`fraud_check_phone_reuse` (on `artisan` insert, >5 accounts/phone),
`fraud_check_rapid_listing_creation` (on `listing` insert, >20/hour),
`fraud_check_rapid_cancellation` (on `bulk_order` update, >3 cancellations/hour).

### Manual review workflow (built, not wired — see Current Status)

```
[Automatic Detection] → [Pending Flag] → [Admin Reviews] → [Resolved OK / Resolved Fraud]
```

This is the intended shape of the review workflow once `pkg/fraud.Detector`
has a caller. Today, the only way to review flags is a direct SQL query
against `fraud_flags` (below).

---

## Detection Rules

### 1. New Buyer Large Order
**Pattern:** New buyer places an order >₹50,000. **Risk:** payment fraud, money
laundering. **False positive rate:** medium (legitimate corporate buyers).

### 2. Phone Number Reuse
**Pattern:** Same phone used for >5 `artisan` rows. **Risk:** fake artisan
accounts, review manipulation. **False positive rate:** low (shared family
phones).

### 3. Rapid Listing Creation
**Pattern:** >20 `listing` rows created by one artisan in <1 hour. **Risk:**
spam, bot activity, stolen content. **False positive rate:** low.

### 4. Rapid Order Cancellation
**Pattern:** >3 `bulk_order` cancellations by one buyer within 1 hour of
placement. **Risk:** payment testing, platform abuse. **False positive rate:**
very low.

---

## Querying Flags (works today, direct SQL)

```sql
-- Pending flags, sorted by severity
SELECT id, resource_type, resource_id, flag_type, severity, description, created_at, status
FROM fraud_flags
WHERE status IN ('pending', 'reviewing')
ORDER BY
    CASE severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END,
    created_at ASC
LIMIT 50;

-- All flags for a specific bulk_order
SELECT * FROM fraud_flags
WHERE resource_type = 'bulk_order'
  AND resource_id = '01234567-89ab-cdef-0123-456789abcdef';

-- Dashboard stats
SELECT
    COUNT(*) FILTER (WHERE status IN ('pending', 'reviewing')) AS pending_count,
    COUNT(*) FILTER (WHERE status IN ('pending', 'reviewing') AND severity IN ('high', 'critical')) AS high_severity_count,
    COUNT(*) FILTER (WHERE status = 'resolved_ok') AS resolved_ok_count,
    COUNT(*) FILTER (WHERE status = 'resolved_fraud') AS resolved_fraud_count
FROM fraud_flags;
```

---

## Application API (`pkg/fraud` — exists, not currently called by any service)

If you wire this up, this is the intended usage. Whoever adds the handler
should also add a BFF route and role-gate it to admin/cluster-officer, the
same way `docs/PORTS_AND_APIS.md`'s ministry-only routes are gated in the
handler layer, not the router.

```go
import "github.com/ZoroNewbie00/kalakriti/pkg/fraud"

detector := fraud.New(db)

// Before processing a payment
hasFraudFlags, err := detector.HasActiveFraudFlags(ctx, "bulk_order", orderID)
if hasFraudFlags {
    return fmt.Errorf("order flagged for fraud review, cannot process payment")
}

// Manual flag creation
err = detector.CreateFlag(ctx, fraud.Flag{
    ResourceType: "artisan",
    ResourceID:   artisanID,
    FlagType:     "suspicious_photos",
    Severity:     "high",
    Description:  "Multiple listings with photos from Pinterest",
})

// List + resolve
flags, err := detector.ListPending(ctx, 50)
err = detector.Resolve(ctx, flagID, adminUserID, false, "Reviewed: legitimate corporate buyer") // false = not fraud
err = detector.Resolve(ctx, flagID, adminUserID, true, "Confirmed fraud: blocked buyer, refunded order") // true = confirmed
```

---

## Configuring Thresholds

Thresholds are hardcoded in the trigger functions. To change them, edit
`migrations/026_fraud_detection.sql` and add a new migration that
`CREATE OR REPLACE FUNCTION`s the trigger with the new threshold — don't edit
the already-applied migration file in place on a running database; goose
tracks migrations by checksum.

```sql
-- Example: raise the new-buyer threshold from 7 days to 14, and from ₹50k to ₹100k
-- Put this in a NEW migration file, e.g. 0NN_fraud_threshold_tuning.sql
CREATE OR REPLACE FUNCTION fraud_check_new_buyer_large_order() ...
```

---

## Adding New Detection Rules

Follow the same pattern as the existing four: a `CREATE OR REPLACE FUNCTION`
plus a trigger, both in a new migration, both referencing the real table/column
names (`bulk_order.state`, `bulk_order.buyer_id`, `artisan`, `listing` —
never `users`, `bulk_orders`, or `status`).

---

## Monitoring

```bash
# Alert on critical pending flags
psql -U kalakriti -d kalakriti -c \
  "SELECT COUNT(*) FROM fraud_flags WHERE status='pending' AND severity='critical';" \
  | grep -v "^0$" && echo "ALERT: Critical fraud flags pending"
```

```sql
-- Daily report
SELECT flag_type, COUNT(*) AS count,
       COUNT(*) FILTER (WHERE status = 'resolved_fraud') AS confirmed_fraud
FROM fraud_flags
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY flag_type
ORDER BY count DESC;
```

Index coverage already exists: `idx_fraud_flags_status` (partial: pending/
reviewing), `idx_fraud_flags_severity` (partial: high/critical),
`idx_fraud_flags_resource`, `idx_fraud_flags_created_at` — see
`docs/DATABASE_INDEXES.md`.

---

## False Positives

| Rule | FP rate | Mitigation |
|------|---------|------------|
| New buyer large order | 20-30% | Check payment method, shipping address |
| Phone reuse | 5-10% | Verify if family members sharing a phone |
| Rapid listing creation | <5% | Check if a bulk-import tool was used |
| Rapid cancellation | <5% | Almost always fraud |

```go
// Mark as false positive (once the API has a caller)
err := detector.Resolve(ctx, flagID, adminUserID, false, "False positive: corporate buyer verified via phone call")
```

---

## Actions on Confirmed Fraud

There is no `blocked` column or account-suspension mechanism today — this
would need its own migration and a check in the auth flow
(`docs/BACKEND_FLOW.md` §1) if built.

```sql
-- Cancel a fraudulent bulk order (real column is `state`, not `status`)
UPDATE bulk_order SET state = 'CANCELLED' WHERE id = '...';
```

Refunds require payment-gateway integration not covered by this doc; log the
action to `audit_log` (see `docs/COMPLIANCE_AUDIT.md`) for compliance.

---

## Testing

```sql
-- Test: new buyer, large order (no `users` table — buyer_id is opaque text)
INSERT INTO bulk_order (id, buyer_id, state, ...)
VALUES (gen_random_uuid(), 'test-buyer-001', 'ALLOCATING', ...);
-- fill in the remaining required columns per migrations/007_orders.sql

SELECT * FROM fraud_flags WHERE flag_type = 'new_buyer_large_order' ORDER BY created_at DESC LIMIT 1;
```

```go
func TestFraudDetection(t *testing.T) {
    db := setupTestDB(t)
    detector := fraud.New(db)

    // ... insert test data that triggers a rule via the real schema ...

    flags, err := detector.ListPending(context.Background(), 10)
    require.NoError(t, err)
    require.NotEmpty(t, flags)
}
```

---

## Performance

Triggers run `AFTER` the insert/update, adding roughly 1-2ms — negligible
against user-facing latency.

---

## Summary

**Detection:** live, 4 rules, Postgres triggers, matches this doc.
**Review:** `pkg/fraud.Detector` exists but has no caller — build a BFF route
and admin handler before relying on the "manual review" workflow.
**Next steps:** wire `Detector` into an admin route, decide on an
auto-block/account-suspension mechanism for confirmed fraud, and consider
whether `HasActiveFraudFlags` should gate payment settlement in collab-svc's
`service/payment.go`.
