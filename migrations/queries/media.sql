-- migrations/queries/media.sql

-- The row is written before the bytes exist: size_bytes holds the size the client
-- declared, and ConfirmUpload overwrites it with what the bucket actually holds.
-- name: CreateMediaPending :one
INSERT INTO media (
    id, artisan_id, product_id, bucket, object_key, kind, mime_type, size_bytes,
    sha256_hex, width_px, height_px, duration_ms, source, state
) VALUES (
    @id, @artisan_id, sqlc.narg('product_id'), @bucket, @object_key, @kind, @mime_type,
    @size_bytes, sqlc.narg('sha256_hex'), sqlc.narg('width_px'), sqlc.narg('height_px'),
    sqlc.narg('duration_ms'), @source, 'PENDING'
)
RETURNING *;

-- name: CreateMedia :one
INSERT INTO media (
    id, artisan_id, product_id, bucket, object_key, kind, mime_type, size_bytes,
    sha256_hex, width_px, height_px, duration_ms, source, state, confirmed_at
) VALUES (
    @id, @artisan_id, sqlc.narg('product_id'), @bucket, @object_key, @kind, @mime_type,
    @size_bytes, sqlc.narg('sha256_hex'), sqlc.narg('width_px'), sqlc.narg('height_px'),
    sqlc.narg('duration_ms'), @source, 'READY', now()
)
RETURNING *;

-- name: GetMedia :one
SELECT * FROM media WHERE id = @id;

-- Dedupe on re-upload: same artisan, same bytes, no second object in MinIO. A
-- PENDING row is not a hit  -  its bytes may never arrive.
-- name: GetMediaByHash :one
SELECT * FROM media
WHERE artisan_id = @artisan_id AND sha256_hex = @sha256_hex AND state <> 'PENDING'
ORDER BY created_at
LIMIT 1;

-- name: ListMediaByProduct :many
SELECT * FROM media WHERE product_id = @product_id ORDER BY uploaded_at, id;

-- Ownership and kind for a set of ids, which is all the catalog needs to decide
-- whether a listing may attach them and in which role.
-- name: ListMediaByIDs :many
SELECT id, artisan_id, kind FROM media WHERE id = ANY(@ids::uuid[]) ORDER BY id;

-- The artisan filter is the guard: attaching another maker's photographs to your
-- own product updates nothing, and the caller is told how many ids were refused.
-- name: AttachMediaToProduct :execrows
UPDATE media SET product_id = @product_id::uuid
WHERE id = ANY(@ids::uuid[]) AND artisan_id = @artisan_id::uuid;

-- Guarded transition, the same shape as the listing's: the caller states the state
-- it believes the row is in, so two confirmations cannot both emit an event.
-- name: TransitionMediaState :one
UPDATE media
SET state          = @next_state,
    size_bytes     = COALESCE(sqlc.narg('size_bytes'), size_bytes),
    sha256_hex     = COALESCE(sqlc.narg('sha256_hex'), sha256_hex),
    confirmed_at   = CASE WHEN @next_state::media_state = 'PENDING'
                          THEN confirmed_at ELSE COALESCE(confirmed_at, now()) END,
    failure_reason = CASE WHEN @next_state::media_state = 'FAILED'
                          THEN sqlc.narg('failure_reason')::text ELSE NULL END
WHERE id = @id AND state = @expected_state
RETURNING *;

-- The enhancement result: a second object beside the original, and the model
-- bundle that produced it, recorded together with the move to READY.
-- name: SetMediaEnhanced :one
UPDATE media
SET state               = 'READY',
    enhanced_object_key = @enhanced_object_key,
    model_version       = sqlc.narg('model_version'),
    failure_reason      = NULL,
    confirmed_at        = COALESCE(confirmed_at, now())
WHERE id = @id AND state = @expected_state
RETURNING *;

-- The reaper's claim: rows whose bytes never arrived. Ordered oldest first so a
-- capped batch always makes progress on the worst offenders.
-- name: ListStalePendingMedia :many
SELECT * FROM media
WHERE state = 'PENDING' AND created_at < @before
ORDER BY created_at
LIMIT @batch_size;

-- name: DeleteMedia :execrows
DELETE FROM media WHERE id = @id;
