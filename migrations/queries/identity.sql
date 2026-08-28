-- migrations/queries/identity.sql
-- Cluster and self-help-group reads and writes. Artisan queries live in artisan.sql.

-- name: CreateCluster :one
INSERT INTO cluster (
    id, name, state_code, district, block, village, pincode, coordinator_phone_e164
) VALUES (
    @id, @name, @state_code, sqlc.narg('district'), sqlc.narg('block'),
    sqlc.narg('village'), sqlc.narg('pincode'), sqlc.narg('coordinator_phone_e164')
)
RETURNING *;

-- name: GetCluster :one
SELECT * FROM cluster WHERE id = @id;

-- name: UpdateCluster :one
UPDATE cluster
SET name                   = COALESCE(sqlc.narg('name'), name),
    state_code             = COALESCE(sqlc.narg('state_code'), state_code),
    district               = COALESCE(sqlc.narg('district'), district),
    block                  = COALESCE(sqlc.narg('block'), block),
    village                = COALESCE(sqlc.narg('village'), village),
    pincode                = COALESCE(sqlc.narg('pincode'), pincode),
    coordinator_phone_e164 = COALESCE(sqlc.narg('coordinator_phone_e164'), coordinator_phone_e164),
    updated_at             = now()
WHERE id = @id
RETURNING *;

-- The cluster page shows a headline member count next to the roster.
-- name: CountClusterMembers :one
SELECT count(*) FROM cluster_member WHERE cluster_id = @cluster_id;

-- name: RemoveClusterMember :execrows
DELETE FROM cluster_member WHERE cluster_id = @cluster_id AND artisan_id = @artisan_id;

-- Joined so a roster page needs one round trip rather than an N+1 on artisan.
-- Keyset pagination on artisan_id, which is a UUIDv7 and so sorts by creation time.
-- name: ListClusterMembers :many
SELECT m.cluster_id, m.artisan_id, m.role, m.joined_at, a.display_name
FROM cluster_member m
JOIN artisan a ON a.id = m.artisan_id
WHERE m.cluster_id = @cluster_id
  AND (sqlc.narg('after')::uuid IS NULL OR m.artisan_id > sqlc.narg('after'))
ORDER BY m.artisan_id
LIMIT @page_size;

-- name: GetClusterMember :one
SELECT m.cluster_id, m.artisan_id, m.role, m.joined_at, a.display_name
FROM cluster_member m
JOIN artisan a ON a.id = m.artisan_id
WHERE m.cluster_id = @cluster_id AND m.artisan_id = @artisan_id;

-- name: CreateShg :one
INSERT INTO shg (id, name, registration_no, cluster_id, signatory_artisan_id)
VALUES (@id, @name, @registration_no, sqlc.narg('cluster_id'), sqlc.narg('signatory_artisan_id'))
RETURNING *;

-- name: GetShg :one
SELECT * FROM shg WHERE id = @id;

-- name: GetShgByRegistrationNo :one
SELECT * FROM shg WHERE registration_no = @registration_no;

-- name: UpdateShg :one
UPDATE shg
SET name                 = COALESCE(sqlc.narg('name'), name),
    cluster_id           = COALESCE(sqlc.narg('cluster_id'), cluster_id),
    signatory_artisan_id = COALESCE(sqlc.narg('signatory_artisan_id'), signatory_artisan_id),
    updated_at           = now()
WHERE id = @id
RETURNING *;

-- Roster replacement is delete-all then insert-all inside one transaction; the
-- shares-sum-to-100 constraint trigger is DEFERRED, so the intermediate empty
-- state never trips it.
-- name: DeleteShgMembers :execrows
DELETE FROM shg_member WHERE shg_id = @shg_id;

-- name: InsertShgMember :exec
INSERT INTO shg_member (shg_id, artisan_id, share_pct)
VALUES (@shg_id, @artisan_id, @share_pct)
ON CONFLICT ON CONSTRAINT shg_member_pkey
DO UPDATE SET share_pct = EXCLUDED.share_pct;

-- name: RemoveShgMember :execrows
DELETE FROM shg_member WHERE shg_id = @shg_id AND artisan_id = @artisan_id;

-- name: ListShgMembers :many
SELECT m.shg_id, m.artisan_id, m.share_pct, m.joined_at, a.display_name
FROM shg_member m
JOIN artisan a ON a.id = m.artisan_id
WHERE m.shg_id = @shg_id
ORDER BY m.share_pct DESC, m.artisan_id;

-- Belt-and-braces read used by the service to report a bad split with a clear
-- message before the deferred trigger would abort the transaction at COMMIT.
-- name: SumShgMemberShares :one
SELECT COALESCE(sum(share_pct), 0)::integer AS total FROM shg_member WHERE shg_id = @shg_id;
