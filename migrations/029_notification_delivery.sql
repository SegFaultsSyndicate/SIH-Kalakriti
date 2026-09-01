-- migrations/029_notification_delivery.sql
-- +goose Up

-- Delivery attempts per channel (WhatsApp, email, push). Distinct from the
-- notification itself — one notification can have delivery attempts to multiple
-- channels, and a failed delivery can be retried.
CREATE TABLE notification_delivery (
    id              uuid        NOT NULL,
    notification_id uuid        NOT NULL,
    channel         text        NOT NULL, -- 'whatsapp', 'email', 'push', 'sms'
    recipient       text        NOT NULL, -- phone number, email, device token
    status          text        NOT NULL DEFAULT 'pending', -- 'pending', 'sent', 'failed'
    attempt_count   int         NOT NULL DEFAULT 0,
    last_error      text,
    sent_at         timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT notification_delivery_pkey PRIMARY KEY (id),
    CONSTRAINT notification_delivery_notification_id_fkey FOREIGN KEY (notification_id)
        REFERENCES notification (id) ON DELETE CASCADE,
    CONSTRAINT notification_delivery_channel_check CHECK (channel IN ('whatsapp', 'email', 'push', 'sms')),
    CONSTRAINT notification_delivery_status_check CHECK (status IN ('pending', 'sent', 'failed'))
);

CREATE INDEX notification_delivery_notification_id_idx ON notification_delivery (notification_id);
CREATE INDEX notification_delivery_status_created_at_idx ON notification_delivery (status, created_at)
    WHERE status = 'pending';

-- +goose Down

DROP TABLE IF EXISTS notification_delivery;
