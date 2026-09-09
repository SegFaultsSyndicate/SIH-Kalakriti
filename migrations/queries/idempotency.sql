-- migrations/queries/idempotency.sql

-- Get-or-insert in one round trip. `inserted` is true for the first caller, which then
-- does the work; false for a replay, which returns the stored response instead.
-- The CTE's conflict clause is what makes the whole thing atomic.
-- name: GetOrInsertIdempotencyKey :one
WITH inserted AS (
    INSERT INTO idempotency_key (id, scope, key, request_hash, expires_at)
    VALUES (@id, @scope, @key, @request_hash, @expires_at)
    ON CONFLICT ON CONSTRAINT idempotency_key_scope_key_key DO NOTHING
    RETURNING id, scope, key, request_hash, response, created_at, expires_at
)
SELECT id, scope, key, request_hash, response, created_at, expires_at, true AS is_new
FROM inserted
UNION ALL
SELECT k.id, k.scope, k.key, k.request_hash, k.response, k.created_at, k.expires_at, false AS is_new
FROM idempotency_key k
WHERE k.scope = @scope AND k.key = @key
  AND NOT EXISTS (SELECT 1 FROM inserted);

-- name: SetIdempotentResponse :execrows
UPDATE idempotency_key SET response = @response
WHERE scope = @scope AND key = @key AND response IS NULL;

-- name: DeleteExpiredIdempotencyKeys :execrows
DELETE FROM idempotency_key WHERE expires_at <= now();

-- name: DeleteIdempotencyKey :execrows
DELETE FROM idempotency_key WHERE scope = @scope AND key = @key AND response IS NULL;
