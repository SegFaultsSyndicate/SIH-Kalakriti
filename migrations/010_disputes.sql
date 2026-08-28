-- migrations/010_disputes.sql
-- +goose Up

-- Mirrors events.v1.DisputeCategory without the UNSPECIFIED member.
CREATE TYPE dispute_category AS ENUM (
    'QUALITY', 'DELAY', 'QUANTITY', 'PAYMENT', 'AUTHENTICITY', 'DAMAGE'
);

-- Mirrors events.v1.DisputeResolution without the UNSPECIFIED member.
CREATE TYPE dispute_resolution AS ENUM (
    'REFUND_FULL', 'REFUND_PARTIAL', 'REWORK', 'REALLOCATE', 'REJECTED',
    'MUTUAL_AGREEMENT'
);

-- Lifecycle of a dispute. DB-only; the proto models raise and resolve as two events.
CREATE TYPE dispute_state AS ENUM ('OPEN', 'UNDER_REVIEW', 'RESOLVED', 'WITHDRAWN');

CREATE TABLE dispute (
    id               uuid             NOT NULL,
    bulk_order_id    uuid             NOT NULL,
    lot_id           uuid,
    raised_by        text             NOT NULL,
    category         dispute_category NOT NULL,
    description      text             NOT NULL,
    state            dispute_state    NOT NULL DEFAULT 'OPEN',
    resolution       dispute_resolution,
    resolved_by      text,
    adjustment_paise bigint,
    currency_code    char(3)          NOT NULL DEFAULT 'INR',
    rationale        text,
    resolved_at      timestamptz,
    created_at       timestamptz      NOT NULL DEFAULT now(),
    updated_at       timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT dispute_pkey PRIMARY KEY (id),
    CONSTRAINT dispute_bulk_order_id_fkey FOREIGN KEY (bulk_order_id)
        REFERENCES bulk_order (id) ON DELETE CASCADE,
    CONSTRAINT dispute_lot_id_fkey FOREIGN KEY (lot_id)
        REFERENCES order_lot (id) ON DELETE SET NULL,
    CONSTRAINT dispute_adjustment_paise_check CHECK (adjustment_paise IS NULL OR adjustment_paise >= 0),
    CONSTRAINT dispute_currency_code_check CHECK (currency_code = 'INR'),
    CONSTRAINT dispute_description_check CHECK (length(description) > 0),
    -- A resolved dispute carries its full outcome; an open one carries none of it.
    CONSTRAINT dispute_resolution_check CHECK (
        (state <> 'RESOLVED' AND resolution IS NULL AND resolved_by IS NULL AND resolved_at IS NULL) OR
        (state =  'RESOLVED' AND resolution IS NOT NULL AND resolved_by IS NOT NULL AND resolved_at IS NOT NULL)
    )
);

-- The operations queue works open disputes oldest first.
CREATE INDEX dispute_state_created_at_idx ON dispute (state, created_at)
    WHERE state IN ('OPEN', 'UNDER_REVIEW');
-- An order page shows every dispute raised against it.
CREATE INDEX dispute_bulk_order_id_idx ON dispute (bulk_order_id);

CREATE TABLE dispute_media (
    dispute_id uuid        NOT NULL,
    media_id   uuid        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT dispute_media_pkey PRIMARY KEY (dispute_id, media_id),
    CONSTRAINT dispute_media_dispute_id_fkey FOREIGN KEY (dispute_id)
        REFERENCES dispute (id) ON DELETE CASCADE,
    CONSTRAINT dispute_media_media_id_fkey FOREIGN KEY (media_id)
        REFERENCES media (id) ON DELETE RESTRICT
);

-- +goose Down

DROP TABLE IF EXISTS dispute_media;
DROP TABLE IF EXISTS dispute;
DROP TYPE IF EXISTS dispute_state;
DROP TYPE IF EXISTS dispute_resolution;
DROP TYPE IF EXISTS dispute_category;
