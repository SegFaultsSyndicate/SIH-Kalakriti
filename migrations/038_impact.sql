-- migrations/038_impact.sql
-- +goose Up

-- F13: real, privacy-safe outcome measurement.
--
-- Uplift cannot be measured from platform data alone (nobody has platform
-- income before they register -- which is exactly why mv_income_comparison
-- was always empty). So: a self-reported baseline captured at registration,
-- plus offline sales the artisan logs (fairs, local markets), measured
-- against settled platform payouts.
CREATE TYPE income_bracket AS ENUM
    ('LT_3K', 'B3K_6K', 'B6K_10K', 'B10K_15K', 'GT_15K', 'PREFER_NOT_TO_SAY');
CREATE TYPE offline_sale_channel AS ENUM ('FAIR', 'LOCAL_MARKET', 'DIRECT', 'OTHER');

CREATE TABLE artisan_income_baseline (
    artisan_id           uuid            NOT NULL,
    monthly_bracket      income_bracket  NOT NULL,
    -- Optional exact figure; when present it wins over the bracket midpoint.
    monthly_paise        bigint,
    fairs_per_year       smallint,
    fair_income_bracket  income_bracket,
    captured_at          timestamptz     NOT NULL DEFAULT now(),
    -- 'SELF' or 'AGENT' (captured in assisted mode).
    source               text            NOT NULL DEFAULT 'SELF',
    CONSTRAINT artisan_income_baseline_pkey PRIMARY KEY (artisan_id),
    CONSTRAINT artisan_income_baseline_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT artisan_income_baseline_monthly_check CHECK (monthly_paise IS NULL OR monthly_paise >= 0),
    CONSTRAINT artisan_income_baseline_fairs_check CHECK (fairs_per_year IS NULL OR fairs_per_year BETWEEN 0 AND 60),
    CONSTRAINT artisan_income_baseline_source_check CHECK (source IN ('SELF', 'AGENT'))
);

CREATE TABLE offline_sale (
    id            uuid                  NOT NULL,
    artisan_id    uuid                  NOT NULL,
    channel       offline_sale_channel  NOT NULL,
    -- e.g. 'Surajkund Mela 2026'.
    event_name    text,
    amount_paise  bigint                NOT NULL,
    sold_on       date                  NOT NULL,
    created_at    timestamptz           NOT NULL DEFAULT now(),
    CONSTRAINT offline_sale_pkey PRIMARY KEY (id),
    CONSTRAINT offline_sale_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT offline_sale_amount_check CHECK (amount_paise > 0),
    CONSTRAINT offline_sale_event_name_check CHECK (event_name IS NULL OR length(event_name) <= 120)
);
CREATE INDEX offline_sale_artisan_date_idx ON offline_sale (artisan_id, sold_on);

-- Per-artisan impact facts. Deliberately per ARTISAN, not pre-grouped:
-- insight-svc groups, takes medians and applies k<5 suppression in Go, so
-- the bracket->rupee mapping lives in exactly one function (pkg/impact) and
-- an artisan with two finance links is never double-counted in a KPI.
CREATE MATERIALIZED VIEW mv_impact_artisan AS
WITH platform AS (
    SELECT ol.artisan_id,
           SUM(psl.net_amount_paise) FILTER (WHERE psl.settled_at >= now() - interval '90 days') AS paise_90d,
           MAX(psl.settled_at) AS last_at
    FROM payment_split_line psl
    JOIN order_lot ol ON ol.id = psl.lot_id
    WHERE psl.payee_type = 'ARTISAN' AND psl.settled_at IS NOT NULL
    GROUP BY ol.artisan_id
),
offline AS (
    SELECT artisan_id,
           SUM(amount_paise) FILTER (WHERE sold_on >= current_date - 90) AS paise_90d,
           SUM(amount_paise) FILTER (WHERE sold_on >= current_date - 90 AND channel = 'FAIR') AS fair_paise_90d,
           MAX(sold_on) AS last_on
    FROM offline_sale
    GROUP BY artisan_id
),
finance AS (
    SELECT artisan_id,
           array_agg(DISTINCT corporation::text ORDER BY corporation::text) AS corporations,
           bool_or(status = 'VERIFIED') AS any_verified
    FROM artisan_finance_link
    WHERE status <> 'REJECTED'
    GROUP BY artisan_id
)
SELECT
    a.id AS artisan_id,
    a.state_code,
    COALESCE(a.district, '') AS district,
    COALESCE(a.social_category::text, '')::text AS social_category,
    COALESCE(f.corporations, '{}')::text[] AS corporations,
    COALESCE(f.any_verified, false) AS finance_verified,
    a.created_at AS registered_at,
    -- '' = no baseline captured (LEFT JOIN miss).
    COALESCE(b.monthly_bracket::text, '')::text AS baseline_bracket,
    b.monthly_paise AS baseline_monthly_paise,
    COALESCE(p.paise_90d, 0)::bigint AS platform_paise_90d,
    COALESCE(o.paise_90d, 0)::bigint AS offline_paise_90d,
    COALESCE(o.fair_paise_90d, 0)::bigint AS fair_paise_90d,
    -- 'epoch' = never sold (sqlc cannot see a nullable expression column).
    COALESCE(GREATEST(p.last_at, o.last_on::timestamptz), 'epoch')::timestamptz AS last_sale_at
