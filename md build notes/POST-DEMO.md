# Post-Demo Roadmap

**Demo Date:** September 20, 2026  
**This document:** Improvements to implement after demo, when time allows

---

## Overview

The platform is **feature-complete for demo**. This roadmap lists production-hardening, operational tooling, and scale improvements deferred until after demo feedback.

Each item includes: effort estimate, priority, and when to do it.

---

## 1. Down Migrations

**What:** Add `-- +goose Down` sections to all migration files (001-023 + 024 if indexes added)

**Why:** Rollback capability during deployment issues. Right now migrations only go up.

**Effort:** 4 hours  
**Priority:** High (do before first production deploy)

**How:**
```sql
-- In each migrations/*.sql file
-- +goose Down
-- +goose StatementBegin
DROP TABLE xyz;
DROP INDEX idx_xyz;
-- +goose StatementEnd
```

**When:** Before pushing to staging/production

---

## 2. Graceful Shutdown

**What:** All services trap SIGTERM and drain in-flight requests before exiting

**Why:** Zero-downtime deploys, no 502s during rolling restart

**Effort:** 8 hours (7 services × 1 hour + testing)  
**Priority:** High (critical for production)

**Pattern:**
```go
// In each service's main.go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

go func() {
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}()

<-ctx.Done()
log.Println("shutting down gracefully...")

shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := srv.Shutdown(shutdownCtx); err != nil {
	log.Printf("shutdown error: %v", err)
}
```

**When:** After demo, before production launch

---

## 3. Structured Logging Everywhere

**What:** Replace all `log.Printf` with `slog` structured logging

**Why:** Grep-able logs, machine-parseable, consistent format

**Effort:** 6 hours (grep all services, replace ~50 call sites)  
**Priority:** Medium

**Pattern:**
```go
// Bad
log.Printf("order %s failed: %v", orderID, err)

// Good
logger.Error(ctx, "order failed", "order_id", orderID, "error", err)
```

**When:** During first production incident retrospective (when logs matter)

---

## 4. Circuit Breakers

**What:** Wrap external API calls (ONDC, WhatsApp, India Post, SMS) with circuit breaker pattern

**Why:** One slow/down external service shouldn't cascade and kill the whole platform

**Effort:** 12 hours (4 integrations × 3 hours)  
**Priority:** Medium (only matters at scale or when external services flake)

