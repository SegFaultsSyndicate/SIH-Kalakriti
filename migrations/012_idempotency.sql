-- migrations/012_idempotency.sql
-- +goose Up

-- Replay guard for mutating RPCs. The handler inserts before doing work; a
-- conflicting insert means the caller is retrying and gets the stored response.
CREATE TABLE idempotency_key (
    id           uuid        NOT NULL,
    scope        text        NOT NULL,
    key          text        NOT NULL,
    request_hash char(64)    NOT NULL,
    response     jsonb,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL DEFAULT now() + interval '7 days',
    CONSTRAINT idempotency_key_pkey PRIMARY KEY (id),
    CONSTRAINT idempotency_key_scope_key_key UNIQUE (scope, key),
    CONSTRAINT idempotency_key_scope_check CHECK (length(scope) > 0),
    CONSTRAINT idempotency_key_key_check CHECK (length(key) > 0),
    CONSTRAINT idempotency_key_request_hash_check CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT idempotency_key_expires_at_check CHECK (expires_at > created_at)
);

-- The reaper deletes expired rows; without this it degenerates to a full scan.
CREATE INDEX idempotency_key_expires_at_idx ON idempotency_key (expires_at);

-- +goose Down

DROP TABLE IF EXISTS idempotency_key;
