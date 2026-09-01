-- migrations/queries/insight.sql
-- Queries for insight-svc: materialized view refresh, aggregate fetches, income statement data.

-- name: RefreshMaterializedViews :exec
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_artisans_by_category;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_listings_by_craft_month;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_earnings_by_district;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_income_comparison;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_dying_crafts;

-- name: GetArtisansByCategory :many
SELECT state_code, district, social_category, artisan_count, verified_count
FROM mv_artisans_by_category
WHERE ($1::text IS NULL OR state_code = $1)
  AND ($2::text IS NULL OR district = $2)
ORDER BY state_code, district, social_category;

-- name: GetListingsByCraftMonth :many
SELECT craft_id, craft_name, month, listing_count, artisan_count
FROM mv_listings_by_craft_month
WHERE ($1::timestamptz IS NULL OR month >= $1)
  AND ($2::timestamptz IS NULL OR month <= $2)
  AND ($3::uuid IS NULL OR craft_id = $3)
ORDER BY month DESC, craft_name;

-- name: GetEarningsByDistrict :many
SELECT state_code, district, total_gmv_paise, total_net_paise, artisan_count, avg_earnings_paise
FROM mv_earnings_by_district
WHERE ($1::text IS NULL OR state_code = $1)
  AND ($2::text IS NULL OR district = $2)
  AND artisan_count >= $3
ORDER BY state_code, district;

-- name: GetIncomeComparison :many
SELECT state_code, district, median_before_paise, median_after_paise, artisan_count
FROM mv_income_comparison
WHERE ($1::text IS NULL OR state_code = $1)
  AND ($2::text IS NULL OR district = $2)
  AND artisan_count >= $3
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
    COALESCE(SUM(psl.gross_amount_paise), 0) AS gross_paise,
    COALESCE(SUM(psl.net_amount_paise), 0) AS net_paise,
    COALESCE(SUM(psl.commission_paise), 0) AS fee_paise
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
    date_trunc('month', psl.settled_at) AS month,
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

-- name: GetArtisanIncomeStatements :many
SELECT id, period_start, period_end, order_count, gross_paise, net_paise,
       fee_paise, short_code, s3_key, created_at
FROM income_statement
WHERE artisan_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
