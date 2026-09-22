# Artisan badges and government scheme guidance — design

Date: 2026-09-15
Status: approved
Scope: two independent features, built and shipped as separate slices in this
implementation pass. Badges first, schemes second.

## Why two specs in one file

They share no tables, no RPCs, no proto package, and no frontend routes.
They are grouped in one document only because they were approved together in
one brainstorming session. Each has its own migration number, proto package,
core-svc vertical slice, BFF routes, and i18n keys. Nothing in Feature 2
depends on Feature 1 landing first, but the plan will sequence badges before
schemes because schemes carry a factual-accuracy risk that deserves the
calmer second half of the work.

## Feature 7: Artisan badges

### Purpose

Give artisans Twitch-style recognition badges: some conferred by Ministry/
cluster staff (verified, master craftsperson, national awardee, GI
practitioner, cluster coordinator), some earned automatically by crossing
activity thresholds (first listing, catalog-building tiers, provenance-
sealing tiers, order-fulfilment tiers). Badges render on the artisan's own
`/badges` screen and on their public buyer-facing storefront, as first-class
content — matching Batch 0's "awardee status is first-class content, not
grey metadata" rule.

### Data model — `migrations/033_badges.sql`

```sql
CREATE TYPE badge_kind AS ENUM ('EARNED', 'CONFERRED');
CREATE TYPE badge_tier AS ENUM ('BRONZE', 'SILVER', 'GOLD');
CREATE TYPE badge_metric AS ENUM (
    'LISTINGS_PUBLISHED', 'PROVENANCE_SEALED',
    'LOTS_ACCEPTED', 'LOTS_COMPLETED'
);

-- Reference catalog. Seeded by this migration; admins may add more later
-- via the admin UI (CONFERRED only — EARNED badges need a metric wired into
-- the consumer, so they are code-defined, not admin-definable).
CREATE TABLE badge (
    id          uuid         NOT NULL,
    code        text         NOT NULL,   -- stable slug, e.g. 'first_listing'
    kind        badge_kind   NOT NULL,
    tier        badge_tier,              -- NULL for untiered badges
    icon_name   text         NOT NULL,   -- key into packages/icons
    metric      badge_metric,            -- NULL for CONFERRED
    threshold   bigint,                  -- NULL for CONFERRED
    sort_order  integer      NOT NULL DEFAULT 0,
    active      boolean      NOT NULL DEFAULT true,
    created_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT badge_pkey PRIMARY KEY (id),
    CONSTRAINT badge_code_key UNIQUE (code),
    CONSTRAINT badge_earned_fields_check CHECK (
        (kind = 'EARNED' AND metric IS NOT NULL AND threshold IS NOT NULL)
        OR (kind = 'CONFERRED' AND metric IS NULL AND threshold IS NULL)
    )
);

-- Grants. One row per artisan per badge they hold. Never computed on read —
-- materialized so a revoke or a changed threshold doesn't retroactively
-- rewrite history, and so storefront reads are a plain indexed lookup.
CREATE TABLE artisan_badge (
    artisan_id    uuid         NOT NULL,
    badge_id      uuid         NOT NULL,
    granted_at    timestamptz  NOT NULL DEFAULT now(),
    granted_by    text         NOT NULL,  -- principal id, or 'system' for EARNED
    evidence      jsonb,                  -- e.g. {"listing_id": "..."} for CONFERRED notes
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

-- Running counters the earned-badge consumer maintains. Kept separate from
-- artisan_badge so a threshold can be raised later without losing the
-- artisan's accumulated count.
CREATE TABLE artisan_badge_progress (
    artisan_id  uuid          NOT NULL,
    metric      badge_metric  NOT NULL,
    value       bigint        NOT NULL DEFAULT 0,
    updated_at  timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT artisan_badge_progress_pkey PRIMARY KEY (artisan_id, metric),
    CONSTRAINT artisan_badge_progress_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE
);
```

Seed rows (inserted by the same migration): conferred —
`verified_artisan`, `master_craftsperson`, `national_awardee`,
`gi_practitioner`, `cluster_coordinator`; earned — `first_listing`
(LISTINGS_PUBLISHED, 1, untiered), `catalog_builder` at BRONZE/SILVER/GOLD
(LISTINGS_PUBLISHED, 5/25/100), `provenance_keeper` at BRONZE/SILVER/GOLD
(PROVENANCE_SEALED, 1/10/50), `order_fulfiller` at BRONZE/SILVER/GOLD
(LOTS_COMPLETED, 1/10/50). 14 badges total.

