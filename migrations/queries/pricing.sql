-- migrations/queries/pricing.sql

-- name: GetListingPricingSource :one
-- The listing's own craft, GI status, materials, embedding, reference size and
-- the owning artisan's state (for the wage floor  -  the artisan cannot pick a
-- more favourable state, it comes from their own registration). language picks
-- which per-language search row to read; the pricing signal does not depend on
-- which one, so callers pass a stable default.
SELECT ls.craft_id, ls.gi_certified, ls.materials, ls.embedding, p.length_mm,
       a.state_code
FROM listing_search ls
JOIN listing l ON l.id = ls.listing_id
JOIN product p ON p.id = l.product_id
JOIN artisan a ON a.id = ls.artisan_id
WHERE ls.listing_id = @listing_id AND ls.language = @language
LIMIT 1;

-- name: GetMinimumWage :one
-- The most recent rate on or before the given date; a wage revision is a new
-- row, so this always cites the rate that was actually in force.
SELECT state_code, effective_date, wage_paise_per_hour
FROM state_minimum_wage
WHERE state_code = @state_code AND effective_date <= @as_of
ORDER BY effective_date DESC
LIMIT 1;

-- name: ListSeasonalityMultipliers :many
-- Every row that could apply to this craft this month: craft-specific rows and
-- the crafts-wide (craft_id IS NULL) rows. The caller picks the driver.
SELECT craft_id, month, festival, multiplier
FROM seasonality_multiplier
WHERE month = @month AND (craft_id = @craft_id OR craft_id IS NULL);

-- name: PricingComparableStats :one
-- Nearest-by-embedding candidates within the requested filter, then p25/p50/p75
-- over their price. The three require_* flags let one query serve both the
-- strict pass (same craft, size band, material class, GI match) and the
-- widened pass (craft only) without duplicating the query.
WITH candidates AS (
    SELECT ls.price_paise
    FROM listing_search ls
    JOIN listing l ON l.id = ls.listing_id
    JOIN product p ON p.id = l.product_id
    WHERE ls.craft_id = @craft_id
      AND ls.listing_id != @exclude_listing_id
      AND ls.embedding IS NOT NULL
      AND (NOT @require_gi::bool OR ls.gi_certified = @gi_certified::bool)
      AND (NOT @require_materials::bool OR ls.materials && @materials::text[])
      AND (NOT @require_size::bool OR (
            p.length_mm IS NOT NULL
            AND p.length_mm BETWEEN @size_min_mm::int AND @size_max_mm::int
      ))
    ORDER BY ls.embedding <=> @embedding::vector
    LIMIT @candidate_limit::int
)
SELECT
    count(*)::bigint AS sample_size,
    coalesce(percentile_cont(0.25) WITHIN GROUP (ORDER BY price_paise), 0)::bigint AS p25_paise,
    coalesce(percentile_cont(0.5)  WITHIN GROUP (ORDER BY price_paise), 0)::bigint AS p50_paise,
    coalesce(percentile_cont(0.75) WITHIN GROUP (ORDER BY price_paise), 0)::bigint AS p75_paise
FROM candidates;
