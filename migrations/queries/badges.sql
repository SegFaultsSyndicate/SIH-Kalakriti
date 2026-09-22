-- migrations/queries/badges.sql

-- name: ListBadgeCatalog :many
SELECT * FROM badge WHERE active ORDER BY sort_order;

-- name: GetBadgeByCode :one
SELECT * FROM badge WHERE code = @code AND (tier IS NULL OR tier = sqlc.narg('tier')::badge_tier);

-- name: ListArtisanBadges :many
SELECT ab.artisan_id, ab.badge_id, ab.granted_at, ab.granted_by, ab.evidence,
       b.code, b.kind, b.tier, b.icon_name, b.metric, b.threshold, b.sort_order
FROM artisan_badge ab
JOIN badge b ON b.id = ab.badge_id
WHERE ab.artisan_id = @artisan_id AND ab.revoked_at IS NULL
ORDER BY b.sort_order;

-- name: GetBadgeProgress :many
SELECT metric, value, updated_at FROM artisan_badge_progress WHERE artisan_id = @artisan_id;

-- name: UpsertBadgeProgress :one
INSERT INTO artisan_badge_progress (artisan_id, metric, value, updated_at)
VALUES (@artisan_id, @metric, @value, now())
ON CONFLICT (artisan_id, metric) DO UPDATE SET value = @value, updated_at = now()
RETURNING *;

-- name: GrantEligibleEarnedBadges :many
-- Idempotent by construction: re-running with the same (artisan_id, value)
-- after a Kafka redelivery inserts nothing new because of the NOT EXISTS
-- guard plus ON CONFLICT DO NOTHING.
INSERT INTO artisan_badge (artisan_id, badge_id, granted_at, granted_by)
SELECT @artisan_id, b.id, now(), 'system'
FROM badge b
WHERE b.kind = 'EARNED' AND b.active AND b.metric = @metric AND b.threshold <= @value
  AND NOT EXISTS (
      SELECT 1 FROM artisan_badge ab WHERE ab.artisan_id = @artisan_id AND ab.badge_id = b.id
  )
ON CONFLICT (artisan_id, badge_id) DO NOTHING
RETURNING *;

-- name: InsertConferredGrant :one
INSERT INTO artisan_badge (artisan_id, badge_id, granted_at, granted_by, evidence)
VALUES (@artisan_id, @badge_id, now(), @granted_by, sqlc.narg('evidence'))
ON CONFLICT (artisan_id, badge_id) DO UPDATE SET
    revoked_at = NULL, revoked_by = NULL, revoke_reason = NULL,
    granted_at = now(), granted_by = @granted_by, evidence = sqlc.narg('evidence')
RETURNING *;

-- name: RevokeGrant :execrows
UPDATE artisan_badge SET revoked_at = now(), revoked_by = @revoked_by, revoke_reason = @revoke_reason
WHERE artisan_id = @artisan_id AND badge_id = @badge_id AND revoked_at IS NULL;

-- name: CountPublishedListings :one
SELECT count(*)::bigint FROM listing WHERE artisan_id = @artisan_id AND state = 'PUBLISHED';

-- name: CountSealedProvenance :one
SELECT count(*)::bigint FROM provenance_record WHERE artisan_id = @artisan_id;

-- name: CountAcceptedLots :one
SELECT count(*)::bigint FROM order_lot
WHERE artisan_id = @artisan_id AND state NOT IN ('OFFERED', 'DECLINED', 'EXPIRED');

-- name: CountCompletedLots :one
SELECT count(*)::bigint FROM order_lot WHERE artisan_id = @artisan_id AND state = 'COMPLETED';
