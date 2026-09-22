# Government Scheme Guidance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Help an artisan discover which government schemes (PM Vishwakarma, Handicrafts Pehchan ID, AHVY, NHDP, SFURTI, Mudra, Stand-Up India, ODOP) they may qualify for, with every result framed as guidance — `MAY_QUALIFY` / `CHECK_REQUIRED` / `UNLIKELY` — never a determination, always paired with the official portal URL.

**Architecture:** `government_scheme` reference rows (Ministry-curated, mirroring the `trend_link`/`badge` catalog shape) plus two child tables: `scheme_criterion` (machine-checkable, deterministic AND-matching against the artisan's own record) and `scheme_manual_check` (not machine-checkable — rendered as a self-confirm checklist, never auto-evaluated). `MatchSchemes` is a pure function with no network/ML/LLM calls, reading only the caller's own `artisan` row plus a `product.craft_id` lookup, `cluster_member`/`shg_member` presence, and the new nullable `artisan.social_category` column.

**Tech Stack:** Go 1.x, pgx/v5, sqlc, buf, segmentio/kafka-go (unused by this feature — no events), chi (bff), SvelteKit 2 + Svelte 5 runes, TypeScript strict, openapi-typescript.

**Spec:** [docs/superpowers/specs/2026-09-15-badges-schemes-design.md](../specs/2026-09-15-badges-schemes-design.md) — "Feature 8: Government scheme eligibility guidance" section, implemented as written.

**Companion plan:** [2026-09-15-artisan-badges.md](2026-09-15-artisan-badges.md) — independent feature, no shared code, but this plan follows the same conventions it establishes (repo methods on `*repo.Repo`, `service.NewX(repository, log)` wired directly in `main.go`, bff client returning `map[string]any`, i18n key-vs-DB-text pattern). Read that plan's Tasks 3-13 first if anything here is under-specified — the idioms are identical.

## Global Constraints

- **No screen anywhere renders the word "eligible" as a determination.** Every scheme match is one of exactly three statuses, each always paired with the scheme's official URL and a "confirm on the official portal" line. This is the single most important constraint in this plan and overrides any instinct to simplify the UI copy.
- The platform never submits an application or links to anything but the scheme's own official government domain.
- `social_category` is optional, nullable, and `PREFER_NOT_TO_SAY` is a first-class value — never silently default it, never make it required.
- Public BFF routes need both layers (`PublicMethods()` in `services/core-svc/internal/core/handler/identity.go` AND no `auth.RequirePrincipal` in the service method) — applies to `ListSchemes` only; `MatchSchemes` reads the caller's own record and stays authed.
- All user-visible strings are i18n keys except admin-added scheme name/summary/manual-check text, which is DB text (mirrors the `trend_link` precedent — an admin adding a scheme later must not require a code change and a 21-locale translation pass).
- Migration number: `034_schemes.sql` (after badges' `033`).
- Never encode a machine-checkable criterion for a rule not confidently, publicly, and unambiguously known — when in doubt, put it in `scheme_manual_check` instead.

---

### Task 1: Migration `034_schemes.sql`

**Files:**
- Create: `migrations/034_schemes.sql`

**Interfaces:**
- Produces: `artisan.social_category` column; tables `government_scheme`, `scheme_criterion`, `scheme_manual_check`; enums `social_category`, `scheme_authority`, `scheme_criterion_type`. 8 seeded schemes with criteria and manual checks.

- [ ] **Step 1: Write the migration**

```sql
-- migrations/034_schemes.sql
-- +goose Up

CREATE TYPE social_category AS ENUM ('GENERAL', 'OBC', 'SC', 'ST', 'EWS', 'PREFER_NOT_TO_SAY');
ALTER TABLE artisan ADD COLUMN social_category social_category;

CREATE TYPE scheme_authority AS ENUM ('CENTRAL', 'STATE');
CREATE TYPE scheme_criterion_type AS ENUM (
    'SOCIAL_CATEGORY', 'STATE_CODE', 'CRAFT_ID', 'MIN_YEARS_EXPERIENCE',
    'HAS_PEHCHAN_ID', 'HAS_PM_VISHWAKARMA_ID', 'CLUSTER_MEMBER', 'SHG_MEMBER'
);

-- Reference catalog, Ministry-curated. Mirrors trend_link's i18n-key/text
-- hybrid: seeded rows use the key (fully translated, 21 locales); a scheme
-- an admin adds later through the admin UI uses free text instead, so
-- adding a scheme never requires a code change or a translation pass.
CREATE TABLE government_scheme (
    id                uuid              NOT NULL,
    code              text              NOT NULL,
    authority         scheme_authority  NOT NULL,
    ministry          text              NOT NULL,
    official_url      text              NOT NULL,
    state_code        text,
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

-- Machine-checkable criteria. All rows for a scheme must pass (AND). negate
-- means "must NOT have" -- e.g. handicrafts_pehchan_id targets artisans who
-- do NOT already have one.
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

-- Not machine-checkable (income ceilings, land holding, prior benefit
-- receipt). Rendered as a checklist the artisan reads and self-confirms;
-- never used to compute status.
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

-- Seed: 8 schemes. curated_by = 'system' for seed data (distinct from a
-- real admin's principal id, so a future audit can tell seed rows apart).
INSERT INTO government_scheme (id, code, authority, ministry, official_url, name_i18n_key, summary_i18n_key, sort_order, curated_by) VALUES
    (gen_random_uuid(), 'pm_vishwakarma',         'CENTRAL', 'Ministry of Micro, Small and Medium Enterprises', 'https://pmvishwakarma.gov.in', 'scheme.pm_vishwakarma.name',         'scheme.pm_vishwakarma.summary',         10, 'system'),
    (gen_random_uuid(), 'handicrafts_pehchan_id', 'CENTRAL', 'Ministry of Textiles',                             'https://www.handicrafts.nic.in',        'scheme.handicrafts_pehchan_id.name', 'scheme.handicrafts_pehchan_id.summary', 20, 'system'),
    (gen_random_uuid(), 'ahvy',                   'CENTRAL', 'Ministry of Textiles',                             'https://www.handicrafts.nic.in',        'scheme.ahvy.name',                   'scheme.ahvy.summary',                   30, 'system'),
    (gen_random_uuid(), 'nhdp',                   'CENTRAL', 'Ministry of Textiles',                             'https://www.handicrafts.nic.in',        'scheme.nhdp.name',                   'scheme.nhdp.summary',                   40, 'system'),
    (gen_random_uuid(), 'sfurti',                 'CENTRAL', 'Ministry of Micro, Small and Medium Enterprises', 'https://sfurti.msme.gov.in',             'scheme.sfurti.name',                 'scheme.sfurti.summary',                 50, 'system'),
    (gen_random_uuid(), 'mudra',                  'CENTRAL', 'Ministry of Finance',                              'https://www.mudra.org.in',               'scheme.mudra.name',                  'scheme.mudra.summary',                  60, 'system'),
    (gen_random_uuid(), 'stand_up_india',         'CENTRAL', 'Ministry of Finance',                              'https://www.standupmitra.in',            'scheme.stand_up_india.name',         'scheme.stand_up_india.summary',         70, 'system'),
    (gen_random_uuid(), 'odop',                   'CENTRAL', 'Department for Promotion of Industry and Internal Trade', 'https://odop.gov.in',                'scheme.odop.name',                   'scheme.odop.summary',                   80, 'system');

-- Criteria: only rules that are unambiguous, public, and stable. Everything
-- else (income ceilings, no-prior-benefit, land holding) is a manual check.
INSERT INTO scheme_criterion (id, scheme_id, type, negate)
SELECT gen_random_uuid(), id, 'HAS_PEHCHAN_ID', true
FROM government_scheme WHERE code = 'handicrafts_pehchan_id';

INSERT INTO scheme_criterion (id, scheme_id, type, string_values)
SELECT gen_random_uuid(), id, 'SOCIAL_CATEGORY', ARRAY['SC', 'ST']
FROM government_scheme WHERE code = 'stand_up_india';
-- Note: Stand-Up India also qualifies women entrepreneurs regardless of
-- category; the platform has no gender field, so this criterion only
-- encodes the SC/ST branch and the scheme's manual checks (below) cover the
-- women-entrepreneur branch as a self-confirm item instead of a false UNLIKELY.

-- Manual checks (self-confirm only, never auto-evaluated).
INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.pm_vishwakarma.check.traditional_trade', 10
FROM government_scheme WHERE code = 'pm_vishwakarma';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.mudra.check.business_plan', 10
FROM government_scheme WHERE code = 'mudra';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.mudra.check.no_existing_default', 20
FROM government_scheme WHERE code = 'mudra';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.stand_up_india.check.first_time_entrepreneur', 10
FROM government_scheme WHERE code = 'stand_up_india';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.stand_up_india.check.women_entrepreneur_alternative', 20
FROM government_scheme WHERE code = 'stand_up_india';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.sfurti.check.cluster_based', 10
FROM government_scheme WHERE code = 'sfurti';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.ahvy.check.registered_artisan', 10
FROM government_scheme WHERE code = 'ahvy';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.nhdp.check.group_or_individual', 10
FROM government_scheme WHERE code = 'nhdp';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.odop.check.district_product_match', 10
FROM government_scheme WHERE code = 'odop';

-- +goose Down

DROP TABLE IF EXISTS scheme_manual_check;
DROP TABLE IF EXISTS scheme_criterion;
DROP TABLE IF EXISTS government_scheme;
DROP TYPE IF EXISTS scheme_criterion_type;
DROP TYPE IF EXISTS scheme_authority;
ALTER TABLE artisan DROP COLUMN IF EXISTS social_category;
DROP TYPE IF EXISTS social_category;
```

Before finalizing: spot-check `pmvishwakarma.gov.in`, `handicrafts.nic.in`, `sfurti.msme.gov.in`, `mudra.org.in`, `standupmitra.in`, `odop.gov.in` are live, correct official domains at implementation time (a domain can change) — this is the one factual claim in this migration that must be verified against the real world, not just written from memory, per the spec's acceptance criterion.

- [ ] **Step 2: Apply and verify**

Run: `goose -dir migrations postgres "$POSTGRES_DSN" up` (or `make migrate-up`).
Expected: `SELECT count(*) FROM government_scheme;` returns 8; `SELECT count(*) FROM scheme_manual_check;` returns 9; `\d artisan` shows the new `social_category` column.

- [ ] **Step 3: Commit**

```bash
git add migrations/034_schemes.sql
git commit -m "feat(migrations): add government scheme catalog and social_category

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 2: sqlc queries — `migrations/queries/schemes.sql`

**Files:**
- Create: `migrations/queries/schemes.sql`
- Modify: `sqlc.yaml` (add `"migrations/queries/schemes.sql"` to the `core` gen block's `queries` list)
- Modify: `migrations/queries/artisan.sql` (add `social_category` to whatever existing `UpdateArtisan`/`GetArtisan`-style query needs it — read that file first to find the right query to extend rather than adding a new one)

**Interfaces:**
- Produces (after `sqlc generate`): `Querier` methods `ListActiveSchemes`, `GetSchemeCriteria`, `GetSchemeManualChecks`, `UpsertScheme`, `DeleteScheme`, `GetSchemeByID`, `UpdateArtisanSocialCategory` (or an extended existing artisan-update query), plus a query the matcher needs for the artisan's own facts: `GetArtisanMatchFacts`.

- [ ] **Step 1: Write the query file**

```sql
-- migrations/queries/schemes.sql

-- name: ListActiveSchemes :many
SELECT * FROM government_scheme WHERE active ORDER BY sort_order;

-- name: GetSchemeByID :one
SELECT * FROM government_scheme WHERE id = @id;

-- name: GetSchemeCriteria :many
SELECT * FROM scheme_criterion WHERE scheme_id = @scheme_id;

-- name: GetSchemeManualChecks :many
SELECT * FROM scheme_manual_check WHERE scheme_id = @scheme_id ORDER BY sort_order;

-- name: GetAllActiveSchemeCriteria :many
-- Bulk fetch for MatchSchemes -- one round trip for every active scheme's
-- criteria rather than N+1 per scheme.
SELECT sc.* FROM scheme_criterion sc
JOIN government_scheme gs ON gs.id = sc.scheme_id
WHERE gs.active;

-- name: GetAllActiveSchemeManualChecks :many
SELECT smc.* FROM scheme_manual_check smc
JOIN government_scheme gs ON gs.id = smc.scheme_id
WHERE gs.active
ORDER BY smc.sort_order;

-- name: UpsertScheme :one
INSERT INTO government_scheme (
    id, code, authority, ministry, official_url, state_code,
    name_i18n_key, name_text, summary_i18n_key, summary_text,
    active, sort_order, curated_by
) VALUES (
    @id, @code, @authority, @ministry, @official_url, sqlc.narg('state_code'),
    sqlc.narg('name_i18n_key'), sqlc.narg('name_text'), sqlc.narg('summary_i18n_key'), sqlc.narg('summary_text'),
    @active, @sort_order, @curated_by
)
ON CONFLICT (id) DO UPDATE SET
    code = @code, authority = @authority, ministry = @ministry, official_url = @official_url,
    state_code = sqlc.narg('state_code'), name_i18n_key = sqlc.narg('name_i18n_key'),
    name_text = sqlc.narg('name_text'), summary_i18n_key = sqlc.narg('summary_i18n_key'),
    summary_text = sqlc.narg('summary_text'), active = @active, sort_order = @sort_order, updated_at = now()
RETURNING *;

-- name: DeleteScheme :execrows
DELETE FROM government_scheme WHERE id = @id;

-- name: GetArtisanMatchFacts :one
-- Everything MatchSchemes needs about one artisan, in one row. craft_ids and
-- cluster/shg membership are computed via correlated subqueries so this
-- stays a single round trip.
SELECT
    a.state_code,
    a.years_of_experience,
    a.social_category,
    (a.pehchan_id IS NOT NULL) AS has_pehchan_id,
    (a.pm_vishwakarma_id IS NOT NULL) AS has_pm_vishwakarma_id,
    (a.primary_cluster_id IS NOT NULL) AS is_cluster_member,
    EXISTS (SELECT 1 FROM shg_member sm WHERE sm.artisan_id = a.id) AS is_shg_member,
    COALESCE(
        (SELECT array_agg(DISTINCT p.craft_id) FROM product p WHERE p.artisan_id = a.id),
        '{}'
    )::uuid[] AS craft_ids
FROM artisan a
WHERE a.id = @artisan_id;
```

- [ ] **Step 2: Extend the artisan update query for `social_category`**

Read `migrations/queries/artisan.sql` for the existing `UpdateArtisan` (or similarly named) query. Add `social_category` as one more `SET` column, following whatever `sqlc.narg(...)`-for-optional-fields pattern that query already uses for its other nullable columns (e.g. `bio`, `years_of_experience`). Do not invent a separate single-purpose query if the existing update query already accepts a broad set of optional fields — extend it.

- [ ] **Step 3: Wire into sqlc.yaml and generate**

Add `- "migrations/queries/schemes.sql"` to `sqlc.yaml`'s `core` block, then:

Run: `sqlc generate && cd services/core-svc && go build ./...`
Expected: no errors; `services/core-svc/internal/core/repo/db/schemes.sql.gen.go` created.

- [ ] **Step 4: Commit**

```bash
git add migrations/queries/schemes.sql migrations/queries/artisan.sql sqlc.yaml services/core-svc/internal/core/repo/db/schemes.sql.gen.go services/core-svc/internal/core/repo/db/artisan.sql.gen.go
git commit -m "feat(core-svc): generate sqlc queries for government schemes

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 3: Domain types and the pure matcher — `domain/scheme.go`

This is the load-bearing correctness task: the matcher is a pure function, unit-testable with zero database, and every status-computation rule the spec's "one rule everything else follows" depends on lives here.

**Files:**
- Create: `services/core-svc/internal/core/domain/scheme.go`
- Test: `services/core-svc/internal/core/domain/scheme_test.go`

**Interfaces:**
- Produces: `MatchStatus` enum (`MayQualify`, `CheckRequired`, `Unlikely`), `SchemeCriterionType` enum, `Scheme`, `SchemeCriterion`, `SchemeManualCheck`, `ArtisanMatchFacts`, `SchemeMatch` structs, and the pure function `MatchScheme(scheme Scheme, criteria []SchemeCriterion, manualChecks []SchemeManualCheck, facts ArtisanMatchFacts) SchemeMatch` — consumed by Task 5 (service).

- [ ] **Step 1: Write the failing tests**

```go
// services/core-svc/internal/core/domain/scheme_test.go
package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMatchScheme_AllCriteriaMetNoManualChecks_MayQualify(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "test_scheme"}
	criteria := []SchemeCriterion{{Type: CriterionMinYearsExperience, IntValue: ptrInt64(2)}}
	facts := ArtisanMatchFacts{YearsOfExperience: ptrInt32(5)}

	match := MatchScheme(scheme, criteria, nil, facts)
	require.Equal(t, MayQualify, match.Status)
}

func TestMatchScheme_UnmetCriterion_Unlikely(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "test_scheme"}
	criteria := []SchemeCriterion{{Type: CriterionMinYearsExperience, IntValue: ptrInt64(10)}}
	facts := ArtisanMatchFacts{YearsOfExperience: ptrInt32(2)}

	match := MatchScheme(scheme, criteria, nil, facts)
	require.Equal(t, Unlikely, match.Status)
	require.Contains(t, match.UnmetCriteriaLabels, "scheme.criterion.min_years_experience")
}

func TestMatchScheme_AllCriteriaMetWithOutstandingManualChecks_CheckRequired(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "test_scheme"}
	manualChecks := []SchemeManualCheck{{I18nKey: ptrString("scheme.test.check.income")}}
	facts := ArtisanMatchFacts{}

	match := MatchScheme(scheme, nil, manualChecks, facts)
	require.Equal(t, CheckRequired, match.Status)
	require.Len(t, match.ManualChecks, 1)
}

func TestMatchScheme_NilSocialCategory_TreatsSocialCategoryCriterionAsUnmet_NoError(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "stand_up_india"}
	criteria := []SchemeCriterion{{Type: CriterionSocialCategory, StringValues: []string{"SC", "ST"}}}
	facts := ArtisanMatchFacts{SocialCategory: nil} // PREFER_NOT_TO_SAY or never answered

	match := MatchScheme(scheme, criteria, nil, facts)
	require.Equal(t, Unlikely, match.Status)
}

func TestMatchScheme_NegatedCriterion_MatchesWhenArtisanDoesNotHaveIt(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "handicrafts_pehchan_id"}
	criteria := []SchemeCriterion{{Type: CriterionHasPehchanID, Negate: true}}
	facts := ArtisanMatchFacts{HasPehchanID: false}

	match := MatchScheme(scheme, criteria, nil, facts)
	require.Equal(t, MayQualify, match.Status)

	factsWithID := ArtisanMatchFacts{HasPehchanID: true}
	matchWithID := MatchScheme(scheme, criteria, nil, factsWithID)
	require.Equal(t, Unlikely, matchWithID.Status)
}

func ptrInt64(v int64) *int64   { return &v }
func ptrInt32(v int32) *int32   { return &v }
func ptrString(v string) *string { return &v }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/core-svc && go test ./internal/core/domain/... -run TestMatchScheme -v`
Expected: FAIL (compile error — types undefined).

- [ ] **Step 3: Write the implementation**

```go
// services/core-svc/internal/core/domain/scheme.go

package domain

import (
	"time"

	"github.com/google/uuid"
)

// MatchStatus is the outcome of matching one artisan against one scheme.
// This is the entire vocabulary this feature is allowed to show an artisan
// -- there is deliberately no fourth "Eligible" status. See the spec's
// "one rule everything else follows".
type MatchStatus string

const (
	MayQualify    MatchStatus = "MAY_QUALIFY"
	CheckRequired MatchStatus = "CHECK_REQUIRED"
	Unlikely      MatchStatus = "UNLIKELY"
)

// SchemeCriterionType is one machine-checkable fact about an artisan.
type SchemeCriterionType string

const (
	CriterionSocialCategory      SchemeCriterionType = "SOCIAL_CATEGORY"
	CriterionStateCode           SchemeCriterionType = "STATE_CODE"
	CriterionCraftID             SchemeCriterionType = "CRAFT_ID"
	CriterionMinYearsExperience  SchemeCriterionType = "MIN_YEARS_EXPERIENCE"
	CriterionHasPehchanID        SchemeCriterionType = "HAS_PEHCHAN_ID"
	CriterionHasPMVishwakarmaID  SchemeCriterionType = "HAS_PM_VISHWAKARMA_ID"
	CriterionClusterMember       SchemeCriterionType = "CLUSTER_MEMBER"
	CriterionSHGMember           SchemeCriterionType = "SHG_MEMBER"
)

// Scheme is one catalog entry.
type Scheme struct {
	ID              uuid.UUID
	Code            string
	Authority       string
	Ministry        string
	OfficialURL     string
	StateCode       *string
	NameI18nKey     *string
	NameText        *string
	SummaryI18nKey  *string
	SummaryText     *string
	SortOrder       int32
}

// SchemeCriterion is one machine-checkable rule attached to a scheme. All of
// a scheme's criteria must pass (AND) for the scheme to be MAY_QUALIFY or
// CHECK_REQUIRED rather than UNLIKELY.
type SchemeCriterion struct {
	Type         SchemeCriterionType
	StringValues []string
	IntValue     *int64
	Negate       bool
}

// SchemeManualCheck is one not-machine-checkable item the artisan must read
// and self-confirm. Never evaluated automatically.
type SchemeManualCheck struct {
	I18nKey   *string
	CheckText *string
}

// ArtisanMatchFacts is everything MatchScheme needs to know about one
// artisan. A nil pointer field means "unknown" and any criterion depending
// on it is treated as unmet, never as an error and never as a pass.
type ArtisanMatchFacts struct {
	StateCode            string
	YearsOfExperience    *int32
	SocialCategory       *string // one of the social_category enum values, or nil
	HasPehchanID         bool
	HasPMVishwakarmaID   bool
	IsClusterMember      bool
	IsSHGMember          bool
	CraftIDs             []uuid.UUID
}

// SchemeMatch is one scheme's result for one artisan.
type SchemeMatch struct {
	Scheme               Scheme
	Status                MatchStatus
	MatchedCriteriaLabels []string
	UnmetCriteriaLabels   []string
	ManualChecks          []SchemeManualCheck
}

// MatchScheme is a pure function: no I/O, no randomness, no clock. Every
// criterion must pass for the scheme to be anything but UNLIKELY; if every
// criterion passes and there is at least one manual check, the status is
// CHECK_REQUIRED; if every criterion passes and there are no manual checks,
// it is MAY_QUALIFY. A criterion whose underlying fact is unknown (a nil
// pointer in facts) is treated as unmet, matching the spec's requirement
// that PREFER_NOT_TO_SAY / never-answered social_category must not error
// and must not silently pass.
func MatchScheme(scheme Scheme, criteria []SchemeCriterion, manualChecks []SchemeManualCheck, facts ArtisanMatchFacts) SchemeMatch {
	match := SchemeMatch{Scheme: scheme, ManualChecks: manualChecks}

	allMet := true
	for _, c := range criteria {
		met := criterionMet(c, facts)
		label := "scheme.criterion." + criterionLabelSuffix(c.Type)
		if met {
			match.MatchedCriteriaLabels = append(match.MatchedCriteriaLabels, label)
		} else {
			match.UnmetCriteriaLabels = append(match.UnmetCriteriaLabels, label)
			allMet = false
		}
	}

	switch {
	case !allMet:
		match.Status = Unlikely
	case len(manualChecks) > 0:
		match.Status = CheckRequired
	default:
		match.Status = MayQualify
	}
	return match
}

func criterionMet(c SchemeCriterion, facts ArtisanMatchFacts) bool {
	var raw bool
	switch c.Type {
	case CriterionSocialCategory:
		raw = facts.SocialCategory != nil && containsString(c.StringValues, *facts.SocialCategory)
	case CriterionStateCode:
		raw = containsString(c.StringValues, facts.StateCode)
	case CriterionCraftID:
		raw = anyUUIDInStrings(facts.CraftIDs, c.StringValues)
	case CriterionMinYearsExperience:
		raw = c.IntValue != nil && facts.YearsOfExperience != nil && int64(*facts.YearsOfExperience) >= *c.IntValue
	case CriterionHasPehchanID:
		raw = facts.HasPehchanID
	case CriterionHasPMVishwakarmaID:
		raw = facts.HasPMVishwakarmaID
	case CriterionClusterMember:
		raw = facts.IsClusterMember
	case CriterionSHGMember:
		raw = facts.IsSHGMember
	default:
		raw = false
	}
	if c.Negate {
		return !raw
	}
	return raw
}

func criterionLabelSuffix(t SchemeCriterionType) string {
	switch t {
	case CriterionSocialCategory:
		return "social_category"
	case CriterionStateCode:
		return "state_code"
	case CriterionCraftID:
		return "craft"
	case CriterionMinYearsExperience:
		return "min_years_experience"
	case CriterionHasPehchanID:
		return "has_pehchan_id"
	case CriterionHasPMVishwakarmaID:
		return "has_pm_vishwakarma_id"
	case CriterionClusterMember:
		return "cluster_member"
	case CriterionSHGMember:
		return "shg_member"
	default:
		return "unknown"
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func anyUUIDInStrings(ids []uuid.UUID, targets []string) bool {
	for _, id := range ids {
		if containsString(targets, id.String()) {
			return true
		}
	}
	return false
}

// SchemeMatchedAt is not part of SchemeMatch -- matching is computed live on
// every request (it is cheap: a handful of in-memory comparisons over data
// already fetched in one query), not cached or timestamped.
var _ = time.Now // remove this line; time is unused in this file -- drop the import too
```

Delete the trailing `var _ = time.Now` line and the `"time"` import — both are dead, left in only as a note-to-self that matching is not cached; the comment above `SchemeMatchedAt` already says that without needing the import. The corrected import block is just:

```go
import (
	"github.com/google/uuid"
)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/core-svc && go test ./internal/core/domain/... -run TestMatchScheme -v`
Expected: all 5 PASS.

- [ ] **Step 5: Commit**

```bash
git add services/core-svc/internal/core/domain/scheme.go services/core-svc/internal/core/domain/scheme_test.go
git commit -m "feat(core-svc): add scheme domain types and pure matcher

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 4: Repository — `repo/scheme.go`

**Files:**
- Create: `services/core-svc/internal/core/repo/scheme.go`
- Test: `services/core-svc/internal/core/repo/scheme_test.go`

**Interfaces:**
- Consumes: `db.Querier` methods (Task 2), `domain.*` (Task 3).
- Produces (methods on `*Repo`):
  - `ListActiveSchemes(ctx) ([]domain.Scheme, error)`
  - `GetSchemeByID(ctx, id uuid.UUID) (domain.Scheme, error)`
  - `GetAllActiveSchemeCriteria(ctx) (map[uuid.UUID][]domain.SchemeCriterion, error)` — keyed by scheme id, one query.
  - `GetAllActiveSchemeManualChecks(ctx) (map[uuid.UUID][]domain.SchemeManualCheck, error)` — same shape.
  - `GetArtisanMatchFacts(ctx, artisanID uuid.UUID) (domain.ArtisanMatchFacts, error)`
  - `UpsertScheme(ctx, s domain.Scheme, criteria []domain.SchemeCriterion, manualChecks []domain.SchemeManualCheck) (domain.Scheme, error)` — replaces the scheme's criteria/manual-check children wholesale (delete-then-insert inside one transaction) since an admin edit form submits the full set, not a diff.
  - `DeleteScheme(ctx, id uuid.UUID) (bool, error)`

- [ ] **Step 1: Write the failing test**

```go
// services/core-svc/internal/core/repo/scheme_test.go
package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepo_ListActiveSchemes_ReturnsSeeded8(t *testing.T) {
	r, cleanup := newTestRepo(t) // reuse this package's existing harness (see badge_test.go's Task 5 for the same pattern)
	defer cleanup()

	schemes, err := r.ListActiveSchemes(context.Background())
	require.NoError(t, err)
	require.Len(t, schemes, 8)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/core-svc && go test ./internal/core/repo/... -run TestRepo_ListActiveSchemes -v`
Expected: FAIL (compile error).

- [ ] **Step 3: Write the implementation**

```go
// services/core-svc/internal/core/repo/scheme.go

package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

func (r *Repo) ListActiveSchemes(ctx context.Context) ([]domain.Scheme, error) {
	rows, err := r.q.ListActiveSchemes(ctx)
	if err != nil {
		return nil, translate(err, "schemes")
	}
	out := make([]domain.Scheme, len(rows))
	for i, row := range rows {
		out[i] = schemeFromRow(row)
	}
	return out, nil
}

func (r *Repo) GetSchemeByID(ctx context.Context, id uuid.UUID) (domain.Scheme, error) {
	row, err := r.q.GetSchemeByID(ctx, id)
	if err != nil {
		return domain.Scheme{}, translate(err, "scheme")
	}
	return schemeFromRow(row), nil
}

// GetAllActiveSchemeCriteria fetches every active scheme's criteria in one
// query and groups them by scheme id, so MatchSchemes (service layer) does
// not issue one query per scheme.
func (r *Repo) GetAllActiveSchemeCriteria(ctx context.Context) (map[uuid.UUID][]domain.SchemeCriterion, error) {
	rows, err := r.q.GetAllActiveSchemeCriteria(ctx)
	if err != nil {
		return nil, translate(err, "scheme criteria")
	}
	out := make(map[uuid.UUID][]domain.SchemeCriterion)
	for _, row := range rows {
		out[row.SchemeID] = append(out[row.SchemeID], domain.SchemeCriterion{
			Type: domain.SchemeCriterionType(row.Type), StringValues: row.StringValues,
			IntValue: row.IntValue, Negate: row.Negate,
		})
	}
	return out, nil
}

func (r *Repo) GetAllActiveSchemeManualChecks(ctx context.Context) (map[uuid.UUID][]domain.SchemeManualCheck, error) {
	rows, err := r.q.GetAllActiveSchemeManualChecks(ctx)
	if err != nil {
		return nil, translate(err, "scheme manual checks")
	}
	out := make(map[uuid.UUID][]domain.SchemeManualCheck)
	for _, row := range rows {
		out[row.SchemeID] = append(out[row.SchemeID], domain.SchemeManualCheck{
			I18nKey: row.I18nKey, CheckText: row.CheckText,
		})
	}
	return out, nil
}

// GetArtisanMatchFacts reads everything MatchSchemes needs about one
// artisan in a single round trip.
func (r *Repo) GetArtisanMatchFacts(ctx context.Context, artisanID uuid.UUID) (domain.ArtisanMatchFacts, error) {
	row, err := r.q.GetArtisanMatchFacts(ctx, artisanID)
	if err != nil {
		return domain.ArtisanMatchFacts{}, translate(err, "artisan match facts")
	}
	var socialCategory *string
	if row.SocialCategory != nil {
		s := string(*row.SocialCategory)
		socialCategory = &s
	}
	return domain.ArtisanMatchFacts{
		StateCode: row.StateCode, YearsOfExperience: row.YearsOfExperience, SocialCategory: socialCategory,
		HasPehchanID: row.HasPehchanID, HasPMVishwakarmaID: row.HasPmVishwakarmaID,
		IsClusterMember: row.IsClusterMember, IsSHGMember: row.IsShgMember, CraftIDs: row.CraftIds,
	}, nil
}

// UpsertScheme replaces the scheme row and wholesale-replaces its criteria
// and manual checks (delete-then-insert in one transaction) since an admin
// edit form always submits the complete set, not an incremental diff --
// simpler and safer than reconciling row-by-row.
func (r *Repo) UpsertScheme(ctx context.Context, s domain.Scheme, criteria []domain.SchemeCriterion, manualChecks []domain.SchemeManualCheck) (domain.Scheme, error) {
	// Uses r.pool directly for a short explicit transaction -- check this
	// package's existing multi-statement write (e.g. repo/catalog.go's
	// listing-with-media insert, referenced in CLAUDE.md's sqlc section) for
	// the actual transaction-acquisition idiom already used elsewhere in
	// this package (likely r.pool.Begin(ctx) with pgx, then r.q.WithTx(tx)),
	// and follow that exact pattern here rather than inventing a new one.
	return domain.Scheme{}, errNotImplemented // replaced in Step 3b below
}

func (r *Repo) DeleteScheme(ctx context.Context, id uuid.UUID) (bool, error) {
	n, err := r.q.DeleteScheme(ctx, id)
	if err != nil {
		return false, translate(err, "scheme delete")
	}
	return n > 0, nil
}

func schemeFromRow(row db.GovernmentScheme) domain.Scheme {
	return domain.Scheme{
		ID: row.ID, Code: row.Code, Authority: string(row.Authority), Ministry: row.Ministry,
		OfficialURL: row.OfficialUrl, StateCode: row.StateCode, NameI18nKey: row.NameI18nKey,
		NameText: row.NameText, SummaryI18nKey: row.SummaryI18nKey, SummaryText: row.SummaryText,
		SortOrder: row.SortOrder,
	}
}
```

**Step 3b — replace the `UpsertScheme` placeholder.** Before writing the real body, find this package's existing transactional multi-statement write pattern: run `grep -n "pool.Begin\|WithTx\|BeginTx" services/core-svc/internal/core/repo/*.go` and copy its exact idiom (likely something like `tx, err := r.pool.Begin(ctx)` then `qtx := r.q.WithTx(tx)` then `defer tx.Rollback(ctx)` then `tx.Commit(ctx)`). Using that exact idiom, the real implementation is:

```go
func (r *Repo) UpsertScheme(ctx context.Context, s domain.Scheme, criteria []domain.SchemeCriterion, manualChecks []domain.SchemeManualCheck) (domain.Scheme, error) {
	tx, err := r.pool.Begin(ctx) // adjust to this package's real transaction-start call
	if err != nil {
		return domain.Scheme{}, translate(err, "scheme upsert")
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx) // adjust to this package's real with-tx call

	row, err := qtx.UpsertScheme(ctx, db.UpsertSchemeParams{
		ID: s.ID, Code: s.Code, Authority: db.SchemeAuthority(s.Authority), Ministry: s.Ministry,
		OfficialUrl: s.OfficialURL, StateCode: s.StateCode, NameI18nKey: s.NameI18nKey,
		NameText: s.NameText, SummaryI18nKey: s.SummaryI18nKey, SummaryText: s.SummaryText,
		Active: true, SortOrder: s.SortOrder, CuratedBy: "admin", // curated_by should come from the caller's principal -- threaded in by the service layer, see Task 5
	})
	if err != nil {
		return domain.Scheme{}, translate(err, "scheme upsert")
	}

	if err := qtx.DeleteSchemeCriteria(ctx, row.ID); err != nil { // add this :exec query to schemes.sql (DELETE FROM scheme_criterion WHERE scheme_id = @scheme_id) if not already present -- Task 2 omitted it, add it there when implementing this step
		return domain.Scheme{}, translate(err, "scheme criteria replace")
	}
	for _, c := range criteria {
		if err := qtx.InsertSchemeCriterion(ctx, db.InsertSchemeCriterionParams{ // add this :exec insert query to schemes.sql alongside DeleteSchemeCriteria
			ID: ids.New(), SchemeID: row.ID, Type: db.SchemeCriterionType(c.Type),
			StringValues: c.StringValues, IntValue: c.IntValue, Negate: c.Negate,
		}); err != nil {
			return domain.Scheme{}, translate(err, "scheme criterion insert")
		}
	}

	if err := qtx.DeleteSchemeManualChecks(ctx, row.ID); err != nil { // add this :exec query too
		return domain.Scheme{}, translate(err, "scheme manual checks replace")
	}
	for _, m := range manualChecks {
		if err := qtx.InsertSchemeManualCheck(ctx, db.InsertSchemeManualCheckParams{ // add this :exec insert query too
			ID: ids.New(), SchemeID: row.ID, I18nKey: m.I18nKey, CheckText: m.CheckText,
		}); err != nil {
			return domain.Scheme{}, translate(err, "scheme manual check insert")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Scheme{}, translate(err, "scheme upsert commit")
	}
	return schemeFromRow(row), nil
}
```

Go back to Task 2's query file and add the four missing queries this step references (`DeleteSchemeCriteria`, `InsertSchemeCriterion`, `DeleteSchemeManualChecks`, `InsertSchemeManualCheck`) before running `sqlc generate` again:

```sql
-- add to migrations/queries/schemes.sql

-- name: DeleteSchemeCriteria :exec
DELETE FROM scheme_criterion WHERE scheme_id = @scheme_id;

-- name: InsertSchemeCriterion :exec
INSERT INTO scheme_criterion (id, scheme_id, type, string_values, int_value, negate)
VALUES (@id, @scheme_id, @type, @string_values, sqlc.narg('int_value'), @negate);

-- name: DeleteSchemeManualChecks :exec
DELETE FROM scheme_manual_check WHERE scheme_id = @scheme_id;

-- name: InsertSchemeManualCheck :exec
INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, check_text)
VALUES (@id, @scheme_id, sqlc.narg('i18n_key'), sqlc.narg('check_text'));
```

Re-run `sqlc generate` after adding these before continuing this task.

Remove the now-dead `errNotImplemented` placeholder and its declaration from the file (it existed only between Step 3 and Step 3b of this task) — the final file contains only the real `UpsertScheme` body above, not both versions.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/core-svc && go test ./internal/core/repo/... -run TestRepo_ListActiveSchemes -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add migrations/queries/schemes.sql services/core-svc/internal/core/repo/db/schemes.sql.gen.go services/core-svc/internal/core/repo/scheme.go services/core-svc/internal/core/repo/scheme_test.go
git commit -m "feat(core-svc): add scheme repository

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 5: Service layer — `service/scheme.go`

**Files:**
- Create: `services/core-svc/internal/core/service/scheme.go`
- Test: `services/core-svc/internal/core/service/scheme_test.go`

**Interfaces:**
- Consumes: `domain.*` (Task 3), a `SchemeStore` interface satisfied by `*repo.Repo` (Task 4), `auth.RequirePrincipal`, `auth.RequireRole`, `auth.RoleMinistry`.
- Produces: `Schemes` struct, `NewSchemes(store SchemeStore, log *slog.Logger) *Schemes`, methods:
  - `ListSchemes(ctx) ([]domain.Scheme, error)` — public, no auth.
  - `MatchSchemes(ctx) ([]domain.SchemeMatch, error)` — authed; resolves the artisan id from the principal itself (never takes an artisan id parameter — this always matches the caller against their own record, so there is no authorization check to get wrong).
  - `UpsertScheme(ctx, s domain.Scheme, criteria []domain.SchemeCriterion, manualChecks []domain.SchemeManualCheck) (domain.Scheme, error)` — MINISTRY only; sets `s.ID` to a fresh uuid when empty (create) or keeps the given id (update).
  - `DeleteScheme(ctx, id uuid.UUID) (bool, error)` — MINISTRY only.

- [ ] **Step 1: Write the failing test**

```go
// services/core-svc/internal/core/service/scheme_test.go
package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

type fakeSchemeStore struct {
	schemes      []domain.Scheme
	criteria     map[uuid.UUID][]domain.SchemeCriterion
	manualChecks map[uuid.UUID][]domain.SchemeManualCheck
	facts        domain.ArtisanMatchFacts
}

func (f *fakeSchemeStore) ListActiveSchemes(ctx context.Context) ([]domain.Scheme, error) { return f.schemes, nil }
func (f *fakeSchemeStore) GetAllActiveSchemeCriteria(ctx context.Context) (map[uuid.UUID][]domain.SchemeCriterion, error) {
	return f.criteria, nil
}
func (f *fakeSchemeStore) GetAllActiveSchemeManualChecks(ctx context.Context) (map[uuid.UUID][]domain.SchemeManualCheck, error) {
	return f.manualChecks, nil
}
func (f *fakeSchemeStore) GetArtisanMatchFacts(ctx context.Context, artisanID uuid.UUID) (domain.ArtisanMatchFacts, error) {
	return f.facts, nil
}
func (f *fakeSchemeStore) GetSchemeByID(ctx context.Context, id uuid.UUID) (domain.Scheme, error) { return domain.Scheme{}, nil }
func (f *fakeSchemeStore) UpsertScheme(ctx context.Context, s domain.Scheme, c []domain.SchemeCriterion, m []domain.SchemeManualCheck) (domain.Scheme, error) {
	return s, nil
}
func (f *fakeSchemeStore) DeleteScheme(ctx context.Context, id uuid.UUID) (bool, error) { return true, nil }

func TestSchemes_MatchSchemes_ReturnsOneMatchPerActiveScheme(t *testing.T) {
	schemeID := uuid.New()
	store := &fakeSchemeStore{
		schemes: []domain.Scheme{{ID: schemeID, Code: "test_scheme"}},
	}
	svc := NewSchemes(store, nil)
	artisanID := uuid.New()
	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: artisanID.String(), Role: auth.RoleArtisan}) // adjust to this repo's real test-principal helper, matching badge_test.go's Task 6 note

	matches, err := svc.MatchSchemes(ctx)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, domain.MayQualify, matches[0].Status) // no criteria, no manual checks -> vacuously MayQualify
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/core-svc && go test ./internal/core/service/... -run TestSchemes_MatchSchemes -v`
Expected: FAIL (compile error).

- [ ] **Step 3: Write the implementation**

```go
// services/core-svc/internal/core/service/scheme.go

package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// SchemeStore is the persistence surface the scheme service depends on.
type SchemeStore interface {
	ListActiveSchemes(ctx context.Context) ([]domain.Scheme, error)
	GetAllActiveSchemeCriteria(ctx context.Context) (map[uuid.UUID][]domain.SchemeCriterion, error)
	GetAllActiveSchemeManualChecks(ctx context.Context) (map[uuid.UUID][]domain.SchemeManualCheck, error)
	GetArtisanMatchFacts(ctx context.Context, artisanID uuid.UUID) (domain.ArtisanMatchFacts, error)
	GetSchemeByID(ctx context.Context, id uuid.UUID) (domain.Scheme, error)
	UpsertScheme(ctx context.Context, s domain.Scheme, criteria []domain.SchemeCriterion, manualChecks []domain.SchemeManualCheck) (domain.Scheme, error)
	DeleteScheme(ctx context.Context, id uuid.UUID) (bool, error)
}

// Schemes is the service for government scheme reference data and matching.
type Schemes struct {
	store SchemeStore
	log   *slog.Logger
}

// NewSchemes builds the scheme service.
func NewSchemes(store SchemeStore, log *slog.Logger) *Schemes {
	if log == nil {
		log = slog.Default()
	}
	return &Schemes{store: store, log: log}
}

// ListSchemes is public reference data -- no principal required.
func (s *Schemes) ListSchemes(ctx context.Context) ([]domain.Scheme, error) {
	return s.store.ListActiveSchemes(ctx)
}

// MatchSchemes matches the CALLER against every active scheme. There is no
// artisan-id parameter and therefore no authorization check to get wrong --
// this method can only ever be asked about the caller's own record.
func (s *Schemes) MatchSchemes(ctx context.Context) ([]domain.SchemeMatch, error) {
	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	artisanID, err := uuid.Parse(principal.Subject)
	if err != nil {
		return nil, err
	}

	schemes, err := s.store.ListActiveSchemes(ctx)
	if err != nil {
		return nil, err
	}
	criteriaByScheme, err := s.store.GetAllActiveSchemeCriteria(ctx)
	if err != nil {
		return nil, err
	}
	manualChecksByScheme, err := s.store.GetAllActiveSchemeManualChecks(ctx)
	if err != nil {
		return nil, err
	}
	facts, err := s.store.GetArtisanMatchFacts(ctx, artisanID)
	if err != nil {
		return nil, err
	}

	matches := make([]domain.SchemeMatch, len(schemes))
	for i, scheme := range schemes {
		matches[i] = domain.MatchScheme(scheme, criteriaByScheme[scheme.ID], manualChecksByScheme[scheme.ID], facts)
	}
	return matches, nil
}

// UpsertScheme creates or updates a scheme. MINISTRY only.
func (s *Schemes) UpsertScheme(ctx context.Context, scheme domain.Scheme, criteria []domain.SchemeCriterion, manualChecks []domain.SchemeManualCheck) (domain.Scheme, error) {
	principal, err := auth.RequireRole(ctx, auth.RoleMinistry)
	if err != nil {
		return domain.Scheme{}, err
	}
	if scheme.ID == uuid.Nil {
		scheme.ID = uuid.New()
	}
	_ = principal // curated_by threading: see the note in Task 4 Step 3b -- if UpsertScheme's store signature is extended to take curatedBy, pass principal.Subject through here.
	return s.store.UpsertScheme(ctx, scheme, criteria, manualChecks)
}

// DeleteScheme removes a scheme. MINISTRY only.
func (s *Schemes) DeleteScheme(ctx context.Context, id uuid.UUID) (bool, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return false, err
	}
	return s.store.DeleteScheme(ctx, id)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/core-svc && go test ./internal/core/service/... -run TestSchemes_MatchSchemes -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/core-svc/internal/core/service/scheme.go services/core-svc/internal/core/service/scheme_test.go
git commit -m "feat(core-svc): add scheme service with self-only matching

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 6: Proto — `proto/schemes/v1/schemes.proto`

**Files:**
- Create: `proto/schemes/v1/schemes.proto`

- [ ] **Step 1: Write the proto file**

```protobuf
// proto/schemes/v1/schemes.proto

syntax = "proto3";

package schemes.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/ZoroNewbie00/kalakriti/pkg/pb/schemes/v1;schemesv1";

// SchemeService surfaces government scheme reference data and matches the
// calling artisan against it. MatchSchemes is guidance only -- see
// MatchStatus's comment.
service SchemeService {
  rpc ListSchemes(ListSchemesRequest) returns (ListSchemesResponse);
  // Matches the CALLER (never an arbitrary artisan id) against every active
  // scheme.
  rpc MatchSchemes(MatchSchemesRequest) returns (MatchSchemesResponse);
  rpc UpsertScheme(UpsertSchemeRequest) returns (UpsertSchemeResponse);
  rpc DeleteScheme(DeleteSchemeRequest) returns (DeleteSchemeResponse);
}

// MatchStatus is guidance, never a determination. There is deliberately no
// "eligible" value.
enum MatchStatus {
  MATCH_STATUS_UNSPECIFIED = 0;
  MATCH_STATUS_MAY_QUALIFY = 1;
  MATCH_STATUS_CHECK_REQUIRED = 2;
  MATCH_STATUS_UNLIKELY = 3;
}

enum SchemeAuthority {
  SCHEME_AUTHORITY_UNSPECIFIED = 0;
  SCHEME_AUTHORITY_CENTRAL = 1;
  SCHEME_AUTHORITY_STATE = 2;
}

enum SchemeCriterionType {
  SCHEME_CRITERION_TYPE_UNSPECIFIED = 0;
  SCHEME_CRITERION_TYPE_SOCIAL_CATEGORY = 1;
  SCHEME_CRITERION_TYPE_STATE_CODE = 2;
  SCHEME_CRITERION_TYPE_CRAFT_ID = 3;
  SCHEME_CRITERION_TYPE_MIN_YEARS_EXPERIENCE = 4;
  SCHEME_CRITERION_TYPE_HAS_PEHCHAN_ID = 5;
  SCHEME_CRITERION_TYPE_HAS_PM_VISHWAKARMA_ID = 6;
  SCHEME_CRITERION_TYPE_CLUSTER_MEMBER = 7;
  SCHEME_CRITERION_TYPE_SHG_MEMBER = 8;
}

message GovernmentScheme {
  string id = 1;
  string code = 2;
  SchemeAuthority authority = 3;
  string ministry = 4;
  string official_url = 5;
  optional string state_code = 6;
  optional string name_i18n_key = 7;
  optional string name_text = 8;
  optional string summary_i18n_key = 9;
  optional string summary_text = 10;
  int32 sort_order = 11;
}

message SchemeCriterionProto {
  SchemeCriterionType type = 1;
  repeated string string_values = 2;
  optional int64 int_value = 3;
  bool negate = 4;
}

message SchemeManualCheckProto {
  optional string i18n_key = 1;
  optional string check_text = 2;
}

message SchemeMatch {
  GovernmentScheme scheme = 1;
  MatchStatus status = 2;
  repeated string matched_criteria_labels = 3;
  repeated string unmet_criteria_labels = 4;
  repeated SchemeManualCheckProto manual_checks = 5;
}

message ListSchemesRequest {}
message ListSchemesResponse { repeated GovernmentScheme schemes = 1; }

message MatchSchemesRequest {}
message MatchSchemesResponse { repeated SchemeMatch matches = 1; }

message UpsertSchemeRequest {
  optional string id = 1; // empty/unset = create
  string code = 2;
  SchemeAuthority authority = 3;
  string ministry = 4;
  string official_url = 5;
  optional string state_code = 6;
  optional string name_i18n_key = 7;
  optional string name_text = 8;
  optional string summary_i18n_key = 9;
  optional string summary_text = 10;
  int32 sort_order = 11;
  repeated SchemeCriterionProto criteria = 12;
  repeated SchemeManualCheckProto manual_checks = 13;
}
message UpsertSchemeResponse { GovernmentScheme scheme = 1; }

message DeleteSchemeRequest { string id = 1; }
message DeleteSchemeResponse { bool deleted = 1; }
```

- [ ] **Step 2: Generate**

Run: `make proto`
Expected: `pkg/pb/schemes/v1/` populated, no errors.

- [ ] **Step 3: Commit**

```bash
git add proto/schemes/v1/schemes.proto
git commit -m "feat(proto): add schemes.v1.SchemeService

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 7: gRPC handler — `handler/scheme.go`

**Files:**
- Create: `services/core-svc/internal/core/handler/scheme.go`

Follow Task 8 of the badges plan (`handler/badge.go`) line for line as the template — same shape: `Schemes` struct embedding `schemesv1.UnimplementedSchemeServiceServer`, `NewSchemes(svc *service.Schemes) *Schemes`, one method per RPC translating between proto and domain types via small `toProto*`/`fromProto*` helper functions.

- [ ] **Step 1: Write the handler**

```go
// services/core-svc/internal/core/handler/scheme.go

package handler

import (
	"context"

	"github.com/google/uuid"

	schemesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/schemes/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Schemes implements schemes.v1.SchemeService.
type Schemes struct {
	schemesv1.UnimplementedSchemeServiceServer
	svc *service.Schemes
}

func NewSchemes(svc *service.Schemes) *Schemes { return &Schemes{svc: svc} }

func (h *Schemes) ListSchemes(ctx context.Context, req *schemesv1.ListSchemesRequest) (*schemesv1.ListSchemesResponse, error) {
	schemes, err := h.svc.ListSchemes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*schemesv1.GovernmentScheme, len(schemes))
	for i, s := range schemes {
		out[i] = toProtoScheme(s)
	}
	return &schemesv1.ListSchemesResponse{Schemes: out}, nil
}

func (h *Schemes) MatchSchemes(ctx context.Context, req *schemesv1.MatchSchemesRequest) (*schemesv1.MatchSchemesResponse, error) {
	matches, err := h.svc.MatchSchemes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*schemesv1.SchemeMatch, len(matches))
	for i, m := range matches {
		checks := make([]*schemesv1.SchemeManualCheckProto, len(m.ManualChecks))
		for j, c := range m.ManualChecks {
			checks[j] = &schemesv1.SchemeManualCheckProto{I18nKey: c.I18nKey, CheckText: c.CheckText}
		}
		out[i] = &schemesv1.SchemeMatch{
			Scheme: toProtoScheme(m.Scheme), Status: toProtoStatus(m.Status),
			MatchedCriteriaLabels: m.MatchedCriteriaLabels, UnmetCriteriaLabels: m.UnmetCriteriaLabels,
			ManualChecks: checks,
		}
	}
	return &schemesv1.MatchSchemesResponse{Matches: out}, nil
}

func (h *Schemes) UpsertScheme(ctx context.Context, req *schemesv1.UpsertSchemeRequest) (*schemesv1.UpsertSchemeResponse, error) {
	var id uuid.UUID
	if req.Id != nil && *req.Id != "" {
		parsed, err := uuid.Parse(*req.Id)
		if err != nil {
			return nil, err
		}
		id = parsed
	}
	criteria := make([]domain.SchemeCriterion, len(req.Criteria))
	for i, c := range req.Criteria {
		criteria[i] = domain.SchemeCriterion{
			Type: fromProtoCriterionType(c.Type), StringValues: c.StringValues, IntValue: c.IntValue, Negate: c.Negate,
		}
	}
	manualChecks := make([]domain.SchemeManualCheck, len(req.ManualChecks))
	for i, m := range req.ManualChecks {
		manualChecks[i] = domain.SchemeManualCheck{I18nKey: m.I18nKey, CheckText: m.CheckText}
	}
	scheme, err := h.svc.UpsertScheme(ctx, domain.Scheme{
		ID: id, Code: req.Code, Authority: fromProtoAuthority(req.Authority), Ministry: req.Ministry,
		OfficialURL: req.OfficialUrl, StateCode: req.StateCode, NameI18nKey: req.NameI18nKey,
		NameText: req.NameText, SummaryI18nKey: req.SummaryI18nKey, SummaryText: req.SummaryText,
		SortOrder: req.SortOrder,
	}, criteria, manualChecks)
	if err != nil {
		return nil, err
	}
	return &schemesv1.UpsertSchemeResponse{Scheme: toProtoScheme(scheme)}, nil
}

func (h *Schemes) DeleteScheme(ctx context.Context, req *schemesv1.DeleteSchemeRequest) (*schemesv1.DeleteSchemeResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}
	deleted, err := h.svc.DeleteScheme(ctx, id)
	if err != nil {
		return nil, err
	}
	return &schemesv1.DeleteSchemeResponse{Deleted: deleted}, nil
}

func toProtoScheme(s domain.Scheme) *schemesv1.GovernmentScheme {
	authority := schemesv1.SchemeAuthority_SCHEME_AUTHORITY_CENTRAL
	if s.Authority == "STATE" {
		authority = schemesv1.SchemeAuthority_SCHEME_AUTHORITY_STATE
	}
	return &schemesv1.GovernmentScheme{
		Id: s.ID.String(), Code: s.Code, Authority: authority, Ministry: s.Ministry, OfficialUrl: s.OfficialURL,
		StateCode: s.StateCode, NameI18nKey: s.NameI18nKey, NameText: s.NameText,
		SummaryI18nKey: s.SummaryI18nKey, SummaryText: s.SummaryText, SortOrder: s.SortOrder,
	}
}

func toProtoStatus(s domain.MatchStatus) schemesv1.MatchStatus {
	switch s {
	case domain.MayQualify:
		return schemesv1.MatchStatus_MATCH_STATUS_MAY_QUALIFY
	case domain.CheckRequired:
		return schemesv1.MatchStatus_MATCH_STATUS_CHECK_REQUIRED
	case domain.Unlikely:
		return schemesv1.MatchStatus_MATCH_STATUS_UNLIKELY
	default:
		return schemesv1.MatchStatus_MATCH_STATUS_UNSPECIFIED
	}
}

func fromProtoAuthority(a schemesv1.SchemeAuthority) string {
	if a == schemesv1.SchemeAuthority_SCHEME_AUTHORITY_STATE {
		return "STATE"
	}
	return "CENTRAL"
}

func fromProtoCriterionType(t schemesv1.SchemeCriterionType) domain.SchemeCriterionType {
	switch t {
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_SOCIAL_CATEGORY:
		return domain.CriterionSocialCategory
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_STATE_CODE:
		return domain.CriterionStateCode
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_CRAFT_ID:
		return domain.CriterionCraftID
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_MIN_YEARS_EXPERIENCE:
		return domain.CriterionMinYearsExperience
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_HAS_PEHCHAN_ID:
		return domain.CriterionHasPehchanID
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_HAS_PM_VISHWAKARMA_ID:
		return domain.CriterionHasPMVishwakarmaID
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_CLUSTER_MEMBER:
		return domain.CriterionClusterMember
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_SHG_MEMBER:
		return domain.CriterionSHGMember
	default:
		return ""
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd services/core-svc && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add services/core-svc/internal/core/handler/scheme.go
git commit -m "feat(core-svc): add scheme gRPC handler

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 8: Wire into `main.go` and `PublicMethods()`

**Files:**
- Modify: `services/core-svc/cmd/core-svc/main.go`
- Modify: `services/core-svc/internal/core/handler/identity.go`

- [ ] **Step 1: Register the service/handler**

Mirror Task 10 Step 1 of the badges plan exactly:

```go
schemesSvc := service.NewSchemes(repository, log)
schemesHandler := handler.NewSchemes(schemesSvc)
```
```go
schemesv1.RegisterSchemeServiceServer(grpcServer, schemesHandler)
```
Add the `schemesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/schemes/v1"` import.

No Kafka consumers for this feature — `MatchSchemes` computes live, on request, from data already committed by other flows.

- [ ] **Step 2: Public-read wiring for `ListSchemes` only**

In `PublicMethods()`, add:
```go
		"/schemes.v1.SchemeService/ListSchemes",
```
`MatchSchemes` stays authed (it is confirmed in Task 5 to call `auth.RequirePrincipal`) — do **not** add it here; it reads the caller's own PII-adjacent fields (social category, experience, IDs) and must never be reachable anonymously.

- [ ] **Step 3: Build and smoke-test**

Run: `cd services/core-svc && go build ./...`
Run: `grpcurl -plaintext localhost:<port> schemes.v1.SchemeService/ListSchemes` → 8 schemes, no auth error.
Run: `grpcurl -plaintext localhost:<port> schemes.v1.SchemeService/MatchSchemes` with no token → expect an auth error, confirming it is NOT public.

- [ ] **Step 4: Commit**

```bash
git add services/core-svc/cmd/core-svc/main.go services/core-svc/internal/core/handler/identity.go
git commit -m "feat(core-svc): wire scheme service into main.go

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 9: BFF client, routes, handlers

**Files:**
- Create: `services/bff/internal/bff/client/scheme.go`
- Modify: `services/bff/internal/bff/server.go`
- Modify: `services/bff/internal/bff/handler/api.go`
- Modify: `services/bff/cmd/bff/main.go`

Follow Tasks 11-12 of the badges plan exactly as the template: `client.Schemes` struct wrapping `schemesv1.SchemeServiceClient`, methods returning `map[string]any`/`[]map[string]any`; routes `GET /schemes` (public), `GET /schemes/match` (authed), `POST /schemes` + `PATCH /schemes/{id}` + `DELETE /schemes/{id}` (authed, MINISTRY, `POST`/`PATCH` wrapped in `withIdempotency`); handler methods in `api.go` following the exact `ListTrendLinks`/`CreateTrendLink` shape.

- [ ] **Step 1: Write `client/scheme.go`**

```go
// services/bff/internal/bff/client/scheme.go
package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	schemesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/schemes/v1"
)

type Schemes struct {
	schemes schemesv1.SchemeServiceClient
}

func NewSchemes(conn grpc.ClientConnInterface) *Schemes {
	return &Schemes{schemes: schemesv1.NewSchemeServiceClient(conn)}
}

func (s *Schemes) ListSchemes(ctx context.Context) ([]map[string]any, error) {
	resp, err := s.schemes.ListSchemes(ctx, &schemesv1.ListSchemesRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(resp.Schemes))
	for i, scheme := range resp.Schemes {
		out[i] = schemeToMap(scheme)
	}
	return out, nil
}

func (s *Schemes) MatchSchemes(ctx context.Context) ([]map[string]any, error) {
	resp, err := s.schemes.MatchSchemes(ctx, &schemesv1.MatchSchemesRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(resp.Matches))
	for i, m := range resp.Matches {
		checks := make([]map[string]any, len(m.ManualChecks))
		for j, c := range m.ManualChecks {
			checks[j] = map[string]any{"i18n_key": c.I18nKey, "check_text": c.CheckText}
		}
		out[i] = map[string]any{
			"scheme": schemeToMap(m.Scheme), "status": m.Status.String(),
			"matched_criteria_labels": m.MatchedCriteriaLabels, "unmet_criteria_labels": m.UnmetCriteriaLabels,
			"manual_checks": checks,
		}
	}
	return out, nil
}

func (s *Schemes) UpsertScheme(ctx context.Context, fields map[string]any) (map[string]any, error) {
	code, _ := fields["code"].(string)
	if code == "" {
		return nil, domain.InvalidInput("code: is required")
	}
	ministry, _ := fields["ministry"].(string)
	officialURL, _ := fields["official_url"].(string)
	if officialURL == "" {
		return nil, domain.InvalidInput("official_url: is required")
	}
	req := &schemesv1.UpsertSchemeRequest{Code: code, Ministry: ministry, OfficialUrl: officialURL}
	if id, ok := fields["id"].(string); ok && id != "" {
		req.Id = &id
	}
	// name_text/summary_text/state_code/sort_order/criteria/manual_checks
	// extraction from fields follows the same pattern as CreateTrendLink in
	// client/trends.go -- copy that pattern for the remaining optional
	// fields when implementing, including criteria/manual_checks as
	// []map[string]any decoded into the proto's repeated message fields.
	resp, err := s.schemes.UpsertScheme(ctx, req)
	if err != nil {
		return nil, err
	}
	return schemeToMap(resp.Scheme), nil
}

func (s *Schemes) DeleteScheme(ctx context.Context, id string) error {
	_, err := s.schemes.DeleteScheme(ctx, &schemesv1.DeleteSchemeRequest{Id: id})
	return err
}

func schemeToMap(s *schemesv1.GovernmentScheme) map[string]any {
	out := map[string]any{
		"id": s.Id, "code": s.Code, "authority": s.Authority.String(), "ministry": s.Ministry,
		"official_url": s.OfficialUrl, "sort_order": s.SortOrder,
	}
	if s.StateCode != nil {
		out["state_code"] = *s.StateCode
	}
	if s.NameI18nKey != nil {
		out["name_i18n_key"] = *s.NameI18nKey
	}
	if s.NameText != nil {
		out["name_text"] = *s.NameText
	}
	if s.SummaryI18nKey != nil {
		out["summary_i18n_key"] = *s.SummaryI18nKey
	}
	if s.SummaryText != nil {
		out["summary_text"] = *s.SummaryText
	}
	return out
}
```

- [ ] **Step 2: Add routes to `server.go`**

```go
api.GET("/schemes", httpx.WrapHandler(apiH.ListSchemes))
authed.GET("/schemes/match", httpx.WrapHandler(apiH.MatchSchemes))
authed.POST("/schemes", httpx.WrapHandler(withIdempotency(apiH.UpsertScheme, cfg.IdempStore)))
authed.PATCH("/schemes/:id", httpx.WrapHandler(withIdempotency(apiH.UpsertScheme, cfg.IdempStore)))
authed.DELETE("/schemes/:id", httpx.WrapHandler(apiH.DeleteScheme))
```
Add `SchemeSvc handler.SchemeService` to `Config` and thread it through the constructor, same as `BadgeSvc` in the badges plan's Task 12.

- [ ] **Step 3: Add handlers to `api.go`**

```go
func (h *APIHandler) ListSchemes(w http.ResponseWriter, r *http.Request) {
	schemes, err := h.schemeSvc.ListSchemes(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"schemes": schemes})
}

func (h *APIHandler) MatchSchemes(w http.ResponseWriter, r *http.Request) {
	matches, err := h.schemeSvc.MatchSchemes(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"matches": matches})
}

func (h *APIHandler) UpsertScheme(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	if id := httpx.URLParam(r, "id"); id != "" {
		body["id"] = id
	}
	scheme, err := h.schemeSvc.UpsertScheme(r.Context(), body)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, scheme)
}

func (h *APIHandler) DeleteScheme(w http.ResponseWriter, r *http.Request) {
	id := httpx.URLParam(r, "id")
	if err := h.schemeSvc.DeleteScheme(r.Context(), id); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "deleted"})
}
```

- [ ] **Step 4: Wire in `bff/cmd/bff/main.go`**

```go
schemeSvc := client.NewSchemes(coreConn)
```
Pass `SchemeSvc: schemeSvc` into the same config struct literal `BadgeSvc` was added to.

- [ ] **Step 5: Build and commit**

Run: `cd services/bff && go build ./...`

```bash
git add services/bff/internal/bff/client/scheme.go services/bff/internal/bff/server.go services/bff/internal/bff/handler/api.go services/bff/cmd/bff/main.go
git commit -m "feat(bff): add scheme routes and handlers

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 10: OpenAPI + generated TS client

**Files:**
- Modify: `services/bff/openapi.json`, `web/packages/api/src/generated/schema.d.ts`, `web/packages/api/src/operations.ts`, `web/packages/api/src/index.ts`

Follow Task 13 of the badges plan exactly. Add path objects for `GET /schemes`, `GET /schemes/match`, `POST /schemes`, `PATCH /schemes/{id}`, `DELETE /schemes/{id}`; component schemas `GovernmentScheme`, `SchemeMatch`, `SchemeManualCheck`.

- [ ] **Step 1-4:** Mirror badges Task 13 Steps 1-4, producing `operations.ts` functions `listSchemes()`, `matchSchemes()`, `upsertScheme(body)`, `deleteScheme(id)`.

```typescript
// --- Government Scheme Operations ---

export type GovernmentScheme = Json<paths['/schemes']['get']['responses'][200]>['schemes'][number];
export type SchemeMatchesResponse = Json<paths['/schemes/match']['get']['responses'][200]>;
export type UpsertSchemeBody = Json<paths['/schemes']['post']['requestBody']>;

export function listSchemes(options?: CallOptions): Promise<{ schemes: GovernmentScheme[] }> {
  return call('/schemes', { ...options, method: 'GET' }) as Promise<{ schemes: GovernmentScheme[] }>;
}

export function matchSchemes(options?: CallOptions): Promise<SchemeMatchesResponse> {
  return call('/schemes/match', { ...options, method: 'GET' }) as Promise<SchemeMatchesResponse>;
}

export function upsertScheme(body: UpsertSchemeBody, id?: string, options?: CallOptions): Promise<GovernmentScheme> {
  const path = id ? `/schemes/${encodeURIComponent(id)}` : '/schemes';
  return call(path, { ...options, method: id ? 'PATCH' : 'POST', body }) as Promise<GovernmentScheme>;
}

export function deleteScheme(id: string, options?: CallOptions): Promise<{ status?: string }> {
  return call(`/schemes/${encodeURIComponent(id)}`, { ...options, method: 'DELETE' }) as Promise<{ status?: string }>;
}
```

- [ ] **Step 5: Commit**

```bash
git add services/bff/openapi.json web/packages/api/src/generated/schema.d.ts web/packages/api/src/operations.ts web/packages/api/src/index.ts
git commit -m "feat(api): add generated scheme client operations

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 11: i18n — scheme names, summaries, checklist items, UI chrome

**Files:**
- Modify: `en.ts` + all 20 non-English catalogues.

**Interfaces:**
- Produces: 8 × 2 (`.name`/`.summary`) = 16 keys for the seeded schemes, plus 9 manual-check keys (one per seeded `scheme_manual_check.i18n_key`), plus UI chrome keys: `nav.schemes`, `schemes.title`, `schemes.subtitle`, `schemes.empty`, `schemes.status.mayQualify`, `schemes.status.checkRequired`, `schemes.status.unlikely`, `schemes.confirmOnPortal`, `schemes.officialLink`, `schemes.matchedOn`, `schemes.manualChecklist`, `schemes.readAloud`, plus 8 `scheme.criterion.*` labels (from Task 3's `criterionLabelSuffix`), plus `registration.socialCategory.*` (label + 6 option labels for the registration step) — roughly 50 keys total.

- [ ] **Step 1: Add keys to `en.ts`**

```typescript
  'nav.schemes': 'Government Schemes',
  'schemes.title': 'Government Schemes',
  'schemes.subtitle': 'Schemes you may qualify for, based on what you have told us. Always confirm on the official portal before applying.',
  'schemes.empty': 'No schemes to show right now.',
  'schemes.status.mayQualify': 'You may qualify',
  'schemes.status.checkRequired': 'Check a few more things',
  'schemes.status.unlikely': 'Likely not a match',
  'schemes.confirmOnPortal': 'Always confirm on the official government portal before applying.',
  'schemes.officialLink': 'Visit official portal',
  'schemes.matchedOn': 'Matched on: {criteria}',
  'schemes.manualChecklist': 'Please confirm for yourself',
  'schemes.readAloud': 'Read this scheme aloud',
  'scheme.criterion.social_category': 'your social category',
  'scheme.criterion.state_code': 'your state',
  'scheme.criterion.craft': 'your craft',
  'scheme.criterion.min_years_experience': 'years of experience',
  'scheme.criterion.has_pehchan_id': 'having a Pehchan ID',
  'scheme.criterion.has_pm_vishwakarma_id': 'having a PM Vishwakarma ID',
  'scheme.criterion.cluster_member': 'cluster membership',
  'scheme.criterion.shg_member': 'SHG membership',
  'registration.socialCategory.label': 'Social category (optional)',
  'registration.socialCategory.general': 'General',
  'registration.socialCategory.obc': 'OBC',
  'registration.socialCategory.sc': 'SC',
  'registration.socialCategory.st': 'ST',
  'registration.socialCategory.ews': 'EWS',
  'registration.socialCategory.preferNotToSay': 'Prefer not to say',
  'scheme.pm_vishwakarma.name': 'PM Vishwakarma',
  'scheme.pm_vishwakarma.summary': 'Support for traditional artisans and craftspeople, including a Vishwakarma certificate, skill training, and collateral-free loans.',
  'scheme.pm_vishwakarma.check.traditional_trade': 'You practise one of the 18 traditional trades covered by this scheme.',
  'scheme.handicrafts_pehchan_id.name': 'Handicrafts Pehchan ID',
  'scheme.handicrafts_pehchan_id.summary': 'An official artisan identity card from the Ministry of Textiles, used to access other handicraft welfare schemes.',
  'scheme.ahvy.name': 'Ambedkar Hastshilp Vikas Yojana',
  'scheme.ahvy.summary': 'Cluster-based development support for handicraft artisans, including design and technical assistance.',
  'scheme.ahvy.check.registered_artisan': 'You are a registered handicrafts artisan in your cluster.',
  'scheme.nhdp.name': 'National Handicrafts Development Programme',
  'scheme.nhdp.summary': 'Infrastructure, marketing, and welfare support for handicraft artisans, individually or through a group.',
  'scheme.nhdp.check.group_or_individual': 'Check whether this component applies to individual artisans or only to registered groups in your area.',
  'scheme.sfurti.name': 'SFURTI',
  'scheme.sfurti.summary': 'Cluster-based support to make traditional industries more productive and competitive.',
  'scheme.sfurti.check.cluster_based': 'This scheme supports registered clusters, not individual artisans directly — check if your cluster is enrolled.',
  'scheme.mudra.name': 'MUDRA Loan',
  'scheme.mudra.summary': 'Collateral-free loans up to a set limit for small, non-farm income-generating businesses.',
  'scheme.mudra.check.business_plan': 'You have a simple business plan or purpose for the loan ready to show.',
  'scheme.mudra.check.no_existing_default': 'You have no existing loan default on record.',
  'scheme.stand_up_india.name': 'Stand-Up India',
  'scheme.stand_up_india.summary': 'Bank loans for setting up a new enterprise, for SC/ST and women entrepreneurs.',
  'scheme.stand_up_india.check.first_time_entrepreneur': 'This is your first enterprise of this kind.',
  'scheme.stand_up_india.check.women_entrepreneur_alternative': 'If you are a woman entrepreneur, you may qualify even if the social-category match above did not apply — check the official portal.',
  'scheme.odop.name': 'One District One Product',
  'scheme.odop.summary': 'Promotion and market linkage support for a district\'s designated signature product.',
  'scheme.odop.check.district_product_match': 'Your craft matches your district\'s designated ODOP product.',
```

- [ ] **Step 2-5:** Mirror badges Task 15 Steps 2-5 exactly: run `npm run audit`, populate all 20 catalogues with real translations (not English placeholders), re-run the audit, run `catalogue-audit.test.ts`, commit.

```bash
git add web/packages/i18n/src/messages/*.ts
git commit -m "feat(i18n): add government scheme guidance text for all 21 locales

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 12: Registration step for `social_category`

**Files:**
- Modify: `web/apps/artisan/src/routes/register/*` (the wizard step sequence — read the existing steps, e.g. `district`, `pehchan`, per Batch 7's spec, to find the exact file/component pattern to copy)

**Interfaces:**
- Consumes: the `registration.socialCategory.*` keys (Task 11), the existing registration Dexie-draft-autosave pattern already used by every other step.

- [ ] **Step 1: Add one new wizard step**

Read the existing optional step (Batch 7 names "PM Vishwakarma / Pehchan ID, with a clear skip") to copy its exact optional-field pattern (voice prompt, skip affordance that reads as a real choice, Dexie autosave, Stepper position). Add a new step immediately after it: six tiles (`GENERAL`, `OBC`, `SC`, `ST`, `EWS`, `PREFER_NOT_TO_SAY`), each spoken aloud via the existing voice layer, `PREFER_NOT_TO_SAY` rendered as an equally-weighted tile (not a smaller "skip" link) so declining reads as a real, respected choice rather than an incomplete answer.

- [ ] **Step 2: Submit `social_category` with the registration payload**

Find `buildRegisterBody` in `web/apps/artisan/src/lib/registration.ts` (referenced in `CLAUDE.md`'s "Artisan registration's real request contract" section) and add `social_category` to the assembled body — but only if the backend's `POST /artisans` actually accepts it. **Check first**: read `services/bff/internal/bff/client/artisan.go`'s `Register()` validation (per CLAUDE.md's own warning to check that file, not the checked-in `openapi.json`, for the real required/accepted fields). If `Register()` does not currently accept `social_category`, this task also needs a small addition there and in `services/core-svc`'s artisan-creation path — treat that as a discovered sub-task, not a blocker: add the field through the same layers Task 2 of this plan already touched for the artisan **update** path (the registration **create** path is a different call site and needs its own equivalent one-line addition).

- [ ] **Step 3: Manual verification**

Run the registration flow end-to-end in the artisan app at 360px width, confirm the new step is spoken, confirm `PREFER_NOT_TO_SAY` is selectable and not visually demoted, confirm the value persists through to the created artisan record.

- [ ] **Step 4: Commit**

```bash
git add web/apps/artisan/src/routes/register/ web/apps/artisan/src/lib/registration.ts
git commit -m "feat(artisan): add optional social category to registration

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 13: Artisan `/schemes` route

**Files:**
- Create: `web/apps/artisan/src/routes/schemes/+page.ts`, `+page.svelte`
- Modify: `web/apps/artisan/src/routes/profile/+page.svelte` (nav entry, same reasoning as the badges plan's Task 17 — link from Profile, not a fifth bottom-nav slot)
- Modify: `web/apps/artisan/src/routes/earnings/+page.svelte` (one added line noting the income statement can serve as proof of income for scheme applications, per the spec)

**Interfaces:**
- Consumes: `matchSchemes` from `@kalakriti/api` (Task 10).

- [ ] **Step 1: Write `+page.ts`**

```typescript
// web/apps/artisan/src/routes/schemes/+page.ts
import { matchSchemes } from '@kalakriti/api';

export const ssr = false;

export async function load() {
  const res = await matchSchemes();
  return { matches: res.matches };
}
```

- [ ] **Step 2: Write `+page.svelte`**

```svelte
<!-- web/apps/artisan/src/routes/schemes/+page.svelte -->
<script lang="ts">
  import { locale } from '@kalakriti/i18n';
  import { SectionHeader, EmptyState, SpeakButton } from '@kalakriti/ui';
  import type { PageData } from './$types';

  interface Props { data: PageData }
  let { data }: Props = $props();

  const t = $derived(locale.t);
  const mayQualify = $derived(data.matches.filter((m) => m.status === 'MATCH_STATUS_MAY_QUALIFY'));
  const checkRequired = $derived(data.matches.filter((m) => m.status === 'MATCH_STATUS_CHECK_REQUIRED'));
  const unlikely = $derived(data.matches.filter((m) => m.status === 'MATCH_STATUS_UNLIKELY'));

  function schemeName(scheme: { name_i18n_key?: string; name_text?: string }) {
    return scheme.name_i18n_key ? t(scheme.name_i18n_key) : (scheme.name_text ?? '');
  }
  function schemeSummary(scheme: { summary_i18n_key?: string; summary_text?: string }) {
    return scheme.summary_i18n_key ? t(scheme.summary_i18n_key) : (scheme.summary_text ?? '');
  }
</script>

<svelte:head><title>{t('schemes.title')}</title></svelte:head>

<main id="main-content">
  <SectionHeader heading={t('schemes.title')} kicker={t('schemes.subtitle')} />

  {#if mayQualify.length === 0 && checkRequired.length === 0}
    <EmptyState heading={t('schemes.empty')} />
  {/if}

  {#each [...mayQualify, ...checkRequired] as match (match.scheme.id)}
    <article class="scheme-card">
      <span class="scheme-card__status scheme-card__status--{match.status.toLowerCase()}">
        {match.status === 'MATCH_STATUS_MAY_QUALIFY' ? t('schemes.status.mayQualify') : t('schemes.status.checkRequired')}
      </span>
      <h3>{schemeName(match.scheme)}</h3>
      <p>{schemeSummary(match.scheme)}</p>
      {#if match.matched_criteria_labels.length > 0}
        <p class="scheme-card__matched">
          {t('schemes.matchedOn', { criteria: match.matched_criteria_labels.map((l) => t(l)).join(', ') })}
        </p>
      {/if}
      {#if match.manual_checks.length > 0}
        <div class="scheme-card__checklist">
          <h4>{t('schemes.manualChecklist')}</h4>
          <ul>
            {#each match.manual_checks as check}
              <li>
                <label>
                  <input type="checkbox" />
                  {check.i18n_key ? t(check.i18n_key) : check.check_text}
                </label>
              </li>
            {/each}
          </ul>
        </div>
      {/if}
      <p class="scheme-card__confirm">{t('schemes.confirmOnPortal')}</p>
      <a href={match.scheme.official_url} target="_blank" rel="noopener noreferrer">{t('schemes.officialLink')}</a>
      <SpeakButton text={`${schemeName(match.scheme)}. ${schemeSummary(match.scheme)}`} label={t('schemes.readAloud')} />
    </article>
  {/each}

  {#if unlikely.length > 0}
    <details>
      <summary>{t('schemes.status.unlikely')} ({unlikely.length})</summary>
      {#each unlikely as match (match.scheme.id)}
        <p>{schemeName(match.scheme)}</p>
      {/each}
    </details>
  {/if}
</main>

<style>
  .scheme-card {
    border: var(--k-hairline) solid var(--k-border-hairline);
    padding: var(--k-space-4);
    margin-block-end: var(--k-space-3);
  }
  .scheme-card__status {
    font-size: var(--k-text-sm);
    font-weight: 600;
  }
  .scheme-card__status--match_status_may_qualify { color: var(--k-success); }
  .scheme-card__status--match_status_check_required { color: var(--k-warning); }
  .scheme-card__confirm {
    font-size: var(--k-text-sm);
    color: var(--k-text-muted);
  }
</style>
```

Note: `unlikely` schemes are collapsed behind `<details>`, not hidden — the spec's acceptance criterion says "never hidden," and a native `<details>`/`<summary>` disclosure satisfies that while keeping the default view focused on what matters, and is itself keyboard- and screen-reader-accessible with no extra ARIA needed.

Check `SpeakButton`'s real prop shape (same caveat as the badges plan's Task 16) and `EmptyState`'s real prop shape before finalizing.

- [ ] **Step 3: Add the earnings-page cross-link**

In `web/apps/artisan/src/routes/earnings/+page.svelte`, near wherever the income statement PDF download action already renders, add one line using a new key `earnings.schemeProofNote` (add to `en.ts` + 20 locales, following Task 11's process): "This statement can also serve as proof of income for government scheme applications." with a link to `/schemes`.

- [ ] **Step 4: Add nav entry from `/profile`**

Same as badges Task 17 Step 3.

- [ ] **Step 5: Manual verification and commit**

Verify at 360px, verify read-aloud, verify the official link opens in a new tab and is visibly external, verify `PREFER_NOT_TO_SAY` social category still returns every scheme's status (none crash) per domain test 4.

```bash
git add web/apps/artisan/src/routes/schemes/ web/apps/artisan/src/routes/profile/+page.svelte web/apps/artisan/src/routes/earnings/+page.svelte web/packages/i18n/src/messages/*.ts
git commit -m "feat(artisan): add /schemes guidance route

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 14: Admin `/schemes` CRUD

**Files:**
- Create: `web/apps/admin/src/routes/schemes/+page.svelte`, `+page.ts`

**Interfaces:**
- Consumes: `listSchemes`, `upsertScheme`, `deleteScheme` (Task 10).

- [ ] **Step 1: Write the loader**

```typescript
// web/apps/admin/src/routes/schemes/+page.ts
import { listSchemes } from '@kalakriti/api';

export async function load() {
  const res = await listSchemes();
  return { schemes: res.schemes };
}
```

- [ ] **Step 2: Write the CRUD page**

Follow this admin app's existing list+form CRUD pattern (check `web/apps/admin/src/routes/companies/+page.svelte`, already in the working tree per this session's git status, for the exact list/edit/delete layout convention this app uses) rather than inventing new admin UI conventions. The form needs: `code`, `authority` (select CENTRAL/STATE), `ministry`, `official_url`, optional `state_code`, `name_text`, `summary_text` (admin-added schemes use free text, not i18n keys, per this plan's constraints), `sort_order`, and a repeatable criteria builder (type select from the 8 `SchemeCriterionType` values + string/int value inputs + a negate checkbox) and a repeatable manual-check list (plain text inputs). Read `companies/+page.svelte` first and match its exact component/styling choices before writing this file.

- [ ] **Step 3: Manual verification and commit**

As MINISTRY, create a new scheme with one criterion and one manual check, confirm it appears in `listSchemes()` and in an artisan's `/schemes` match results if their facts satisfy it. Edit it, confirm the update. Delete it, confirm removal.

```bash
git add web/apps/admin/src/routes/schemes/
git commit -m "feat(admin): add scheme catalog CRUD

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 15: End-to-end verification against acceptance criteria

- [ ] **Step 1: Full backend test suite**

Run: `cd services/core-svc && go test ./... && cd ../bff && go test ./...`

- [ ] **Step 2: Full frontend checks**

Run: `cd web && pnpm check && pnpm --filter @kalakriti/i18n test`

- [ ] **Step 3: Manual acceptance walkthrough**

1. Search every rendered string in the artisan `/schemes` page and the admin `/schemes` page for the literal word "eligible" — it must not appear anywhere as a status claim (the spec's primary acceptance criterion).
2. Set an artisan's `social_category` to `PREFER_NOT_TO_SAY`, call `MatchSchemes`, confirm every scheme still returns a status (no crash, no 500) and `stand_up_india` specifically shows `UNLIKELY` or `CHECK_REQUIRED` (never `MAY_QUALIFY` from the social-category criterion alone).
3. Click every seeded scheme's official link, confirm it resolves to a real, correct government domain.
4. As MINISTRY, add a new scheme through the admin panel with only `name_text` (no i18n key) — confirm it renders correctly on the artisan `/schemes` page without needing a code deploy or translation pass.
5. Switch locale to Hindi and Bengali, confirm every seeded scheme's name/summary/manual-check text and all UI chrome renders correctly with no English fallback visible.

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "test: verify government scheme guidance end-to-end

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```
