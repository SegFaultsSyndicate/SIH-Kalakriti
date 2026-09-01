-- migrations/023_insight.sql
-- +goose Up

-- Ministry dashboard materialized views for aggregates. Refreshed on a ticker
-- (default 15 min) and via manual endpoint, never computed live per request.

-- Artisans onboarded by social category and district.
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

-- Listings published by craft and month.
CREATE MATERIALIZED VIEW mv_listings_by_craft_month AS
SELECT
    p.craft_id,
    c.display_name AS craft_name,
    date_trunc('month', l.published_at) AS month,
    COUNT(*) AS listing_count,
    COUNT(DISTINCT l.artisan_id) AS artisan_count
FROM listing l
JOIN product p ON l.product_id = p.id
JOIN craft c ON p.craft_id = c.id
WHERE l.published_at IS NOT NULL
GROUP BY p.craft_id, c.display_name, date_trunc('month', l.published_at);

CREATE INDEX mv_listings_by_craft_month_craft_month_idx
    ON mv_listings_by_craft_month (craft_id, month DESC);

-- GMV and average artisan earnings by district.
CREATE MATERIALIZED VIEW mv_earnings_by_district AS
SELECT
    a.state_code,
    a.district,
    SUM(psl.gross_amount_paise) AS total_gmv_paise,
    SUM(psl.net_amount_paise) AS total_net_paise,
    COUNT(DISTINCT psl.payee_id) AS artisan_count,
    CASE WHEN COUNT(DISTINCT psl.payee_id) > 0
         THEN SUM(psl.net_amount_paise) / COUNT(DISTINCT psl.payee_id)
         ELSE 0
    END AS avg_earnings_paise
FROM payment_split_line psl
JOIN order_lot ol ON psl.lot_id = ol.id
JOIN artisan a ON ol.artisan_id = a.id
WHERE psl.payee_type = 'ARTISAN'
  AND psl.settled_at IS NOT NULL
GROUP BY a.state_code, a.district;

CREATE INDEX mv_earnings_by_district_state_district_idx
    ON mv_earnings_by_district (state_code, district);

-- Median artisan income before vs after 90 days. Tracks artisan_id registration
-- and compares income in [-90d, registration) window vs [registration, +90d].
-- Only artisans with at least one settled payment in each window are included.
CREATE MATERIALIZED VIEW mv_income_comparison AS
WITH artisan_windows AS (
    SELECT
        a.id AS artisan_id,
        a.state_code,
        a.district,
        a.created_at AS registered_at,
        a.created_at - interval '90 days' AS before_start,
        a.created_at AS before_end,
        a.created_at AS after_start,
        a.created_at + interval '90 days' AS after_end
    FROM artisan a
    WHERE a.created_at <= now() - interval '90 days'
),
before_income AS (
    SELECT
        aw.artisan_id,
        SUM(psl.net_amount_paise) AS income_paise
    FROM artisan_windows aw
    JOIN order_lot ol ON ol.artisan_id = aw.artisan_id
    JOIN payment_split_line psl ON psl.lot_id = ol.id
    WHERE psl.payee_type = 'ARTISAN'
      AND psl.settled_at IS NOT NULL
      AND psl.settled_at >= aw.before_start
      AND psl.settled_at < aw.before_end
    GROUP BY aw.artisan_id
),
after_income AS (
    SELECT
        aw.artisan_id,
        SUM(psl.net_amount_paise) AS income_paise
    FROM artisan_windows aw
    JOIN order_lot ol ON ol.artisan_id = aw.artisan_id
    JOIN payment_split_line psl ON psl.lot_id = ol.id
    WHERE psl.payee_type = 'ARTISAN'
      AND psl.settled_at IS NOT NULL
      AND psl.settled_at >= aw.after_start
      AND psl.settled_at < aw.after_end
    GROUP BY aw.artisan_id
)
SELECT
    aw.state_code,
    aw.district,
    percentile_cont(0.5) WITHIN GROUP (ORDER BY bi.income_paise) AS median_before_paise,
    percentile_cont(0.5) WITHIN GROUP (ORDER BY ai.income_paise) AS median_after_paise,
    COUNT(*) AS artisan_count
