-- migrations/queries/insight.sql
-- Queries for insight-svc: materialized view refresh, aggregate fetches, income statement data.

-- name: RefreshMaterializedViews :exec
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_artisans_by_category;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_listings_by_craft_month;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_earnings_by_district;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_dying_crafts;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_impact_artisan;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_sales_by_artisan_month;

-- name: GetArtisansByCategory :many
SELECT state_code, district, social_category, artisan_count, verified_count
FROM mv_artisans_by_category
WHERE (sqlc.narg('state_code')::text IS NULL OR state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('district')::text IS NULL OR district = sqlc.narg('district'))
ORDER BY state_code, district, social_category;

-- name: GetListingsByCraftMonth :many
SELECT craft_id, craft_name, month, listing_count, artisan_count
FROM mv_listings_by_craft_month
WHERE (sqlc.narg('from_date')::timestamptz IS NULL OR month >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR month <= sqlc.narg('to_date'))
  AND (sqlc.narg('craft_id')::uuid IS NULL OR craft_id = sqlc.narg('craft_id'))
ORDER BY month DESC, craft_name;

-- name: GetEarningsByDistrict :many
SELECT state_code, district, total_gmv_paise, total_net_paise, artisan_count, avg_earnings_paise
FROM mv_earnings_by_district
WHERE (sqlc.narg('state_code')::text IS NULL OR state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('district')::text IS NULL OR district = sqlc.narg('district'))
  AND artisan_count >= @min_bucket
ORDER BY state_code, district;

-- name: GetDyingCrafts :many
SELECT craft_id, craft_name, decline_rate, peak_artisans, current_artisans
FROM mv_dying_crafts
ORDER BY decline_rate ASC
LIMIT $1;

-- name: GetArtisanEarningsByPeriod :one
SELECT
    a.id,
    a.display_name,
    COUNT(DISTINCT ol.id) AS order_count,
    COALESCE(SUM(psl.gross_amount_paise), 0)::bigint AS gross_paise,
    COALESCE(SUM(psl.net_amount_paise), 0)::bigint AS net_paise,
    COALESCE(SUM(psl.commission_paise), 0)::bigint AS fee_paise
FROM artisan a
LEFT JOIN order_lot ol ON ol.artisan_id = a.id
LEFT JOIN payment_split_line psl ON psl.lot_id = ol.id
    AND psl.payee_type = 'ARTISAN'
    AND psl.settled_at IS NOT NULL
    AND psl.settled_at >= $2
    AND psl.settled_at < $3
WHERE a.id = $1
GROUP BY a.id, a.display_name;

-- name: GetArtisanEarningsMonthly :many
SELECT
    date_trunc('month', psl.settled_at)::timestamptz AS month,
    COUNT(DISTINCT ol.id) AS order_count,
    SUM(psl.gross_amount_paise) AS gross_paise,
    SUM(psl.net_amount_paise) AS net_paise,
    SUM(psl.commission_paise) AS fee_paise
FROM payment_split_line psl
JOIN order_lot ol ON psl.lot_id = ol.id
WHERE ol.artisan_id = $1
  AND psl.payee_type = 'ARTISAN'
  AND psl.settled_at IS NOT NULL
  AND psl.settled_at >= $2
  AND psl.settled_at < $3
GROUP BY date_trunc('month', psl.settled_at)
ORDER BY month ASC;

