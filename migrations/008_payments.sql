-- migrations/008_payments.sql
-- +goose Up

-- Mirrors fulfilment.v1.PayeeType without the UNSPECIFIED member.
CREATE TYPE payee_type AS ENUM ('ARTISAN', 'SELF_HELP_GROUP', 'CLUSTER');

-- Mirrors fulfilment.v1.MilestoneTrigger without the UNSPECIFIED member.
CREATE TYPE milestone_trigger AS ENUM (
    'ADVANCE', 'PRODUCTION_START', 'QC_PASSED', 'DISPATCH', 'DELIVERY'
);

CREATE TABLE payment_split (
    id                     uuid        NOT NULL,
    bulk_order_id          uuid        NOT NULL,
    gross_total_paise      bigint      NOT NULL,
    commission_total_paise bigint      NOT NULL,
    net_total_paise        bigint      NOT NULL,
    currency_code          char(3)     NOT NULL DEFAULT 'INR',
    settled_at             timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payment_split_pkey PRIMARY KEY (id),
    CONSTRAINT payment_split_bulk_order_id_key UNIQUE (bulk_order_id),
    CONSTRAINT payment_split_bulk_order_id_fkey FOREIGN KEY (bulk_order_id)
        REFERENCES bulk_order (id) ON DELETE CASCADE,
    CONSTRAINT payment_split_gross_total_paise_check CHECK (gross_total_paise >= 0),
    CONSTRAINT payment_split_commission_total_paise_check CHECK (commission_total_paise >= 0),
    CONSTRAINT payment_split_net_total_paise_check CHECK (net_total_paise >= 0),
    CONSTRAINT payment_split_currency_code_check CHECK (currency_code = 'INR'),
    -- Money is conserved: nothing may be created or lost between gross and net.
    CONSTRAINT payment_split_totals_check
        CHECK (net_total_paise + commission_total_paise = gross_total_paise)
);

CREATE TABLE payment_split_line (
    id                uuid        NOT NULL,
    payment_split_id  uuid        NOT NULL,
    payee_id          text        NOT NULL,
    payee_type        payee_type  NOT NULL,
    lot_id            uuid        NOT NULL,
    gross_amount_paise bigint     NOT NULL,
    commission_paise  bigint      NOT NULL,
    net_amount_paise  bigint      NOT NULL,
    payout_ref        text        NOT NULL,
    settlement_ref    text,
    settled_at        timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payment_split_line_pkey PRIMARY KEY (id),
    CONSTRAINT payment_split_line_split_lot_payee_key UNIQUE (payment_split_id, lot_id, payee_id),
    CONSTRAINT payment_split_line_payment_split_id_fkey FOREIGN KEY (payment_split_id)
        REFERENCES payment_split (id) ON DELETE CASCADE,
    CONSTRAINT payment_split_line_lot_id_fkey FOREIGN KEY (lot_id)
        REFERENCES order_lot (id) ON DELETE RESTRICT,
    CONSTRAINT payment_split_line_gross_amount_paise_check CHECK (gross_amount_paise >= 0),
    CONSTRAINT payment_split_line_commission_paise_check CHECK (commission_paise >= 0),
    CONSTRAINT payment_split_line_net_amount_paise_check CHECK (net_amount_paise >= 0),
    CONSTRAINT payment_split_line_amounts_check
        CHECK (net_amount_paise + commission_paise = gross_amount_paise),
    CONSTRAINT payment_split_line_settlement_check
        CHECK ((settled_at IS NULL) = (settlement_ref IS NULL))
);

-- The artisan earnings screen lists everything paid to one payee.
CREATE INDEX payment_split_line_payee_id_created_at_idx
    ON payment_split_line (payee_id, created_at DESC);
-- The settlement worker picks up lines that have not moved yet.
CREATE INDEX payment_split_line_unsettled_idx ON payment_split_line (payment_split_id)
    WHERE settled_at IS NULL;

CREATE TABLE escrow_milestone (
    id            uuid              NOT NULL,
    bulk_order_id uuid              NOT NULL,
    lot_id        uuid,
    trigger       milestone_trigger NOT NULL,
    amount_paise  bigint            NOT NULL,
    currency_code char(3)           NOT NULL DEFAULT 'INR',
    released      boolean           NOT NULL DEFAULT false,
    released_at   timestamptz,
    release_ref   text,
    created_at    timestamptz       NOT NULL DEFAULT now(),
    updated_at    timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT escrow_milestone_pkey PRIMARY KEY (id),
    CONSTRAINT escrow_milestone_order_lot_trigger_key
        UNIQUE NULLS NOT DISTINCT (bulk_order_id, lot_id, trigger),
    CONSTRAINT escrow_milestone_bulk_order_id_fkey FOREIGN KEY (bulk_order_id)
        REFERENCES bulk_order (id) ON DELETE CASCADE,
    CONSTRAINT escrow_milestone_lot_id_fkey FOREIGN KEY (lot_id)
        REFERENCES order_lot (id) ON DELETE CASCADE,
    CONSTRAINT escrow_milestone_amount_paise_check CHECK (amount_paise >= 0),
    CONSTRAINT escrow_milestone_currency_code_check CHECK (currency_code = 'INR'),
    -- A released tranche always records when and against what reference.
    CONSTRAINT escrow_milestone_release_check CHECK (
        (released = false AND released_at IS NULL AND release_ref IS NULL) OR
        (released = true  AND released_at IS NOT NULL AND release_ref IS NOT NULL)
    )
);

-- Progress events release tranches by lot and trigger.
CREATE INDEX escrow_milestone_lot_id_trigger_idx ON escrow_milestone (lot_id, trigger)
    WHERE released = false;

-- +goose Down

DROP TABLE IF EXISTS escrow_milestone;
DROP TABLE IF EXISTS payment_split_line;
DROP TABLE IF EXISTS payment_split;
DROP TYPE IF EXISTS milestone_trigger;
DROP TYPE IF EXISTS payee_type;
