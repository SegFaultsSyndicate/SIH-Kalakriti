-- migrations/queries/fulfilment_saga.sql

-- name: InsertBulkOrderEvent :exec
INSERT INTO bulk_order_event (id, bulk_order_id, lot_id, event_type, payload)
VALUES (@id, @bulk_order_id, sqlc.narg('lot_id'), @event_type, @payload);

-- name: ListBulkOrderEventsSince :many
-- WatchOrder's replay: every event on one order at or after `since`, oldest
-- first. since is nullable  -  a null replays the whole history.
SELECT * FROM bulk_order_event
WHERE bulk_order_id = @bulk_order_id
  AND (sqlc.narg('since')::timestamptz IS NULL OR occurred_at >= sqlc.narg('since'))
ORDER BY occurred_at, id;

-- name: GetReservationForLot :one
SELECT * FROM capacity_reservation WHERE lot_id = @lot_id;

-- name: GetListingForFulfilment :one
-- The saga's own view of a listing: craft, artisan, price and typical lead
-- time. Joins product for craft_id since listing itself only carries
-- product_id.
SELECT
    l.id                AS listing_id,
    l.product_id,
    l.artisan_id,
    l.price_paise       AS unit_price_paise,
    COALESCE(l.lead_time_days, 0)::integer AS typical_lead_time_days,
    p.craft_id,
    l.owner_type,
    l.owner_shg_id
FROM listing l
JOIN product p ON p.id = l.product_id
WHERE l.id = @listing_id;

-- name: InsertQCResult :exec
INSERT INTO qc_result (id, lot_id, inspector_id, passed, notes, media_ids, inspected_at)
VALUES (@id, @lot_id, @inspector_id, @passed, sqlc.narg('notes'), @media_ids, @inspected_at);

-- name: InsertQCDefect :exec
INSERT INTO qc_defect (id, qc_result_id, code, description, severity, media_ids, affected_units)
VALUES (@id, @qc_result_id, @code, @description, @severity, @media_ids, @affected_units);

-- name: ExpireLotsByID :many
-- Companion to ReleaseExpiredReservations (orders.sql): that query releases
-- the expired capacity holds, this one moves the OFFERED lots each of those
-- reservations was backing to EXPIRED. Kept as two statements rather than
-- one because a lot without a live reservation (should not happen given the
-- 1:1 lot/reservation coupling this batch establishes, but the schema does
-- not enforce it) must not be silently expired by a reservation sweep that
-- was never really its own.
UPDATE order_lot
SET state = 'EXPIRED', updated_at = now()
WHERE id = ANY(@lot_ids::uuid[]) AND state = 'OFFERED'
RETURNING *;

-- name: ListAllocationCandidates :many
-- Candidate artisans for one craft: verified artisans whose most recent
-- listing for this craft states a capacity_per_month, minus units already
-- committed (accepted lots plus live reservations) in the requested period,
-- plus a rolling on-time completion rate over their last 20 completed lots.
--
-- ponytail: "verified technique where the buyer requires it" is implemented
-- as artisan.verified only  -  the schema has no per-technique verification
-- flag (checked: listing_attribute has no such row shape, and the proto's
-- CreateBulkOrderRequest has no technique field to check against). Upgrade
-- to a real per-(artisan, technique) verification table if/when the product
-- spec names which technique field drives this.
WITH committed AS (
    SELECT ol.artisan_id, COALESCE(SUM(ol.quantity), 0)::integer AS units
    FROM order_lot ol
    WHERE ol.state IN ('ACCEPTED', 'IN_PRODUCTION', 'QC_PENDING', 'COMPLETED')
    GROUP BY ol.artisan_id
    UNION ALL
    SELECT cr.artisan_id, COALESCE(SUM(cr.units), 0)::integer
    FROM capacity_reservation cr
    WHERE cr.state = 'HELD' AND cr.expires_at > now()
      AND cr.period_start = @period_start
    GROUP BY cr.artisan_id
),
committed_totals AS (
    SELECT artisan_id, SUM(units)::integer AS units FROM committed GROUP BY artisan_id
),
on_time AS (
    SELECT ol.artisan_id,
           AVG(CASE WHEN ol.updated_at <= ol.promised_ship_date THEN 1.0 ELSE 0.0 END) AS rate
    FROM (
        SELECT *, ROW_NUMBER() OVER (PARTITION BY artisan_id ORDER BY updated_at DESC) AS rn
        FROM order_lot
        WHERE state = 'COMPLETED'
    ) ol
    WHERE ol.rn <= 20
    GROUP BY ol.artisan_id
)
SELECT
    a.id AS artisan_id,
    a.primary_cluster_id,
    GREATEST(l.capacity_per_month - COALESCE(ct.units, 0), 0)::integer AS available_units,
    COALESCE(ot.rate, 1.0)::real AS on_time_rate,
    COALESCE(a.primary_cluster_id = @hub_cluster_id::uuid, false) AS same_cluster