Display text (name, one-line description, criteria sentence) is **not**
stored in the DB — it is i18n keys, one triad per badge code:
`badge.<code>.name`, `badge.<code>.desc`, `badge.<code>.criteria`. ~14
badges × 3 keys = 42 keys × 21 locales. This is the deliberate exception to
the `trend_link` DB-text precedent: the badge catalog is closed and small,
and an artisan seeing their own badge name in their own script matters more
here than it does for admin-authored trend links.

### proto — `proto/badges/v1/badges.proto`

```
service BadgeService {
  rpc ListBadgeCatalog(ListBadgeCatalogRequest) returns (ListBadgeCatalogResponse);       // public
  rpc ListArtisanBadges(ListArtisanBadgesRequest) returns (ListArtisanBadgesResponse);     // public
  rpc GetBadgeProgress(GetBadgeProgressRequest) returns (GetBadgeProgressResponse);        // self
  rpc GrantBadge(GrantBadgeRequest) returns (GrantBadgeResponse);                          // MINISTRY
  rpc RevokeBadge(RevokeBadgeRequest) returns (RevokeBadgeResponse);                       // MINISTRY
}

message Badge {
  string id = 1; string code = 2; BadgeKind kind = 3; BadgeTier tier = 4; // tier optional
  string icon_name = 5; BadgeMetric metric = 6; optional int64 threshold = 7;
  int32 sort_order = 8;
}
message ArtisanBadge {
  Badge badge = 1; google.protobuf.Timestamp granted_at = 2; string granted_by = 3;
}
message BadgeProgressEntry { BadgeMetric metric = 1; int64 value = 2; }
```

`GrantBadge`/`RevokeBadge` requests only accept `badge_id`s where
`kind = CONFERRED` — the service layer rejects a grant attempt against an
EARNED badge with a validation error, since those are only ever written by
the consumer.

### core-svc

- `internal/core/domain/badge.go` — `Badge`, `ArtisanBadge`, `BadgeProgress` domain types.
- `internal/core/repo/badge.go` — sqlc-backed repo: `ListBadgeCatalog`,
  `ListArtisanBadges(artisanID)`, `GetBadgeProgress(artisanID)`,
  `InsertGrant` (used by both the admin path and the consumer, idempotent via
  the `(artisan_id, badge_id)` PK — a duplicate insert on replay is caught
  and treated as success, not an error), `RevokeGrant`, `UpsertProgress`.
- `internal/core/service/badge.go` — `ListBadgeCatalog`/`ListArtisanBadges`
  use `auth.PrincipalFrom` (never error for an anonymous caller);
  `GetBadgeProgress` requires the caller to be that artisan;
  `GrantBadge`/`RevokeBadge` require `auth.RequireRole(MINISTRY)`.
- `internal/core/handler/badge.go` — gRPC handler wiring.
- **New**: `internal/core/consumer/badge.go` — a Kafka consumer group,
  modelled on `services/channel-svc/internal/channel/consumer/consumer.go`
  (core-svc has no consumer package today; this introduces the pattern).
  Subscribes to `topics.CatalogListingPublished`, `topics.CatalogProvenanceSealed`,
  `topics.OrderLotAccepted`, `topics.OrderLotCompleted`. On each event: in one
  transaction, increments `artisan_badge_progress` for the relevant metric,
  then for every EARNED badge on that metric whose threshold is now met and
  which the artisan doesn't already hold, inserts an `artisan_badge` row with
  `granted_by = 'system'`. At-least-once delivery is safe: the progress
  increment uses `ON CONFLICT ... DO UPDATE SET value = artisan_badge_progress.value + 1`
  keyed by a per-event dedupe check against the event's own id stored
  alongside (reuses the existing idempotency/outbox event-id pattern already
  in the codebase — see `migrations/011_outbox.sql` /
  `migrations/012_idempotency.sql` for the convention to follow) so a
  redelivered event does not double-count.
- `internal/core/wiring/wiring.go` — register the repo, service, and start
  the new consumer alongside existing wiring.
- `internal/core/handler/identity.go`'s `PublicMethods()` — add
  `badges.v1.BadgeService/ListBadgeCatalog` and `.../ListArtisanBadges`.

### BFF

`services/bff/internal/bff/client/badge.go` (gRPC client wrapper), routes in
`server.go`:
- `GET /badges` — public, catalog.
- `GET /artisans/:id/badges` — public.
- `GET /badges/me/progress` — authed.
- `POST /artisans/:id/badges` — authed, MINISTRY, idempotent (`withIdempotency`), body `{badge_code}`.
- `DELETE /artisans/:id/badges/:code` — authed, MINISTRY, body `{reason}`.

