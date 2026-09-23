-- migrations/037_finance_link.sql
-- +goose Up

-- F12: tie an artisan's MoSJE finance-corporation loan / scheme beneficiary
-- record to their shop, so platform sales can be shown against their EMI.
CREATE TYPE finance_corporation AS ENUM
    ('NSFDC', 'NBCFDC', 'NSKFDC', 'NDFDC', 'PM_DAKSH', 'PM_AJAY', 'OTHER');
CREATE TYPE finance_link_status AS ENUM ('SELF_REPORTED', 'VERIFIED', 'REJECTED');

CREATE TABLE artisan_finance_link (
    id                  uuid                 NOT NULL,
    artisan_id          uuid                 NOT NULL,
    corporation         finance_corporation  NOT NULL,
    -- State Channelizing Agency / bank name.
    channelizing_agency text,
    -- Only the last 4 characters of the loan/beneficiary reference are
    -- stored readable; reference_hash is HMAC-SHA256(server salt, full ref)
    -- for de-duplication. The plaintext never reaches the database or logs.
    reference_last4     text                 NOT NULL,
    reference_hash      bytea                NOT NULL,
    sanctioned_paise    bigint,
    emi_paise           bigint,
    emi_day_of_month    smallint,
    repayment_start     date,
    status              finance_link_status  NOT NULL DEFAULT 'SELF_REPORTED',
    verified_by         text,
    verified_at         timestamptz,
    reject_reason       text,
    -- DPDP Act 2023: record when and to which consent text the artisan agreed.
    consent_at          timestamptz          NOT NULL,
    consent_version     text                 NOT NULL,
    created_at          timestamptz          NOT NULL DEFAULT now(),
    updated_at          timestamptz          NOT NULL DEFAULT now(),
    CONSTRAINT artisan_finance_link_pkey PRIMARY KEY (id),
    CONSTRAINT artisan_finance_link_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT finance_link_last4_check CHECK (length(reference_last4) BETWEEN 1 AND 4),
    CONSTRAINT finance_link_sanctioned_check CHECK (sanctioned_paise IS NULL OR sanctioned_paise >= 0),
    CONSTRAINT finance_link_emi_check CHECK (emi_paise IS NULL OR emi_paise >= 0),
    CONSTRAINT finance_link_emi_day_check CHECK (emi_day_of_month IS NULL OR emi_day_of_month BETWEEN 1 AND 28),
    CONSTRAINT finance_link_verified_check CHECK ((status = 'VERIFIED') = (verified_at IS NOT NULL)),
    CONSTRAINT finance_link_unique UNIQUE (artisan_id, corporation, reference_hash)
);

CREATE INDEX artisan_finance_link_artisan_idx ON artisan_finance_link (artisan_id);
CREATE INDEX artisan_finance_link_status_idx ON artisan_finance_link (status);

-- EMI reminders ride the existing notification table (009) and its delivery
-- pipeline (029). One reminder per link per due month, however many core-svc
-- replicas run the daily job.
ALTER TYPE notification_kind ADD VALUE IF NOT EXISTS 'EMI_REMINDER';

CREATE TABLE emi_reminder_sent (
    link_id    uuid        NOT NULL,
    due_month  date        NOT NULL,
    sent_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT emi_reminder_sent_pkey PRIMARY KEY (link_id, due_month),
    CONSTRAINT emi_reminder_sent_link_id_fkey FOREIGN KEY (link_id)
        REFERENCES artisan_finance_link (id) ON DELETE CASCADE
);

-- Seed the MoSJE finance corporations and schemes into the scheme catalog so
-- /schemes recommends the right corporation to an artisan who has not linked
-- one. NSKFDC and PM-DAKSH point at the ministry portal: their own domains
-- (nskfdc.nic.in, pmdaksh.dosje.gov.in) did not resolve when this was written
-- (2026-09-23); the other four were checked and resolve.
INSERT INTO government_scheme (id, code, authority, ministry, official_url, name_i18n_key, summary_i18n_key, sort_order, curated_by) VALUES
    (gen_random_uuid(), 'nsfdc',    'CENTRAL', 'Ministry of Social Justice and Empowerment', 'https://nsfdc.nic.in',         'scheme.nsfdc.name',    'scheme.nsfdc.summary',    90,  'system'),
    (gen_random_uuid(), 'nbcfdc',   'CENTRAL', 'Ministry of Social Justice and Empowerment', 'https://nbcfdc.gov.in',        'scheme.nbcfdc.name',   'scheme.nbcfdc.summary',   100, 'system'),
    (gen_random_uuid(), 'nskfdc',   'CENTRAL', 'Ministry of Social Justice and Empowerment', 'https://socialjustice.gov.in', 'scheme.nskfdc.name',   'scheme.nskfdc.summary',   110, 'system'),
    (gen_random_uuid(), 'ndfdc',    'CENTRAL', 'Ministry of Social Justice and Empowerment', 'https://ndfdc.nic.in',         'scheme.ndfdc.name',    'scheme.ndfdc.summary',    120, 'system'),
    (gen_random_uuid(), 'pm_daksh', 'CENTRAL', 'Ministry of Social Justice and Empowerment', 'https://socialjustice.gov.in', 'scheme.pm_daksh.name', 'scheme.pm_daksh.summary', 130, 'system'),
    (gen_random_uuid(), 'pm_ajay',  'CENTRAL', 'Ministry of Social Justice and Empowerment', 'https://pmajay.dosje.gov.in',  'scheme.pm_ajay.name',  'scheme.pm_ajay.summary',  140, 'system');

INSERT INTO scheme_criterion (id, scheme_id, type, string_values)
SELECT gen_random_uuid(), id, 'SOCIAL_CATEGORY', ARRAY['SC']
FROM government_scheme WHERE code IN ('nsfdc', 'pm_ajay');

INSERT INTO scheme_criterion (id, scheme_id, type, string_values)
SELECT gen_random_uuid(), id, 'SOCIAL_CATEGORY', ARRAY['OBC', 'EWS']
FROM government_scheme WHERE code = 'nbcfdc';

INSERT INTO scheme_criterion (id, scheme_id, type, string_values)
SELECT gen_random_uuid(), id, 'SOCIAL_CATEGORY', ARRAY['SC', 'OBC', 'EWS']
FROM government_scheme WHERE code = 'pm_daksh';

-- NSKFDC (sanitation workers) and NDFDC (persons with disabilities) turn on
-- facts the platform does not hold, so they are self-confirm checks only.
INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.nskfdc.check.sanitation_worker', 10
FROM government_scheme WHERE code = 'nskfdc';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.ndfdc.check.disability_certificate', 10
FROM government_scheme WHERE code = 'ndfdc';

INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, sort_order)
SELECT gen_random_uuid(), id, 'scheme.finance.check.income_limit', 20
FROM government_scheme WHERE code IN ('nsfdc', 'nbcfdc', 'nskfdc', 'ndfdc');

-- +goose Down

DELETE FROM government_scheme WHERE code IN ('nsfdc', 'nbcfdc', 'nskfdc', 'ndfdc', 'pm_daksh', 'pm_ajay');
DROP TABLE IF EXISTS emi_reminder_sent;
DROP TABLE IF EXISTS artisan_finance_link;
DROP TYPE IF EXISTS finance_link_status;
DROP TYPE IF EXISTS finance_corporation;
-- notification_kind's EMI_REMINDER value is left in place: Postgres cannot
-- drop an enum value, and an unused one is harmless.