FROM artisan a
LEFT JOIN platform p ON p.artisan_id = a.id
LEFT JOIN offline o ON o.artisan_id = a.id
LEFT JOIN finance f ON f.artisan_id = a.id
LEFT JOIN artisan_income_baseline b ON b.artisan_id = a.id;

CREATE UNIQUE INDEX mv_impact_artisan_artisan_idx ON mv_impact_artisan (artisan_id);
CREATE INDEX mv_impact_artisan_region_idx ON mv_impact_artisan (state_code, district);

-- Monthly sales mix per artisan (last 24 months): platform vs fair vs other
-- offline. Per artisan so a filtered cohort (state/district/category/
-- corporation) can be joined against mv_impact_artisan before summing.
CREATE MATERIALIZED VIEW mv_sales_by_artisan_month AS
WITH months AS (
    SELECT ol.artisan_id, date_trunc('month', psl.settled_at)::date AS month,
           psl.net_amount_paise AS platform, 0::bigint AS fair, 0::bigint AS other
    FROM payment_split_line psl
    JOIN order_lot ol ON ol.id = psl.lot_id
    WHERE psl.payee_type = 'ARTISAN' AND psl.settled_at >= now() - interval '24 months'
    UNION ALL
    SELECT artisan_id, date_trunc('month', sold_on)::date,
           0, CASE WHEN channel = 'FAIR' THEN amount_paise ELSE 0 END,
           CASE WHEN channel <> 'FAIR' THEN amount_paise ELSE 0 END
    FROM offline_sale
    WHERE sold_on >= current_date - interval '24 months'
)
SELECT artisan_id, month,
       SUM(platform)::bigint AS platform_paise,
       SUM(fair)::bigint AS fair_paise,
       SUM(other)::bigint AS other_offline_paise
FROM months
GROUP BY artisan_id, month;

CREATE UNIQUE INDEX mv_sales_by_artisan_month_idx ON mv_sales_by_artisan_month (artisan_id, month);

-- Postgres keeps no refresh timestamp for a materialized view; insight-svc
-- stamps one here after each successful refresh so the dashboard can say
-- "Updated 2 hours ago" honestly.
CREATE TABLE insight_refresh_log (
    refreshed_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT insight_refresh_log_pkey PRIMARY KEY (refreshed_at)
);

-- mv_income_comparison compared platform payouts before vs after
-- registration; nobody has payouts before registering, so it was always
-- empty. GetIncomeComparison now reads mv_impact_artisan (self-reported
-- baseline vs trailing-90-day income) instead.
DROP MATERIALIZED VIEW IF EXISTS mv_income_comparison;

-- REFRESH MATERIALIZED VIEW CONCURRENTLY (insight.sql) requires a unique
-- index on every view it refreshes; the 023/034 views never had one, so a
-- refresh always failed. District is NULL-able; NULLs are distinct in a
-- unique index, and each view emits at most one NULL-district row per key.
CREATE UNIQUE INDEX mv_artisans_by_category_key_idx
    ON mv_artisans_by_category (state_code, district, social_category);
CREATE UNIQUE INDEX mv_listings_by_craft_month_key_idx
    ON mv_listings_by_craft_month (craft_id, month);
CREATE UNIQUE INDEX mv_earnings_by_district_key_idx
    ON mv_earnings_by_district (state_code, district);
CREATE UNIQUE INDEX mv_dying_crafts_key_idx
    ON mv_dying_crafts (craft_id);

-- +goose Down

DROP INDEX IF EXISTS mv_dying_crafts_key_idx;
DROP INDEX IF EXISTS mv_earnings_by_district_key_idx;
DROP INDEX IF EXISTS mv_listings_by_craft_month_key_idx;
DROP INDEX IF EXISTS mv_artisans_by_category_key_idx;

CREATE MATERIALIZED VIEW mv_income_comparison AS
SELECT ''::text AS state_code, ''::text AS district,
       0::double precision AS median_before_paise, 0::double precision AS median_after_paise,
       0::bigint AS artisan_count
WHERE false;
CREATE INDEX mv_income_comparison_state_district_idx ON mv_income_comparison (state_code, district);

DROP TABLE IF EXISTS insight_refresh_log;
DROP MATERIALIZED VIEW IF EXISTS mv_sales_by_artisan_month;
DROP MATERIALIZED VIEW IF EXISTS mv_impact_artisan;
DROP TABLE IF EXISTS offline_sale;
DROP TABLE IF EXISTS artisan_income_baseline;
DROP TYPE IF EXISTS offline_sale_channel;
DROP TYPE IF EXISTS income_bracket;