Add all five to `services/bff/openapi.json`, regenerate
`web/packages/api/src/generated/schema.d.ts`
(`pnpm --filter @kalakriti/api api:gen`), add typed wrappers to
`operations.ts` + the `index.ts` barrel.

### Frontend

- `packages/ui`: `<BadgeChip>` (icon + label, hairline border, tier-coloured
  edge not tier-coloured fill — never colour-alone for meaning, tier name is
  always in the accessible label too), `<BadgeGrid>` (earned first, then
  in-progress with a real progress bar sourced from `GetBadgeProgress`, then
  locked-with-criteria last).
- Artisan app: `/badges` route using `<BadgeGrid>`, "read aloud" per badge
  via the existing voice layer.
- Buyer app: badge row added to `/artisan/{slug}` storefront, next to
  awardee status.
- Admin app: `/artisans/{id}` gets a grant/revoke panel (CONFERRED badges
  only, dropdown restricted to `kind = CONFERRED` catalog entries) — human
  decision, no bulk action, consistent with the moderation-queue rule in
  Batch 0.

### Acceptance criteria

- Publishing a listing, sealing provenance, and completing a lot each bump
  progress and grant the correct tier exactly once, even if the consumer
  restarts mid-batch (replay does not double-grant or double-count).
- `GET /artisans/:id/badges` and `GET /badges` work with no Authorization
  header.
- A MINISTRY admin can grant/revoke a CONFERRED badge; attempting to grant
  an EARNED badge through the API is rejected.
- Badge names/descriptions render correctly in all 21 locales.

---

## Feature 8: Government scheme eligibility guidance

### Purpose

Help an artisan discover which government schemes (PM Vishwakarma, Handicrafts
Pehchan ID, AHVY, NHDP, SFURTI, Mudra, Stand-Up India, ODOP, and others a
Ministry official adds later) they may qualify for, and what to do next. This
is guidance, never a determination, and never an application channel.

### The one rule everything else follows

**No screen in this feature ever tells an artisan "you are eligible."** Every
match renders as `MAY_QUALIFY`, `CHECK_REQUIRED`, or `UNLIKELY`, always paired
with the scheme's real official URL and the sentence "confirm on the official
portal before applying." Machine-checkable criteria (state, craft, years of
experience, existing IDs, cluster/SHG membership, social category) drive the
status; anything not safely machine-checkable (income ceiling, land holding,
prior benefit receipt) is a manual checklist item the artisan reads and
self-confirms, never inferred. The platform does not submit or link out to
any application flow on the artisan's behalf — this is discovery and
paperwork-readiness only.

### Data model — `migrations/034_schemes.sql`