**Library:** [`github.com/sony/gobreaker`](https://github.com/sony/gobreaker) (already battle-tested)

**Pattern:**
```go
cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
	Name:        "ONDC",
	MaxRequests: 3,
	Timeout:     60 * time.Second,
})

resp, err := cb.Execute(func() (interface{}, error) {
	return ondcClient.SubmitOrder(ctx, req)
})
```

**When:** After seeing external API flakiness in production logs

---

## 5. Chaos Testing

**What:** Inject faults (slow queries, Redis down, Kafka lag, service crashes) and verify graceful degradation

**Why:** Confidence that the system survives real-world failures

**Effort:** 16 hours (write chaos scenarios, automate in CI)  
**Priority:** Low (luxury item, do if time)

**Tools:**
- [`toxiproxy`](https://github.com/Shopify/toxiproxy) — network faults (latency, connection cuts)
- `docker pause <service>` — simulate crashes
- Redis flush mid-request — test fail-open rate limiter

**Scenarios:**
1. PostgreSQL connection pool exhausted → requests queue, no crash
2. Redis down → rate limiter fails open, idempotency disabled (logged)
3. Kafka down → outbox accumulates, relay retries when back
4. ml-svc 5s response time → BFF times out gracefully, returns 503

**When:** 2-3 weeks post-launch, once confident in happy path

---

## 6. Idempotency Key Enforcement

**What:** Return 400 if idempotency key reused with different request body (conflict detection)

**Why:** Right now duplicate key with same body returns cached response (correct), but different body also returns cached response (silent wrong behavior)

**Effort:** 4 hours  
**Priority:** High (correctness bug)

**Fix:**
```go
// In middleware/idempotency.go
// Store hash of request body alongside response
// On cache hit, compare stored hash with current request hash
// If mismatch: return 409 Conflict "idempotency key reused with different payload"
```

**When:** Immediately after demo (before any real money flows)

---

## 7. Webhook Delivery System

**What:** Notify buyers/artisans of order events (accepted, shipped, completed) via webhook + retry queue

**Why:** Buyers want to integrate Kalakriti into their order management systems

**Effort:** 24 hours (webhook subscription table, delivery worker, retry logic, signature verification)  
**Priority:** Low (only matters when buyers ask for it)

**Design:**
- Table: `webhook_subscriptions(url, secret, events[])`
- On event: insert into `webhook_deliveries(subscription_id, event, payload, attempts, next_retry_at)`
- Worker polls deliveries, POSTs to URL with HMAC signature, exponential backoff on failure
- Max 10 retries over 7 days, then mark `failed`

**When:** When first buyer requests webhook integration

---

## 8. Fraud Detection Hooks

**What:** Flag suspicious patterns (bulk orders from new buyer, rapid listing churn, phone number reuse)

**Why:** Prevent platform abuse before it scales

**Effort:** 16 hours (define rules, implement scoring, admin dashboard flag)  
**Priority:** Medium (fraud only matters at scale)

**Rules to flag:**
1. Buyer with <7 days account age places >₹50k order
2. Same phone number used for >5 accounts
3. Artisan uploads >20 listings in <1 hour
4. Order cancelled within 1 hour of placement (>3 times)

**Action:** Mark account `flagged_for_review`, admin approves/rejects in dashboard

**When:** After seeing first fraudulent activity in production

---

## 9. Backup & Restore Automation

**What:** Daily PostgreSQL dumps to S3, documented restore procedure, monthly restore drill

**Why:** Disasters happen. Backups you can't restore are useless.

**Effort:** 8 hours (write backup script, test restore, schedule cron)  
**Priority:** High (do before launch)

**Backup script:**
```bash
#!/bin/bash
# infra/backup.sh
DATE=$(date +%Y%m%d-%H%M%S)
DUMP_FILE="kalakriti-$DATE.sql.gz"

docker compose exec -T postgres pg_dump -U kalakriti -Fc kalakriti | gzip > "/backups/$DUMP_FILE"
aws s3 cp "/backups/$DUMP_FILE" "s3://kalakriti-backups/$DUMP_FILE"

# Keep local copy for 7 days
find /backups -name "*.sql.gz" -mtime +7 -delete
```

**Restore procedure:**
```bash
# Download latest backup
aws s3 cp s3://kalakriti-backups/kalakriti-20260920-030000.sql.gz ./

# Restore (drops existing DB!)
gunzip -c kalakriti-20260920-030000.sql.gz | \
  docker compose exec -T postgres psql -U kalakriti kalakriti
```

**When:** 1 week before production launch

---

## 10. Load Testing

**What:** k6 scripts simulating 500 concurrent users across all API endpoints

**Why:** Know the breaking point before users find it

**Effort:** 12 hours (write realistic load scenarios, analyze bottlenecks)  
**Priority:** Medium (nice to have before launch)

**Scenarios:**
1. **Browse listings:** 200 users × 10 req/min → `/listings?status=published`
2. **Search:** 100 users × 5 req/min → `/search?q=madhubani`
3. **Create orders:** 50 users × 1 req/min → `POST /orders`
4. **Upload media:** 20 users × 1 upload/min → media 3-step flow

**Target SLOs:**
- p95 response time <500ms (reads)
- p95 response time <2s (writes)
- Error rate <0.1%

**When:** 1 week before expected traffic spike

---

## 11. Compliance Logging (Audit Trail)

**What:** Immutable log of sensitive operations (payment splits, order amendments, QC failures) for compliance

**Why:** Regulators, customer support, fraud investigations need "who did what when"

**Effort:** 10 hours  
**Priority:** Low → High (if govt tender requires it)

**Design:**
- Table: `audit_log(id, timestamp, actor_id, action, resource_type, resource_id, changes jsonb, ip_address)`
- Trigger on: payment splits created, order status changed, QC result recorded, artisan verified
- Append-only (no updates/deletes), retention 7 years

**When:** If compliance audit or govt contract requires it

---

## 12. Multi-Language Support (i18n)

**What:** Localize error messages, UI strings, email templates to Hindi + English

**Why:** Right now Hindi search works, but error messages are English-only

**Effort:** 20 hours (extract strings, translate, wire Accept-Language header)  
**Priority:** Low (English acceptable for MVP, Hindi critical for scale)

**Pattern:**
```go
// pkg/i18n/i18n.go
func T(ctx context.Context, key string) string {
	lang := LanguageFromContext(ctx) // from Accept-Language header
	return translations[lang][key]
}

// Usage in handlers
httpx.Error(w, fmt.Errorf(i18n.T(r.Context(), "error.unauthenticated")))
```

**Translations:**
- `error.unauthenticated` → "Authentication required" / "प्रमाणीकरण आवश्यक है"
- `error.rate_limit_exceeded` → "Too many requests" / "बहुत अधिक अनुरोध"
- Email templates (OTP, order confirmation, income statement)

**When:** After user feedback says "I don't understand these errors"

---

## 13. A/B Testing Framework

**What:** Feature flags + variant assignment + metrics collection for experiments

**Why:** Test "does showing similar artisans increase conversions?" without shipping to everyone

**Effort:** 16 hours  
**Priority:** Low (growth tool, not launch blocker)

**Pattern:**
```go
// pkg/experiments/experiments.go
if experiments.IsEnabled(ctx, "show_similar_artisans") {
	similarArtisans := fetchSimilar(artisanID)
	// ...
}

// Track metrics
experiments.Track(ctx, "listing_view", map[string]any{
	"listing_id": listingID,
	"variant": experiments.Variant(ctx, "show_similar_artisans"),
})
```

**Backend:** LaunchDarkly or self-hosted feature flag service

**When:** 3+ months post-launch, when product team wants to experiment

---

## 14. Polish: Error Messages, Logging, Edge Cases

**What:** Review all error messages (are they actionable?), add retries where sensible, handle edge cases discovered in testing

**Why:** Death by 1000 papercuts — small bugs compound

**Effort:** 16 hours (sweep codebase, fix ~20 small issues)  
**Priority:** Medium (continuous improvement)

**Examples:**
- Error: "invalid request" → "price_paise must be positive"
- Add retry to Kafka producer (transient network blips)
- Handle 0-quantity bulk order (return 400 before hitting DB)
- Validate district is in state (reject "Madhubani, Tamil Nadu")

**When:** During first week of production, based on real user errors

---

## Summary Table

| # | Item | Effort | Priority | When |
|---|------|--------|----------|------|
| 1 | Down migrations | 4h | High | Before prod deploy |
| 2 | Graceful shutdown | 8h | High | After demo |
| 3 | Structured logging | 6h | Medium | After first incident |
| 4 | Circuit breakers | 12h | Medium | After external API flakiness |
| 5 | Chaos testing | 16h | Low | 2-3 weeks post-launch |
| 6 | Idempotency enforcement | 4h | High | Immediately after demo |
| 7 | Webhook delivery | 24h | Low | When buyer requests it |
| 8 | Fraud detection | 16h | Medium | After first fraud case |
| 9 | Backup & restore | 8h | High | 1 week before launch |
| 10 | Load testing | 12h | Medium | Before traffic spike |
| 11 | Compliance logging | 10h | Low→High | If compliance required |
| 12 | Multi-language (i18n) | 20h | Low | After user feedback |
| 13 | A/B testing | 16h | Low | 3+ months post-launch |
| 14 | Polish | 16h | Medium | Continuous |

**Total deferred effort:** ~172 hours (~4-5 weeks)

---

## Recommended Order

**Week 1 after demo:**
1. Idempotency enforcement (4h) — correctness bug
2. Down migrations (4h) — deployment safety
3. Graceful shutdown (8h) — zero-downtime deploys

**Before production launch:**
4. Backup & restore (8h) — disaster recovery
5. Load testing (12h) — capacity planning

**First 2 months in production:**
6. Structured logging (6h) — operational visibility
7. Circuit breakers (12h) — resilience
8. Fraud detection (16h) — platform safety
9. Polish (16h) — user experience

**Later (as needed):**
10. Compliance logging — if required
11. Multi-language — if user demand
12. Webhook delivery — if buyer demand
13. A/B testing — growth experiments
14. Chaos testing — confidence building

---

## What NOT to Do

**Don't add these unless explicitly requested:**
- GraphQL API (REST works fine)
- Real-time dashboard (SSE watch endpoint covers it)
- Mobile SDKs (web API is mobile-ready)
- Blockchain provenance (Ed25519 signatures sufficient)
- ML model retraining pipeline (ml-svc mock mode works for demo)
- Multi-tenancy (single platform for now)
- Elasticsearch (PostgreSQL FTS + pgvector sufficient)

---

**Last Updated:** 2026-08-28  
**Next Review:** After demo feedback (Sept 21, 2026)
