-- migrations/031_b2b.sql
-- +goose Up

-- Company types mirror b2b.v1.CompanyType without the UNSPECIFIED member.
CREATE TYPE company_type AS ENUM ('RETAILER', 'BOUTIQUE', 'EXPORTER', 'INSTITUTION');

-- Verification lifecycle for company legitimacy checks.
CREATE TYPE verification_status AS ENUM ('PENDING', 'VERIFIED', 'REJECTED');

-- Interest lifecycle mirror b2b.v1.InterestStatus.
CREATE TYPE interest_status AS ENUM ('PENDING', 'ACCEPTED', 'DECLINED');

-- Boutique-artisan match status mirror b2b.v1.MatchStatus.
CREATE TYPE match_status AS ENUM ('SUGGESTED', 'CONTACTED', 'ACTIVE', 'DECLINED');

CREATE TABLE company (
    id                      uuid                 NOT NULL,
    user_id                 text                 NOT NULL,
    name                    text                 NOT NULL,
    company_type            company_type         NOT NULL,
    gstin                   text,
    contact_name            text                 NOT NULL,
    contact_phone           text                 NOT NULL,
    contact_email           text,
    website                 text,
    state_code              text                 NOT NULL,
    district                text,
    verification_status     verification_status  NOT NULL DEFAULT 'PENDING',
    verified                boolean              NOT NULL DEFAULT false,
    verified_by             text,
    verified_at             timestamptz,
    rejection_reason        text,
    income_statement_url    text                 NOT NULL DEFAULT '',
    commission_rate_bps     integer              NOT NULL DEFAULT 50,
    total_sales_paise       bigint               NOT NULL DEFAULT 0,
    commission_earned_paise bigint               NOT NULL DEFAULT 0,
    -- Boutique-specific fields.
    accepts_consignment     boolean              NOT NULL DEFAULT false,
    min_order_value_paise   bigint,
    preferred_craft_ids     uuid[],
    -- Physical store location for proximity matching.
    store_latitude          double precision,
    store_longitude         double precision,
    store_address           text,
    store_city              text,
    store_pincode           text,
    created_at              timestamptz          NOT NULL DEFAULT now(),
    updated_at              timestamptz          NOT NULL DEFAULT now(),
    CONSTRAINT company_pkey PRIMARY KEY (id),
    CONSTRAINT company_user_id_key UNIQUE (user_id),
    CONSTRAINT company_gstin_key UNIQUE (gstin),
    CONSTRAINT company_contact_phone_check CHECK (contact_phone ~ '^\+[1-9][0-9]{7,14}$'),
    CONSTRAINT company_min_order_value_check CHECK (min_order_value_paise IS NULL OR min_order_value_paise >= 0),
    CONSTRAINT company_commission_rate_check CHECK (commission_rate_bps IN (50, 100)),
    CONSTRAINT company_sales_totals_check CHECK (total_sales_paise >= 0 AND commission_earned_paise >= 0),
    CONSTRAINT company_store_coords_check CHECK (
        (store_latitude IS NULL AND store_longitude IS NULL) OR
        (store_latitude IS NOT NULL AND store_longitude IS NOT NULL)
    )
);

-- Discovery: companies by type, for artisan browsing.
CREATE INDEX company_type_idx ON company (company_type);
-- Admin verification queue.
CREATE INDEX company_verification_status_idx ON company (verification_status, created_at DESC);
-- Discovery: boutiques by location for proximity search.
CREATE INDEX company_boutique_location_idx ON company (store_latitude, store_longitude)
    WHERE company_type = 'BOUTIQUE' AND store_latitude IS NOT NULL;
-- Lookup by user_id for "my company" endpoint.
CREATE INDEX company_user_id_idx ON company (user_id);

CREATE TABLE company_sale_settlement (
    id                      uuid          NOT NULL,
    company_id              uuid          NOT NULL,
    order_id                text          NOT NULL,
    product_name            text          NOT NULL,
    buyer_id                text          NOT NULL,
    gross_amount_paise      bigint        NOT NULL CHECK (gross_amount_paise > 0),
    commission_rate_bps     integer       NOT NULL CHECK (commission_rate_bps IN (50, 100)),
    platform_fee_paise      bigint        NOT NULL CHECK (platform_fee_paise >= 0),
    net_payout_paise        bigint        NOT NULL CHECK (net_payout_paise >= 0),
    settled_at              timestamptz   NOT NULL DEFAULT now(),
    created_at              timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT company_sale_settlement_pkey PRIMARY KEY (id),
    CONSTRAINT company_sale_settlement_company_id_fkey FOREIGN KEY (company_id)
        REFERENCES company (id) ON DELETE RESTRICT
);

