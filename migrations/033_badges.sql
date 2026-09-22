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
    CONSTRAINT badge_code_key UNIQUE (code, tier),
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
