# Artisan Badges Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give artisans Twitch-style recognition badges — five Ministry-conferred (verified, master craftsperson, national awardee, GI practitioner, cluster coordinator) and nine activity-earned across three tiers (catalog building, provenance sealing, order fulfilment) plus an untiered first-listing badge — rendered on the artisan's own screen and on their public buyer storefront.

**Architecture:** A `badge` reference catalog + `artisan_badge` grant table (materialized, never computed on read) live in core-svc, matching the `trend_link` slice's shape. Conferred badges are granted by a MINISTRY admin through a gRPC RPC. Earned badges are granted by a set of Kafka consumer handlers, one per triggering event, that **recompute** the artisan's true count from source tables (published listings, sealed provenance records, accepted/completed lots) and grant any newly-crossed threshold — this is a deliberate simplification over the spec's increment-with-ledger sketch: recompute-and-set is naturally idempotent under at-least-once delivery with no separate dedupe table, because replaying the same event just recomputes the same count.

**Tech Stack:** Go 1.x, pgx/v5, sqlc, buf (protoc-gen-go/protoc-gen-go-grpc), segmentio/kafka-go via `pkg/kafka`, chi (bff), SvelteKit 2 + Svelte 5 runes, TypeScript strict, openapi-typescript.

**Spec:** [docs/superpowers/specs/2026-09-15-badges-schemes-design.md](../specs/2026-09-15-badges-schemes-design.md) — "Feature 7: Artisan badges" section. This plan's consumer design (recompute-and-set) supersedes that section's increment-with-event-ledger sketch for the reason above; everything else in that section is implemented as written.

## Global Constraints

- No hand-written REST request/response types anywhere in the frontend — everything flows through the generated `schema.d.ts` (regenerate after every `openapi.json` change: `pnpm --filter @kalakriti/api api:gen`).
- Money is never rendered as raw output here (no paise fields in this feature) — N/A, noted for completeness.
- Public BFF routes need **both** layers: the RPC in `PublicMethods()` in `services/core-svc/internal/core/handler/identity.go`, AND the service method must not call `auth.RequirePrincipal`.
- All user-visible strings are i18n keys — never hardcode English in a `.svelte` file. New `en.ts` keys require all 20 non-English catalogues updated in the same task (the ratchet baseline is 0 issues per locale; a missing key is a TypeScript compile error since every catalogue is typed `export const xx: Messages`).
- Never convey meaning by colour alone (badge tier must also be in the accessible label/text, not just a border colour).
- Touch targets ≥44×44 CSS px in the artisan app; every list has empty/loading/error states.
- Migration numbers: this feature owns `033_badges.sql`. Do not reuse gaps 024/028.
- `services/bff/openapi.json` is hand-maintained JSON (not generated) — edit it directly, matching existing path-object shape exactly, then regenerate the TS types from it.

---

### Task 1: Close the pre-existing gap — publish `OrderLotCompleted`