FROM artisan a
JOIN listing l ON l.artisan_id = a.id
JOIN product p ON p.id = l.product_id
LEFT JOIN committed_totals ct ON ct.artisan_id = a.id
LEFT JOIN on_time ot ON ot.artisan_id = a.id
WHERE p.craft_id = @craft_id
  AND l.capacity_per_month IS NOT NULL
  AND a.verified = true
  AND (@restrict_to_preferred::boolean = false OR a.id = ANY(@preferred_artisan_ids::uuid[]))
ORDER BY a.id;

-- name: ReleaseExpiredReservationsReturningOrders :many
-- Batch 13 variant of ReleaseExpiredReservations (orders.sql) that also
-- surfaces which bulk order each expired reservation's lot belongs to, so the
-- service layer can re-run ProposeAllocation (the reoffer step) for exactly
-- the orders affected by this sweep, without a second query.
UPDATE capacity_reservation cr
SET state = 'RELEASED', updated_at = now()
FROM (
    SELECT id FROM capacity_reservation
    WHERE state = 'HELD' AND expires_at <= now()
    ORDER BY expires_at
    LIMIT @batch_size
    FOR UPDATE SKIP LOCKED
) claimed
WHERE cr.id = claimed.id
RETURNING cr.*;

-- name: ExpireLotsByIDReturningOrders :many
UPDATE order_lot
SET state = 'EXPIRED', updated_at = now()
WHERE id = ANY(@lot_ids::uuid[]) AND state = 'OFFERED'
RETURNING *;

-- name: DropoutLot :one
-- Mid-production withdrawal: the lot returns to the pool as REALLOCATED with
-- no other lot on the order touched. Guarded to the two states production can
-- actually be dropped from.
UPDATE order_lot
SET state = 'REALLOCATED', dropout_reason = sqlc.narg('dropout_reason'), updated_at = now()
WHERE id = @id AND state IN ('ACCEPTED', 'IN_PRODUCTION')
RETURNING *;

-- name: StartRework :one
-- Rework resubmission: an artisan has addressed a non-critical QC failure and
-- the lot goes back for a fresh inspection, guarded by the rework window set
-- when it first failed.
UPDATE order_lot
SET state = 'QC_PENDING', updated_at = now()
WHERE id = @id AND state = 'QC_FAILED' AND rework_deadline > now()
RETURNING *;

-- name: SetLotReworkDeadline :one
UPDATE order_lot
SET state = 'QC_FAILED', rework_deadline = @rework_deadline, updated_at = now()
WHERE id = @id AND state = 'QC_PENDING'
RETURNING *;

-- name: ReallocateLot :one
-- Shared terminal move for both a second QC failure and a rework-window
-- timeout  -  either way the lot will not deliver and its units go back to the
-- allocation pool via a fresh lot.
UPDATE order_lot
SET state = 'REALLOCATED', updated_at = now()
WHERE id = @id AND state IN ('QC_PENDING', 'QC_FAILED')
RETURNING *;

-- name: CountFailedQC :one
SELECT count(*)::integer FROM qc_result WHERE lot_id = @lot_id AND passed = false;

-- name: ListSHGMemberShares :many
SELECT artisan_id, share_pct FROM shg_member WHERE shg_id = @shg_id ORDER BY artisan_id;

-- name: CreateAmendment :one
INSERT INTO bulk_order_amendment (
    id, bulk_order_id, amendment_type, proposed_quantity, proposed_required_by, reason
) VALUES (
    @id, @bulk_order_id, @amendment_type, sqlc.narg('proposed_quantity'),
    sqlc.narg('proposed_required_by'), @reason
)
RETURNING *;

-- name: GetAmendment :one
SELECT * FROM bulk_order_amendment WHERE id = @id;

-- name: DecideAmendment :one
UPDATE bulk_order_amendment
SET status = @status, decided_at = now(), updated_at = now()
WHERE id = @id AND status = 'PENDING'
RETURNING *;

-- name: ReduceBulkOrderQuantity :one
-- Guarded to AMENDMENT_PENDING, same optimistic-concurrency shape as
-- TransitionBulkOrderState (orders.sql)  -  an amendment is only ever applied
-- to an order still waiting on it.
UPDATE bulk_order
SET quantity = @quantity, total_value_paise = @total_value_paise, state = @state, updated_at = now()
WHERE id = @id AND state = 'AMENDMENT_PENDING'
RETURNING *;

-- name: ExtendBulkOrderDeadline :one
UPDATE bulk_order
SET required_by = @required_by, state = @state, updated_at = now()
WHERE id = @id AND state = 'AMENDMENT_PENDING'
RETURNING *;
