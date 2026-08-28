-- migrations/026_fraud_detection.sql
-- +goose Up
-- +goose StatementBegin

-- Fraud detection: flag suspicious patterns for manual review
CREATE TABLE fraud_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type TEXT NOT NULL CHECK (resource_type IN ('user', 'artisan', 'bulk_order', 'listing')),
    resource_id UUID NOT NULL,
    flag_type TEXT NOT NULL,  -- e.g., 'new_buyer_large_order', 'phone_reuse', 'rapid_listing_churn'
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    description TEXT NOT NULL,
    metadata JSONB,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'reviewing', 'resolved_ok', 'resolved_fraud')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID,
    resolution_notes TEXT
);

-- Indexes for admin dashboard
CREATE INDEX idx_fraud_flags_status ON fraud_flags(status) WHERE status IN ('pending', 'reviewing');
CREATE INDEX idx_fraud_flags_resource ON fraud_flags(resource_type, resource_id);
CREATE INDEX idx_fraud_flags_created_at ON fraud_flags(created_at DESC);
CREATE INDEX idx_fraud_flags_severity ON fraud_flags(severity) WHERE severity IN ('high', 'critical');

-- Function to check for new buyer large order (>₹50k order from buyer with <7 days account age)
CREATE OR REPLACE FUNCTION fraud_check_new_buyer_large_order()
RETURNS TRIGGER AS $$
DECLARE
    buyer_created_at TIMESTAMPTZ;
    account_age_days INT;
    order_value_paise BIGINT;
BEGIN
    -- Get buyer account creation date
    SELECT created_at INTO buyer_created_at
    FROM users
    WHERE id = NEW.buyer_id;

    account_age_days := EXTRACT(EPOCH FROM (NOW() - buyer_created_at)) / 86400;

    -- Calculate order value (sum of lot quantities × prices)
    SELECT COALESCE(SUM(ol.quantity * ol.unit_price_paise), 0) INTO order_value_paise
    FROM order_lots ol
    WHERE ol.bulk_order_id = NEW.id;

    -- Flag if: buyer account <7 days old AND order >₹50,000
    IF account_age_days < 7 AND order_value_paise > 5000000 THEN
        INSERT INTO fraud_flags (
            resource_type,
            resource_id,
            flag_type,
            severity,
            description,
            metadata
        ) VALUES (
            'bulk_order',
            NEW.id,
            'new_buyer_large_order',
            'high',
            format('New buyer (account age: %s days) placed large order (₹%s)',
                   account_age_days,
                   order_value_paise / 100.0),
            jsonb_build_object(
                'buyer_id', NEW.buyer_id,
                'account_age_days', account_age_days,
                'order_value_paise', order_value_paise
            )
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_fraud_check_new_buyer_large_order
    AFTER INSERT ON bulk_orders
    FOR EACH ROW
    EXECUTE FUNCTION fraud_check_new_buyer_large_order();

-- Function to check for phone number reuse (same phone used for >5 accounts)
CREATE OR REPLACE FUNCTION fraud_check_phone_reuse()
RETURNS TRIGGER AS $$
DECLARE
    phone_usage_count INT;
BEGIN
    -- Count how many artisans use this phone
    SELECT COUNT(*) INTO phone_usage_count
    FROM artisan
    WHERE phone_e164 = NEW.phone_e164;

    -- Flag if phone used by >5 artisans
    IF phone_usage_count > 5 THEN
        INSERT INTO fraud_flags (
            resource_type,
            resource_id,
            flag_type,
            severity,
            description,
            metadata
        ) VALUES (
            'artisan',
            NEW.id,
            'phone_reuse',
            'medium',
            format('Phone number %s used by %s artisan accounts',
                   NEW.phone_e164,
                   phone_usage_count),
            jsonb_build_object(
                'phone_e164', NEW.phone_e164,
                'usage_count', phone_usage_count
            )
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_fraud_check_phone_reuse
    AFTER INSERT ON artisan
    FOR EACH ROW
    EXECUTE FUNCTION fraud_check_phone_reuse();

-- Function to check for rapid listing creation (>20 listings in <1 hour)
CREATE OR REPLACE FUNCTION fraud_check_rapid_listing_creation()
RETURNS TRIGGER AS $$
DECLARE
    recent_listing_count INT;
BEGIN
    -- Count listings by this artisan in last hour
    SELECT COUNT(*) INTO recent_listing_count
    FROM listing
    WHERE artisan_id = NEW.artisan_id
      AND created_at > NOW() - INTERVAL '1 hour';

    -- Flag if >20 listings in last hour
    IF recent_listing_count > 20 THEN
        INSERT INTO fraud_flags (
            resource_type,
            resource_id,
            flag_type,
            severity,
            description,
            metadata
        ) VALUES (
            'artisan',
            NEW.artisan_id,
            'rapid_listing_creation',
            'medium',
            format('Artisan created %s listings in the last hour', recent_listing_count),
            jsonb_build_object(
                'artisan_id', NEW.artisan_id,
                'listing_count', recent_listing_count
            )
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_fraud_check_rapid_listing_creation
    AFTER INSERT ON listing
    FOR EACH ROW
    EXECUTE FUNCTION fraud_check_rapid_listing_creation();

-- Function to check for rapid order cancellations (>3 cancellations within 1 hour of placement)
CREATE OR REPLACE FUNCTION fraud_check_rapid_cancellation()
RETURNS TRIGGER AS $$
DECLARE
    quick_cancellation_count INT;
BEGIN
    -- Only check if order was cancelled
    IF NEW.status = 'cancelled' AND OLD.status != 'cancelled' THEN
        -- Check if cancelled within 1 hour of placement
        IF NEW.created_at > NOW() - INTERVAL '1 hour' THEN
            -- Count quick cancellations by this buyer
            SELECT COUNT(*) INTO quick_cancellation_count
            FROM bulk_orders
            WHERE buyer_id = NEW.buyer_id
              AND status = 'cancelled'
              AND created_at > NOW() - INTERVAL '1 hour';

            -- Flag if >3 quick cancellations
            IF quick_cancellation_count > 3 THEN
                INSERT INTO fraud_flags (
                    resource_type,
                    resource_id,
                    flag_type,
                    severity,
                    description,
                    metadata
                ) VALUES (
                    'user',
                    NEW.buyer_id,
                    'rapid_cancellation',
                    'high',
                    format('Buyer cancelled %s orders within 1 hour of placement', quick_cancellation_count),
                    jsonb_build_object(
                        'buyer_id', NEW.buyer_id,
                        'cancellation_count', quick_cancellation_count
                    )
                );
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_fraud_check_rapid_cancellation
    AFTER UPDATE ON bulk_orders
    FOR EACH ROW
    EXECUTE FUNCTION fraud_check_rapid_cancellation();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trigger_fraud_check_rapid_cancellation ON bulk_orders;
DROP TRIGGER IF EXISTS trigger_fraud_check_rapid_listing_creation ON listing;
DROP TRIGGER IF EXISTS trigger_fraud_check_phone_reuse ON artisan;
DROP TRIGGER IF EXISTS trigger_fraud_check_new_buyer_large_order ON bulk_orders;

DROP FUNCTION IF EXISTS fraud_check_rapid_cancellation();
DROP FUNCTION IF EXISTS fraud_check_rapid_listing_creation();
DROP FUNCTION IF EXISTS fraud_check_phone_reuse();
DROP FUNCTION IF EXISTS fraud_check_new_buyer_large_order();

DROP TABLE IF EXISTS fraud_flags;

-- +goose StatementEnd