FROM artisan_windows aw
JOIN before_income bi ON bi.artisan_id = aw.artisan_id
JOIN after_income ai ON ai.artisan_id = aw.artisan_id
GROUP BY aw.state_code, aw.district;

CREATE INDEX mv_income_comparison_state_district_idx
    ON mv_income_comparison (state_code, district);

-- Dying-craft watch: crafts with declining active-artisan count over trailing
-- 12 months, ranked by decline rate, minimum sample size 5 artisans.
CREATE MATERIALIZED VIEW mv_dying_crafts AS
WITH monthly_active AS (
    SELECT
        p.craft_id,
        date_trunc('month', l.published_at) AS month,
        COUNT(DISTINCT l.artisan_id) AS active_artisan_count
    FROM listing l
    JOIN product p ON l.product_id = p.id
    WHERE l.published_at >= now() - interval '12 months'
      AND l.published_at IS NOT NULL
    GROUP BY p.craft_id, date_trunc('month', l.published_at)
),
craft_stats AS (
    SELECT
        craft_id,
        MIN(active_artisan_count) AS min_count,
        MAX(active_artisan_count) AS max_count,
        AVG(active_artisan_count) AS avg_count,
        (MIN(active_artisan_count)::real - MAX(active_artisan_count)::real) / NULLIF(MAX(active_artisan_count)::real, 0) AS decline_rate
    FROM monthly_active
    GROUP BY craft_id
    HAVING MAX(active_artisan_count) >= 5
       AND MIN(active_artisan_count) < MAX(active_artisan_count)
)
SELECT
    cs.craft_id,
    c.display_name AS craft_name,
    cs.decline_rate,
    cs.max_count AS peak_artisans,
    cs.min_count AS current_artisans
FROM craft_stats cs
JOIN craft c ON c.id = cs.craft_id
WHERE cs.decline_rate < 0
ORDER BY cs.decline_rate ASC;

CREATE INDEX mv_dying_crafts_decline_rate_idx
    ON mv_dying_crafts (decline_rate ASC);

-- Income statement verification table. Maps short codes to artisan/period for
-- public verification at /v/statement/{code}.
CREATE TABLE income_statement (
    id              uuid        NOT NULL,
    artisan_id      uuid        NOT NULL,
    period_start    timestamptz NOT NULL,
    period_end      timestamptz NOT NULL,
    order_count     integer     NOT NULL,
    gross_paise     bigint      NOT NULL,
    net_paise       bigint      NOT NULL,
    fee_paise       bigint      NOT NULL,
    signature       bytea       NOT NULL,
    signature_algo  text        NOT NULL DEFAULT 'ed25519',
    public_key_id   text        NOT NULL,
    short_code      text        NOT NULL,
    s3_key          text        NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT income_statement_pkey PRIMARY KEY (id),
    CONSTRAINT income_statement_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE RESTRICT,
    CONSTRAINT income_statement_short_code_key UNIQUE (short_code),
    CONSTRAINT income_statement_period_check CHECK (period_end > period_start),
    CONSTRAINT income_statement_amounts_check CHECK (
        gross_paise >= 0 AND net_paise >= 0 AND fee_paise >= 0
        AND gross_paise = net_paise + fee_paise
    ),
    CONSTRAINT income_statement_short_code_check CHECK (length(short_code) = 10)
);

-- Public verification lookups.
CREATE INDEX income_statement_short_code_idx ON income_statement (short_code);
-- Artisan's own statements.
CREATE INDEX income_statement_artisan_id_created_at_idx
    ON income_statement (artisan_id, created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS income_statement;
DROP MATERIALIZED VIEW IF EXISTS mv_dying_crafts;
DROP MATERIALIZED VIEW IF EXISTS mv_income_comparison;
DROP MATERIALIZED VIEW IF EXISTS mv_earnings_by_district;
DROP MATERIALIZED VIEW IF EXISTS mv_listings_by_craft_month;
DROP MATERIALIZED VIEW IF EXISTS mv_artisans_by_category;
