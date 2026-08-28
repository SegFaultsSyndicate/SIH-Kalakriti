-- migrations/007_orders.sql
-- +goose Up

-- Mirrors fulfilment.v1.BulkOrderState without the UNSPECIFIED member.
CREATE TYPE bulk_order_state AS ENUM (
    'ALLOCATING', 'PARTIALLY_ALLOCATED', 'CONFIRMED', 'IN_PRODUCTION',
    'AMENDMENT_PENDING', 'COMPLETED', 'CANCELLED'
);

-- Mirrors fulfilment.v1.LotState without the UNSPECIFIED member.
CREATE TYPE lot_state AS ENUM (
    'OFFERED', 'ACCEPTED', 'DECLINED', 'EXPIRED', 'IN_PRODUCTION',
    'QC_PENDING', 'QC_FAILED', 'COMPLETED', 'REALLOCATED'
);

-- Mirrors fulfilment.v1.ReservationState without the UNSPECIFIED member.
CREATE TYPE reservation_state AS ENUM ('HELD', 'CONSUMED', 'RELEASED');

CREATE TABLE bulk_order (
    id                 uuid             NOT NULL,
    buyer_id           text             NOT NULL,
    listing_id         uuid             NOT NULL,
    product_id         uuid             NOT NULL,
    quantity           integer          NOT NULL,
    unit_price_paise   bigint           NOT NULL,
    total_value_paise  bigint           NOT NULL,
    currency_code      char(3)          NOT NULL DEFAULT 'INR',
    required_by        timestamptz      NOT NULL,
    state              bulk_order_state NOT NULL DEFAULT 'ALLOCATING',
    allocated_quantity integer          NOT NULL DEFAULT 0,
    customisations     jsonb            NOT NULL DEFAULT '{}'::jsonb,
    notes              text,
    created_by         text             NOT NULL DEFAULT 'system',
    created_at         timestamptz      NOT NULL DEFAULT now(),
    updated_at         timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT bulk_order_pkey PRIMARY KEY (id),
    CONSTRAINT bulk_order_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listing (id) ON DELETE RESTRICT,
    CONSTRAINT bulk_order_product_id_fkey FOREIGN KEY (product_id)
        REFERENCES product (id) ON DELETE RESTRICT,
    CONSTRAINT bulk_order_quantity_check CHECK (quantity > 0),
    CONSTRAINT bulk_order_unit_price_paise_check CHECK (unit_price_paise >= 0),
    CONSTRAINT bulk_order_total_value_paise_check CHECK (total_value_paise >= 0),
    CONSTRAINT bulk_order_currency_code_check CHECK (currency_code = 'INR'),
    CONSTRAINT bulk_order_allocated_quantity_check
        CHECK (allocated_quantity >= 0 AND allocated_quantity <= quantity)
);

-- The buyer's order list, newest first.
CREATE INDEX bulk_order_buyer_id_created_at_idx ON bulk_order (buyer_id, created_at DESC);
-- The allocator sweeps orders that still need coverage.
CREATE INDEX bulk_order_state_required_by_idx ON bulk_order (state, required_by)
    WHERE state IN ('ALLOCATING', 'PARTIALLY_ALLOCATED');

