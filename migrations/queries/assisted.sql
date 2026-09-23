-- migrations/queries/assisted.sql
-- Staff accounts (FIELD_AGENT/CLUSTER_OFFICER/MINISTRY), assisted links, and
-- the assisted-mode audit trail.

-- name: GetActiveStaffByPhone :one
SELECT * FROM staff_account WHERE phone_e164 = @phone_e164 AND active;

-- name: GetStaffAccount :one
SELECT * FROM staff_account WHERE id = @id;

-- name: CreateStaffAccount :one
INSERT INTO staff_account (
    id, phone_e164, display_name, role, state_code, district, cluster_id, csc_id, created_by
) VALUES (
    @id, @phone_e164, @display_name, @role, sqlc.narg('state_code'), sqlc.narg('district'),
    sqlc.narg('cluster_id'), sqlc.narg('csc_id'), @created_by
)
RETURNING *;

-- name: ListStaffAccounts :many
SELECT * FROM staff_account
WHERE (sqlc.narg('state_code')::text IS NULL OR state_code = sqlc.narg('state_code'))
ORDER BY active DESC, created_at DESC
LIMIT 500;

-- name: SetStaffActive :one
UPDATE staff_account SET active = @active WHERE id = @id RETURNING *;

-- name: UpsertAssistedLink :one
-- Re-linking after a revoke reuses the (agent, artisan) row: a fresh consent
-- clears revoked_at rather than inserting a duplicate the unique key forbids.
INSERT INTO assisted_link (id, agent_id, artisan_id, consent_method, consent_ref, consent_at, needs_review)
VALUES (@id, @agent_id, @artisan_id, @consent_method, @consent_ref, now(), @needs_review)
ON CONFLICT (agent_id, artisan_id) DO UPDATE SET
    consent_method = EXCLUDED.consent_method,
    consent_ref    = EXCLUDED.consent_ref,
    consent_at     = now(),
    needs_review   = EXCLUDED.needs_review,
    reviewed_by    = NULL,
    reviewed_at    = NULL,
    revoked_at     = NULL
RETURNING *;

-- name: HasActiveAssistedLink :one
SELECT EXISTS (
    SELECT 1 FROM assisted_link l
    JOIN staff_account s ON s.id = l.agent_id
    WHERE l.agent_id = @agent_id AND l.artisan_id = @artisan_id
      AND l.revoked_at IS NULL AND s.active
)::boolean AS linked;

-- name: RevokeAssistedLink :execrows
-- Only the artisan themself revokes (consent withdrawal); scoped by artisan_id
-- so one artisan can never revoke another's link by guessing an id.
UPDATE assisted_link SET revoked_at = now()
WHERE id = @id AND artisan_id = @artisan_id AND revoked_at IS NULL;

-- name: ListAgentArtisans :many
SELECT
    a.id, a.display_name, a.village, a.district, a.state_code, a.photo_media_id,
    l.consent_method, l.needs_review, l.created_at AS linked_at,
    COALESCE(act.last_activity_at, 'epoch'::timestamptz)::timestamptz AS last_activity_at,
    (SELECT count(*) FROM listing li WHERE li.artisan_id = a.id AND li.state = 'DRAFT')::bigint AS draft_count
FROM assisted_link l
JOIN artisan a ON a.id = l.artisan_id
LEFT JOIN LATERAL (
    SELECT al.timestamp AS last_activity_at FROM audit_log al
    WHERE al.subject_id = a.id ORDER BY al.timestamp DESC LIMIT 1
) act ON true
WHERE l.agent_id = @agent_id AND l.revoked_at IS NULL
ORDER BY a.display_name;

-- name: ListArtisanHelpers :many
SELECT l.id AS link_id, s.id AS agent_id, s.display_name, s.role, s.csc_id,
       l.consent_method, l.created_at AS linked_at
FROM assisted_link l
JOIN staff_account s ON s.id = l.agent_id
WHERE l.artisan_id = @artisan_id AND l.revoked_at IS NULL
ORDER BY l.created_at;

-- name: ListLinksNeedingReview :many
SELECT l.id, l.agent_id, s.display_name AS agent_name, l.artisan_id, a.display_name AS artisan_name,
       a.district, a.state_code, l.consent_ref, l.created_at
FROM assisted_link l
JOIN staff_account s ON s.id = l.agent_id
JOIN artisan a ON a.id = l.artisan_id
WHERE l.needs_review AND l.reviewed_at IS NULL AND l.revoked_at IS NULL
  AND (sqlc.narg('state_code')::text IS NULL OR a.state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('district')::text IS NULL OR a.district = sqlc.narg('district'))
ORDER BY l.created_at
LIMIT 200;

-- name: MarkLinkReviewed :execrows
UPDATE assisted_link SET reviewed_by = @reviewed_by, reviewed_at = now()
WHERE id = @id AND needs_review AND reviewed_at IS NULL;

-- name: InsertAssistedAudit :exec
INSERT INTO audit_log (actor_id, actor_type, action, resource_type, resource_id, subject_id, metadata)
VALUES (@actor_id, 'user', @action, @resource_type, @resource_id, @subject_id, @metadata);

-- name: ListAgentProductivity :many
-- One row per agent: artisans onboarded (active links), listings created on
-- behalf (CreateListing audit rows), and their most recent on-behalf write.
SELECT
    s.id, s.display_name, s.role, s.state_code, s.district, s.csc_id, s.active,
    (SELECT count(*) FROM assisted_link l WHERE l.agent_id = s.id AND l.revoked_at IS NULL)::bigint AS artisans_onboarded,
    (SELECT count(*) FROM audit_log al WHERE al.actor_id = s.id
        AND al.action = '/catalog.v1.CatalogService/CreateListing')::bigint AS listings_created,
    COALESCE(act.last_active_at, 'epoch'::timestamptz)::timestamptz AS last_active_at
FROM staff_account s
LEFT JOIN LATERAL (
    SELECT al.timestamp AS last_active_at FROM audit_log al
    WHERE al.actor_id = s.id ORDER BY al.timestamp DESC LIMIT 1
) act ON true
WHERE s.role IN ('FIELD_AGENT', 'CLUSTER_OFFICER')
  AND (sqlc.narg('state_code')::text IS NULL OR s.state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('district')::text IS NULL OR s.district = sqlc.narg('district'))
ORDER BY listings_created DESC, s.display_name;

-- name: GetListingHelper :one
-- Who, if anyone, created this listing on the artisan's behalf.
SELECT s.id, s.display_name, s.csc_id
FROM audit_log al
JOIN staff_account s ON s.id = al.actor_id
WHERE al.resource_id = @listing_id AND al.action = '/catalog.v1.CatalogService/CreateListing'
ORDER BY al.timestamp
LIMIT 1;
