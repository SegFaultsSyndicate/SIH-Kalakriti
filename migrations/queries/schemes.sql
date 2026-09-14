-- migrations/queries/schemes.sql

-- name: ListActiveSchemes :many
SELECT * FROM government_scheme WHERE active ORDER BY sort_order;

-- name: GetSchemeByID :one
SELECT * FROM government_scheme WHERE id = @id;

-- name: GetSchemeCriteria :many
SELECT * FROM scheme_criterion WHERE scheme_id = @scheme_id;

-- name: GetSchemeManualChecks :many
SELECT * FROM scheme_manual_check WHERE scheme_id = @scheme_id ORDER BY sort_order;

-- name: GetAllActiveSchemeCriteria :many
-- Bulk fetch for MatchSchemes -- one round trip for every active scheme's
-- criteria rather than N+1 per scheme.
SELECT sc.* FROM scheme_criterion sc
JOIN government_scheme gs ON gs.id = sc.scheme_id
WHERE gs.active;

-- name: GetAllActiveSchemeManualChecks :many
SELECT smc.* FROM scheme_manual_check smc
JOIN government_scheme gs ON gs.id = smc.scheme_id
WHERE gs.active
ORDER BY smc.sort_order;

-- name: UpsertScheme :one
INSERT INTO government_scheme (
    id, code, authority, ministry, official_url, state_code,
    name_i18n_key, name_text, summary_i18n_key, summary_text,
    active, sort_order, curated_by
) VALUES (
    @id, @code, @authority, @ministry, @official_url, sqlc.narg('state_code'),
    sqlc.narg('name_i18n_key'), sqlc.narg('name_text'), sqlc.narg('summary_i18n_key'), sqlc.narg('summary_text'),
    @active, @sort_order, @curated_by
)
ON CONFLICT (id) DO UPDATE SET
    code = @code, authority = @authority, ministry = @ministry, official_url = @official_url,
    state_code = sqlc.narg('state_code'), name_i18n_key = sqlc.narg('name_i18n_key'),
    name_text = sqlc.narg('name_text'), summary_i18n_key = sqlc.narg('summary_i18n_key'),
    summary_text = sqlc.narg('summary_text'), active = @active, sort_order = @sort_order, updated_at = now()
RETURNING *;

-- name: DeleteScheme :execrows
DELETE FROM government_scheme WHERE id = @id;

-- name: DeleteSchemeCriteria :exec
DELETE FROM scheme_criterion WHERE scheme_id = @scheme_id;

-- name: InsertSchemeCriterion :exec
INSERT INTO scheme_criterion (id, scheme_id, type, string_values, int_value, negate)
VALUES (@id, @scheme_id, @type, @string_values, @int_value, @negate);

-- name: DeleteSchemeManualChecks :exec
DELETE FROM scheme_manual_check WHERE scheme_id = @scheme_id;

-- name: InsertSchemeManualCheck :exec
INSERT INTO scheme_manual_check (id, scheme_id, i18n_key, check_text, sort_order)
VALUES (@id, @scheme_id, @i18n_key, @check_text, @sort_order);

-- name: GetArtisanMatchFacts :one
-- Everything MatchSchemes needs about one artisan, in one row. craft_ids and
-- cluster/shg membership are computed via correlated subqueries so this
-- stays a single round trip.
SELECT
    a.state_code,
    a.years_of_experience,
    a.social_category,
    (a.pehchan_id IS NOT NULL)::boolean AS has_pehchan_id,
    (a.pm_vishwakarma_id IS NOT NULL)::boolean AS has_pm_vishwakarma_id,
    (a.primary_cluster_id IS NOT NULL)::boolean AS is_cluster_member,
    EXISTS (SELECT 1 FROM shg_member sm WHERE sm.artisan_id = a.id) AS is_shg_member,
    COALESCE(
        (SELECT array_agg(DISTINCT p.craft_id) FROM product p WHERE p.artisan_id = a.id),
        '{}'
    )::uuid[] AS craft_ids
FROM artisan a
WHERE a.id = @artisan_id;
