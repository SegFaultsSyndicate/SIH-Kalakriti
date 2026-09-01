-- migrations/019_bulk_order_events.sql
-- +goose Up

-- bulk_order gained no idempotency guard in migration 007: CreateBulkOrder is a
-- multi-step saga entry point (feasibility check, then an insert, then an
-- outbox row, same transaction), so it needs the same DB-level idempotency the
-- rest of the project uses for anything that must survive a crash mid-flight  - 
-- pkg/idempotency.Do is deliberately not used here, per the project's standing
-- rule that it cannot safely wrap an operation like this one. A caller-supplied
-- key scoped to the buyer is the guard: a retried CreateBulkOrder with the same
-- key hits this constraint and the repo returns the row that already exists
-- instead of creating a second one.
ALTER TABLE bulk_order ADD COLUMN idempotency_key text;
ALTER TABLE bulk_order ADD CONSTRAINT bulk_order_buyer_id_idempotency_key_key
    UNIQUE (buyer_id, idempotency_key);

-- bulk_order_event is the append-only audit trail for the fulfilment saga: one
-- row per state transition on the order or one of its lots. This is separate
-- from the outbox (migrations/011_outbox.sql)  -  the outbox is a transient
-- publish queue that rows are deleted from implicitly by never being
-- re-selected, this table is a permanent record kept even after the outbox
-- row that carried the same fact to Kafka has long been published.
CREATE TABLE bulk_order_event (
    id            uuid        NOT NULL,
    bulk_order_id uuid        NOT NULL,
    lot_id        uuid,
    event_type    text        NOT NULL,
    payload       jsonb       NOT NULL DEFAULT '{}'::jsonb,
    occurred_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT bulk_order_event_pkey PRIMARY KEY (id),
    CONSTRAINT bulk_order_event_bulk_order_id_fkey FOREIGN KEY (bulk_order_id)
        REFERENCES bulk_order (id) ON DELETE CASCADE,
    CONSTRAINT bulk_order_event_lot_id_fkey FOREIGN KEY (lot_id)
        REFERENCES order_lot (id) ON DELETE SET NULL
);

-- WatchOrder's since-replay and the order timeline screen both read one
-- order's events oldest first.
CREATE INDEX bulk_order_event_bulk_order_id_occurred_at_idx
    ON bulk_order_event (bulk_order_id, occurred_at);

-- +goose Down

DROP TABLE IF EXISTS bulk_order_event;
ALTER TABLE bulk_order DROP CONSTRAINT IF EXISTS bulk_order_buyer_id_idempotency_key_key;
ALTER TABLE bulk_order DROP COLUMN IF EXISTS idempotency_key;