CREATE INDEX company_sale_settlement_company_idx ON company_sale_settlement (company_id, settled_at DESC);
CREATE INDEX company_sale_settlement_order_idx ON company_sale_settlement (order_id);

CREATE TABLE company_interest (
    id              uuid            NOT NULL,
    company_id      uuid            NOT NULL,
    artisan_id      uuid            NOT NULL,
    message         text            NOT NULL DEFAULT '',
    status          interest_status NOT NULL DEFAULT 'PENDING',
    responded_at    timestamptz,
    created_at      timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT company_interest_pkey PRIMARY KEY (id),
    CONSTRAINT company_interest_company_id_fkey FOREIGN KEY (company_id)
        REFERENCES company (id) ON DELETE CASCADE,
    CONSTRAINT company_interest_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    -- A company can only have one pending interest per artisan.
    CONSTRAINT company_interest_unique_pending UNIQUE (company_id, artisan_id),
    CONSTRAINT company_interest_responded_check CHECK (
        (status = 'PENDING' AND responded_at IS NULL) OR
        (status != 'PENDING' AND responded_at IS NOT NULL)
    )
);

-- Artisan's inbox of inbound leads, newest first.
CREATE INDEX company_interest_artisan_status_idx ON company_interest (artisan_id, status, created_at DESC);
-- Company's outbound interest list.
CREATE INDEX company_interest_company_idx ON company_interest (company_id, created_at DESC);

CREATE TABLE supply_partnership (
    id              uuid        NOT NULL,
    company_id      uuid        NOT NULL,
    artisan_id      uuid        NOT NULL,
    craft_id        uuid        NOT NULL,
    terms           text        NOT NULL DEFAULT '',
    renewal_date    timestamptz,
    active          boolean     NOT NULL DEFAULT true,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT supply_partnership_pkey PRIMARY KEY (id),
    CONSTRAINT supply_partnership_company_id_fkey FOREIGN KEY (company_id)
        REFERENCES company (id) ON DELETE CASCADE,
    CONSTRAINT supply_partnership_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT supply_partnership_craft_id_fkey FOREIGN KEY (craft_id)
        REFERENCES craft (id) ON DELETE RESTRICT
);

-- Artisan's active partnerships.
CREATE INDEX supply_partnership_artisan_active_idx ON supply_partnership (artisan_id, active)
    WHERE active = true;
-- Company's active partnerships.
CREATE INDEX supply_partnership_company_active_idx ON supply_partnership (company_id, active)
    WHERE active = true;

CREATE TABLE boutique_artisan_match (
    id              uuid          NOT NULL,
    company_id      uuid          NOT NULL,
    artisan_id      uuid          NOT NULL,
    match_score     real          NOT NULL DEFAULT 0.0,
    status          match_status  NOT NULL DEFAULT 'SUGGESTED',
    created_at      timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT boutique_artisan_match_pkey PRIMARY KEY (id),
    CONSTRAINT boutique_artisan_match_company_id_fkey FOREIGN KEY (company_id)
        REFERENCES company (id) ON DELETE CASCADE,
    CONSTRAINT boutique_artisan_match_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    -- One match suggestion per boutique-artisan pair.
    CONSTRAINT boutique_artisan_match_unique UNIQUE (company_id, artisan_id),
    CONSTRAINT boutique_artisan_match_score_check CHECK (match_score BETWEEN 0.0 AND 1.0)
);

-- Artisan's recommended boutiques, highest score first.
CREATE INDEX boutique_match_artisan_score_idx ON boutique_artisan_match (artisan_id, match_score DESC);

-- +goose Down

DROP TABLE IF EXISTS boutique_artisan_match;
DROP TABLE IF EXISTS supply_partnership;
DROP TABLE IF EXISTS company_interest;
DROP TABLE IF EXISTS company_sale_settlement;
DROP TABLE IF EXISTS company;
DROP TYPE IF EXISTS match_status;
DROP TYPE IF EXISTS interest_status;
DROP TYPE IF EXISTS verification_status;
DROP TYPE IF EXISTS company_type;
