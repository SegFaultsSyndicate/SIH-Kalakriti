-- migrations/queries/impact.sql
-- F13 artisan-side: income baseline, logged offline sales, and the live
-- per-artisan income summary (the ministry aggregates live in insight.sql).

-- name: UpsertIncomeBaseline :one
INSERT INTO artisan_income_baseline (
    artisan_id, monthly_bracket, monthly_paise, fairs_per_year, fair_income_bracket, source, captured_at
) VALUES (
    @artisan_id, @monthly_bracket, sqlc.narg('monthly_paise'), sqlc.narg('fairs_per_year'),
    sqlc.narg('fair_income_bracket'), @source, now()
)
ON CONFLICT (artisan_id) DO UPDATE SET
    monthly_bracket     = EXCLUDED.monthly_bracket,
    monthly_paise       = EXCLUDED.monthly_paise,
    fairs_per_year      = EXCLUDED.fairs_per_year,
    fair_income_bracket = EXCLUDED.fair_income_bracket,
    source              = EXCLUDED.source,
    captured_at         = now()
RETURNING *;

-- name: GetIncomeBaseline :one
SELECT * FROM artisan_income_baseline WHERE artisan_id = @artisan_id;

-- name: InsertOfflineSale :one
INSERT INTO offline_sale (id, artisan_id, channel, event_name, amount_paise, sold_on)
VALUES (@id, @artisan_id, @channel, sqlc.narg('event_name'), @amount_paise, @sold_on)
ON CONFLICT (id) DO NOTHING
RETURNING *;

-- name: GetOfflineSale :one
SELECT * FROM offline_sale WHERE id = @id;

-- name: ListOfflineSales :many
SELECT * FROM offline_sale
WHERE artisan_id = @artisan_id
ORDER BY sold_on DESC, created_at DESC
LIMIT @page_size;

-- name: DeleteOfflineSale :execrows
DELETE FROM offline_sale WHERE id = @id AND artisan_id = @artisan_id;

-- name: GetArtisanIncomeFacts :one
-- Everything the income summary needs in one round trip. Platform income is
-- settled ARTISAN payment_split_line net only; unsettled lines are reported
-- separately as "on the way", never counted as earned.
SELECT
    a.created_at AS registered_at,
    COALESCE((SELECT SUM(psl.net_amount_paise) FROM payment_split_line psl
              JOIN order_lot ol ON ol.id = psl.lot_id
              WHERE ol.artisan_id = a.id AND psl.payee_type = 'ARTISAN'
                AND psl.settled_at >= now() - interval '90 days'), 0)::bigint AS platform_paise_90d,
    COALESCE((SELECT SUM(psl.net_amount_paise) FROM payment_split_line psl
              JOIN order_lot ol ON ol.id = psl.lot_id
              WHERE ol.artisan_id = a.id AND psl.payee_type = 'ARTISAN'
                AND psl.settled_at IS NULL), 0)::bigint AS platform_pending_paise,
    COALESCE((SELECT SUM(os.amount_paise) FROM offline_sale os
              WHERE os.artisan_id = a.id AND os.sold_on >= current_date - 90), 0)::bigint AS offline_paise_90d,
    COALESCE((SELECT SUM(os.amount_paise) FROM offline_sale os
              WHERE os.artisan_id = a.id AND os.sold_on >= current_date - 90
                AND os.channel = 'FAIR'), 0)::bigint AS fair_paise_90d
FROM artisan a
WHERE a.id = @artisan_id;

-- name: GetArtisanMonthlyIncome :many
-- Platform (settled) and offline totals per calendar month since @since.
WITH m AS (
    SELECT date_trunc('month', psl.settled_at)::date AS month,
           psl.net_amount_paise AS platform, 0::bigint AS offline
    FROM payment_split_line psl
    JOIN order_lot ol ON ol.id = psl.lot_id
    WHERE ol.artisan_id = @artisan_id AND psl.payee_type = 'ARTISAN'
      AND psl.settled_at >= @since::date
    UNION ALL
    SELECT date_trunc('month', os.sold_on)::date, 0, os.amount_paise
    FROM offline_sale os
    WHERE os.artisan_id = @artisan_id AND os.sold_on >= @since::date
)
SELECT month, SUM(platform)::bigint AS platform_paise, SUM(offline)::bigint AS offline_paise
FROM m
GROUP BY month
ORDER BY month;
