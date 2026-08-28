-- migrations/queries/notification_delivery.sql

-- name: InsertNotificationDelivery :one
INSERT INTO notification_delivery (id, notification_id, channel, recipient, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateDeliveryStatus :exec
UPDATE notification_delivery
SET status = $2, sent_at = $3, attempt_count = attempt_count + 1, last_error = $4
WHERE id = $1;

-- name: ListPendingDeliveries :many
SELECT * FROM notification_delivery
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: GetDeliveriesForNotification :many
SELECT * FROM notification_delivery
WHERE notification_id = $1
ORDER BY created_at DESC;