```sql
-- Optional, nullable, self-reported. PREFER_NOT_TO_SAY is a first-class
-- value, not a missing one — registration must offer it as a real choice,
-- not silently default to NULL only when skipped.
CREATE TYPE social_category AS ENUM (
    'GENERAL', 'OBC', 'SC', 'ST', 'EWS', 'PREFER_NOT_TO_SAY'
);
ALTER TABLE artisan ADD COLUMN social_category social_category;

CREATE TYPE scheme_authority AS ENUM ('CENTRAL', 'STATE');
CREATE TYPE scheme_criterion_type AS ENUM (
    'SOCIAL_CATEGORY', 'STATE_CODE', 'CRAFT_ID', 'MIN_YEARS_EXPERIENCE',
    'HAS_PEHCHAN_ID', 'HAS_PM_VISHWAKARMA_ID', 'CLUSTER_MEMBER', 'SHG_MEMBER'
);

CREATE TABLE government_scheme (
    id                uuid              NOT NULL,
    code              text              NOT NULL,
    authority         scheme_authority  NOT NULL,
    ministry          text              NOT NULL,
    official_url      text              NOT NULL,
    -- NULL = all-India; set for a state-specific scheme.
    state_code        text,
    -- Seeded schemes use the i18n key; admin-added schemes use free text.
    -- Renderer prefers the key when present, falls back to the text column.
    name_i18n_key     text,
    name_text         text,
    summary_i18n_key  text,
    summary_text      text,
    active            boolean           NOT NULL DEFAULT true,
    sort_order        integer           NOT NULL DEFAULT 0,
    curated_by        text              NOT NULL,
    created_at        timestamptz       NOT NULL DEFAULT now(),
    updated_at        timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT government_scheme_pkey PRIMARY KEY (id),
    CONSTRAINT government_scheme_code_key UNIQUE (code),
    CONSTRAINT government_scheme_official_url_check CHECK (official_url ~ '^https://'),
    CONSTRAINT government_scheme_name_check CHECK (name_i18n_key IS NOT NULL OR name_text IS NOT NULL),
    CONSTRAINT government_scheme_summary_check CHECK (summary_i18n_key IS NOT NULL OR summary_text IS NOT NULL)
);

-- Machine-checkable criteria. All rows for a scheme must pass (AND).
-- negate=true means "must NOT have" (e.g. NOT already HAS_PEHCHAN_ID for a
-- scheme that is specifically the on-ramp to getting one).
CREATE TABLE scheme_criterion (
    id             uuid                   NOT NULL,
    scheme_id      uuid                   NOT NULL,
    type           scheme_criterion_type  NOT NULL,
    string_values  text[]                 NOT NULL DEFAULT '{}',
    int_value      bigint,
    negate         boolean                NOT NULL DEFAULT false,
    CONSTRAINT scheme_criterion_pkey PRIMARY KEY (id),
    CONSTRAINT scheme_criterion_scheme_id_fkey FOREIGN KEY (scheme_id)
        REFERENCES government_scheme (id) ON DELETE CASCADE
);

-- Not machine-checkable. Rendered as a checklist the artisan reads and
-- self-confirms; never used to auto-compute a status.
CREATE TABLE scheme_manual_check (
    id          uuid         NOT NULL,
    scheme_id   uuid         NOT NULL,
    i18n_key    text,
    check_text  text,
    sort_order  integer      NOT NULL DEFAULT 0,
    CONSTRAINT scheme_manual_check_pkey PRIMARY KEY (id),
    CONSTRAINT scheme_manual_check_scheme_id_fkey FOREIGN KEY (scheme_id)
        REFERENCES government_scheme (id) ON DELETE CASCADE,
    CONSTRAINT scheme_manual_check_text_check CHECK (i18n_key IS NOT NULL OR check_text IS NOT NULL)
);

CREATE INDEX scheme_criterion_scheme_idx ON scheme_criterion (scheme_id);
CREATE INDEX scheme_manual_check_scheme_idx ON scheme_manual_check (scheme_id);
CREATE INDEX government_scheme_active_idx ON government_scheme (active, sort_order);
```

Seed rows (this migration): 8 schemes with real, currently-known official
URLs — `pm_vishwakarma`, `handicrafts_pehchan_id`, `ahvy`
(Ambedkar Hastshilp Vikas Yojana), `nhdp` (National Handicrafts Development
Programme), `sfurti`, `mudra`, `stand_up_india`, `odop`. Only rules that are
unambiguous and stable go in as `scheme_criterion` rows (e.g. PM Vishwakarma
is craft/artisan-occupation-based with no state restriction; Stand-Up India
requires SC/ST or woman entrepreneur — encoded as a `SOCIAL_CATEGORY`
criterion; `handicrafts_pehchan_id` uses a `negate` criterion on
`HAS_PEHCHAN_ID` since it's the on-ramp to getting one, not for those who
already have it). Anything involving income ceilings, loan-history checks,
or wording I am not fully certain of goes into `scheme_manual_check` instead
of being encoded as a criterion — deliberately erring toward "ask the artisan
to check" over "assert a machine-checkable rule I reconstructed from
memory."

### proto — `proto/schemes/v1/schemes.proto`

```
service SchemeService {
  rpc ListSchemes(ListSchemesRequest) returns (ListSchemesResponse);       // public, reference data
  rpc MatchSchemes(MatchSchemesRequest) returns (MatchSchemesResponse);    // self, authed
  rpc UpsertScheme(UpsertSchemeRequest) returns (UpsertSchemeResponse);    // MINISTRY
  rpc DeleteScheme(DeleteSchemeRequest) returns (DeleteSchemeResponse);    // MINISTRY
}

enum MatchStatus { MATCH_STATUS_UNSPECIFIED = 0; MAY_QUALIFY = 1; CHECK_REQUIRED = 2; UNLIKELY = 3; }

message SchemeMatch {
  GovernmentScheme scheme = 1;
  MatchStatus status = 2;
  repeated string matched_criteria_labels = 3;   // i18n keys for what matched, for legibility
  repeated string unmet_criteria_labels = 4;
  repeated ManualCheck manual_checks = 5;
}
```

