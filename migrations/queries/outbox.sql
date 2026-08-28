-- migrations/queries/outbox.sql

-- Inserted in the same transaction as the business write. The conflict clause makes a
-- replayed handler a no-op rather than a duplicate publish.
-- name: InsertOutbox :execrows
INSERT INTO outbox (id, aggregate_id, topic, idempotency_key, payload)
VALUES (@id, @aggregate_id, @topic, @idempotency_key, @payload)
ON CONFLICT ON CONSTRAINT outbox_topic_idempotency_key_key DO NOTHING;

-- The relay's claim. SKIP LOCKED lets several relay instances share the table without
-- blocking each other; the rows stay locked until the publishing transaction ends.
-- name: FetchUnpublishedOutbox :many
SELECT * FROM outbox
WHERE published_at IS NULL
ORDER BY created_at, id
LIMIT @batch_size
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :execrows
UPDATE outbox SET published_at = now() WHERE id = ANY(@ids::uuid[]);

-- name: RecordOutboxFailure :exec
UPDATE outbox SET attempts = attempts + 1, last_error = @last_error WHERE id = @id;

-- name: DeletePublishedOutboxBefore :execrows
DELETE FROM outbox WHERE published_at IS NOT NULL AND published_at < @before;
