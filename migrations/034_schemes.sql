-- migrations/034_schemes.sql
-- +goose Up

CREATE TYPE social_category AS ENUM ('GENERAL', 'OBC', 'SC', 'ST', 'EWS', 'PREFER_NOT_TO_SAY');

-- mv_artisans_by_category (023_insight.sql) reads artisan.social_category,
-- so Postgres refuses to retype the column while the view depends on it.
-- Drop and recreate around the type change.
DROP MATERIALIZED VIEW mv_artisans_by_category;
ALTER TABLE artisan ALTER COLUMN social_category TYPE social_category USING social_category::social_category;
CREATE MATERIALIZED VIEW mv_artisans_by_category AS
SELECT
    state_code,
    district,
    social_category,
    COUNT(*) AS artisan_count,
    COUNT(*) FILTER (WHERE verified = true) AS verified_count
FROM artisan
WHERE social_category IS NOT NULL
GROUP BY state_code, district, social_category;

CREATE INDEX mv_artisans_by_category_state_district_idx
    ON mv_artisans_by_category (state_code, district);

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

DROP MATERIALIZED VIEW IF EXISTS mv_artisans_by_category;
ALTER TABLE artisan ALTER COLUMN social_category TYPE text;
CREATE MATERIALIZED VIEW mv_artisans_by_category AS
SELECT
    state_code,
    district,
    social_category,
    COUNT(*) AS artisan_count,
    COUNT(*) FILTER (WHERE verified = true) AS verified_count
FROM artisan
WHERE social_category IS NOT NULL
GROUP BY state_code, district, social_category;

CREATE INDEX mv_artisans_by_category_state_district_idx
    ON mv_artisans_by_category (state_code, district);

DROP TYPE IF EXISTS social_category;
