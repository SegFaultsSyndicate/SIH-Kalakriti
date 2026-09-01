-- migrations/queries/payments.sql

-- name: CreatePaymentSplit :one
INSERT INTO payment_split (
    id, bulk_order_id, gross_total_paise, commission_total_paise, net_total_paise
) VALUES (
    @id, @bulk_order_id, @gross_total_paise, @commission_total_paise, @net_total_paise
)
ON CONFLICT ON CONSTRAINT payment_split_bulk_order_id_key DO NOTHING
RETURNING *;

-- name: GetPaymentSplitByOrder :one
SELECT * FROM payment_split WHERE bulk_order_id = @bulk_order_id;

-- name: InsertPaymentSplitLine :one
INSERT INTO payment_split_line (
    id, payment_split_id, payee_id, payee_type, lot_id,
    gross_amount_paise, commission_paise, net_amount_paise, payout_ref
) VALUES (
    @id, @payment_split_id, @payee_id, @payee_type, @lot_id,
    @gross_amount_paise, @commission_paise, @net_amount_paise, @payout_ref
)
ON CONFLICT ON CONSTRAINT payment_split_line_split_lot_payee_key DO NOTHING
RETURNING *;

-- name: ListUnsettledSplitLines :many
SELECT * FROM payment_split_line
WHERE payment_split_id = @payment_split_id AND settled_at IS NULL
ORDER BY id;

-- name: MarkSplitLineSettled :one
UPDATE payment_split_line
SET settled_at = now(), settlement_ref = sqlc.narg('settlement_ref'), updated_at = now()
WHERE id = @id AND settled_at IS NULL
RETURNING *;

-- name: MarkPaymentSplitSettled :one
UPDATE payment_split SET settled_at = now(), updated_at = now()
WHERE payment_split.id = @id AND settled_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM payment_split_line l
      WHERE l.payment_split_id = payment_split.id AND l.settled_at IS NULL
  )
RETURNING *;

-- name: CreateEscrowMilestone :one
INSERT INTO escrow_milestone (id, bulk_order_id, lot_id, trigger, amount_paise)
VALUES (@id, @bulk_order_id, sqlc.narg('lot_id'), @trigger, @amount_paise)
ON CONFLICT ON CONSTRAINT escrow_milestone_order_lot_trigger_key DO NOTHING
RETURNING *;

-- name: ListPendingMilestones :many
SELECT * FROM escrow_milestone
WHERE bulk_order_id = @bulk_order_id AND released = false
ORDER BY trigger;

-- name: ReleaseEscrowMilestone :one
UPDATE escrow_milestone
SET released = true, released_at = now(), release_ref = sqlc.narg('release_ref'), updated_at = now()
WHERE id = @id AND released = false
RETURNING *;

-- Batch 13: get-or-insert so a retried RequestPaymentSplit call never creates
-- a second header row and never re-derives amounts from a possibly-changed
-- world  -  same idiom as CreateBulkOrder (orders.sql).
-- name: CreatePaymentSplitIdempotent :one
WITH inserted AS (
    INSERT INTO payment_split (id, bulk_order_id, gross_total_paise, commission_total_paise, net_total_paise)
    VALUES (@id, @bulk_order_id, @gross_total_paise, @commission_total_paise, @net_total_paise)
    ON CONFLICT ON CONSTRAINT payment_split_bulk_order_id_key DO NOTHING
    RETURNING *
)
SELECT *, true AS is_new FROM inserted
UNION ALL
SELECT p.*, false AS is_new FROM payment_split p
WHERE p.bulk_order_id = @bulk_order_id AND NOT EXISTS (SELECT 1 FROM inserted);

-- name: InsertPaymentSplitLineIdempotent :one
WITH inserted AS (
    INSERT INTO payment_split_line (
        id, payment_split_id, payee_id, payee_type, lot_id,
        gross_amount_paise, commission_paise, net_amount_paise, payout_ref
    ) VALUES (
        @id, @payment_split_id, @payee_id, @payee_type, @lot_id,
        @gross_amount_paise, @commission_paise, @net_amount_paise, @payout_ref
    )
    ON CONFLICT ON CONSTRAINT payment_split_line_split_lot_payee_key DO NOTHING
    RETURNING *
)
SELECT *, true AS is_new FROM inserted
UNION ALL
SELECT l.*, false AS is_new FROM payment_split_line l
WHERE l.payment_split_id = @payment_split_id AND l.lot_id = @lot_id AND l.payee_id = @payee_id
  AND NOT EXISTS (SELECT 1 FROM inserted);

-- name: ListSplitLinesByOrder :many
SELECT l.* FROM payment_split_line l
JOIN payment_split p ON p.id = l.payment_split_id
WHERE p.bulk_order_id = @bulk_order_id
ORDER BY l.created_at, l.id;

-- name: ListMilestonesForLot :many
SELECT * FROM escrow_milestone WHERE lot_id = @lot_id AND released = false ORDER BY trigger;
