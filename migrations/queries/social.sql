-- migrations/queries/social.sql

-- name: InsertNotification :one
INSERT INTO notification (id, recipient_id, kind, language, title, body, payload, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetNotification :one
SELECT * FROM notification WHERE id = $1;

-- name: ListNotificationsForUser :many
SELECT * FROM notification
WHERE recipient_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListUnreadNotificationsForUser :many
SELECT * FROM notification
WHERE recipient_id = $1 AND read_at IS NULL
ORDER BY created_at DESC
LIMIT $2;

-- name: MarkNotificationRead :exec
UPDATE notification
SET read_at = $2
WHERE id = $1 AND read_at IS NULL;

-- name: CountUnreadForUser :one
SELECT count(*) FROM notification
WHERE recipient_id = $1 AND read_at IS NULL;

-- name: GetFollowers :many
SELECT follower_id FROM follow
WHERE artisan_id = $1
ORDER BY created_at DESC;

-- name: InsertFollow :exec
INSERT INTO follow (artisan_id, follower_id, source, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (artisan_id, follower_id) DO NOTHING;