-- name: CreateIncomeStatement :one
INSERT INTO income_statement (
    id, artisan_id, period_start, period_end, order_count,
    gross_paise, net_paise, fee_paise, signature, signature_algo,
    public_key_id, short_code, s3_key
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING id, created_at;

-- name: GetIncomeStatementByCode :one
SELECT
    s.id, s.artisan_id, s.period_start, s.period_end, s.order_count,
    s.gross_paise, s.net_paise, s.fee_paise, s.signature, s.signature_algo,
    s.public_key_id, s.short_code, s.s3_key, s.created_at,
    a.display_name AS artisan_name
FROM income_statement s
JOIN artisan a ON a.id = s.artisan_id
WHERE s.short_code = $1;

-- name: IncomeStatementShortCodeExists :one
SELECT EXISTS(SELECT 1 FROM income_statement WHERE short_code = @short_code) AS exists;

-- name: GetArtisanIncomeStatements :many
SELECT id, period_start, period_end, order_count, gross_paise, net_paise,
       fee_paise, short_code, s3_key, created_at
FROM income_statement
WHERE artisan_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListImpactArtisans :many
-- Per-artisan impact facts for one filtered cohort. Grouping, medians and
-- k<5 suppression happen in insight-svc (Go), never here.
SELECT artisan_id, state_code, district, social_category, corporations, finance_verified,
       registered_at, baseline_bracket, baseline_monthly_paise,
       platform_paise_90d, offline_paise_90d, fair_paise_90d, last_sale_at
FROM mv_impact_artisan
WHERE (sqlc.narg('state_code')::text IS NULL OR state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('district')::text IS NULL OR district = sqlc.narg('district'))
  AND (sqlc.narg('social_category')::text IS NULL OR social_category = sqlc.narg('social_category'))
  AND (sqlc.narg('corporation')::text IS NULL OR sqlc.narg('corporation')::text = ANY (corporations));

-- name: GetSalesMixByMonth :many
-- Monthly platform vs fair vs other-offline totals for a filtered cohort,
-- with the number of distinct artisans behind each month (for suppression).
SELECT s.month,
       SUM(s.platform_paise)::bigint AS platform_paise,
       SUM(s.fair_paise)::bigint AS fair_paise,
       SUM(s.other_offline_paise)::bigint AS other_offline_paise,
       COUNT(DISTINCT s.artisan_id)::bigint AS artisan_count
FROM mv_sales_by_artisan_month s
JOIN mv_impact_artisan i ON i.artisan_id = s.artisan_id
WHERE (sqlc.narg('state_code')::text IS NULL OR i.state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('district')::text IS NULL OR i.district = sqlc.narg('district'))
  AND (sqlc.narg('social_category')::text IS NULL OR i.social_category = sqlc.narg('social_category'))
  AND (sqlc.narg('corporation')::text IS NULL OR sqlc.narg('corporation')::text = ANY (i.corporations))
  AND (sqlc.narg('from_month')::date IS NULL OR s.month >= sqlc.narg('from_month'))
  AND (sqlc.narg('to_month')::date IS NULL OR s.month <= sqlc.narg('to_month'))
GROUP BY s.month
ORDER BY s.month;

-- name: GetImpactViewsRefreshedAt :one
-- Postgres records no refresh time for a materialized view, so the dashboard
-- reads the time insight-svc stamped after its last successful refresh
-- (epoch = never).
SELECT COALESCE(MAX(refreshed_at), 'epoch'::timestamptz)::timestamptz AS refreshed_at FROM insight_refresh_log;

-- name: RecordInsightRefresh :exec
INSERT INTO insight_refresh_log (refreshed_at) VALUES (now()) ON CONFLICT DO NOTHING;

-- name: ListFinanceCoverageFacts :many
-- One row per non-rejected finance link in a filtered cohort, with the
-- artisan's 90-day income, for per-corporation coverage in Go.
SELECT fl.artisan_id, fl.corporation::text AS corporation, fl.status::text AS status,
       COALESCE(fl.emi_paise, 0)::bigint AS emi_paise,
       i.platform_paise_90d, i.offline_paise_90d
FROM artisan_finance_link fl
JOIN mv_impact_artisan i ON i.artisan_id = fl.artisan_id
WHERE fl.status <> 'REJECTED'
  AND (sqlc.narg('state_code')::text IS NULL OR i.state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('district')::text IS NULL OR i.district = sqlc.narg('district'))
  AND (sqlc.narg('social_category')::text IS NULL OR i.social_category = sqlc.narg('social_category'))
  AND (sqlc.narg('corporation')::text IS NULL OR fl.corporation::text = sqlc.narg('corporation'));

-- name: GetLiteracyFunnel :many
-- Per district: registered artisans, how many started the literacy track,
-- how many completed at least half of it (4 of 8), and how many hold a
-- certificate. Live tables, not a view: the track is small and new.
SELECT i.state_code, i.district,
       COUNT(*)::bigint AS artisans,
       COUNT(*) FILTER (WHERE lp.started)::bigint AS started,
       COUNT(*) FILTER (WHERE lp.completed >= 4)::bigint AS half_way,
       COUNT(c.id)::bigint AS certified
FROM mv_impact_artisan i
LEFT JOIN LATERAL (
    SELECT count(*) > 0 AS started, count(completed_at) AS completed
    FROM literacy_progress p WHERE p.artisan_id = i.artisan_id
) lp ON true
LEFT JOIN literacy_certificate c ON c.artisan_id = i.artisan_id
WHERE (sqlc.narg('state_code')::text IS NULL OR i.state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('district')::text IS NULL OR i.district = sqlc.narg('district'))
  AND (sqlc.narg('social_category')::text IS NULL OR i.social_category = sqlc.narg('social_category'))
  AND (sqlc.narg('corporation')::text IS NULL OR sqlc.narg('corporation')::text = ANY (i.corporations))
GROUP BY i.state_code, i.district
ORDER BY i.state_code, i.district;

-- name: CountCompletedLessons :one
SELECT count(*)::bigint FROM literacy_progress WHERE artisan_id = $1 AND completed_at IS NOT NULL;

-- name: CreateLiteracyCertificate :one
-- ON CONFLICT: a certificate is issued once; a racing second issue returns nothing.
INSERT INTO literacy_certificate (id, artisan_id, short_code, signature, public_key_id, s3_key)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (artisan_id) DO NOTHING
RETURNING id, artisan_id, issued_at, short_code, s3_key;

-- name: GetLiteracyCertificateByArtisan :one
SELECT id, artisan_id, issued_at, short_code, s3_key FROM literacy_certificate WHERE artisan_id = $1;

-- name: GetLiteracyCertificateByCode :one
-- Everything the public verify page shows: name, primary craft and place --
-- never the phone or any other identifier.
SELECT c.id, c.issued_at, c.short_code, c.signature, c.public_key_id, c.s3_key,
       a.display_name AS artisan_name, a.state_code, COALESCE(a.district, '')::text AS district,
       COALESCE((SELECT cr.display_name FROM artisan_craft ac JOIN craft cr ON cr.id = ac.craft_id
                 WHERE ac.artisan_id = a.id ORDER BY ac.is_primary DESC, ac.created_at LIMIT 1), '')::text AS craft_name
FROM literacy_certificate c
JOIN artisan a ON a.id = c.artisan_id
WHERE c.short_code = $1;

-- name: LiteracyCertificateShortCodeExists :one
SELECT EXISTS(SELECT 1 FROM literacy_certificate WHERE short_code = @short_code) AS exists;

-- name: GetArtisanDisplayName :one
SELECT display_name FROM artisan WHERE id = $1;
