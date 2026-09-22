-- migrations/queries/trends.sql

-- name: CreateTrendLink :one
INSERT INTO trend_link (
    id, title, description, url, source_type, craft_id,
    thumbnail_url, embed_html, embed_metadata, pinned, created_by, curator_role, curator_name, expires_at
) VALUES (
    @id, @title, @description, @url, @source_type, sqlc.narg('craft_id'),
    sqlc.narg('thumbnail_url'), sqlc.narg('embed_html'),
    sqlc.narg('embed_metadata'), @pinned, @created_by, @curator_role, @curator_name, sqlc.narg('expires_at')
)
RETURNING *;

-- name: GetTrendLink :one
SELECT * FROM trend_link WHERE id = @id;

-- name: ListTrendLinks :many
SELECT tl.*, c.display_name AS craft_name
FROM trend_link tl
LEFT JOIN craft c ON c.id = tl.craft_id
WHERE (sqlc.narg('craft_id')::uuid IS NULL OR tl.craft_id = sqlc.narg('craft_id'))
  AND (sqlc.narg('source_type')::trend_source_type IS NULL OR tl.source_type = sqlc.narg('source_type'))
  AND (NOT @exclude_expired::boolean OR tl.expires_at IS NULL OR tl.expires_at > now())
  AND (sqlc.narg('after')::uuid IS NULL OR tl.id > sqlc.narg('after'))
ORDER BY tl.pinned DESC, tl.created_at DESC
LIMIT @page_size;

-- name: DeleteTrendLink :execrows
DELETE FROM trend_link WHERE id = @id;

-- name: PinTrendLink :one
UPDATE trend_link SET pinned = @pinned
WHERE id = @id
RETURNING *;
