-- migrations/queries/artisan.sql

-- name: CreateArtisan :one
INSERT INTO artisan (
    id, user_id, display_name, phone_e164, pehchan_id, pm_vishwakarma_id,
    primary_cluster_id, state_code, district, block, village, pincode,
    languages, years_of_experience, bio, created_by
) VALUES (
    @id, sqlc.narg('user_id'), @display_name, @phone_e164, sqlc.narg('pehchan_id'),
    sqlc.narg('pm_vishwakarma_id'), sqlc.narg('primary_cluster_id'), @state_code,
    sqlc.narg('district'), sqlc.narg('block'), sqlc.narg('village'), sqlc.narg('pincode'),
    @languages, sqlc.narg('years_of_experience'), sqlc.narg('bio'), @created_by
)
RETURNING *;

-- name: GetArtisan :one
SELECT * FROM artisan WHERE id = @id;

-- name: GetArtisanByPhone :one
SELECT * FROM artisan WHERE phone_e164 = @phone_e164;

-- name: GetArtisanByPehchanID :one
SELECT * FROM artisan WHERE pehchan_id = @pehchan_id;

-- name: GetArtisanByPMVishwakarmaID :one
SELECT * FROM artisan WHERE pm_vishwakarma_id = @pm_vishwakarma_id;

-- name: UpdateArtisan :one
UPDATE artisan
SET display_name        = COALESCE(sqlc.narg('display_name'), display_name),
    primary_cluster_id  = COALESCE(sqlc.narg('primary_cluster_id'), primary_cluster_id),
    state_code          = COALESCE(sqlc.narg('state_code'), state_code),
    district            = COALESCE(sqlc.narg('district'), district),
    block               = COALESCE(sqlc.narg('block'), block),
    village             = COALESCE(sqlc.narg('village'), village),
    pincode             = COALESCE(sqlc.narg('pincode'), pincode),
    languages           = COALESCE(sqlc.narg('languages'), languages),
    years_of_experience = COALESCE(sqlc.narg('years_of_experience'), years_of_experience),
    bio                 = COALESCE(sqlc.narg('bio'), bio),
    photo_media_id      = COALESCE(sqlc.narg('photo_media_id'), photo_media_id),
    updated_at          = now()
WHERE id = @id
RETURNING *;

-- name: SetArtisanVerified :one
UPDATE artisan SET verified = @verified, updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteArtisan :execrows
DELETE FROM artisan WHERE id = @id;

-- Keyset pagination on UUIDv7, which sorts by creation time, so no separate cursor column.
-- name: ListArtisansByCluster :many
SELECT a.* FROM artisan a
WHERE a.primary_cluster_id = @cluster_id
  AND (sqlc.narg('after')::uuid IS NULL OR a.id > sqlc.narg('after'))
ORDER BY a.id
LIMIT @page_size;

-- Candidate scan for the allocator: everyone who practises a craft, most experienced first.
-- name: ListArtisansByCraft :many
SELECT a.* FROM artisan a
JOIN artisan_craft ac ON ac.artisan_id = a.id
WHERE ac.craft_id = @craft_id
  AND (NOT @verified_only::boolean OR a.verified)
ORDER BY a.years_of_experience DESC NULLS LAST, a.id
LIMIT @page_size;

-- name: AddArtisanCraft :exec
INSERT INTO artisan_craft (artisan_id, craft_id, is_primary)
VALUES (@artisan_id, @craft_id, @is_primary)
ON CONFLICT ON CONSTRAINT artisan_craft_pkey
DO UPDATE SET is_primary = EXCLUDED.is_primary;

-- name: UpsertClusterMember :exec
INSERT INTO cluster_member (cluster_id, artisan_id, role)
VALUES (@cluster_id, @artisan_id, @role)
ON CONFLICT ON CONSTRAINT cluster_member_pkey
DO UPDATE SET role = EXCLUDED.role;

-- name: ListArtisanCrafts :many
SELECT c.* FROM craft c
JOIN artisan_craft ac ON ac.craft_id = c.id
WHERE ac.artisan_id = @artisan_id
ORDER BY ac.is_primary DESC, c.display_name;

-- name: RemoveArtisanCraft :execrows
DELETE FROM artisan_craft WHERE artisan_id = @artisan_id AND craft_id = @craft_id;

-- SHG membership is written with a share percentage; see InsertShgMember in identity.sql.
-- name: GetShgForArtisan :one
SELECT s.* FROM shg s
JOIN shg_member m ON m.shg_id = s.id
WHERE m.artisan_id = @artisan_id
LIMIT 1;