CREATE TABLE order_lot (
    id                      uuid        NOT NULL,
    bulk_order_id           uuid        NOT NULL,
    artisan_id              uuid        NOT NULL,
    cluster_id              uuid,
    quantity                integer     NOT NULL,
    unit_price_paise        bigint      NOT NULL,
    lot_value_paise         bigint      NOT NULL,
    currency_code           char(3)     NOT NULL DEFAULT 'INR',
    state                   lot_state   NOT NULL DEFAULT 'OFFERED',
    offered_at              timestamptz NOT NULL DEFAULT now(),
    responds_by             timestamptz NOT NULL,
    accepted_at             timestamptz,
    promised_ship_date      timestamptz,
    progress_pct            integer     NOT NULL DEFAULT 0,
    capacity_reservation_id uuid,
    decline_reason          text,
    reallocated_from_lot_id uuid,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT order_lot_pkey PRIMARY KEY (id),
    CONSTRAINT order_lot_bulk_order_id_fkey FOREIGN KEY (bulk_order_id)
        REFERENCES bulk_order (id) ON DELETE CASCADE,
    CONSTRAINT order_lot_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE RESTRICT,
    CONSTRAINT order_lot_cluster_id_fkey FOREIGN KEY (cluster_id)
        REFERENCES cluster (id) ON DELETE SET NULL,
    CONSTRAINT order_lot_reallocated_from_lot_id_fkey FOREIGN KEY (reallocated_from_lot_id)
        REFERENCES order_lot (id) ON DELETE SET NULL,
    CONSTRAINT order_lot_quantity_check CHECK (quantity > 0),
    CONSTRAINT order_lot_unit_price_paise_check CHECK (unit_price_paise >= 0),
    CONSTRAINT order_lot_lot_value_paise_check CHECK (lot_value_paise >= 0),
    CONSTRAINT order_lot_currency_code_check CHECK (currency_code = 'INR'),
    CONSTRAINT order_lot_progress_pct_check CHECK (progress_pct BETWEEN 0 AND 100),
    CONSTRAINT order_lot_responds_by_check CHECK (responds_by > offered_at),
    -- An accepted lot has both an acceptance stamp and a promised ship date; a lot
    -- that was never accepted has neither.
    CONSTRAINT order_lot_acceptance_check CHECK (
        (accepted_at IS NULL AND promised_ship_date IS NULL) OR
        (accepted_at IS NOT NULL AND promised_ship_date IS NOT NULL)
    ),
    CONSTRAINT order_lot_decline_reason_check
        CHECK ((state = 'DECLINED') = (decline_reason IS NOT NULL))
);

-- The artisan's inbox: offers awaiting an answer, soonest deadline first.
CREATE INDEX order_lot_artisan_id_state_idx ON order_lot (artisan_id, state);
-- Loading an order hydrates all of its lots.
CREATE INDEX order_lot_bulk_order_id_idx ON order_lot (bulk_order_id);
-- The expiry sweeper scans only offers that can still lapse.
CREATE INDEX order_lot_responds_by_idx ON order_lot (responds_by) WHERE state = 'OFFERED';

CREATE TABLE capacity_reservation (
    id           uuid              NOT NULL,
    artisan_id   uuid              NOT NULL,
    listing_id   uuid              NOT NULL,
    lot_id       uuid              NOT NULL,
    units        integer           NOT NULL,
    period_start timestamptz       NOT NULL,
    period_end   timestamptz       NOT NULL,
    expires_at   timestamptz       NOT NULL,
    state        reservation_state NOT NULL DEFAULT 'HELD',
    created_at   timestamptz       NOT NULL DEFAULT now(),
    updated_at   timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT capacity_reservation_pkey PRIMARY KEY (id),
    CONSTRAINT capacity_reservation_lot_id_key UNIQUE (lot_id),
    CONSTRAINT capacity_reservation_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT capacity_reservation_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listing (id) ON DELETE CASCADE,
    CONSTRAINT capacity_reservation_lot_id_fkey FOREIGN KEY (lot_id)
        REFERENCES order_lot (id) ON DELETE CASCADE,
    CONSTRAINT capacity_reservation_units_check CHECK (units > 0),
    CONSTRAINT capacity_reservation_period_check CHECK (period_end > period_start)
);

ALTER TABLE order_lot
    ADD CONSTRAINT order_lot_capacity_reservation_id_fkey FOREIGN KEY (capacity_reservation_id)
        REFERENCES capacity_reservation (id) ON DELETE SET NULL;

-- Summing live holds for one artisan-month is the allocator's capacity check. The
-- predicate cannot include expires_at > now(), which is not immutable, so the
-- expiry comparison stays in the query and only HELD rows are indexed.
CREATE INDEX capacity_reservation_held_idx
    ON capacity_reservation (artisan_id, listing_id, period_start, expires_at)
    WHERE state = 'HELD';

-- +goose Down

ALTER TABLE order_lot DROP CONSTRAINT IF EXISTS order_lot_capacity_reservation_id_fkey;
DROP TABLE IF EXISTS capacity_reservation;
DROP TABLE IF EXISTS order_lot;
DROP TABLE IF EXISTS bulk_order;
DROP TYPE IF EXISTS reservation_state;
DROP TYPE IF EXISTS lot_state;
DROP TYPE IF EXISTS bulk_order_state;
