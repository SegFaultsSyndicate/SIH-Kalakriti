-- migrations/021_compensation.sql
-- +goose Up

-- Rework and dropout tracking on order_lot. dropout_reason is free text
-- recorded when a lot is reallocated because the artisan withdrew (as
-- opposed to a QC-driven reallocation, which already has its reason on the
-- qc_defect rows)  -  kept separate from decline_reason because
-- order_lot_decline_reason_check ties that column to state = 'DECLINED'
-- specifically. rework_deadline is set when a lot first fails QC with a
-- non-critical defect; a second failure or a rework past this deadline both
-- lead to REALLOCATED (service layer, see domain.lotTransitions).
ALTER TABLE order_lot
    ADD COLUMN dropout_reason  text,
    ADD COLUMN rework_deadline timestamptz;

-- Escrow only ever applies to a bulk order (never a single-artisan direct
-- sale, which this schema does not model in bulk_order at all), and is
-- opt-in per order.
ALTER TABLE bulk_order
    ADD COLUMN escrow_enabled boolean NOT NULL DEFAULT false;

-- Mirrors the amendment shape fulfilment.v1 will grow for the buyer
-- amendment flow: a reduced quantity or an extended deadline, never both in
-- one proposal.
CREATE TYPE amendment_type AS ENUM ('REDUCE_QUANTITY', 'EXTEND_DEADLINE');
CREATE TYPE amendment_status AS ENUM ('PENDING', 'ACCEPTED', 'DECLINED');

CREATE TABLE bulk_order_amendment (
    id                    uuid             NOT NULL,
    bulk_order_id         uuid             NOT NULL,
    amendment_type        amendment_type   NOT NULL,
    proposed_quantity     integer,
    proposed_required_by  timestamptz,
    reason                text             NOT NULL,
    status                amendment_status NOT NULL DEFAULT 'PENDING',
    decided_at            timestamptz,
    created_at            timestamptz      NOT NULL DEFAULT now(),
    updated_at            timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT bulk_order_amendment_pkey PRIMARY KEY (id),
    CONSTRAINT bulk_order_amendment_bulk_order_id_fkey FOREIGN KEY (bulk_order_id)
        REFERENCES bulk_order (id) ON DELETE CASCADE,
    CONSTRAINT bulk_order_amendment_quantity_check
        CHECK ((amendment_type = 'REDUCE_QUANTITY') = (proposed_quantity IS NOT NULL)),
    CONSTRAINT bulk_order_amendment_deadline_check
        CHECK ((amendment_type = 'EXTEND_DEADLINE') = (proposed_required_by IS NOT NULL)),
    CONSTRAINT bulk_order_amendment_decided_check
        CHECK ((status = 'PENDING') = (decided_at IS NULL))
);

CREATE INDEX bulk_order_amendment_bulk_order_id_idx
    ON bulk_order_amendment (bulk_order_id, created_at DESC);
-- Only one amendment may be awaiting a buyer decision at a time.
CREATE UNIQUE INDEX bulk_order_amendment_one_pending_idx
    ON bulk_order_amendment (bulk_order_id) WHERE status = 'PENDING';

-- +goose Down

DROP TABLE IF EXISTS bulk_order_amendment;
DROP TYPE IF EXISTS amendment_status;
DROP TYPE IF EXISTS amendment_type;

ALTER TABLE bulk_order DROP COLUMN IF EXISTS escrow_enabled;

ALTER TABLE order_lot
    DROP COLUMN IF EXISTS rework_deadline,
    DROP COLUMN IF EXISTS dropout_reason;