### core-svc

- `internal/core/domain/scheme.go`, `internal/core/repo/scheme.go`,
  `internal/core/service/scheme.go`, `internal/core/handler/scheme.go` —
  same layering as `badge`.
- `MatchSchemes` is a pure function reading the caller's own `artisan` row
  (state_code, craft_ids via existing listing/ontology join, years_of_experience,
  pehchan_id/pm_vishwakarma_id presence, cluster_id, shg membership,
  social_category) against each active scheme's criteria. No network calls,
  no ML, no LLM — deterministic and auditable. `ListSchemes` and
  `MatchSchemes` both use `auth.PrincipalFrom`/require-self, not open
  anonymous access for `MatchSchemes` (it reads the caller's own PII-adjacent
  fields); `ListSchemes` (scheme reference data with no artisan data) is
  public and goes in `PublicMethods()`.
- `UpsertScheme`/`DeleteScheme` require `auth.RequireRole(MINISTRY)`.

### BFF

- `GET /schemes` — public.
- `GET /schemes/match` — authed, self.
- `POST /schemes`, `PATCH /schemes/:id`, `DELETE /schemes/:id` — authed, MINISTRY.

Same openapi.json + codegen + operations.ts steps as badges.

### Frontend

- Artisan registration (`web/apps/artisan/src/routes/register/*`): one new
  optional step, "social category (optional)" with a clear skip that reads
  as a real choice and `PREFER_NOT_TO_SAY` offered as its own tile, spoken
  aloud like every other registration question per Batch 7's rules.
- Artisan `/schemes`: `MAY_QUALIFY` schemes first, each card shows ministry,
  what you get (summary), which of the artisan's own facts matched (chips,
  e.g. "your state", "your craft", "5+ years experience"), the manual
  checklist as plain checkboxes (self-confirm only, not submitted anywhere),
  "read aloud," and an outbound link to the official portal clearly marked
  as leaving the platform. `CHECK_REQUIRED` and `UNLIKELY` sections below,
  collapsed by default but never hidden.
- Ties to `/earnings`: the income statement PDF already generated there gets
  one added line noting it can serve as income proof for scheme applications
  — reuses existing functionality, no new document type.
- Admin `/schemes`: CRUD list + form, criteria builder (type + value pickers
  from the existing enums), manual-check list editor.

### Acceptance criteria

- No screen anywhere renders the word "eligible" as a determination; only
  the three defined statuses, each paired with the official URL.
- `MatchSchemes` for an artisan with `social_category = PREFER_NOT_TO_SAY`
  correctly treats any `SOCIAL_CATEGORY` criterion as unmet (not an error,
  not a crash) and still returns every other scheme's status.
- Every seeded scheme's `official_url` resolves to a real government domain
  (spot-checked, not assumed) at implementation time.
- An insights aggregate that ever groups by `social_category` (none exist
  yet in this scope, but the column exists) is left for a future spec if
  requested — this spec adds the column and the artisan-facing matcher only,
  it does not add a ministry-facing social-category breakdown.

---

## Cross-cutting notes for the implementation plan

- Migration numbers: `033_badges.sql`, `034_schemes.sql` (024 and 028 are
  pre-existing gaps in the sequence — do not reuse them).
- Both migrations add to `sqlc.yaml`'s `core` gen block's `queries` list
  (`migrations/queries/badges.sql`, `migrations/queries/schemes.sql`), same
  as the existing `b2b.sql`/`trends.sql` entries.
- `buf`/`protoc-gen-go`/`protoc-gen-go-grpc`/`sqlc` are confirmed on PATH;
  `make proto sqlc` (already a `make up` prerequisite per CLAUDE.md) picks
  up both new proto packages automatically once they exist under `proto/`.
- i18n: after adding keys to `en.ts`, all 20 non-English catalogues must be
  populated in the same pass to keep `i18n-baseline.json` at 0 issues per
  locale — translations are done directly in this implementation (~75 keys
  total across both features is a bounded batch, unlike the large ongoing
  2,363-key catalogue effort tracked separately in project memory).
- Both features' public RPCs need the two-layer public-read fix documented
  in CLAUDE.md: `PublicMethods()` in
  `services/core-svc/internal/core/handler/identity.go` AND the service
  method itself must not call `auth.RequirePrincipal`.
- New pattern introduced: `services/core-svc/internal/core/consumer/` did
  not exist before this work; it is modelled on
  `services/channel-svc/internal/channel/consumer/`.
