-- migrations/027_webhooks.sql
-- +goose Up
-- +goose StatementBegin

-- Webhook subscriptions: buyers register URLs to receive event notifications
CREATE TABLE webhook_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscriber_id UUID NOT NULL,  -- user who owns this subscription
    subscriber_type TEXT NOT NULL CHECK (subscriber_type IN ('buyer', 'seller', 'admin')),
    url TEXT NOT NULL,
    secret TEXT NOT NULL,  -- HMAC secret for signature verification
    events TEXT[] NOT NULL,  -- e.g., {'order.created', 'order.shipped', 'order.completed'}
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_success_at TIMESTAMPTZ,
    last_failure_at TIMESTAMPTZ,
    consecutive_failures INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_webhook_subscriptions_subscriber ON webhook_subscriptions(subscriber_id);
CREATE INDEX idx_webhook_subscriptions_active ON webhook_subscriptions(active) WHERE active = true;

-- Webhook deliveries: queue of outgoing webhooks with retry logic
CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    event TEXT NOT NULL,
    payload JSONB NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 10,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'succeeded', 'failed')),
    http_status INT,
    response_body TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ
);

CREATE INDEX idx_webhook_deliveries_pending ON webhook_deliveries(next_retry_at)
    WHERE status = 'pending' AND attempts < max_attempts;
CREATE INDEX idx_webhook_deliveries_subscription ON webhook_deliveries(subscription_id);
CREATE INDEX idx_webhook_deliveries_event ON webhook_deliveries(event);

-- Function to enqueue webhook delivery when order is created
CREATE OR REPLACE FUNCTION webhook_enqueue_order_created()
RETURNS TRIGGER AS $$
BEGIN
    -- Find all active subscriptions for 'order.created' event
    INSERT INTO webhook_deliveries (subscription_id, event, payload)
    SELECT
        ws.id,
        'order.created',
        jsonb_build_object(
            'event', 'order.created',
            'timestamp', NOW(),
            'data', jsonb_build_object(
                'id', NEW.id,
                'buyer_id', NEW.buyer_id,
                'status', NEW.state,
                'created_at', NEW.created_at
            )
        )
    FROM webhook_subscriptions ws
    WHERE ws.active = true
      AND 'order.created' = ANY(ws.events)
      AND ws.subscriber_id::text = NEW.buyer_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_webhook_order_created
    AFTER INSERT ON bulk_order
    FOR EACH ROW
    EXECUTE FUNCTION webhook_enqueue_order_created();

-- Function to enqueue webhook delivery when order status changes
CREATE OR REPLACE FUNCTION webhook_enqueue_order_updated()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.state IS DISTINCT FROM NEW.state THEN
        -- Find all active subscriptions for 'order.updated' event
        INSERT INTO webhook_deliveries (subscription_id, event, payload)
        SELECT
            ws.id,
            'order.updated',
            jsonb_build_object(
                'event', 'order.updated',
                'timestamp', NOW(),
                'data', jsonb_build_object(
                    'id', NEW.id,
                    'buyer_id', NEW.buyer_id,
                    'old_status', OLD.state,
                    'new_status', NEW.state,
                    'updated_at', NOW()
                )
            )
        FROM webhook_subscriptions ws
        WHERE ws.active = true
          AND 'order.updated' = ANY(ws.events)
          AND ws.subscriber_id::text = NEW.buyer_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_webhook_order_updated
    AFTER UPDATE ON bulk_order
    FOR EACH ROW
    EXECUTE FUNCTION webhook_enqueue_order_updated();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trigger_webhook_order_updated ON bulk_order;
DROP TRIGGER IF EXISTS trigger_webhook_order_created ON bulk_order;

DROP FUNCTION IF EXISTS webhook_enqueue_order_updated();
DROP FUNCTION IF EXISTS webhook_enqueue_order_created();

DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhook_subscriptions;

-- +goose StatementEnd
