-- migrations/009_social.sql
-- +goose Up

-- What a notification is about. DB-only; the proto carries notifications as events.
CREATE TYPE notification_kind AS ENUM (
    'LOT_OFFERED', 'LOT_EXPIRING', 'LISTING_APPROVAL_DUE', 'PAYMENT_SETTLED',
    'QC_FAILED', 'DISPUTE_RAISED', 'SHIPMENT_DELIVERED', 'ARTISAN_FOLLOWED'
);

CREATE TABLE follow (
    artisan_id  uuid        NOT NULL,
    follower_id text        NOT NULL,
    source      text        NOT NULL DEFAULT 'listing-page',
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT follow_pkey PRIMARY KEY (artisan_id, follower_id),
    CONSTRAINT follow_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE
);

-- A buyer's "makers I follow" feed.
CREATE INDEX follow_follower_id_created_at_idx ON follow (follower_id, created_at DESC);

CREATE TABLE notification (
    id           uuid              NOT NULL,
    recipient_id text              NOT NULL,
    kind         notification_kind NOT NULL,
    language     language_code     NOT NULL DEFAULT 'ENGLISH',
    title        text              NOT NULL,
    body         text              NOT NULL,
    payload      jsonb             NOT NULL DEFAULT '{}'::jsonb,
    read_at      timestamptz,
    created_at   timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT notification_pkey PRIMARY KEY (id),
    CONSTRAINT notification_title_check CHECK (length(title) > 0)
);

-- The unread badge and the notification drawer both read exactly this slice.
CREATE INDEX notification_recipient_unread_idx ON notification (recipient_id, created_at DESC)
    WHERE read_at IS NULL;

-- +goose Down

DROP TABLE IF EXISTS notification;
DROP TABLE IF EXISTS follow;
DROP TYPE IF EXISTS notification_kind;