`pkg/topics.OrderLotCompleted` is defined but never published anywhere in the codebase (`services/collab-svc/internal/collab/service/fulfilment.go`'s `SubmitQC` transitions a lot to `COMPLETED` and writes a `bulk_order_event` row for the SSE timeline, but never enqueues the Kafka event). The `order_fulfiller` earned badge needs this event to exist. This is a minimal, in-pattern fix, not a refactor.

**Files:**
- Modify: `services/collab-svc/internal/collab/service/fulfilment.go:895-905` (inside `SubmitQC`, right after the `QC_RECORDED` event write, before the `if target == domain.LotReallocated` check)
- Modify: `services/collab-svc/internal/collab/service/fulfilment.go:1177-1186` (add `lotCompletedPayload` next to `lotAcceptedPayload`/`lotDeclinedPayload`/`lotProgressedPayload`)
- Test: `services/collab-svc/internal/collab/service/fulfilment_test.go`

**Interfaces:**
- Consumes: `outbox.Enqueue(ctx, tx, id, aggregateID, topic, idempotencyKey string, payload any) error` (existing, `pkg/outbox`), `topics.OrderLotCompleted` (existing constant, `pkg/topics`), `toLotEventPayload(l domain.OrderLot) lotEventPayload` (existing helper in the same file).
- Produces: `lotCompletedPayload(l domain.OrderLot) lotEventPayload` — new helper, same signature shape as its three siblings, for Task 9's consumer to depend on (payload has `lot_id`, `bulk_order_id`, `artisan_id`, `quantity`, `state`, `progress_pct`).

- [ ] **Step 1: Add the missing helper function**

Add immediately after `lotProgressedPayload` (currently line 1179):

```go
func lotCompletedPayload(l domain.OrderLot) lotEventPayload { return toLotEventPayload(l) }
```

- [ ] **Step 2: Enqueue the event when a lot completes**

In `SubmitQC`, find this existing block (around line 899-905):

```go
		// target == LotCompleted: a passed lot releases its QC_PASSED
		// tranche and, once every lot that will ever complete has, closes
		// out the order.
		if err := f.releaseLotMilestone(ctx, tx, lot.BulkOrderID, updated, domain.MilestoneQCPassed, in.IdempotencyKey); err != nil {
			return err
		}
```

Insert an outbox enqueue immediately before that comment block, still inside the same `f.store.InTx` closure and still guarded by `target == domain.LotCompleted` being the only path that reaches this code (the `LotReallocated` and `LotQCFailed` branches both `return nil` earlier in the same closure, so no extra `if` is needed — this code only runs when `target == domain.LotCompleted`):

```go
		if err := outbox.Enqueue(ctx, tx, ids.New().String(), updated.ID.String(),
			topics.OrderLotCompleted, in.IdempotencyKey+":lot-completed", lotCompletedPayload(updated)); err != nil {
			return err
		}

		// target == LotCompleted: a passed lot releases its QC_PASSED
		// tranche and, once every lot that will ever complete has, closes
		// out the order.
		if err := f.releaseLotMilestone(ctx, tx, lot.BulkOrderID, updated, domain.MilestoneQCPassed, in.IdempotencyKey); err != nil {
			return err
		}
```

Confirm `ids` (`github.com/ZoroNewbie00/kalakriti/pkg/ids`) is already imported in this file (it is, used elsewhere in `SubmitQC`'s surrounding code) — no new import needed.

- [ ] **Step 3: Write the failing/passing test**

Find the existing test for a passing QC submission (search `TestFulfilment_SubmitQC` or similar in `fulfilment_test.go`) and add an assertion that the fake outbox/producer recorded a `topics.OrderLotCompleted` publish. Follow the exact assertion style already used in that test file for `topics.OrderLotAccepted` (there is an existing passing-QC test scenario to extend — locate it with `grep -n "SubmitQC" services/collab-svc/internal/collab/service/fulfilment_test.go`). Add:

```go
func TestFulfilment_SubmitQC_PublishesLotCompleted(t *testing.T) {
	// Reuse this test file's existing fixture/harness setup (same pattern as
	// the neighboring SubmitQC pass-case test — a lot in QC_PENDING state).
	f, store, outboxSpy := newFulfilmentTestHarness(t) // use whatever the file's existing harness constructor is named
	lot := seedLotInQCPending(t, store)                // use whatever the file's existing seed helper is named

	_, err := f.SubmitQC(context.Background(), SubmitQCInput{
		LotID: lot.ID, InspectorID: "inspector-1", Passed: true,
	})
	require.NoError(t, err)

	require.True(t, outboxSpy.Published(topics.OrderLotCompleted),
		"expected topics.OrderLotCompleted to be enqueued on QC pass")
}
```

Adjust the harness/seed helper names to match whatever already exists in `fulfilment_test.go` — read that file first (`grep -n "^func Test\|^func new\|^func seed" services/collab-svc/internal/collab/service/fulfilment_test.go`) and reuse its established fixtures rather than inventing new ones.

- [ ] **Step 4: Run the test**

Run: `cd services/collab-svc && go test ./internal/collab/service/... -run TestFulfilment_SubmitQC -v`
Expected: PASS (after Step 2's fix; write the test first if following strict TDD ordering — it will FAIL with "expected event not found" before Step 2).

- [ ] **Step 5: Commit**

```bash
git add services/collab-svc/internal/collab/service/fulfilment.go services/collab-svc/internal/collab/service/fulfilment_test.go
git commit -m "fix(collab-svc): publish OrderLotCompleted on QC pass

topics.OrderLotCompleted was defined but never enqueued; SubmitQC only
wrote the SSE-facing bulk_order_event row. Needed by the badge feature's
order_fulfiller earned-badge consumer.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 2: Migration `033_badges.sql`

**Files:**
- Create: `migrations/033_badges.sql`

**Interfaces:**
- Produces: tables `badge`, `artisan_badge`, `artisan_badge_progress`; enums `badge_kind`, `badge_tier`, `badge_metric`. 14 seeded `badge` rows with fixed, stable `code` values used by every later task.

- [ ] **Step 1: Write the migration**

```sql
-- migrations/033_badges.sql
-- +goose Up

CREATE TYPE badge_kind AS ENUM ('EARNED', 'CONFERRED');
CREATE TYPE badge_tier AS ENUM ('BRONZE', 'SILVER', 'GOLD');
CREATE TYPE badge_metric AS ENUM (
    'LISTINGS_PUBLISHED', 'PROVENANCE_SEALED', 'LOTS_ACCEPTED', 'LOTS_COMPLETED'
);

-- Reference catalog. CONFERRED badges are grantable by a MINISTRY admin at
-- runtime; EARNED badges are granted only by the badge-tracking consumer
-- (see services/core-svc/internal/core/handler/consumer.go), since their
-- threshold logic is code, not data.
CREATE TABLE badge (
    id          uuid         NOT NULL,
    code        text         NOT NULL,
    kind        badge_kind   NOT NULL,
    tier        badge_tier,
    icon_name   text         NOT NULL,
    metric      badge_metric,
    threshold   bigint,
    sort_order  integer      NOT NULL DEFAULT 0,
    active      boolean      NOT NULL DEFAULT true,
    created_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT badge_pkey PRIMARY KEY (id),
    CONSTRAINT badge_code_key UNIQUE (code),
    CONSTRAINT badge_earned_fields_check CHECK (
        (kind = 'EARNED' AND metric IS NOT NULL AND threshold IS NOT NULL)
        OR (kind = 'CONFERRED' AND metric IS NULL AND threshold IS NULL)
    ),
    CONSTRAINT badge_threshold_check CHECK (threshold IS NULL OR threshold > 0)
);

-- Grants. Materialized, never computed on read. A revoke sets revoked_at
-- rather than deleting, so history survives; a re-grant after revoke
-- overwrites the revoke fields back to NULL (see the InsertConferredGrant
-- query in Task 3, used by both the admin grant path and re-grants).
CREATE TABLE artisan_badge (
    artisan_id    uuid         NOT NULL,
    badge_id      uuid         NOT NULL,
    granted_at    timestamptz  NOT NULL DEFAULT now(),
    granted_by    text         NOT NULL,
    evidence      jsonb,
    revoked_at    timestamptz,
    revoked_by    text,
    revoke_reason text,
    CONSTRAINT artisan_badge_pkey PRIMARY KEY (artisan_id, badge_id),
    CONSTRAINT artisan_badge_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT artisan_badge_badge_id_fkey FOREIGN KEY (badge_id)
        REFERENCES badge (id) ON DELETE CASCADE
);

CREATE INDEX artisan_badge_artisan_active_idx ON artisan_badge (artisan_id)
    WHERE revoked_at IS NULL;

-- Running per-metric counters, recomputed (never incremented) by the badge
-- consumer from source-of-truth tables on every relevant event, so at-least-
-- once Kafka delivery cannot double-count.
CREATE TABLE artisan_badge_progress (
    artisan_id  uuid          NOT NULL,
    metric      badge_metric  NOT NULL,
    value       bigint        NOT NULL DEFAULT 0,
    updated_at  timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT artisan_badge_progress_pkey PRIMARY KEY (artisan_id, metric),
    CONSTRAINT artisan_badge_progress_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE
);

-- Seed catalog. Display text is i18n keys badge.<code>.name / .desc /
-- .criteria (see Task 15) -- icon_name references packages/icons (Task 14).
INSERT INTO badge (id, code, kind, tier, icon_name, metric, threshold, sort_order) VALUES
    (gen_random_uuid(), 'verified_artisan',    'CONFERRED', NULL,     'badge-verified',    NULL,                  NULL, 10),
    (gen_random_uuid(), 'master_craftsperson', 'CONFERRED', NULL,     'badge-master',      NULL,                  NULL, 20),
    (gen_random_uuid(), 'national_awardee',    'CONFERRED', NULL,     'badge-award',       NULL,                  NULL, 30),
    (gen_random_uuid(), 'gi_practitioner',     'CONFERRED', NULL,     'badge-gi',          NULL,                  NULL, 40),
    (gen_random_uuid(), 'cluster_coordinator', 'CONFERRED', NULL,     'badge-coordinator', NULL,                  NULL, 50),
    (gen_random_uuid(), 'first_listing',       'EARNED',    NULL,     'badge-milestone',   'LISTINGS_PUBLISHED', 1,    60),
    (gen_random_uuid(), 'catalog_builder',     'EARNED',    'BRONZE', 'badge-milestone',   'LISTINGS_PUBLISHED', 5,    70),
    (gen_random_uuid(), 'catalog_builder',     'EARNED',    'SILVER', 'badge-milestone',   'LISTINGS_PUBLISHED', 25,   71),
    (gen_random_uuid(), 'catalog_builder',     'EARNED',    'GOLD',   'badge-milestone',   'LISTINGS_PUBLISHED', 100,  72),
    (gen_random_uuid(), 'provenance_keeper',   'EARNED',    'BRONZE', 'badge-milestone',   'PROVENANCE_SEALED',  1,    80),
    (gen_random_uuid(), 'provenance_keeper',   'EARNED',    'SILVER', 'badge-milestone',   'PROVENANCE_SEALED',  10,   81),
    (gen_random_uuid(), 'provenance_keeper',   'EARNED',    'GOLD',   'badge-milestone',   'PROVENANCE_SEALED',  50,   82),
    (gen_random_uuid(), 'order_fulfiller',     'EARNED',    'BRONZE', 'badge-milestone',   'LOTS_COMPLETED',     1,    90),
    (gen_random_uuid(), 'order_fulfiller',     'EARNED',    'SILVER', 'badge-milestone',   'LOTS_COMPLETED',     10,   91),
    (gen_random_uuid(), 'order_fulfiller',     'EARNED',    'GOLD',   'badge-milestone',   'LOTS_COMPLETED',     50,   92);

-- +goose Down

DROP TABLE IF EXISTS artisan_badge_progress;
DROP TABLE IF EXISTS artisan_badge;
DROP TABLE IF EXISTS badge;
DROP TYPE IF EXISTS badge_metric;
DROP TYPE IF EXISTS badge_tier;
DROP TYPE IF EXISTS badge_kind;
```

Note: `catalog_builder` and `provenance_keeper` and `order_fulfiller` each appear three times with the same `code` — this violates the `UNIQUE (code)` constraint as written. Fix: the unique constraint must be on `(code, tier)`, not `code` alone, since a tiered badge is one logical badge family with three rows. Correct the migration before running it:

```sql
    CONSTRAINT badge_code_key UNIQUE (code, tier),
```

Re-verify: `first_listing`, `verified_artisan`, etc. (untiered) have `tier IS NULL`, and `UNIQUE (code, tier)` with `tier IS NULL` treats each NULL as distinct in Postgres by default — since none of the untiered codes repeat, this is safe (no two rows share both `code='first_listing'` and `tier=NULL` twice).

- [ ] **Step 2: Apply and verify**

Run: `cd C:\projects\kalakriti && goose -dir migrations postgres "$POSTGRES_DSN" up` (or `make migrate-up` if a local stack is running per `CLAUDE.md`).
Expected: migration `033` applies cleanly; `SELECT count(*) FROM badge;` returns 14.

- [ ] **Step 3: Commit**

```bash
git add migrations/033_badges.sql
git commit -m "feat(migrations): add badge catalog and grant tables

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 3: sqlc queries — `migrations/queries/badges.sql`

**Files:**
- Create: `migrations/queries/badges.sql`
- Modify: `sqlc.yaml` (add `"migrations/queries/badges.sql"` to the `core` gen block's `queries` list, alongside the existing `"migrations/queries/trends.sql"` entry)

**Interfaces:**
- Produces (after `sqlc generate`, in `services/core-svc/internal/core/repo/db/badges.sql.gen.go`): `Querier` methods `ListBadgeCatalog`, `GetBadgeByCode`, `ListArtisanBadges`, `GetBadgeProgress`, `UpsertBadgeProgress`, `GrantEligibleEarnedBadges`, `InsertConferredGrant`, `RevokeGrant`, and their `*Params`/row types — consumed by Task 4's `repo/badge.go`.

- [ ] **Step 1: Write the query file**

```sql
-- migrations/queries/badges.sql

-- name: ListBadgeCatalog :many
SELECT * FROM badge WHERE active ORDER BY sort_order;

-- name: GetBadgeByCode :one
SELECT * FROM badge WHERE code = @code AND (tier IS NULL OR tier = sqlc.narg('tier')::badge_tier);

-- name: ListArtisanBadges :many
SELECT ab.artisan_id, ab.badge_id, ab.granted_at, ab.granted_by, ab.evidence,
       b.code, b.kind, b.tier, b.icon_name, b.metric, b.threshold, b.sort_order
FROM artisan_badge ab
JOIN badge b ON b.id = ab.badge_id
WHERE ab.artisan_id = @artisan_id AND ab.revoked_at IS NULL
ORDER BY b.sort_order;

-- name: GetBadgeProgress :many
SELECT metric, value, updated_at FROM artisan_badge_progress WHERE artisan_id = @artisan_id;

-- name: UpsertBadgeProgress :one
INSERT INTO artisan_badge_progress (artisan_id, metric, value, updated_at)
VALUES (@artisan_id, @metric, @value, now())
ON CONFLICT (artisan_id, metric) DO UPDATE SET value = @value, updated_at = now()
RETURNING *;

-- name: GrantEligibleEarnedBadges :many
-- Idempotent by construction: re-running with the same (artisan_id, value)
-- after a Kafka redelivery inserts nothing new because of the NOT EXISTS
-- guard plus ON CONFLICT DO NOTHING.
INSERT INTO artisan_badge (artisan_id, badge_id, granted_at, granted_by)
SELECT @artisan_id, b.id, now(), 'system'
FROM badge b
WHERE b.kind = 'EARNED' AND b.active AND b.metric = @metric AND b.threshold <= @value
  AND NOT EXISTS (
      SELECT 1 FROM artisan_badge ab WHERE ab.artisan_id = @artisan_id AND ab.badge_id = b.id
  )
ON CONFLICT (artisan_id, badge_id) DO NOTHING
RETURNING *;

-- name: InsertConferredGrant :one
INSERT INTO artisan_badge (artisan_id, badge_id, granted_at, granted_by, evidence)
VALUES (@artisan_id, @badge_id, now(), @granted_by, sqlc.narg('evidence'))
ON CONFLICT (artisan_id, badge_id) DO UPDATE SET
    revoked_at = NULL, revoked_by = NULL, revoke_reason = NULL,
    granted_at = now(), granted_by = @granted_by, evidence = sqlc.narg('evidence')
RETURNING *;

-- name: RevokeGrant :execrows
UPDATE artisan_badge SET revoked_at = now(), revoked_by = @revoked_by, revoke_reason = @revoke_reason
WHERE artisan_id = @artisan_id AND badge_id = @badge_id AND revoked_at IS NULL;

-- name: CountPublishedListings :one
SELECT count(*)::bigint FROM listing WHERE artisan_id = @artisan_id AND state = 'PUBLISHED';

-- name: CountSealedProvenance :one
SELECT count(*)::bigint FROM provenance_record WHERE artisan_id = @artisan_id;

-- name: CountAcceptedLots :one
SELECT count(*)::bigint FROM order_lot
WHERE artisan_id = @artisan_id AND state NOT IN ('OFFERED', 'DECLINED', 'EXPIRED');

-- name: CountCompletedLots :one
SELECT count(*)::bigint FROM order_lot WHERE artisan_id = @artisan_id AND state = 'COMPLETED';
```

- [ ] **Step 2: Wire into sqlc.yaml**

In `sqlc.yaml`, under the `core` gen block's `queries:` list, add a new line right after the existing `- "migrations/queries/trends.sql"`:

```yaml
      - "migrations/queries/badges.sql"
```

- [ ] **Step 3: Generate and verify it compiles**

Run: `sqlc generate` (from repo root)
Expected: `services/core-svc/internal/core/repo/db/badges.sql.gen.go` is created with no errors. Run `cd services/core-svc && go build ./...` to confirm the generated package compiles.

- [ ] **Step 4: Commit**

```bash
git add migrations/queries/badges.sql sqlc.yaml services/core-svc/internal/core/repo/db/badges.sql.gen.go
git commit -m "feat(core-svc): generate sqlc queries for badges

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 4: Domain types — `domain/badge.go`

**Files:**
- Create: `services/core-svc/internal/core/domain/badge.go`
- Test: `services/core-svc/internal/core/domain/badge_test.go`

**Interfaces:**
- Produces: `BadgeKind`, `BadgeTier`, `BadgeMetric` string enums; `Badge`, `ArtisanBadge`, `BadgeProgressEntry` structs; `GrantBadgeInput` with `Validate() error` — consumed by Task 5 (repo), Task 6 (service), Task 7 (handler).

- [ ] **Step 1: Write the failing test**

```go
// services/core-svc/internal/core/domain/badge_test.go
package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGrantBadgeInput_Validate(t *testing.T) {
	valid := GrantBadgeInput{ArtisanID: uuid.New(), BadgeID: uuid.New(), GrantedBy: "admin-1"}
	require.NoError(t, valid.Validate())

	missing := GrantBadgeInput{BadgeID: uuid.New(), GrantedBy: "admin-1"}
	require.Error(t, missing.Validate())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/core-svc && go test ./internal/core/domain/... -run TestGrantBadgeInput_Validate -v`
Expected: FAIL (compile error — `GrantBadgeInput` undefined).

- [ ] **Step 3: Write the implementation**

```go
// services/core-svc/internal/core/domain/badge.go

package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// BadgeKind distinguishes an admin-conferred badge from one the badge
// consumer grants automatically when an activity threshold is crossed.
type BadgeKind string

const (
	BadgeKindEarned    BadgeKind = "EARNED"
	BadgeKindConferred BadgeKind = "CONFERRED"
)

// BadgeTier is the tier of a tiered earned badge; untiered badges carry nil.
type BadgeTier string

const (
	BadgeTierBronze BadgeTier = "BRONZE"
	BadgeTierSilver BadgeTier = "SILVER"
	BadgeTierGold   BadgeTier = "GOLD"
)

// BadgeMetric is a countable artisan activity an earned badge's threshold is
// measured against.
type BadgeMetric string

const (
	MetricListingsPublished BadgeMetric = "LISTINGS_PUBLISHED"
	MetricProvenanceSealed  BadgeMetric = "PROVENANCE_SEALED"
	MetricLotsAccepted      BadgeMetric = "LOTS_ACCEPTED"
	MetricLotsCompleted     BadgeMetric = "LOTS_COMPLETED"
)

// Badge is one catalog entry: either a conferred recognition or one tier of
// an earned badge family.
type Badge struct {
	ID        uuid.UUID
	Code      string
	Kind      BadgeKind
	Tier      *BadgeTier
	IconName  string
	Metric    *BadgeMetric
	Threshold *int64
	SortOrder int32
}

// ArtisanBadge is one active grant, joined with its catalog entry.
type ArtisanBadge struct {
	Badge     Badge
	GrantedAt time.Time
	GrantedBy string
	Evidence  *string // raw JSON, if present
}

// BadgeProgressEntry is the artisan's current count toward one metric.
type BadgeProgressEntry struct {
	Metric    BadgeMetric
	Value     int64
	UpdatedAt time.Time
}

// GrantBadgeInput is what an admin supplies to confer a badge.
type GrantBadgeInput struct {
	ArtisanID uuid.UUID
	BadgeID   uuid.UUID
	GrantedBy string
	Evidence  *string
}

// Validate checks a grant request has everything it needs. It does not check
// that BadgeID refers to a CONFERRED badge -- that check needs a DB read and
// lives in the service layer (see service.Badges.GrantBadge).
func (in GrantBadgeInput) Validate() error {
	if in.ArtisanID == uuid.Nil {
		return fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.BadgeID == uuid.Nil {
		return fmt.Errorf("badge_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.GrantedBy == "" {
		return fmt.Errorf("granted_by is required: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// RevokeBadgeInput is what an admin supplies to revoke a badge.
type RevokeBadgeInput struct {
	ArtisanID uuid.UUID
	BadgeID   uuid.UUID
	RevokedBy string
	Reason    string
}

// Validate checks a revoke request has everything it needs.
func (in RevokeBadgeInput) Validate() error {
	if in.ArtisanID == uuid.Nil {
		return fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.BadgeID == uuid.Nil {
		return fmt.Errorf("badge_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.RevokedBy == "" {
		return fmt.Errorf("revoked_by is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.Reason == "" {
		return fmt.Errorf("reason is required: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}
```

Check `pkgdomain.ErrInvalidInput` is the exact exported name used elsewhere (confirm via `grep -n "ErrInvalidInput" pkg/domain/*.go`); if the package instead exposes only a constructor like `domain.InvalidInput(msg string) error` (seen used in `services/bff/internal/bff/client/trends.go` as `domain.InvalidInput(...)`), use that constructor form instead of wrapping a sentinel — match whichever pattern `services/core-svc/internal/core/domain/trends.go`'s `Validate()` actually uses (it uses `fmt.Errorf("...: %w", pkgdomain.ErrInvalidInput)` per the code already read in this plan's research — keep it as written above).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/core-svc && go test ./internal/core/domain/... -run TestGrantBadgeInput_Validate -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/core-svc/internal/core/domain/badge.go services/core-svc/internal/core/domain/badge_test.go
git commit -m "feat(core-svc): add badge domain types

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 5: Repository — `repo/badge.go`

**Files:**
- Create: `services/core-svc/internal/core/repo/badge.go`
- Test: `services/core-svc/internal/core/repo/badge_test.go`

**Interfaces:**
- Consumes: `db.Querier` methods from Task 3, `domain.*` types from Task 4, existing `translate(err, entity string) error` helper (same file pattern as `repo/trends.go`).
- Produces (methods on `*Repo`, satisfying Task 6's `service.BadgeStore` interface by structural typing — no explicit interface assertion needed since `main.go` passes `*repo.Repo` directly, matching `service.NewTrends(repository, log)`):
  - `ListBadgeCatalog(ctx) ([]domain.Badge, error)`
  - `ListArtisanBadges(ctx, artisanID uuid.UUID) ([]domain.ArtisanBadge, error)`
  - `GetBadgeProgress(ctx, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error)`
  - `RecomputeAndGrant(ctx, artisanID uuid.UUID, metric domain.BadgeMetric, count int64) ([]domain.Badge, error)` — upserts progress then grants any newly-eligible badge; returns the badges newly granted (empty slice if none).
  - `CountPublishedListings(ctx, artisanID uuid.UUID) (int64, error)`
  - `CountSealedProvenance(ctx, artisanID uuid.UUID) (int64, error)`
  - `CountAcceptedLots(ctx, artisanID uuid.UUID) (int64, error)`
  - `CountCompletedLots(ctx, artisanID uuid.UUID) (int64, error)`
  - `GetBadgeByCode(ctx, code string, tier *domain.BadgeTier) (domain.Badge, error)`
  - `GrantConferredBadge(ctx, in domain.GrantBadgeInput) (domain.ArtisanBadge, error)`
  - `RevokeBadge(ctx, in domain.RevokeBadgeInput) (bool, error)`

- [ ] **Step 1: Write the failing test**

```go
// services/core-svc/internal/core/repo/badge_test.go
package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// Uses this package's existing integration-test harness (a real Postgres via
// testcontainers or a DSN from env -- follow whatever TestMain / newTestRepo
// helper repo_test.go or trends_test.go in this same package already sets
// up; do not invent a new harness).
func TestRepo_RecomputeAndGrant_GrantsFirstListingBadge(t *testing.T) {
	r, cleanup := newTestRepo(t) // reuse the existing helper name from this package's other _test.go files
	defer cleanup()
	ctx := context.Background()
	artisanID := seedArtisan(t, r) // reuse the existing seed helper name from this package's other _test.go files

	granted, err := r.RecomputeAndGrant(ctx, artisanID, domain.MetricListingsPublished, 1)
	require.NoError(t, err)
	require.Len(t, granted, 1)
	require.Equal(t, "first_listing", granted[0].Code)

	// Replaying the same count must not grant again or error.
	granted2, err := r.RecomputeAndGrant(ctx, artisanID, domain.MetricListingsPublished, 1)
	require.NoError(t, err)
	require.Empty(t, granted2)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/core-svc && go test ./internal/core/repo/... -run TestRepo_RecomputeAndGrant -v`
Expected: FAIL (compile error — `RecomputeAndGrant` undefined).

- [ ] **Step 3: Write the implementation**

```go
// services/core-svc/internal/core/repo/badge.go

package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// ListBadgeCatalog lists every active badge, sort_order ascending.
func (r *Repo) ListBadgeCatalog(ctx context.Context) ([]domain.Badge, error) {
	rows, err := r.q.ListBadgeCatalog(ctx)
	if err != nil {
		return nil, translate(err, "badge catalog")
	}
	out := make([]domain.Badge, len(rows))
	for i, row := range rows {
		out[i] = badgeFromRow(row)
	}
	return out, nil
}

// GetBadgeByCode looks up one catalog entry by its code and tier (tier nil
// for an untiered badge).
func (r *Repo) GetBadgeByCode(ctx context.Context, code string, tier *domain.BadgeTier) (domain.Badge, error) {
	var dbTier *db.BadgeTier
	if tier != nil {
		t := db.BadgeTier(*tier)
		dbTier = &t
	}
	row, err := r.q.GetBadgeByCode(ctx, db.GetBadgeByCodeParams{Code: code, Tier: dbTier})
	if err != nil {
		return domain.Badge{}, translate(err, "badge")
	}
	return badgeFromRow(row), nil
}

// ListArtisanBadges lists an artisan's active (non-revoked) grants, joined
// with their catalog entries.
func (r *Repo) ListArtisanBadges(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanBadge, error) {
	rows, err := r.q.ListArtisanBadges(ctx, artisanID)
	if err != nil {
		return nil, translate(err, "artisan badges")
	}
	out := make([]domain.ArtisanBadge, len(rows))
	for i, row := range rows {
		var tier *domain.BadgeTier
		if row.Tier != nil {
			t := domain.BadgeTier(*row.Tier)
			tier = &t
		}
		var metric *domain.BadgeMetric
		if row.Metric != nil {
			m := domain.BadgeMetric(*row.Metric)
			metric = &m
		}
		var evidence *string
		if len(row.Evidence) > 0 {
			s := string(row.Evidence)
			evidence = &s
		}
		out[i] = domain.ArtisanBadge{
			Badge: domain.Badge{
				ID: row.BadgeID, Code: row.Code, Kind: domain.BadgeKind(row.Kind),
				Tier: tier, IconName: row.IconName, Metric: metric,
				Threshold: row.Threshold, SortOrder: row.SortOrder,
			},
			GrantedAt: row.GrantedAt, GrantedBy: row.GrantedBy, Evidence: evidence,
		}
	}
	return out, nil
}

// GetBadgeProgress reads an artisan's current counters, one row per metric
// they have any activity in (a metric never touched simply has no row).
func (r *Repo) GetBadgeProgress(ctx context.Context, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error) {
	rows, err := r.q.GetBadgeProgress(ctx, artisanID)
	if err != nil {
		return nil, translate(err, "badge progress")
	}
	out := make([]domain.BadgeProgressEntry, len(rows))
	for i, row := range rows {
		out[i] = domain.BadgeProgressEntry{
			Metric: domain.BadgeMetric(row.Metric), Value: row.Value, UpdatedAt: row.UpdatedAt,
		}
	}
	return out, nil
}

// RecomputeAndGrant sets an artisan's progress counter for metric to the
// given (freshly recomputed) count, then grants every EARNED badge on that
// metric whose threshold is now met and which the artisan does not already
// hold. Both statements are naturally idempotent: an UPSERT that sets an
// absolute value, and an INSERT ... SELECT ... WHERE NOT EXISTS ... ON
// CONFLICT DO NOTHING. A Kafka redelivery calling this twice with the same
// count is a no-op the second time -- no explicit transaction or dedupe
// ledger is needed.
func (r *Repo) RecomputeAndGrant(ctx context.Context, artisanID uuid.UUID, metric domain.BadgeMetric, count int64) ([]domain.Badge, error) {
	if _, err := r.q.UpsertBadgeProgress(ctx, db.UpsertBadgeProgressParams{
		ArtisanID: artisanID, Metric: db.BadgeMetric(metric), Value: count,
	}); err != nil {
		return nil, translate(err, "badge progress")
	}

	rows, err := r.q.GrantEligibleEarnedBadges(ctx, db.GrantEligibleEarnedBadgesParams{
		ArtisanID: artisanID, Metric: db.BadgeMetric(metric), Threshold: count,
	})
	if err != nil {
		return nil, translate(err, "badge grant")
	}
	return r.hydrateGrantedBadges(ctx, rows)
}

// hydrateGrantedBadges looks up each newly-granted row's catalog entry by id
// so the caller (and ultimately the consumer's log line / any future
// notification) has the badge's code, not just its id. GrantEligibleEarnedBadges
// returns artisan_badge rows, which do not carry the catalog columns.
func (r *Repo) hydrateGrantedBadges(ctx context.Context, rows []db.ArtisanBadge) ([]domain.Badge, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	catalog, err := r.q.ListBadgeCatalog(ctx)
	if err != nil {
		return nil, translate(err, "badge catalog")
	}
	byID := make(map[uuid.UUID]db.Badge, len(catalog))
	for _, b := range catalog {
		byID[b.ID] = b
	}
	out := make([]domain.Badge, 0, len(rows))
	for _, row := range rows {
		if b, ok := byID[row.BadgeID]; ok {
			out = append(out, badgeFromRow(b))
		}
	}
	return out, nil
}

// CountPublishedListings, CountSealedProvenance, CountAcceptedLots, and
// CountCompletedLots are the source-of-truth counts RecomputeAndGrant's
// caller (service.Badges, see Task 6) passes in as count.

func (r *Repo) CountPublishedListings(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	n, err := r.q.CountPublishedListings(ctx, artisanID)
	return n, translate(err, "published listing count")
}

func (r *Repo) CountSealedProvenance(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	n, err := r.q.CountSealedProvenance(ctx, artisanID)
	return n, translate(err, "sealed provenance count")
}

func (r *Repo) CountAcceptedLots(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	n, err := r.q.CountAcceptedLots(ctx, artisanID)
	return n, translate(err, "accepted lot count")
}

func (r *Repo) CountCompletedLots(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	n, err := r.q.CountCompletedLots(ctx, artisanID)
	return n, translate(err, "completed lot count")
}

// GrantConferredBadge writes (or re-writes, clearing any prior revoke) an
// admin-conferred grant.
func (r *Repo) GrantConferredBadge(ctx context.Context, in domain.GrantBadgeInput) (domain.ArtisanBadge, error) {
	var evidence []byte
	if in.Evidence != nil {
		evidence = []byte(*in.Evidence)
	}
	row, err := r.q.InsertConferredGrant(ctx, db.InsertConferredGrantParams{
		ArtisanID: in.ArtisanID, BadgeID: in.BadgeID, GrantedBy: in.GrantedBy, Evidence: evidence,
	})
	if err != nil {
		return domain.ArtisanBadge{}, translate(err, "badge grant")
	}
	badge, err := r.hydrateGrantedBadges(ctx, []db.ArtisanBadge{row})
	if err != nil || len(badge) == 0 {
		return domain.ArtisanBadge{}, translate(err, "badge")
	}
	return domain.ArtisanBadge{Badge: badge[0], GrantedAt: row.GrantedAt, GrantedBy: row.GrantedBy}, nil
}

// RevokeBadge marks a grant revoked. Returns false if there was no active
// grant to revoke (already revoked, or never granted).
func (r *Repo) RevokeBadge(ctx context.Context, in domain.RevokeBadgeInput) (bool, error) {
	n, err := r.q.RevokeGrant(ctx, db.RevokeGrantParams{
		ArtisanID: in.ArtisanID, BadgeID: in.BadgeID, RevokedBy: in.RevokedBy, RevokeReason: in.Reason,
	})
	if err != nil {
		return false, translate(err, "badge revoke")
	}
	return n > 0, nil
}

func badgeFromRow(row db.Badge) domain.Badge {
	var tier *domain.BadgeTier
	if row.Tier != nil {
		t := domain.BadgeTier(*row.Tier)
		tier = &t
	}
	var metric *domain.BadgeMetric
	if row.Metric != nil {
		m := domain.BadgeMetric(*row.Metric)
		metric = &m
	}
	return domain.Badge{
		ID: row.ID, Code: row.Code, Kind: domain.BadgeKind(row.Kind), Tier: tier,
		IconName: row.IconName, Metric: metric, Threshold: row.Threshold, SortOrder: row.SortOrder,
	}
}
```

Check the exact generated field names (`db.GetBadgeByCodeParams`, `db.UpsertBadgeProgressParams`, `db.GrantEligibleEarnedBadgesParams`, `db.InsertConferredGrantParams`, `db.RevokeGrantParams`, `db.ArtisanBadge`, `db.Badge`) against what `sqlc generate` actually produced in Task 3, Step 3 — sqlc's exact casing (`Url` not `URL`, seen earlier in `db.CreateTrendLinkParams.Url`) may differ slightly from what's guessed here; adjust field names to match the generated file, not this plan, if they differ.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/core-svc && go test ./internal/core/repo/... -run TestRepo_RecomputeAndGrant -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/core-svc/internal/core/repo/badge.go services/core-svc/internal/core/repo/badge_test.go
git commit -m "feat(core-svc): add badge repository

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 6: Service layer — `service/badge.go`

**Files:**
- Create: `services/core-svc/internal/core/service/badge.go`
- Test: `services/core-svc/internal/core/service/badge_test.go`

**Interfaces:**
- Consumes: `domain.*` (Task 4), a `BadgeStore` interface satisfied by `*repo.Repo` (Task 5), `auth.PrincipalFrom(ctx) (auth.Principal, bool)`, `auth.RequirePrincipal(ctx) (auth.Principal, error)`, `auth.RequireRole(ctx, roles ...auth.Role) (auth.Principal, error)`, `auth.RoleMinistry`.
- Produces: `Badges` struct, `NewBadges(store BadgeStore, log *slog.Logger) *Badges`, methods:
  - `ListBadgeCatalog(ctx) ([]domain.Badge, error)` — no auth check (public).
  - `ListArtisanBadges(ctx, artisanID uuid.UUID) ([]domain.ArtisanBadge, error)` — no auth check (public).
  - `GetBadgeProgress(ctx, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error)` — caller must be that artisan.
  - `GrantBadge(ctx, in domain.GrantBadgeInput) (domain.ArtisanBadge, error)` — MINISTRY only; rejects a `BadgeID` whose catalog `Kind != CONFERRED`.
  - `RevokeBadge(ctx, in domain.RevokeBadgeInput) (bool, error)` — MINISTRY only.
  - `TrackListingPublished(ctx, artisanID uuid.UUID) ([]domain.Badge, error)` — no auth check (called only from the internal Kafka consumer, never from a network-facing RPC).
  - `TrackProvenanceSealed(ctx, artisanID uuid.UUID) ([]domain.Badge, error)` — same.
  - `TrackLotAccepted(ctx, artisanID uuid.UUID) ([]domain.Badge, error)` — same.
  - `TrackLotCompleted(ctx, artisanID uuid.UUID) ([]domain.Badge, error)` — same.

- [ ] **Step 1: Write the failing test**

```go
// services/core-svc/internal/core/service/badge_test.go
package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

type fakeBadgeStore struct {
	catalog          []domain.Badge
	countPublished   int64
	recomputeCalls   []domain.BadgeMetric
	grantedOnRecompute []domain.Badge
}

func (f *fakeBadgeStore) ListBadgeCatalog(ctx context.Context) ([]domain.Badge, error) { return f.catalog, nil }
func (f *fakeBadgeStore) ListArtisanBadges(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanBadge, error) { return nil, nil }
func (f *fakeBadgeStore) GetBadgeProgress(ctx context.Context, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error) { return nil, nil }
func (f *fakeBadgeStore) RecomputeAndGrant(ctx context.Context, artisanID uuid.UUID, metric domain.BadgeMetric, count int64) ([]domain.Badge, error) {
	f.recomputeCalls = append(f.recomputeCalls, metric)
	return f.grantedOnRecompute, nil
}
func (f *fakeBadgeStore) CountPublishedListings(ctx context.Context, artisanID uuid.UUID) (int64, error) { return f.countPublished, nil }
func (f *fakeBadgeStore) CountSealedProvenance(ctx context.Context, artisanID uuid.UUID) (int64, error) { return 0, nil }
func (f *fakeBadgeStore) CountAcceptedLots(ctx context.Context, artisanID uuid.UUID) (int64, error)     { return 0, nil }
func (f *fakeBadgeStore) CountCompletedLots(ctx context.Context, artisanID uuid.UUID) (int64, error)    { return 0, nil }
func (f *fakeBadgeStore) GetBadgeByCode(ctx context.Context, code string, tier *domain.BadgeTier) (domain.Badge, error) { return domain.Badge{}, nil }
func (f *fakeBadgeStore) GrantConferredBadge(ctx context.Context, in domain.GrantBadgeInput) (domain.ArtisanBadge, error) { return domain.ArtisanBadge{}, nil }
func (f *fakeBadgeStore) RevokeBadge(ctx context.Context, in domain.RevokeBadgeInput) (bool, error) { return false, nil }

func TestBadges_TrackListingPublished_RecomputesFromRealCount(t *testing.T) {
	store := &fakeBadgeStore{countPublished: 5}
	svc := NewBadges(store, nil)

	_, err := svc.TrackListingPublished(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, []domain.BadgeMetric{domain.MetricListingsPublished}, store.recomputeCalls)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/core-svc && go test ./internal/core/service/... -run TestBadges_TrackListingPublished -v`
Expected: FAIL (compile error — `NewBadges` undefined).

- [ ] **Step 3: Write the implementation**

```go
// services/core-svc/internal/core/service/badge.go

package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// BadgeStore is the persistence surface the badge service depends on.
type BadgeStore interface {
	ListBadgeCatalog(ctx context.Context) ([]domain.Badge, error)
	ListArtisanBadges(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanBadge, error)
	GetBadgeProgress(ctx context.Context, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error)
	RecomputeAndGrant(ctx context.Context, artisanID uuid.UUID, metric domain.BadgeMetric, count int64) ([]domain.Badge, error)
	CountPublishedListings(ctx context.Context, artisanID uuid.UUID) (int64, error)
	CountSealedProvenance(ctx context.Context, artisanID uuid.UUID) (int64, error)
	CountAcceptedLots(ctx context.Context, artisanID uuid.UUID) (int64, error)
	CountCompletedLots(ctx context.Context, artisanID uuid.UUID) (int64, error)
	GetBadgeByCode(ctx context.Context, code string, tier *domain.BadgeTier) (domain.Badge, error)
	GrantConferredBadge(ctx context.Context, in domain.GrantBadgeInput) (domain.ArtisanBadge, error)
	RevokeBadge(ctx context.Context, in domain.RevokeBadgeInput) (bool, error)
}

// Badges is the service for the artisan badge catalog, grants, and
// activity-based auto-grants.
type Badges struct {
	store BadgeStore
	log   *slog.Logger
}

// NewBadges builds the badge service.
func NewBadges(store BadgeStore, log *slog.Logger) *Badges {
	if log == nil {
		log = slog.Default()
	}
	return &Badges{store: store, log: log}
}

// ListBadgeCatalog is public reference data -- no principal required.
func (b *Badges) ListBadgeCatalog(ctx context.Context) ([]domain.Badge, error) {
	return b.store.ListBadgeCatalog(ctx)
}

// ListArtisanBadges is public -- it backs the buyer storefront's badge row,
// which an anonymous buyer must be able to see.
func (b *Badges) ListArtisanBadges(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanBadge, error) {
	return b.store.ListArtisanBadges(ctx, artisanID)
}

// GetBadgeProgress requires the caller to be the artisan whose progress is
// being read -- unlike grants, in-progress counters are not public.
func (b *Badges) GetBadgeProgress(ctx context.Context, artisanID uuid.UUID) ([]domain.BadgeProgressEntry, error) {
	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if principal.Subject != artisanID.String() {
		return nil, pkgdomain.Forbidden("cannot read another artisan's badge progress")
	}
	return b.store.GetBadgeProgress(ctx, artisanID)
}

// GrantBadge confers a CONFERRED badge. Only a MINISTRY principal may call
// this, and only against a badge whose catalog kind is CONFERRED -- EARNED
// badges are only ever written by the Track* methods below.
func (b *Badges) GrantBadge(ctx context.Context, in domain.GrantBadgeInput) (domain.ArtisanBadge, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return domain.ArtisanBadge{}, err
	}
	if err := in.Validate(); err != nil {
		return domain.ArtisanBadge{}, err
	}
	badge, err := b.badgeByID(ctx, in.BadgeID)
	if err != nil {
		return domain.ArtisanBadge{}, err
	}
	if badge.Kind != domain.BadgeKindConferred {
		return domain.ArtisanBadge{}, pkgdomain.InvalidInput("badge " + badge.Code + " is earned automatically and cannot be granted directly")
	}
	return b.store.GrantConferredBadge(ctx, in)
}

// RevokeBadge revokes a badge grant. MINISTRY only.
func (b *Badges) RevokeBadge(ctx context.Context, in domain.RevokeBadgeInput) (bool, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return false, err
	}
	if err := in.Validate(); err != nil {
		return false, err
	}
	return b.store.RevokeBadge(ctx, in)
}

// badgeByID is a small helper: the store has no GetBadgeByID, only
// GetBadgeByCode, so GrantBadge instead reads the full catalog and finds the
// row -- the catalog is 14 rows, so this is cheap, and it avoids adding a
// query used from exactly one call site.
func (b *Badges) badgeByID(ctx context.Context, id uuid.UUID) (domain.Badge, error) {
	catalog, err := b.store.ListBadgeCatalog(ctx)
	if err != nil {
		return domain.Badge{}, err
	}
	for _, badge := range catalog {
		if badge.ID == id {
			return badge, nil
		}
	}
	return domain.Badge{}, pkgdomain.NotFound("badge not found")
}

// TrackListingPublished recomputes LISTINGS_PUBLISHED from the real count of
// the artisan's published listings and grants any newly-crossed tier. Called
// only from the Kafka consumer (handler.CatalogListingPublishedHandler),
// never from a network-facing RPC, so it takes no principal.
func (b *Badges) TrackListingPublished(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	count, err := b.store.CountPublishedListings(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	return b.recomputeAndLog(ctx, artisanID, domain.MetricListingsPublished, count)
}

// TrackProvenanceSealed recomputes PROVENANCE_SEALED.
func (b *Badges) TrackProvenanceSealed(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	count, err := b.store.CountSealedProvenance(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	return b.recomputeAndLog(ctx, artisanID, domain.MetricProvenanceSealed, count)
}

// TrackLotAccepted recomputes LOTS_ACCEPTED. No badge in the current catalog
// is keyed on this metric yet -- it is tracked so a future badge can be
// seeded against it without a code change, matching how CountAcceptedLots
// already exists in the repo (see spec's LOTS_ACCEPTED enum value).
func (b *Badges) TrackLotAccepted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	count, err := b.store.CountAcceptedLots(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	return b.recomputeAndLog(ctx, artisanID, domain.MetricLotsAccepted, count)
}

// TrackLotCompleted recomputes LOTS_COMPLETED (the order_fulfiller family).
func (b *Badges) TrackLotCompleted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	count, err := b.store.CountCompletedLots(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	return b.recomputeAndLog(ctx, artisanID, domain.MetricLotsCompleted, count)
}

func (b *Badges) recomputeAndLog(ctx context.Context, artisanID uuid.UUID, metric domain.BadgeMetric, count int64) ([]domain.Badge, error) {
	granted, err := b.store.RecomputeAndGrant(ctx, artisanID, metric, count)
	if err != nil {
		return nil, err
	}
	for _, badge := range granted {
		b.log.InfoContext(ctx, "badge granted", "artisan_id", artisanID, "badge_code", badge.Code, "metric", metric, "count", count)
	}
	return granted, nil
}

var _ = errors.New // keep errors imported if pkgdomain helpers below don't already need it; remove this line if unused after final edit
```

Remove the trailing `var _ = errors.New` line — it was left in only as a placeholder reminder and is not needed; `errors` is not otherwise used in this file, so drop the import too if `go vet` flags it unused.

Check `pkgdomain.Forbidden`, `pkgdomain.InvalidInput`, `pkgdomain.NotFound` are the actual exported constructor names in `pkg/domain` (confirm with `grep -n "^func Forbidden\|^func InvalidInput\|^func NotFound" pkg/domain/*.go`) before finalizing — adjust names if the package uses different casing or a different constructor set.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/core-svc && go test ./internal/core/service/... -run TestBadges_TrackListingPublished -v`
Expected: PASS

- [ ] **Step 5: Add and pass one auth-focused test**

```go
func TestBadges_GrantBadge_RejectsEarnedBadge(t *testing.T) {
	earnedBadge := domain.Badge{ID: uuid.New(), Code: "first_listing", Kind: domain.BadgeKindEarned}
	store := &fakeBadgeStore{catalog: []domain.Badge{earnedBadge}}
	svc := NewBadges(store, nil)

	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: "admin-1", Role: auth.RoleMinistry}) // use this package's real helper to inject a test principal -- check pkg/auth/context.go for its exact name (e.g. WithPrincipal) and adjust

	_, err := svc.GrantBadge(ctx, domain.GrantBadgeInput{ArtisanID: uuid.New(), BadgeID: earnedBadge.ID, GrantedBy: "admin-1"})
	require.Error(t, err)
}
```

Check `pkg/auth/context.go` for the actual exported function that injects a `Principal` into a `context.Context` for tests (search `grep -n "^func.*Context" pkg/auth/context.go`) — it is used by other service tests in this same package (e.g. `trends_test.go` or `catalog_test.go` likely has an auth-required test already); copy that exact call from an existing test rather than guessing the name.

Run: `cd services/core-svc && go test ./internal/core/service/... -run TestBadges_GrantBadge_RejectsEarnedBadge -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add services/core-svc/internal/core/service/badge.go services/core-svc/internal/core/service/badge_test.go
git commit -m "feat(core-svc): add badge service with recompute-based earned grants

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 7: Proto — `proto/badges/v1/badges.proto`

**Files:**
- Create: `proto/badges/v1/badges.proto`

**Interfaces:**
- Produces (after `make proto`, in `pkg/pb/badges/v1/`): `BadgeServiceServer`/`BadgeServiceClient`, `Badge`, `ArtisanBadge`, `BadgeProgressEntry` messages, request/response types — consumed by Task 8 (core-svc handler) and Task 11 (bff client).

- [ ] **Step 1: Write the proto file**

```protobuf
// proto/badges/v1/badges.proto

syntax = "proto3";

package badges.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/ZoroNewbie00/kalakriti/pkg/pb/badges/v1;badgesv1";

// BadgeService manages the artisan recognition badge catalog and grants.
service BadgeService {
  // List the active badge catalog (public reference data).
  rpc ListBadgeCatalog(ListBadgeCatalogRequest) returns (ListBadgeCatalogResponse);
  // List one artisan's active badge grants (public).
  rpc ListArtisanBadges(ListArtisanBadgesRequest) returns (ListArtisanBadgesResponse);
  // Get the caller's own progress toward each earned-badge metric.
  rpc GetBadgeProgress(GetBadgeProgressRequest) returns (GetBadgeProgressResponse);
  // Confer a CONFERRED badge on an artisan (MINISTRY only).
  rpc GrantBadge(GrantBadgeRequest) returns (GrantBadgeResponse);
  // Revoke a badge grant (MINISTRY only).
  rpc RevokeBadge(RevokeBadgeRequest) returns (RevokeBadgeResponse);
}

enum BadgeKind {
  BADGE_KIND_UNSPECIFIED = 0;
  BADGE_KIND_EARNED = 1;
  BADGE_KIND_CONFERRED = 2;
}

enum BadgeTier {
  BADGE_TIER_UNSPECIFIED = 0;
  BADGE_TIER_BRONZE = 1;
  BADGE_TIER_SILVER = 2;
  BADGE_TIER_GOLD = 3;
}

enum BadgeMetric {
  BADGE_METRIC_UNSPECIFIED = 0;
  BADGE_METRIC_LISTINGS_PUBLISHED = 1;
  BADGE_METRIC_PROVENANCE_SEALED = 2;
  BADGE_METRIC_LOTS_ACCEPTED = 3;
  BADGE_METRIC_LOTS_COMPLETED = 4;
}

message Badge {
  string id = 1;
  string code = 2;
  BadgeKind kind = 3;
  BadgeTier tier = 4; // BADGE_TIER_UNSPECIFIED for an untiered badge
  string icon_name = 5;
  BadgeMetric metric = 6; // BADGE_METRIC_UNSPECIFIED for a conferred badge
  optional int64 threshold = 7;
  int32 sort_order = 8;
}

message ArtisanBadge {
  Badge badge = 1;
  google.protobuf.Timestamp granted_at = 2;
  string granted_by = 3;
}

message BadgeProgressEntry {
  BadgeMetric metric = 1;
  int64 value = 2;
  google.protobuf.Timestamp updated_at = 3;
}

message ListBadgeCatalogRequest {}
message ListBadgeCatalogResponse { repeated Badge badges = 1; }

message ListArtisanBadgesRequest { string artisan_id = 1; }
message ListArtisanBadgesResponse { repeated ArtisanBadge artisan_badges = 1; }

message GetBadgeProgressRequest { string artisan_id = 1; }
message GetBadgeProgressResponse { repeated BadgeProgressEntry entries = 1; }

message GrantBadgeRequest {
  string artisan_id = 1;
  string badge_id = 2;
  optional string evidence_json = 3;
}
message GrantBadgeResponse { ArtisanBadge artisan_badge = 1; }

message RevokeBadgeRequest {
  string artisan_id = 1;
  string badge_id = 2;
  string reason = 3;
}
message RevokeBadgeResponse { bool revoked = 1; }
```

- [ ] **Step 2: Generate and verify**

Run: `make proto` (from repo root)
Expected: `pkg/pb/badges/v1/badges.pb.go` and `badges_grpc.pb.go` are created with no errors. Run `go build ./...` from repo root to confirm nothing else broke.

- [ ] **Step 3: Commit**

`pkg/pb/*` is `.gitignore`d per `CLAUDE.md` ("generated code, correctly not committed") — only the `.proto` source is committed:

```bash
git add proto/badges/v1/badges.proto
git commit -m "feat(proto): add badges.v1.BadgeService

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 8: gRPC handler — `handler/badge.go`

**Files:**
- Create: `services/core-svc/internal/core/handler/badge.go`

**Interfaces:**
- Consumes: `service.Badges` (Task 6), `badgesv1.*` (Task 7).
- Produces: `Badges` handler struct, `NewBadges(svc *service.Badges) *Badges` implementing `badgesv1.BadgeServiceServer` — registered in Task 10.

- [ ] **Step 1: Write the handler**

```go
// services/core-svc/internal/core/handler/badge.go

package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	badgesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/badges/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Badges implements badges.v1.BadgeService.
type Badges struct {
	badgesv1.UnimplementedBadgeServiceServer
	svc *service.Badges
}

// NewBadges builds the badge handler.
func NewBadges(svc *service.Badges) *Badges {
	return &Badges{svc: svc}
}

func (h *Badges) ListBadgeCatalog(ctx context.Context, req *badgesv1.ListBadgeCatalogRequest) (*badgesv1.ListBadgeCatalogResponse, error) {
	catalog, err := h.svc.ListBadgeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*badgesv1.Badge, len(catalog))
	for i, b := range catalog {
		out[i] = toProtoBadge(b)
	}
	return &badgesv1.ListBadgeCatalogResponse{Badges: out}, nil
}

func (h *Badges) ListArtisanBadges(ctx context.Context, req *badgesv1.ListArtisanBadgesRequest) (*badgesv1.ListArtisanBadgesResponse, error) {
	artisanID, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, err
	}
	grants, err := h.svc.ListArtisanBadges(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	out := make([]*badgesv1.ArtisanBadge, len(grants))
	for i, g := range grants {
		out[i] = &badgesv1.ArtisanBadge{
			Badge: toProtoBadge(g.Badge), GrantedAt: timestamppb.New(g.GrantedAt), GrantedBy: g.GrantedBy,
		}
	}
	return &badgesv1.ListArtisanBadgesResponse{ArtisanBadges: out}, nil
}

func (h *Badges) GetBadgeProgress(ctx context.Context, req *badgesv1.GetBadgeProgressRequest) (*badgesv1.GetBadgeProgressResponse, error) {
	artisanID, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, err
	}
	entries, err := h.svc.GetBadgeProgress(ctx, artisanID)
	if err != nil {
		return nil, err
	}
	out := make([]*badgesv1.BadgeProgressEntry, len(entries))
	for i, e := range entries {
		out[i] = &badgesv1.BadgeProgressEntry{
			Metric: toProtoMetric(e.Metric), Value: e.Value, UpdatedAt: timestamppb.New(e.UpdatedAt),
		}
	}
	return &badgesv1.GetBadgeProgressResponse{Entries: out}, nil
}

func (h *Badges) GrantBadge(ctx context.Context, req *badgesv1.GrantBadgeRequest) (*badgesv1.GrantBadgeResponse, error) {
	artisanID, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, err
	}
	badgeID, err := uuid.Parse(req.GetBadgeId())
	if err != nil {
		return nil, err
	}
	grant, err := h.svc.GrantBadge(ctx, domain.GrantBadgeInput{
		ArtisanID: artisanID, BadgeID: badgeID, GrantedBy: principalSubjectOrEmpty(ctx), Evidence: req.EvidenceJson,
	})
	if err != nil {
		return nil, err
	}
	return &badgesv1.GrantBadgeResponse{ArtisanBadge: &badgesv1.ArtisanBadge{
		Badge: toProtoBadge(grant.Badge), GrantedAt: timestamppb.New(grant.GrantedAt), GrantedBy: grant.GrantedBy,
	}}, nil
}

func (h *Badges) RevokeBadge(ctx context.Context, req *badgesv1.RevokeBadgeRequest) (*badgesv1.RevokeBadgeResponse, error) {
	artisanID, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, err
	}
	badgeID, err := uuid.Parse(req.GetBadgeId())
	if err != nil {
		return nil, err
	}
	revoked, err := h.svc.RevokeBadge(ctx, domain.RevokeBadgeInput{
		ArtisanID: artisanID, BadgeID: badgeID, RevokedBy: principalSubjectOrEmpty(ctx), Reason: req.GetReason(),
	})
	if err != nil {
		return nil, err
	}
	return &badgesv1.RevokeBadgeResponse{Revoked: revoked}, nil
}

func toProtoBadge(b domain.Badge) *badgesv1.Badge {
	out := &badgesv1.Badge{
		Id: b.ID.String(), Code: b.Code, Kind: toProtoKind(b.Kind), IconName: b.IconName, SortOrder: b.SortOrder,
	}
	if b.Tier != nil {
		out.Tier = toProtoTier(*b.Tier)
	}
	if b.Metric != nil {
		out.Metric = toProtoMetric(*b.Metric)
	}
	out.Threshold = b.Threshold
	return out
}

func toProtoKind(k domain.BadgeKind) badgesv1.BadgeKind {
	if k == domain.BadgeKindConferred {
		return badgesv1.BadgeKind_BADGE_KIND_CONFERRED
	}
	return badgesv1.BadgeKind_BADGE_KIND_EARNED
}

func toProtoTier(t domain.BadgeTier) badgesv1.BadgeTier {
	switch t {
	case domain.BadgeTierBronze:
		return badgesv1.BadgeTier_BADGE_TIER_BRONZE
	case domain.BadgeTierSilver:
		return badgesv1.BadgeTier_BADGE_TIER_SILVER
	case domain.BadgeTierGold:
		return badgesv1.BadgeTier_BADGE_TIER_GOLD
	default:
		return badgesv1.BadgeTier_BADGE_TIER_UNSPECIFIED
	}
}

func toProtoMetric(m domain.BadgeMetric) badgesv1.BadgeMetric {
	switch m {
	case domain.MetricListingsPublished:
		return badgesv1.BadgeMetric_BADGE_METRIC_LISTINGS_PUBLISHED
	case domain.MetricProvenanceSealed:
		return badgesv1.BadgeMetric_BADGE_METRIC_PROVENANCE_SEALED
	case domain.MetricLotsAccepted:
		return badgesv1.BadgeMetric_BADGE_METRIC_LOTS_ACCEPTED
	case domain.MetricLotsCompleted:
		return badgesv1.BadgeMetric_BADGE_METRIC_LOTS_COMPLETED
	default:
		return badgesv1.BadgeMetric_BADGE_METRIC_UNSPECIFIED
	}
}
```

`principalSubjectOrEmpty(ctx)` is a small helper needed here that does not yet exist: add it (or reuse an identical existing helper if `handler` package already has one — check with `grep -rn "func principalSubjectOrEmpty\|auth.PrincipalFrom" services/core-svc/internal/core/handler/*.go` first). If it does not exist, add to the bottom of `handler/badge.go`:

```go
func principalSubjectOrEmpty(ctx context.Context) string {
	if p, ok := auth.PrincipalFrom(ctx); ok {
		return p.Subject
	}
	return ""
}
```

(add `"github.com/ZoroNewbie00/kalakriti/pkg/auth"` to the imports if this helper is added here).

- [ ] **Step 2: Verify it compiles**

Run: `cd services/core-svc && go build ./...`
Expected: no errors (will only succeed once Task 9/10 wire it in, or once this file alone type-checks against Task 6/7's real generated types — run `go vet ./internal/core/handler/...` to type-check in isolation first).

- [ ] **Step 3: Commit**

```bash
git add services/core-svc/internal/core/handler/badge.go
git commit -m "feat(core-svc): add badge gRPC handler

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 9: Kafka consumer handlers in `handler/consumer.go`

**Files:**
- Modify: `services/core-svc/internal/core/handler/consumer.go`
- Test: `services/core-svc/internal/core/handler/consumer_test.go`

**Interfaces:**
- Consumes: `service.Badges` (Task 6, specifically `TrackListingPublished`/`TrackProvenanceSealed`/`TrackLotAccepted`/`TrackLotCompleted`), existing `decodeAggregate(msg segmentio.Message, payload any) (uuid.UUID, error)` helper, `kafka.HandlerFunc` type.
- Produces: `CatalogListingPublishedBadgeHandler(svc *service.Badges, log *slog.Logger) kafka.HandlerFunc`, `CatalogProvenanceSealedHandler(svc *service.Badges, log *slog.Logger) kafka.HandlerFunc`, `OrderLotAcceptedHandler(svc *service.Badges, log *slog.Logger) kafka.HandlerFunc`, `OrderLotCompletedHandler(svc *service.Badges, log *slog.Logger) kafka.HandlerFunc` — wired into `main.go` in Task 10.

- [ ] **Step 1: Write the failing test**

```go
// services/core-svc/internal/core/handler/consumer_test.go — add to this file if it
// already exists (it likely does, given ListingPublishedHandler is tested
// somewhere in this package); otherwise create it following this package's
// existing test style for other *Handler functions.
package handler

import (
	"context"
	"encoding/json"
	"testing"

	segmentio "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

type fakeBadgeTracker struct {
	trackedArtisanIDs []string
}

func (f *fakeBadgeTracker) TrackListingPublished(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	f.trackedArtisanIDs = append(f.trackedArtisanIDs, artisanID.String())
	return nil, nil
}

func TestCatalogListingPublishedBadgeHandler_TracksArtisan(t *testing.T) {
	artisanID := "018f1e2a-0000-7000-8000-000000000001"
	envelope := map[string]any{
		"header":  map[string]any{"aggregate_id": "018f1e2a-0000-7000-8000-000000000099"},
		"payload": map[string]any{"listing_id": "018f1e2a-0000-7000-8000-000000000099", "artisan_id": artisanID, "craft_id": "018f1e2a-0000-7000-8000-000000000002"},
	}
	value, err := json.Marshal(envelope)
	require.NoError(t, err)

	tracker := &fakeBadgeTracker{}
	handle := CatalogListingPublishedBadgeHandler(tracker, nil)

	err = handle(context.Background(), segmentio.Message{Value: value})
	require.NoError(t, err)
	require.Equal(t, []string{artisanID}, tracker.trackedArtisanIDs)
}
```

`CatalogListingPublishedBadgeHandler`'s first parameter should be typed against a small local interface (`badgeTracker` below), not the concrete `*service.Badges`, so this fake can substitute for it — see Step 3.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/core-svc && go test ./internal/core/handler/... -run TestCatalogListingPublishedBadgeHandler -v`
Expected: FAIL (compile error — handler undefined).

- [ ] **Step 3: Write the implementation**

Add to `services/core-svc/internal/core/handler/consumer.go` (existing file; add these payload structs near the top alongside `listingPublished`/`mediaUploaded`, and the handler functions near `ListingPublishedHandler`):

```go
// provenanceSealed mirrors events.v1.CatalogProvenanceSealed's payload fields
// needed by the badge tracker.
type provenanceSealed struct {
	ProvenanceID string `json:"provenance_id"`
	ListingID    string `json:"listing_id"`
	ArtisanID    string `json:"artisan_id"`
}

// lotEvent mirrors the lot event payload shape emitted by collab-svc for
// OrderLotAccepted and OrderLotCompleted (see
// services/collab-svc/internal/collab/service/fulfilment.go's
// lotEventPayload) -- only the fields the badge tracker needs are decoded.
type lotEvent struct {
	LotID       string `json:"lot_id"`
	BulkOrderID string `json:"bulk_order_id"`
	ArtisanID   string `json:"artisan_id"`
}

// badgeTracker is the subset of *service.Badges the badge consumer handlers
// need, seamed as an interface so they are testable without a database.
type badgeTracker interface {
	TrackListingPublished(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error)
	TrackProvenanceSealed(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error)
	TrackLotAccepted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error)
	TrackLotCompleted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error)
}

// CatalogListingPublishedBadgeHandler recomputes the LISTINGS_PUBLISHED
// badge metric whenever a listing is published.
func CatalogListingPublishedBadgeHandler(tracker badgeTracker, log *slog.Logger) kafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		var payload listingPublished
		if _, err := decodeAggregate(msg, &payload); err != nil {
			return err
		}
		artisanID, err := uuid.Parse(payload.ArtisanID)
		if err != nil {
			return fmt.Errorf("artisan_id %q is not a uuid: %w", payload.ArtisanID, err)
		}
		if _, err := tracker.TrackListingPublished(ctx, artisanID); err != nil {
			return fmt.Errorf("tracking listing-published badge for artisan %s: %w", artisanID, err)
		}
		return nil
	}
}

// CatalogProvenanceSealedBadgeHandler recomputes PROVENANCE_SEALED.
func CatalogProvenanceSealedBadgeHandler(tracker badgeTracker, log *slog.Logger) kafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		var payload provenanceSealed
		if _, err := decodeAggregate(msg, &payload); err != nil {
			return err
		}
		artisanID, err := uuid.Parse(payload.ArtisanID)
		if err != nil {
			return fmt.Errorf("artisan_id %q is not a uuid: %w", payload.ArtisanID, err)
		}
		if _, err := tracker.TrackProvenanceSealed(ctx, artisanID); err != nil {
			return fmt.Errorf("tracking provenance-sealed badge for artisan %s: %w", artisanID, err)
		}
		return nil
	}
}

// OrderLotAcceptedBadgeHandler recomputes LOTS_ACCEPTED.
func OrderLotAcceptedBadgeHandler(tracker badgeTracker, log *slog.Logger) kafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		var payload lotEvent
		if _, err := decodeAggregate(msg, &payload); err != nil {
			return err
		}
		artisanID, err := uuid.Parse(payload.ArtisanID)
		if err != nil {
			return fmt.Errorf("artisan_id %q is not a uuid: %w", payload.ArtisanID, err)
		}
		if _, err := tracker.TrackLotAccepted(ctx, artisanID); err != nil {
			return fmt.Errorf("tracking lot-accepted badge for artisan %s: %w", artisanID, err)
		}
		return nil
	}
}

// OrderLotCompletedBadgeHandler recomputes LOTS_COMPLETED.
func OrderLotCompletedBadgeHandler(tracker badgeTracker, log *slog.Logger) kafka.HandlerFunc {
	return func(ctx context.Context, msg segmentio.Message) error {
		var payload lotEvent
		if _, err := decodeAggregate(msg, &payload); err != nil {
			return err
		}
		artisanID, err := uuid.Parse(payload.ArtisanID)
		if err != nil {
			return fmt.Errorf("artisan_id %q is not a uuid: %w", payload.ArtisanID, err)
		}
		if _, err := tracker.TrackLotCompleted(ctx, artisanID); err != nil {
			return fmt.Errorf("tracking lot-completed badge for artisan %s: %w", artisanID, err)
		}
		return nil
	}
}
```

Add `"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"` to this file's imports if not already present (it is not, per the file's imports read earlier in this plan's research — `handler/consumer.go` currently imports only `service`, not `domain`).

Rewrite the test's `fakeBadgeTracker` to implement all four `badgeTracker` methods (not just `TrackListingPublished`), returning `nil, nil` for the three unused ones, and fix its import of `uuid`/`domain`:

```go
type fakeBadgeTracker struct {
	trackedArtisanIDs []string
}

func (f *fakeBadgeTracker) TrackListingPublished(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) {
	f.trackedArtisanIDs = append(f.trackedArtisanIDs, artisanID.String())
	return nil, nil
}
func (f *fakeBadgeTracker) TrackProvenanceSealed(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error) { return nil, nil }
func (f *fakeBadgeTracker) TrackLotAccepted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error)     { return nil, nil }
func (f *fakeBadgeTracker) TrackLotCompleted(ctx context.Context, artisanID uuid.UUID) ([]domain.Badge, error)    { return nil, nil }
```

Add `"github.com/google/uuid"` and the `domain` import to `consumer_test.go`'s imports.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/core-svc && go test ./internal/core/handler/... -run TestCatalogListingPublishedBadgeHandler -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/core-svc/internal/core/handler/consumer.go services/core-svc/internal/core/handler/consumer_test.go
git commit -m "feat(core-svc): add Kafka handlers for earned-badge tracking

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 10: Wire it all into `main.go` and `PublicMethods()`

**Files:**
- Modify: `services/core-svc/cmd/core-svc/main.go`
- Modify: `services/core-svc/internal/core/handler/identity.go`

**Interfaces:**
- Consumes: everything from Tasks 5-9.
- Produces: a running `badges.v1.BadgeService` gRPC endpoint and four running Kafka consumer goroutines.

- [ ] **Step 1: Build and register the service/handler**

In `services/core-svc/cmd/core-svc/main.go`, near the existing `trendsSvc := service.NewTrends(repository, log)` / `trendsHandler := handler.NewTrends(trendsSvc)` lines, add:

```go
badgesSvc := service.NewBadges(repository, log)
badgesHandler := handler.NewBadges(badgesSvc)
```

Near `trendsv1.RegisterTrendServiceServer(grpcServer, trendsHandler)`, add:

```go
badgesv1.RegisterBadgeServiceServer(grpcServer, badgesHandler)
```

Add the import `badgesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/badges/v1"` alongside the existing `trendsv1` import.

- [ ] **Step 2: Wire the four consumer goroutines**

Near the existing `translationConsumer := pkgkafka.NewConsumerGroup(...)` block, add four more consumer groups — one per topic. Two of these topics (`OrderLotAccepted`, `OrderLotCompleted`) are published by **collab-svc**, not core-svc itself; that is fine, Kafka topics are cross-service:

```go
badgeListingConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
	Brokers: cfg.kafka.Brokers,
	Topic:   topics.CatalogListingPublished,
	GroupID: cfg.kafka.ConsumerGroupPrefix + "." + serviceName + ".badges.listing",
}, log)

badgeProvenanceConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
	Brokers: cfg.kafka.Brokers,
	Topic:   topics.CatalogProvenanceSealed,
	GroupID: cfg.kafka.ConsumerGroupPrefix + "." + serviceName + ".badges.provenance",
}, log)

badgeLotAcceptedConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
	Brokers: cfg.kafka.Brokers,
	Topic:   topics.OrderLotAccepted,
	GroupID: cfg.kafka.ConsumerGroupPrefix + "." + serviceName + ".badges.lot-accepted",
}, log)

badgeLotCompletedConsumer := pkgkafka.NewConsumerGroup(pkgkafka.ConsumerConfig{
	Brokers: cfg.kafka.Brokers,
	Topic:   topics.OrderLotCompleted,
	GroupID: cfg.kafka.ConsumerGroupPrefix + "." + serviceName + ".badges.lot-completed",
}, log)
```

A distinct `GroupID` per consumer is required (a shared group ID across different topics would silently drop most messages under Kafka's consumer-group partition assignment) — follow the existing `serviceName + "." + purpose` naming already used by `translationConsumer`'s group id, just with a `.badges.<subpurpose>` suffix so all four are visibly related in broker tooling.

Near the existing `wg.Add(1); go func() { ... translationConsumer.Run(...) ... }()` block, add four matching goroutines:

```go
wg.Add(1)
go func() {
	defer wg.Done()
	log.Info("badge listing-published consumer started", "topic", topics.CatalogListingPublished)
	if err := badgeListingConsumer.Run(bgCtx, handler.CatalogListingPublishedBadgeHandler(badgesSvc, log)); err != nil {
		log.Error("badge listing-published consumer stopped", "error", err)
	}
}()

wg.Add(1)
go func() {
	defer wg.Done()
	log.Info("badge provenance-sealed consumer started", "topic", topics.CatalogProvenanceSealed)
	if err := badgeProvenanceConsumer.Run(bgCtx, handler.CatalogProvenanceSealedBadgeHandler(badgesSvc, log)); err != nil {
		log.Error("badge provenance-sealed consumer stopped", "error", err)
	}
}()

wg.Add(1)
go func() {
	defer wg.Done()
	log.Info("badge lot-accepted consumer started", "topic", topics.OrderLotAccepted)
	if err := badgeLotAcceptedConsumer.Run(bgCtx, handler.OrderLotAcceptedBadgeHandler(badgesSvc, log)); err != nil {
		log.Error("badge lot-accepted consumer stopped", "error", err)
	}
}()

wg.Add(1)
go func() {
	defer wg.Done()
	log.Info("badge lot-completed consumer started", "topic", topics.OrderLotCompleted)
	if err := badgeLotCompletedConsumer.Run(bgCtx, handler.OrderLotCompletedBadgeHandler(badgesSvc, log)); err != nil {
		log.Error("badge lot-completed consumer stopped", "error", err)
	}
}()
```

`badgesSvc` (concrete `*service.Badges`) satisfies the `badgeTracker` interface from Task 9 structurally — no explicit cast needed.

- [ ] **Step 3: Public-read wiring, both layers**

In `services/core-svc/internal/core/handler/identity.go`'s `PublicMethods()`, add two lines alongside the existing `"/trends.v1.TrendService/ListTrendLinks"` entry:

```go
		"/badges.v1.BadgeService/ListBadgeCatalog",
		"/badges.v1.BadgeService/ListArtisanBadges",
```

This alone is not sufficient — confirm (it already is, per Task 6's implementation) that `Badges.ListBadgeCatalog` and `Badges.ListArtisanBadges` do not call `auth.RequirePrincipal`/`auth.RequireRole` (they don't; they take no principal at all). This satisfies both halves of the CLAUDE.md public-read rule.

- [ ] **Step 4: Build and smoke-test**

Run: `cd services/core-svc && go build ./...`
Expected: no errors.

Run (with a local stack up, per README): `grpcurl -plaintext localhost:<core-svc grpc port> badges.v1.BadgeService/ListBadgeCatalog`
Expected: 14 badges returned, no auth error.

- [ ] **Step 5: Commit**

```bash
git add services/core-svc/cmd/core-svc/main.go services/core-svc/internal/core/handler/identity.go
git commit -m "feat(core-svc): wire badge service, handler, and consumers into main.go

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 11: BFF client — `client/badge.go`

**Files:**
- Create: `services/bff/internal/bff/client/badge.go`

**Interfaces:**
- Consumes: `badgesv1.BadgeServiceClient` (Task 7).
- Produces: `Badges` client struct, `NewBadges(conn grpc.ClientConnInterface) *Badges`, methods returning `map[string]any` (matching the existing `client/trends.go` convention of untyped maps that `api.go` serializes directly to JSON):
  - `ListBadgeCatalog(ctx) ([]map[string]any, error)`
  - `ListArtisanBadges(ctx, artisanID string) ([]map[string]any, error)`
  - `GetBadgeProgress(ctx, artisanID string) ([]map[string]any, error)`
  - `GrantBadge(ctx, idempotencyKey, artisanID string, fields map[string]any) (map[string]any, error)`
  - `RevokeBadge(ctx, artisanID, badgeCode string, fields map[string]any) error`

- [ ] **Step 1: Write the client**

```go
// services/bff/internal/bff/client/badge.go
package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	badgesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/badges/v1"
)

// Badges is bff's outbound client to core-svc's BadgeService.
type Badges struct {
	badges badgesv1.BadgeServiceClient
}

// NewBadges builds the badges client.
func NewBadges(conn grpc.ClientConnInterface) *Badges {
	return &Badges{badges: badgesv1.NewBadgeServiceClient(conn)}
}

// ListBadgeCatalog lists the active badge catalog.
func (b *Badges) ListBadgeCatalog(ctx context.Context) ([]map[string]any, error) {
	resp, err := b.badges.ListBadgeCatalog(ctx, &badgesv1.ListBadgeCatalogRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(resp.Badges))
	for i, badge := range resp.Badges {
		out[i] = badgeToMap(badge)
	}
	return out, nil
}

// ListArtisanBadges lists one artisan's active grants.
func (b *Badges) ListArtisanBadges(ctx context.Context, artisanID string) ([]map[string]any, error) {
	resp, err := b.badges.ListArtisanBadges(ctx, &badgesv1.ListArtisanBadgesRequest{ArtisanId: artisanID})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(resp.ArtisanBadges))
	for i, ab := range resp.ArtisanBadges {
		out[i] = map[string]any{
			"badge":      badgeToMap(ab.Badge),
			"granted_at": ab.GetGrantedAt().AsTime(),
			"granted_by": ab.GrantedBy,
		}
	}
	return out, nil
}

// GetBadgeProgress returns the caller's own progress entries.
func (b *Badges) GetBadgeProgress(ctx context.Context, artisanID string) ([]map[string]any, error) {
	resp, err := b.badges.GetBadgeProgress(ctx, &badgesv1.GetBadgeProgressRequest{ArtisanId: artisanID})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(resp.Entries))
	for i, e := range resp.Entries {
		out[i] = map[string]any{
			"metric": e.Metric.String(), "value": e.Value, "updated_at": e.GetUpdatedAt().AsTime(),
		}
	}
	return out, nil
}

// GrantBadge confers a CONFERRED badge on an artisan.
func (b *Badges) GrantBadge(ctx context.Context, artisanID string, fields map[string]any) (map[string]any, error) {
	badgeID, _ := fields["badge_id"].(string)
	if badgeID == "" {
		return nil, domain.InvalidInput("badge_id: is required")
	}
	req := &badgesv1.GrantBadgeRequest{ArtisanId: artisanID, BadgeId: badgeID}
	if evidence, ok := fields["evidence_json"].(string); ok && evidence != "" {
		req.EvidenceJson = &evidence
	}
	resp, err := b.badges.GrantBadge(ctx, req)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"badge":      badgeToMap(resp.ArtisanBadge.Badge),
		"granted_at": resp.ArtisanBadge.GetGrantedAt().AsTime(),
		"granted_by": resp.ArtisanBadge.GrantedBy,
	}, nil
}

// RevokeBadge revokes a badge grant.
func (b *Badges) RevokeBadge(ctx context.Context, artisanID, badgeID string, fields map[string]any) error {
	reason, _ := fields["reason"].(string)
	if reason == "" {
		return domain.InvalidInput("reason: is required")
	}
	_, err := b.badges.RevokeBadge(ctx, &badgesv1.RevokeBadgeRequest{ArtisanId: artisanID, BadgeId: badgeID, Reason: reason})
	return err
}

func badgeToMap(b *badgesv1.Badge) map[string]any {
	out := map[string]any{
		"id": b.Id, "code": b.Code, "kind": b.Kind.String(), "icon_name": b.IconName, "sort_order": b.SortOrder,
	}
	if b.Tier != badgesv1.BadgeTier_BADGE_TIER_UNSPECIFIED {
		out["tier"] = b.Tier.String()
	}
	if b.Metric != badgesv1.BadgeMetric_BADGE_METRIC_UNSPECIFIED {
		out["metric"] = b.Metric.String()
	}
	if b.Threshold != nil {
		out["threshold"] = *b.Threshold
	}
	return out
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd services/bff && go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add services/bff/internal/bff/client/badge.go
git commit -m "feat(bff): add badges gRPC client wrapper

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 12: BFF routes and handlers

**Files:**
- Modify: `services/bff/internal/bff/server.go`
- Modify: `services/bff/internal/bff/handler/api.go`
- Modify: `services/bff/cmd/bff/main.go` (wire `client.NewBadges` and pass into `APIHandler`, matching how `cfg.TrendSvc` is threaded — check `main.go:103`'s call site for the exact `handler.NewAPI(...)`-style constructor and add `badgeSvc` alongside `b2bSvc, trendSvc`)

**Interfaces:**
- Consumes: `client.Badges` (Task 11).
- Produces: five new routes and their handlers, following the exact `TrendLink` route/handler pairing already in the file.

- [ ] **Step 1: Add routes to `server.go`**

Find the existing block:
```go
api.GET("/trends", httpx.WrapHandler(apiH.ListTrendLinks))
```
Add nearby (public group):
```go
api.GET("/badges", httpx.WrapHandler(apiH.ListBadgeCatalog))
api.GET("/artisans/:id/badges", httpx.WrapHandler(apiH.ListArtisanBadges))
```

Find the existing authed block containing:
```go
authed.POST("/trends", httpx.WrapHandler(withIdempotency(apiH.CreateTrendLink, cfg.IdempStore)))
```
Add nearby:
```go
authed.GET("/badges/me/progress", httpx.WrapHandler(apiH.GetBadgeProgress))
authed.POST("/artisans/:id/badges", httpx.WrapHandler(withIdempotency(apiH.GrantBadge, cfg.IdempStore)))
authed.DELETE("/artisans/:id/badges/:code", httpx.WrapHandler(apiH.RevokeBadge))
```

Also add `cfg.BadgeSvc` to the `Config` struct near the existing `B2BSvc, cfg.TrendSvc,` field (mirror `TrendSvc handler.TrendService` at line 55: add `BadgeSvc handler.BadgeService`), and thread it through the `NewAPI`/`New` constructor call at line 103 alongside `cfg.B2BSvc, cfg.TrendSvc,`.

- [ ] **Step 2: Add handler methods to `api.go`**

Add near the existing `CreateTrendLink`/`ListTrendLinks`/`DeleteTrendLink`/`PinTrendLink` block:

```go
func (h *APIHandler) ListBadgeCatalog(w http.ResponseWriter, r *http.Request) {
	badges, err := h.badgeSvc.ListBadgeCatalog(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"badges": badges})
}

func (h *APIHandler) ListArtisanBadges(w http.ResponseWriter, r *http.Request) {
	artisanID := httpx.URLParam(r, "id")
	badges, err := h.badgeSvc.ListArtisanBadges(r.Context(), artisanID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"artisan_badges": badges})
}

func (h *APIHandler) GetBadgeProgress(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		httpx.Error(w, domain.Unauthenticated("missing principal"))
		return
	}
	progress, err := h.badgeSvc.GetBadgeProgress(r.Context(), principal.Subject)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"progress": progress})
}

func (h *APIHandler) GrantBadge(w http.ResponseWriter, r *http.Request) {
	artisanID := httpx.URLParam(r, "id")
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	grant, err := h.badgeSvc.GrantBadge(r.Context(), artisanID, body)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, grant)
}

func (h *APIHandler) RevokeBadge(w http.ResponseWriter, r *http.Request) {
	artisanID := httpx.URLParam(r, "id")
	badgeCode := httpx.URLParam(r, "code")
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	if err := h.badgeSvc.RevokeBadge(r.Context(), artisanID, badgeCode, body); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "revoked"})
}
```

Note: `RevokeBadge`'s route is `DELETE /artisans/:id/badges/:code` (badge **code**, human-readable and stable, not the internal UUID — friendlier for an admin UI's URL and for API consumers) but `client.Badges.RevokeBadge` (Task 11) takes a `badgeID` (UUID), matching the gRPC contract's `RevokeBadgeRequest.badge_id`. Resolve this mismatch by having `APIHandler.RevokeBadge` look up the badge's UUID from its code before calling the client — add a `GetBadgeByCode` method to `client.Badges` (mirroring `ListBadgeCatalog` but filtering client-side, since there is no dedicated RPC for it and adding one for a single admin-facing lookup is not worth a new proto method):

```go
// Add to client/badge.go from Task 11:
func (b *Badges) GetBadgeByCode(ctx context.Context, code string) (map[string]any, error) {
	catalog, err := b.ListBadgeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	for _, badge := range catalog {
		if badge["code"] == code {
			return badge, nil
		}
	}
	return nil, domain.NotFound("badge not found: " + code)
}
```

And use it in `api.go`'s `RevokeBadge`:

```go
func (h *APIHandler) RevokeBadge(w http.ResponseWriter, r *http.Request) {
	artisanID := httpx.URLParam(r, "id")
	badgeCode := httpx.URLParam(r, "code")
	badge, err := h.badgeSvc.GetBadgeByCode(r.Context(), badgeCode)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	badgeID, _ := badge["id"].(string)
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	if err := h.badgeSvc.RevokeBadge(r.Context(), artisanID, badgeID, body); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "revoked"})
}
```

Add a `handler.BadgeService` interface (matching the existing `handler.TrendService` interface pattern — check `server.go`'s `TrendService` interface definition and copy its shape) listing every method `client.Badges` implements, and an `h.badgeSvc BadgeService` field on `APIHandler`, set from the constructor.

Add imports as needed: `"github.com/ZoroNewbie00/kalakriti/pkg/auth"` if not already imported in `api.go` (check first — other handlers in this file already read `auth.PrincipalFrom`, so it is very likely already imported).

- [ ] **Step 3: Wire `badgeSvc` through `bff/cmd/bff/main.go`**

Mirror the exact existing lines wiring `trendsSvc`/`TrendSvc` (find them with `grep -n "NewTrends\|TrendSvc" services/bff/cmd/bff/main.go`) and add:

```go
badgeSvc := client.NewBadges(coreConn) // reuse the same coreConn variable trendsSvc/b2bSvc are built from
```

and pass `BadgeSvc: badgeSvc` into whatever `bff.Config{...}` or `handler.NewAPI(...)` struct literal already carries `TrendSvc: trendsSvc`.

- [ ] **Step 4: Build and verify**

Run: `cd services/bff && go build ./...`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add services/bff/internal/bff/server.go services/bff/internal/bff/handler/api.go services/bff/cmd/bff/main.go services/bff/internal/bff/client/badge.go
git commit -m "feat(bff): add badge routes and handlers

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 13: OpenAPI spec + generated TS client

**Files:**
- Modify: `services/bff/openapi.json`
- Modify: `web/packages/api/src/generated/schema.d.ts` (regenerated, not hand-edited)
- Modify: `web/packages/api/src/operations.ts`
- Modify: `web/packages/api/src/index.ts`

**Interfaces:**
- Produces: `listBadgeCatalog()`, `listArtisanBadges(artisanId)`, `getBadgeProgress()`, `grantBadge(artisanId, body)`, `revokeBadge(artisanId, code, body)` — typed functions consumed by Tasks 17-19's Svelte routes.

- [ ] **Step 1: Add five paths to `openapi.json`**

Following the exact shape of the existing `/trends` path object (read earlier in this plan's research), add path objects for:
- `GET /badges` → `{ badges: Badge[] }`
- `GET /artisans/{id}/badges` → `{ artisan_badges: ArtisanBadge[] }`
- `GET /badges/me/progress` → `{ progress: BadgeProgressEntry[] }` (authed)
- `POST /artisans/{id}/badges` → 201, body `{ badge_id: string, evidence_json?: string }` (authed, MINISTRY)
- `DELETE /artisans/{id}/badges/{code}` → 200, body `{ reason: string }` (authed, MINISTRY)

Also add three reusable schemas under `components.schemas` (check where `TrendLink` is defined there and mirror the placement): `Badge` (`id, code, kind, tier?, icon_name, metric?, threshold?, sort_order`), `ArtisanBadge` (`badge, granted_at, granted_by`), `BadgeProgressEntry` (`metric, value, updated_at`).

- [ ] **Step 2: Regenerate the TS types**

Run: `cd web/packages/api && pnpm api:gen`
Expected: `schema.d.ts` updates with `paths['/badges']`, `paths['/artisans/{id}/badges']`, etc. Verify with `grep -n "badges" web/packages/api/src/generated/schema.d.ts | head`.

- [ ] **Step 3: Add typed wrapper functions to `operations.ts`**

Following the exact `listTrendLinks`/`createTrendLink` pattern read earlier, add near the bottom of the file:

```typescript
// --- Badge Operations ---

export type Badge = Json<paths['/badges']['get']['responses'][200]>['badges'][number];
export type ArtisanBadgesResponse = Json<paths['/artisans/{id}/badges']['get']['responses'][200]>;
export type BadgeProgressResponse = Json<paths['/badges/me/progress']['get']['responses'][200]>;
export type GrantBadgeBody = Json<paths['/artisans/{id}/badges']['post']['requestBody']>;

export function listBadgeCatalog(options?: CallOptions): Promise<{ badges: Badge[] }> {
  return call('/badges', { ...options, method: 'GET' }) as Promise<{ badges: Badge[] }>;
}

export function listArtisanBadges(artisanId: string, options?: CallOptions): Promise<ArtisanBadgesResponse> {
  return call(`/artisans/${encodeURIComponent(artisanId)}/badges`, { ...options, method: 'GET' }) as Promise<ArtisanBadgesResponse>;
}

export function getBadgeProgress(options?: CallOptions): Promise<BadgeProgressResponse> {
  return call('/badges/me/progress', { ...options, method: 'GET' }) as Promise<BadgeProgressResponse>;
}

export function grantBadge(artisanId: string, body: GrantBadgeBody, options?: CallOptions): Promise<unknown> {
  return call(`/artisans/${encodeURIComponent(artisanId)}/badges`, { ...options, method: 'POST', body });
}

export function revokeBadge(artisanId: string, code: string, body: { reason: string }, options?: CallOptions): Promise<{ status?: string }> {
  return call(`/artisans/${encodeURIComponent(artisanId)}/badges/${encodeURIComponent(code)}`, { ...options, method: 'DELETE', body }) as Promise<{ status?: string }>;
}
```

- [ ] **Step 4: Export from the barrel**

In `web/packages/api/src/index.ts`, add near the existing `listTrendLinks, createTrendLink, deleteTrendLink, pinTrendLink,` line:
```typescript
  listBadgeCatalog,
  listArtisanBadges,
  getBadgeProgress,
  grantBadge,
  revokeBadge,
```
and near the existing `type TrendLink, type CreateTrendLinkBody, type TrendLinksResponse,` line:
```typescript
  type Badge,
  type ArtisanBadgesResponse,
  type BadgeProgressResponse,
  type GrantBadgeBody,
```

- [ ] **Step 5: Type-check**

Run: `cd web && pnpm -w check` (or `pnpm --filter @kalakriti/api check` if that's the package's own script name — check `package.json`)
Expected: no new type errors.

- [ ] **Step 6: Commit**

```bash
git add services/bff/openapi.json web/packages/api/src/generated/schema.d.ts web/packages/api/src/operations.ts web/packages/api/src/index.ts
git commit -m "feat(api): add generated badge client operations

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 14: Badge icons

**Files:**
- Create: `web/packages/icons/src/badge-verified.svg`, `badge-master.svg`, `badge-award.svg`, `badge-gi.svg`, `badge-coordinator.svg`, `badge-milestone.svg`, `badge-locked.svg` (seven icons: the six `icon_name` values seeded in Task 2 plus one shared "not yet earned" outline used by every locked badge regardless of its target icon, per the frontend's badge grid design in Task 16)

**Interfaces:**
- Produces (after `pnpm generate`): `'badge-verified' | 'badge-master' | 'badge-award' | 'badge-gi' | 'badge-coordinator' | 'badge-milestone' | 'badge-locked'` added to `IconName` in `icons.d.ts`, usable as `<Icon name="badge-verified" />`.

- [ ] **Step 1: Write seven simple line-icon SVGs**

Follow `check.svg`'s exact structural convention (`viewBox="0 0 24 24"`, `aria-hidden="true"`, `focusable="false"`, `fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"`, a one-line descriptive comment naming the icon and its licensing status). Seven distinct, simple geometric badge-shaped outlines (a shield for verified, a shield with a star for master, a ribbon-medal for award, a shield with a small circle motif for gi, two overlapping circles for coordinator, a plain hexagon for milestone, a hexagon with a dashed/lighter stroke for locked):

```xml
<!-- web/packages/icons/src/badge-verified.svg -->
<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">
  <!-- badge-verified (core icon). none -- global UI iconography, not a licensed motif. Use: verified_artisan badge. -->
  <path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" d="M12 3 L19 6 V12 C19 16.5 16 19.5 12 21 C8 19.5 5 16.5 5 12 V6 Z"/>
  <path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" d="M9 12 L11 14 L15.5 9.5"/>
</svg>
```

```xml
<!-- web/packages/icons/src/badge-master.svg -->
<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">
  <!-- badge-master (core icon). none -- global UI iconography, not a licensed motif. Use: master_craftsperson badge. -->
  <path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" d="M12 3 L19 6 V12 C19 16.5 16 19.5 12 21 C8 19.5 5 16.5 5 12 V6 Z"/>
  <path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" d="M12 8.5 L13.2 11 L16 11.4 L14 13.3 L14.5 16 L12 14.6 L9.5 16 L10 13.3 L8 11.4 L10.8 11 Z"/>
</svg>
```

```xml
<!-- web/packages/icons/src/badge-award.svg -->
<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">
  <!-- badge-award (core icon). none -- global UI iconography, not a licensed motif. Use: national_awardee badge. -->
  <circle cx="12" cy="9" r="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" d="M9 13.2 L7.5 21 L12 18.5 L16.5 21 L15 13.2"/>
</svg>
```

```xml
<!-- web/packages/icons/src/badge-gi.svg -->
<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">
  <!-- badge-gi (core icon). none -- global UI iconography, not a licensed motif. Use: gi_practitioner badge. -->
  <path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" d="M12 3 L19 6 V12 C19 16.5 16 19.5 12 21 C8 19.5 5 16.5 5 12 V6 Z"/>
  <circle cx="12" cy="11.5" r="3" fill="none" stroke="currentColor" stroke-width="1.5"/>
</svg>
```

```xml
<!-- web/packages/icons/src/badge-coordinator.svg -->
<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">
  <!-- badge-coordinator (core icon). none -- global UI iconography, not a licensed motif. Use: cluster_coordinator badge. -->
  <circle cx="9" cy="10" r="4" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <circle cx="15" cy="14" r="4" fill="none" stroke="currentColor" stroke-width="1.5"/>
</svg>
```

```xml
<!-- web/packages/icons/src/badge-milestone.svg -->
<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">
  <!-- badge-milestone (core icon). none -- global UI iconography, not a licensed motif. Use: earned tiered badges (catalog_builder, provenance_keeper, order_fulfiller, first_listing). -->
  <path fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" d="M12 3 L20 8 V16 L12 21 L4 16 V8 Z"/>
</svg>
```

```xml
<!-- web/packages/icons/src/badge-locked.svg -->
<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false">
  <!-- badge-locked (core icon). none -- global UI iconography, not a licensed motif. Use: any not-yet-earned badge slot. -->
  <path fill="none" stroke="currentColor" stroke-width="1.25" stroke-linejoin="round" stroke-dasharray="2.5 2" d="M12 3 L20 8 V16 L12 21 L4 16 V8 Z"/>
</svg>
```

- [ ] **Step 2: Generate the icon registry**

Run: `cd web/packages/icons && pnpm generate`
Expected: `icons.js` and `icons.d.ts` regenerate with the seven new names present. Verify: `grep -n "badge-verified" web/packages/icons/icons.js`.

- [ ] **Step 3: Commit**

```bash
git add web/packages/icons/src/badge-*.svg web/packages/icons/icons.js web/packages/icons/icons.d.ts
git commit -m "feat(icons): add seven badge icons

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 15: i18n — badge display text (en.ts + 20 locales)

**Files:**
- Modify: `web/packages/i18n/src/messages/en.ts`
- Modify: all 20 non-English catalogues in `web/packages/i18n/src/messages/*.ts` (`as.ts, bn.ts, brx.ts, doi.ts, gu.ts, hi.ts, kn.ts, kok.ts, ks.ts, mai.ts, ml.ts, mr.ts, ne.ts, or.ts, pa.ts, sa.ts, sd.ts, ta.ts, te.ts, ur.ts`)

**Interfaces:**
- Produces: `badge.<code>.name`, `badge.<code>.desc`, `badge.<code>.criteria` for all 14 catalog codes (tiered families share one key triad per tier-independent name, e.g. `badge.catalog_builder.name` covers all three tiers, with the tier itself rendered separately via a `badge.tier.bronze/silver/gold` triad — 12 badge-family key triads, not 14, since the three tiered families collapse to one triad each) plus `nav.badges`, `badges.title`, `badges.subtitle`, `badges.empty`, `badges.inProgress`, `badges.locked`, `badges.tier.bronze/silver/gold`, `badges.progressLabel` (with `{current}`/`{total}` placeholders), `badges.grantedOn` (with `{date}` placeholder), `badges.readAloud` — 39 keys total, consumed by Tasks 16-19.

- [ ] **Step 1: Add keys to `en.ts`**

Add near the existing `trends.*` block (same file, same section style):

```typescript
  'nav.badges': 'Badges',
  'badges.title': 'Badges',
  'badges.subtitle': 'Recognition for your craft and your work on the platform.',
  'badges.empty': 'No badges yet. Publish your first listing to earn one.',
  'badges.inProgress': 'In progress',
  'badges.locked': 'Not yet earned',
  'badges.progressLabel': '{current} of {total}',
  'badges.grantedOn': 'Earned on {date}',
  'badges.readAloud': 'Read this badge aloud',
  'badges.tier.bronze': 'Bronze',
  'badges.tier.silver': 'Silver',
  'badges.tier.gold': 'Gold',
  'badge.verified_artisan.name': 'Verified Artisan',
  'badge.verified_artisan.desc': 'Identity and craft confirmed by the Ministry.',
  'badge.verified_artisan.criteria': 'Granted by a Ministry or cluster officer after verification.',
  'badge.master_craftsperson.name': 'Master Craftsperson',
  'badge.master_craftsperson.desc': 'Recognised for exceptional skill in their craft.',
  'badge.master_craftsperson.criteria': 'Granted by a Ministry or cluster officer.',
  'badge.national_awardee.name': 'National Awardee',
  'badge.national_awardee.desc': 'Recipient of a national craft award.',
  'badge.national_awardee.criteria': 'Granted by a Ministry officer with award evidence on file.',
  'badge.gi_practitioner.name': 'GI Practitioner',
  'badge.gi_practitioner.desc': 'Practises a Geographical Indication registered craft.',
  'badge.gi_practitioner.criteria': 'Granted by a Ministry or cluster officer.',
  'badge.cluster_coordinator.name': 'Cluster Coordinator',
  'badge.cluster_coordinator.desc': 'Coordinates artisans within their cluster.',
  'badge.cluster_coordinator.criteria': 'Granted by a Ministry officer.',
  'badge.first_listing.name': 'First Listing',
  'badge.first_listing.desc': 'Published their first catalog listing.',
  'badge.first_listing.criteria': 'Publish 1 listing.',
  'badge.catalog_builder.name': 'Catalog Builder',
  'badge.catalog_builder.desc': 'Built a growing catalog of published listings.',
  'badge.catalog_builder.criteria': 'Publish {threshold} listings.',
  'badge.provenance_keeper.name': 'Provenance Keeper',
  'badge.provenance_keeper.desc': 'Seals provenance records for their work.',
  'badge.provenance_keeper.criteria': 'Seal provenance on {threshold} listings.',
  'badge.order_fulfiller.name': 'Order Fulfiller',
  'badge.order_fulfiller.desc': 'Completes bulk order lots reliably.',
  'badge.order_fulfiller.criteria': 'Complete {threshold} order lots.',
```

- [ ] **Step 2: Run the audit script to confirm the new keys are detected**

Run: `cd web/packages/i18n && npm run audit`
Expected: output reports 39 new keys missing from all 20 non-English locales (baseline currently 0 — this will show as new failures until Step 3 is done).

- [ ] **Step 3: Populate all 20 non-English catalogues**

For each of the 20 locale files, add the same 39 keys (identical key names, translated values) to the file's `export const <code>: Messages = { ... }` object, in the same relative position (near any existing `trends.*` keys in that file, matching where Step 1 placed them in `en.ts`). Translate each string faithfully into that locale's language and script — do not leave any locale with an English placeholder value, since the ratchet baseline requires 0 issues and a placeholder-in-disguise (English text under a non-English key) would pass the type checker but fail the audit script's translation-completeness check.

This is 20 repetitions of the same 39-key block with translated values — mechanically identical in structure to Step 1, differing only in the string values and target script. Do this file by file; after each file, the TypeScript compiler enforces the key set is complete (the file won't compile with a missing key, since `Messages` is a fixed type), so a missing key surfaces immediately as a build error rather than silently.

- [ ] **Step 4: Run the audit script again**

Run: `cd web/packages/i18n && npm run audit`
Expected: 0 issues across all 21 locales (matching the existing baseline).

- [ ] **Step 5: Run the catalogue-audit test**

Run: `cd web/packages/i18n && npx vitest run catalogue-audit.test.ts`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/packages/i18n/src/messages/*.ts
git commit -m "feat(i18n): add badge display text for all 21 locales

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 16: Shared UI components — `BadgeChip` and `BadgeGrid`

**Files:**
- Create: `web/packages/ui/src/BadgeChip.svelte`
- Create: `web/packages/ui/src/BadgeGrid.svelte`
- Test: `web/packages/ui/src/BadgeChip.test.ts` (or `.svelte.test.ts`, matching whatever extension this package's existing component tests use — check `ls web/packages/ui/src/*.test.*` first)

**Interfaces:**
- Consumes: `Badge`, `ArtisanBadgesResponse`, `BadgeProgressResponse` types (Task 13), `Icon` from `@kalakriti/icons`, `locale` from `@kalakriti/i18n` (same import shape as `Chip.svelte`).
- Produces: `<BadgeChip badge={...} granted={true} progress={...} />`, `<BadgeGrid catalog={...} granted={...} progress={...} />` — consumed by Tasks 17-19.

- [ ] **Step 1: Write `BadgeChip.svelte`**

```svelte
<!--
  packages/ui/src/BadgeChip.svelte

    <BadgeChip badge={catalogEntry} granted={true} grantedAt={date} />
    <BadgeChip badge={catalogEntry} granted={false} progress={{ current: 3, total: 5 }} />

  One badge, in one of three states: granted, in-progress (earned badges
  only, shows a progress bar), or locked. Tier is shown as TEXT (the tier
  name), never only as a border colour -- colour alone must never carry
  meaning. Hairline border, no shadow, per Batch 0's design law.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { Icon, type IconName } from '@kalakriti/icons';
  import SpeakButton from './SpeakButton.svelte';

  interface BadgeCatalogEntry {
    code: string;
    kind: 'BADGE_KIND_EARNED' | 'BADGE_KIND_CONFERRED';
    tier?: 'BADGE_TIER_BRONZE' | 'BADGE_TIER_SILVER' | 'BADGE_TIER_GOLD';
    icon_name: string;
    threshold?: number;
  }

  interface Props {
    badge: BadgeCatalogEntry;
    granted: boolean;
    grantedAt?: string;
    progress?: { current: number; total: number };
    class?: string;
  }

  let { badge, granted, grantedAt, progress, class: className }: Props = $props();

  const t = $derived(locale.t);
  const iconName = $derived((granted ? badge.icon_name : 'badge-locked') as IconName);
  const name = $derived(t(`badge.${badge.code}.name`));
  const desc = $derived(t(`badge.${badge.code}.desc`));
  const tierLabel = $derived(
    badge.tier === 'BADGE_TIER_BRONZE' ? t('badges.tier.bronze') :
    badge.tier === 'BADGE_TIER_SILVER' ? t('badges.tier.silver') :
    badge.tier === 'BADGE_TIER_GOLD' ? t('badges.tier.gold') : null,
  );
  const speakText = $derived(
    tierLabel ? `${name}. ${tierLabel}. ${desc}` : `${name}. ${desc}`,
  );
</script>

<div class="k-badge-chip {className || ''}" class:k-badge-chip--granted={granted} class:k-badge-chip--locked={!granted}>
  <Icon name={iconName} size="2rem" />
  <div class="k-badge-chip__body">
    <span class="k-badge-chip__name">
      {name}
      {#if tierLabel}<span class="k-badge-chip__tier">— {tierLabel}</span>{/if}
    </span>
    <span class="k-badge-chip__desc">{desc}</span>
    {#if granted && grantedAt}
      <span class="k-badge-chip__meta">{t('badges.grantedOn', { date: grantedAt })}</span>
    {:else if progress}
      <div class="k-badge-chip__progress" role="progressbar" aria-valuenow={progress.current} aria-valuemin={0} aria-valuemax={progress.total} aria-label={t('badges.progressLabel', { current: String(progress.current), total: String(progress.total) })}>
        <div class="k-badge-chip__progress-fill" style="width: {Math.min(100, (progress.current / progress.total) * 100)}%"></div>
      </div>
      <span class="k-badge-chip__meta">{t('badges.progressLabel', { current: String(progress.current), total: String(progress.total) })}</span>
    {:else}
      <span class="k-badge-chip__meta">{t(`badge.${badge.code}.criteria`, { threshold: String(badge.threshold ?? '') })}</span>
    {/if}
  </div>
  <SpeakButton text={speakText} label={t('badges.readAloud')} />
</div>

<style>
  .k-badge-chip {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-3);
    padding: var(--k-space-3);
    border: var(--k-hairline) solid var(--k-border-hairline);
    border-radius: var(--k-radius-sm, 4px);
    background-color: var(--k-surface);
  }

  .k-badge-chip--locked {
    color: var(--k-text-muted);
    border-style: dashed;
  }

  .k-badge-chip__body {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-1);
    flex: 1;
    min-width: 0;
  }

  .k-badge-chip__name {
    font-weight: 600;
  }

  .k-badge-chip__tier {
    font-weight: 400;
    color: var(--k-text-muted);
  }

  .k-badge-chip__desc,
  .k-badge-chip__meta {
    font-size: var(--k-text-sm);
    color: var(--k-text-muted);
  }

  .k-badge-chip__progress {
    height: 6px;
    border-radius: var(--k-radius-full, 999px);
    background-color: var(--k-border-hairline);
    overflow: hidden;
  }

  .k-badge-chip__progress-fill {
    height: 100%;
    background-color: var(--k-accent);
  }
</style>
```

Check `SpeakButton.svelte`'s actual prop names (`text`/`label` guessed here from its filename and Batch 3's "AudioPlayback plays a TTS clip" description) — read `web/packages/ui/src/SpeakButton.svelte`'s `Props` interface first and adjust the two props passed to it above if they differ.

Check `@kalakriti/icons`'s actual named export for the `IconName` type (guessed as `IconName` here, matching the JSDoc `@typedef` seen in `Icon.svelte`'s research — `import('./icons.d.ts').IconName`) — if `icons.d.ts` does not re-export `IconName` as a public type from the package's own index, either import it from the exact path already used elsewhere in this codebase or drop the type annotation and use `string` with a comment.

Check `locale.t`'s interpolation call signature — `Chip.svelte` (read earlier) calls `t('search.filters.remove', { label })`, i.e. an object of string values keyed by placeholder name, matching what's used above (`{ current: String(...), total: String(...) }`, `{ date: grantedAt }`, `{ threshold: String(...) }`) — no change needed, this matches.

- [ ] **Step 2: Write `BadgeGrid.svelte`**

```svelte
<!--
  packages/ui/src/BadgeGrid.svelte

    <BadgeGrid catalog={badgeCatalog} granted={artisanBadges} progress={progressEntries} />

  Earned-first ordering: granted badges, then in-progress earned badges
  (grouped by their tier family, showing only the next unearned tier),
  then locked/conferred badges the artisan doesn't hold. Per Batch 3's
  EmptyState rule, an artisan with zero badges sees a real empty state,
  not a blank grid.
-->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import BadgeChip from './BadgeChip.svelte';
  import EmptyState from './EmptyState.svelte';

  interface BadgeCatalogEntry {
    id: string;
    code: string;
    kind: 'BADGE_KIND_EARNED' | 'BADGE_KIND_CONFERRED';
    tier?: 'BADGE_TIER_BRONZE' | 'BADGE_TIER_SILVER' | 'BADGE_TIER_GOLD';
    icon_name: string;
    metric?: string;
    threshold?: number;
  }
  interface GrantedBadge { badge: BadgeCatalogEntry; granted_at: string; granted_by: string }
  interface ProgressEntry { metric: string; value: number }

  interface Props {
    catalog: BadgeCatalogEntry[];
    granted: GrantedBadge[];
    progress: ProgressEntry[];
  }

  let { catalog, granted, progress }: Props = $props();

  const t = $derived(locale.t);
  const grantedIds = $derived(new Set(granted.map((g) => g.badge.id)));
  const grantedByAt = $derived(new Map(granted.map((g) => [g.badge.id, g.granted_at])));
  const progressByMetric = $derived(new Map(progress.map((p) => [p.metric, p.value])));

  const grantedEntries = $derived(catalog.filter((b) => grantedIds.has(b.id)));

  // For each EARNED metric family, find the lowest-threshold tier not yet
  // granted -- that is the "in progress" row; higher tiers in the same
  // family are not shown separately until that one is reached.
  const inProgressEntries = $derived(
    catalog
      .filter((b) => b.kind === 'BADGE_KIND_EARNED' && !grantedIds.has(b.id))
      .filter((b, _, all) => {
        const sameFamily = all.filter((x) => x.metric === b.metric && !grantedIds.has(x.id));
        const lowest = sameFamily.reduce((min, x) => ((x.threshold ?? 0) < (min.threshold ?? 0) ? x : min), sameFamily[0]);
        return b.id === lowest?.id;
      }),
  );

  const lockedConferredEntries = $derived(
    catalog.filter((b) => b.kind === 'BADGE_KIND_CONFERRED' && !grantedIds.has(b.id)),
  );
</script>

<div class="k-badge-grid">
  {#if grantedEntries.length === 0 && inProgressEntries.length === 0}
    <EmptyState heading={t('badges.empty')} />
  {:else}
    {#each grantedEntries as badge (badge.id)}
      <BadgeChip {badge} granted={true} grantedAt={grantedByAt.get(badge.id)} />
    {/each}
    {#each inProgressEntries as badge (badge.id)}
      <BadgeChip
        {badge}
        granted={false}
        progress={{ current: progressByMetric.get(badge.metric ?? '') ?? 0, total: badge.threshold ?? 1 }}
      />
    {/each}
  {/if}
  {#each lockedConferredEntries as badge (badge.id)}
    <BadgeChip {badge} granted={false} />
  {/each}
</div>

<style>
  .k-badge-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: var(--k-space-3);
  }
</style>
```

Check `EmptyState.svelte`'s actual `Props` (guessed `heading` here from Batch 3's spec text "heading, a body line, and a primary action") — read the real file and adjust the prop name/shape used above if it differs (e.g. it may require a `body` and be a required, not optional, prop — pass a minimal valid set matching its actual interface).

- [ ] **Step 3: Write a component test**

```typescript
// web/packages/ui/src/BadgeGrid.test.ts — match this package's existing test
// file naming/extension convention (check `ls web/packages/ui/src/*.test.*`
// first; Svelte 5 component tests in this repo likely use
// @testing-library/svelte per Batch 0's stack, with a `.svelte.test.ts` or
// plain `.test.ts` extension -- follow whatever Chip.test.ts or
// SectionHeader.test.ts (if one exists) already uses).
import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import BadgeGrid from './BadgeGrid.svelte';

describe('BadgeGrid', () => {
  it('shows the empty state when the artisan has no badges', () => {
    render(BadgeGrid, { props: { catalog: [], granted: [], progress: [] } });
    expect(screen.getByText(/no badges yet/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 4: Run the test**

Run: `cd web/packages/ui && npx vitest run BadgeGrid.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/packages/ui/src/BadgeChip.svelte web/packages/ui/src/BadgeGrid.svelte web/packages/ui/src/BadgeGrid.test.ts
git commit -m "feat(ui): add BadgeChip and BadgeGrid components

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 17: Artisan `/badges` route

**Files:**
- Create: `web/apps/artisan/src/routes/badges/+page.ts`
- Create: `web/apps/artisan/src/routes/badges/+page.svelte`
- Modify: whatever file defines the artisan app's persistent bottom navigation (search `grep -rln "nav.myWork\|nav.orders" web/apps/artisan/src` to find it) to add a `/badges` entry — or confirm with the user/existing nav pattern whether badges belongs in the bottom nav (4 slots already full: Home, My Work, Orders, Profile per Batch 7) or is reached from `/profile` instead. **Default to linking it from `/profile`**, not adding a fifth bottom-nav slot, since Batch 7 fixed the bottom nav at exactly four items.

**Interfaces:**
- Consumes: `listBadgeCatalog`, `listArtisanBadges`, `getBadgeProgress` from `@kalakriti/api` (Task 13), `BadgeGrid` from `@kalakriti/ui` (Task 16), the artisan's own id from the session store (check `packages/api/src/auth-flow.ts`'s `session` store for the exact field name, likely `session.claims.sub` or similar).

- [ ] **Step 1: Write `+page.ts`**

```typescript
// web/apps/artisan/src/routes/badges/+page.ts
import { listBadgeCatalog, listArtisanBadges, getBadgeProgress } from '@kalakriti/api';
import { session } from '@kalakriti/api'; // adjust import path/name to match this repo's actual session store export -- check an existing authed +page.ts in this app (e.g. /orders or /listings) for the real pattern

export const ssr = false;

export async function load() {
  const artisanId = session.claims?.sub ?? ''; // adjust to the real session shape -- copy the exact access pattern from an existing authed +page.ts in this app
  const [catalogRes, grantedRes, progressRes] = await Promise.all([
    listBadgeCatalog(),
    listArtisanBadges(artisanId),
    getBadgeProgress(),
  ]);
  return {
    catalog: catalogRes.badges,
    granted: grantedRes.artisan_badges,
    progress: progressRes.progress,
  };
}
```

Before writing this file, read one existing authed `+page.ts` in `web/apps/artisan/src/routes/` (e.g. `orders/+page.ts` or `listings/+page.ts`) to copy its exact session-access pattern and error-handling convention (loading/error states per Batch 0's "every list has an explicit empty state, loading state, and error state" rule) rather than the placeholder guess above.

- [ ] **Step 2: Write `+page.svelte`**

```svelte
<!-- web/apps/artisan/src/routes/badges/+page.svelte -->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { BadgeGrid, SectionHeader } from '@kalakriti/ui';
  import type { PageData } from './$types';

  interface Props { data: PageData }
  let { data }: Props = $props();

  const t = $derived(locale.t);
</script>

<svelte:head><title>{t('badges.title')}</title></svelte:head>

<main id="main-content">
  <SectionHeader heading={t('badges.title')} kicker={t('badges.subtitle')} />
  <BadgeGrid catalog={data.catalog} granted={data.granted} progress={data.progress} />
</main>
```

Check `SectionHeader.svelte`'s real `Props` (guessed `heading`/`kicker` here from Batch 0's "kicker line, then a large section heading, then a View all link" description) — read the actual file and correct prop names if they differ.

- [ ] **Step 3: Add navigation entry point from `/profile`**

Read the artisan app's `/profile` route (`web/apps/artisan/src/routes/profile/+page.svelte`) and add a link to `/badges` in its existing list-of-links section (matching however that page already links to, e.g., settings or accessibility statement), with the label `t('nav.badges')`.

- [ ] **Step 4: Manual verification**

Run: `pnpm --filter artisan dev` and navigate to `/badges` at 360px width (per Batch 1's "primary user is on a cheap Android phone" rule). Confirm: empty state renders with no data, grid renders with seed data, "read aloud" works, keyboard-only traversal reaches every chip.

- [ ] **Step 5: Commit**

```bash
git add web/apps/artisan/src/routes/badges/ web/apps/artisan/src/routes/profile/+page.svelte
git commit -m "feat(artisan): add /badges route

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 18: Buyer storefront badge row

**Files:**
- Modify: `web/apps/buyer/src/routes/artisan/[slug]/+page.ts` (or wherever the storefront's loader lives — check `ls web/apps/buyer/src/routes/artisan/`)
- Modify: `web/apps/buyer/src/routes/artisan/[slug]/+page.svelte`

**Interfaces:**
- Consumes: `listArtisanBadges` (Task 13, public route, no auth needed), `BadgeChip` (Task 16).

- [ ] **Step 1: Add badges to the storefront loader**

Read the existing storefront `+page.ts` load function (it already fetches the artisan's storefront data per Batch 11's spec) and add one more parallel fetch:

```typescript
import { listArtisanBadges } from '@kalakriti/api';
// ... inside the existing load(), alongside whatever other Promise.all entries already exist:
const badgesRes = await listArtisanBadges(artisanId); // use whatever variable name the existing loader already has for the artisan id
// return { ...existingReturn, badges: badgesRes.artisan_badges };
```

- [ ] **Step 2: Render the badge row on the storefront page**

In the storefront `+page.svelte`, near wherever "awardee status" or similar first-class artisan metadata already renders (per this plan's spec, badges sit "next to awardee status" — find that existing block), add:

```svelte
{#if data.badges.length > 0}
  <div class="storefront-badges">
    {#each data.badges as grant (grant.badge.id)}
      <BadgeChip badge={grant.badge} granted={true} grantedAt={grant.granted_at} />
    {/each}
  </div>
{/if}
```

Import `BadgeChip` from `@kalakriti/ui` at the top of the file.

- [ ] **Step 3: Manual verification**

Navigate to a seeded artisan's storefront page while logged out (anonymous buyer) and confirm badges render — this is the concrete proof of the public-read wiring from Task 10, Step 3.

- [ ] **Step 4: Commit**

```bash
git add web/apps/buyer/src/routes/artisan/
git commit -m "feat(buyer): show artisan badges on storefront

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 19: Admin grant/revoke panel

**Files:**
- Modify: whatever page renders a single artisan's admin detail view (search `grep -rln "artisan" web/apps/admin/src/routes/` to find it — likely `web/apps/admin/src/routes/artisans/[id]/+page.svelte`, following the `/clusters` and `/companies` naming convention already in the admin app)
- Modify: that route's `+page.ts` loader

**Interfaces:**
- Consumes: `listBadgeCatalog`, `listArtisanBadges`, `grantBadge`, `revokeBadge` (Task 13).

- [ ] **Step 1: Add catalog + grants to the loader**

```typescript
// added to the existing artisan-detail +page.ts load():
import { listBadgeCatalog, listArtisanBadges } from '@kalakriti/api';
// ... alongside existing fetches:
const [catalogRes, grantedRes] = await Promise.all([listBadgeCatalog(), listArtisanBadges(artisanId)]);
// return { ...existingReturn, badgeCatalog: catalogRes.badges, artisanBadges: grantedRes.artisan_badges };
```

- [ ] **Step 2: Add a grant/revoke panel**

In the artisan detail `+page.svelte`, add a section (matching this admin app's existing section styling — check how `/clusters` or `/moderation` structures a panel):

```svelte
<script lang="ts">
  import { grantBadge, revokeBadge } from '@kalakriti/api';
  // ... existing imports

  const conferredCatalog = $derived(data.badgeCatalog.filter((b) => b.kind === 'BADGE_KIND_CONFERRED'));
  const grantedIds = $derived(new Set(data.artisanBadges.map((g) => g.badge.id)));

  let selectedBadgeId = $state('');
  let revokeReason = $state('');

  async function handleGrant() {
    if (!selectedBadgeId) return;
    await grantBadge(data.artisan.id, { badge_id: selectedBadgeId });
    location.reload(); // simplest correct refresh; matches this admin app's existing post-mutation pattern -- check an existing admin mutation call site (e.g. /moderation's approve action) and use its actual refresh convention (invalidate() vs reload()) instead if it differs
  }

  async function handleRevoke(code: string) {
    if (!revokeReason) return;
    await revokeBadge(data.artisan.id, code, { reason: revokeReason });
    location.reload();
  }
</script>

<section class="admin-badges-panel">
  <h2>{t('badges.title')}</h2>
  <ul>
    {#each data.artisanBadges as grant (grant.badge.id)}
      <li>
        {grant.badge.code}
        <button type="button" onclick={() => handleRevoke(grant.badge.code)}>{t('common.revoke')}</button>
      </li>
    {/each}
  </ul>
  <form onsubmit={(e) => { e.preventDefault(); handleGrant(); }}>
    <select bind:value={selectedBadgeId}>
      <option value="">{t('common.selectBadge')}</option>
      {#each conferredCatalog.filter((b) => !grantedIds.has(b.id)) as badge (badge.id)}
        <option value={badge.id}>{badge.code}</option>
      {/each}
    </select>
    <button type="submit">{t('common.grant')}</button>
  </form>
</section>
```

This is a human-decision panel — one badge at a time, no bulk action, matching the moderation-queue rule from Batch 13 ("HUMAN DECISION REQUIRED on every item... No bulk auto-action"). Add `common.revoke`, `common.selectBadge`, `common.grant` i18n keys (3 more keys × 21 locales) via the same process as Task 15 if they don't already exist in `en.ts` (check first — generic "grant"/"revoke" labels may already exist under a different key from another admin panel; reuse rather than duplicate if so).

Read the actual admin artisan-detail page's existing styling, imports, and post-mutation convention before finalizing this task — the code above is illustrative of the interaction, not a verbatim drop-in, since this plan's research did not read that specific file's current contents.

- [ ] **Step 3: Manual verification**

As a MINISTRY-role admin, grant a conferred badge to a seeded artisan, confirm it appears on their storefront (Task 18) and in their own `/badges` view (Task 17). Revoke it, confirm it disappears from both.

- [ ] **Step 4: Commit**

```bash
git add web/apps/admin/src/routes/
git commit -m "feat(admin): add badge grant/revoke panel to artisan detail

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 20: End-to-end verification against acceptance criteria

**Files:** none created; this task runs the full stack and checks the spec's acceptance criteria.

- [ ] **Step 1: Full backend test suite**

Run: `cd services/core-svc && go test ./... && cd ../collab-svc && go test ./... && cd ../bff && go test ./...`
Expected: all PASS.

- [ ] **Step 2: Full frontend checks**

Run: `cd web && pnpm check && pnpm --filter @kalakriti/i18n test`
Expected: no type errors, i18n audit at 0 issues.

- [ ] **Step 3: Manual acceptance walkthrough (local stack via `make demo-up`)**

1. Publish 5 listings for a fresh test artisan → confirm `catalog_builder` BRONZE grants automatically (Task 9's consumer path) and `first_listing` grants after the first.
2. Restart core-svc mid-way through step 1's events (kill and restart the container) → confirm no duplicate grants and no error (idempotent recompute, Task 5).
3. Seal provenance on a listing → confirm `provenance_keeper` BRONZE grants.
4. Accept and complete a bulk-order lot for the artisan → confirm `topics.OrderLotCompleted` fires (Task 1's fix) and `order_fulfiller` BRONZE grants.
5. As MINISTRY, grant `verified_artisan` (Task 19) → confirm it appears on the artisan's public storefront while logged out (Task 18, proving Task 10 Step 3's public-read wiring) and on their own `/badges` (Task 17).
6. Attempt to grant an EARNED badge (e.g. `first_listing`) via the admin panel or a direct API call → confirm it is rejected (Task 6's `GrantBadge` kind check).
7. Switch locale to Hindi and Tamil → confirm every badge name/description/criteria renders correctly (Task 15) with no fallback-to-English text visible.

- [ ] **Step 4: Final commit (if any fixups were needed)**

```bash
git add -A
git commit -m "test: verify artisan badges feature end-to-end

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```
