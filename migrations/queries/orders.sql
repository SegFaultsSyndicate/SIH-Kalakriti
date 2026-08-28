-- migrations/queries/orders.sql

-- Get-or-insert in one round trip, same idiom as idempotency.sql's
-- GetOrInsertIdempotencyKey: is_new is true only for the caller that actually
-- inserted the row, so the service can skip re-emitting the created event and
-- outbox row on a replay.
-- name: CreateBulkOrder :one
WITH inserted AS (
    INSERT INTO bulk_order (
        id, buyer_id, listing_id, product_id, quantity, unit_price_paise,
        total_value_paise, required_by, customisations, notes, idempotency_key,
        created_by, escrow_enabled
    ) VALUES (
        @id, @buyer_id, @listing_id, @product_id, @quantity, @unit_price_paise,
        @total_value_paise, @required_by, @customisations, sqlc.narg('notes'),
        @idempotency_key, @created_by, @escrow_enabled
    )
    ON CONFLICT ON CONSTRAINT bulk_order_buyer_id_idempotency_key_key DO NOTHING
    RETURNING *
)
SELECT *, true AS is_new FROM inserted
UNION ALL
SELECT o.*, false AS is_new FROM bulk_order o
WHERE o.buyer_id = @buyer_id AND o.idempotency_key = @idempotency_key
  AND NOT EXISTS (SELECT 1 FROM inserted);

-- name: GetBulkOrder :one
SELECT * FROM bulk_order WHERE id = @id;

-- name: GetBulkOrderForUpdate :one
SELECT * FROM bulk_order WHERE id = @id FOR UPDATE;

-- name: TransitionBulkOrderState :one
UPDATE bulk_order
SET state = @next_state, updated_at = now()
WHERE id = @id AND state = @expected_state
RETURNING *;

-- name: UpdateBulkOrderAllocation :one
UPDATE bulk_order
SET allocated_quantity = @allocated_quantity,
    state              = @state,
    updated_at         = now()
WHERE id = @id
RETURNING *;

-- name: InsertOrderLot :one
INSERT INTO order_lot (
    id, bulk_order_id, artisan_id, cluster_id, quantity, unit_price_paise,
    lot_value_paise, responds_by, reallocated_from_lot_id
) VALUES (
    @id, @bulk_order_id, @artisan_id, sqlc.narg('cluster_id'), @quantity,
    @unit_price_paise, @lot_value_paise, @responds_by, sqlc.narg('reallocated_from_lot_id')
)
RETURNING *;

-- name: GetOrderLot :one
SELECT * FROM order_lot WHERE id = @id;

-- name: ListLotsByOrder :many
SELECT * FROM order_lot WHERE bulk_order_id = @bulk_order_id ORDER BY offered_at, id;

-- name: ListOpenOffersForArtisan :many
SELECT * FROM order_lot
WHERE artisan_id = @artisan_id AND state = 'OFFERED'
ORDER BY responds_by
LIMIT @page_size;

-- Guarded transition; zero rows means another writer moved the lot first.
-- name: TransitionLotState :one
UPDATE order_lot
SET state          = @next_state,
    decline_reason = CASE WHEN @next_state::lot_state = 'DECLINED'
                          THEN sqlc.narg('decline_reason')::text ELSE decline_reason END,
    updated_at     = now()
WHERE id = @id AND state = @expected_state
RETURNING *;

-- name: AcceptOrderLot :one
UPDATE order_lot
SET state              = 'ACCEPTED',
    accepted_at        = now(),
    promised_ship_date = @promised_ship_date,
    updated_at         = now()
WHERE id = @id AND state = 'OFFERED' AND responds_by > now()
RETURNING *;

-- name: SetLotProgress :one
UPDATE order_lot
SET progress_pct = @progress_pct,
    state        = @state,
    updated_at   = now()
WHERE id = @id AND state IN ('ACCEPTED', 'IN_PRODUCTION')
RETURNING *;

-- Offers past their deadline, claimed one batch at a time by the expiry sweeper.
-- name: ExpireLots :many
UPDATE order_lot
SET state = 'EXPIRED', updated_at = now()
WHERE id IN (
    SELECT id FROM order_lot
    WHERE state = 'OFFERED' AND responds_by <= now()
    ORDER BY responds_by
    LIMIT @batch_size
    FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: SumDeliveredLots :one
SELECT COALESCE(SUM(quantity), 0)::bigint AS delivered_quantity,
       COALESCE(SUM(lot_value_paise), 0)::bigint AS delivered_value_paise
FROM order_lot
WHERE bulk_order_id = @bulk_order_id AND state = 'COMPLETED';

-- name: ReserveCapacity :one
INSERT INTO capacity_reservation (
    id, artisan_id, listing_id, lot_id, units, period_start, period_end, expires_at
) VALUES (
    @id, @artisan_id, @listing_id, @lot_id, @units, @period_start, @period_end, @expires_at
)
RETURNING *;

-- Live holds for one artisan-month. The expiry test lives here rather than in the
-- index predicate because now() is not immutable.
-- name: SumHeldCapacity :one
SELECT COALESCE(SUM(units), 0)::bigint AS held_units
FROM capacity_reservation
WHERE artisan_id = @artisan_id
  AND listing_id = @listing_id
  AND period_start = @period_start
  AND state = 'HELD'
  AND expires_at > now();

-- name: SetReservationState :one
UPDATE capacity_reservation
SET state = @state, updated_at = now()
WHERE id = @id
RETURNING *;

-- name: ReleaseReservationForLot :one
UPDATE capacity_reservation
SET state = 'RELEASED', updated_at = now()
WHERE lot_id = @lot_id AND state = 'HELD'
RETURNING *;

-- name: ReleaseExpiredReservations :many
UPDATE capacity_reservation
SET state = 'RELEASED', updated_at = now()
WHERE id IN (
    SELECT id FROM capacity_reservation
    WHERE state = 'HELD' AND expires_at <= now()
    ORDER BY expires_at
    LIMIT @batch_size
    FOR UPDATE SKIP LOCKED
)
RETURNING *;
