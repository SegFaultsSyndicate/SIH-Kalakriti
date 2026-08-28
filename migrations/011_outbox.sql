-- migrations/011_outbox.sql
-- +goose Up

-- Transactional outbox. Every Kafka-producing write inserts here in the same
-- transaction as the business row; the relay publishes and stamps published_at.
CREATE TABLE outbox (
    id              uuid        NOT NULL,
    aggregate_id    uuid        NOT NULL,
    topic           text        NOT NULL,
    idempotency_key text        NOT NULL,
    payload         jsonb       NOT NULL,
    attempts        integer     NOT NULL DEFAULT 0,
    last_error      text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    published_at    timestamptz,
    CONSTRAINT outbox_pkey PRIMARY KEY (id),
    -- Re-running a handler must not enqueue the same fact twice.
    CONSTRAINT outbox_topic_idempotency_key_key UNIQUE (topic, idempotency_key),
    CONSTRAINT outbox_topic_check CHECK (length(topic) > 0),
    CONSTRAINT outbox_attempts_check CHECK (attempts >= 0)
);

-- The relay's only read: the oldest unpublished rows. Published rows fall out of
-- the index, so it stays small however large the table grows.
CREATE INDEX outbox_unpublished_idx ON outbox (created_at) WHERE published_at IS NULL;

-- +goose Down

DROP TABLE IF EXISTS outbox;
