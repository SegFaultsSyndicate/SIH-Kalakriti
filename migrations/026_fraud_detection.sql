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

-- Function to check for a large first order from a buyer (>Rs. 50k).
--
-- ponytail: the original design also gated this on buyer account age
-- (<7 days), but buyer_id here is free text from the external identity
-- provider (bulk_order.buyer_id has no local FK — see migrations/007) and
-- this schema has no local users/buyer table to read a creation date from.
-- A DB trigger cannot reach across services for it, so this flags on
-- "first bulk order over the threshold" instead. Upgrade by having core-svc
-- pass account age into the order-create call and checking it there instead
-- of in a trigger, if the false-positive rate on repeat buyers matters.
-- ponytail: also note order_lot rows for NEW.id don't exist yet at INSERT
-- time on bulk_order (lots are created later, during allocation) — the
-- original design's join to order_lot would always sum to zero. bulk_order
-- already carries its own total_value_paise at insert time, so use that
-- directly instead.
CREATE OR REPLACE FUNCTION fraud_check_new_buyer_large_order()
RETURNS TRIGGER AS $$
DECLARE
    prior_order_count INT;
BEGIN
    SELECT COUNT(*) INTO prior_order_count
    FROM bulk_order
    WHERE buyer_id = NEW.buyer_id AND id != NEW.id;

    -- Flag if: this is the buyer's first order AND it's >Rs. 50,000
    IF prior_order_count = 0 AND NEW.total_value_paise > 5000000 THEN
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
            format('New buyer placed large first order (Rs. %s)',
                   NEW.total_value_paise / 100.0),
            jsonb_build_object(
                'buyer_id', NEW.buyer_id,
                'order_value_paise', NEW.total_value_paise
            )
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_fraud_check_new_buyer_large_order
    AFTER INSERT ON bulk_order
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
    -- buyer_id is free text from the external identity provider (see
    -- migrations/007), not guaranteed UUID-shaped; resource_id below is
    -- uuid NOT NULL, so a non-UUID buyer_id would abort this transaction
    -- with invalid_text_representation. Skip the flag rather than break
    -- the cancellation for exactly the buyers this is meant to catch.
    IF NEW.state = 'CANCELLED' AND OLD.state != 'CANCELLED'
       AND NEW.buyer_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$' THEN
        -- Check if cancelled within 1 hour of placement
        IF NEW.created_at > NOW() - INTERVAL '1 hour' THEN
            -- Count quick cancellations by this buyer
            SELECT COUNT(*) INTO quick_cancellation_count
            FROM bulk_order
            WHERE buyer_id = NEW.buyer_id
              AND state = 'CANCELLED'
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
                    NEW.buyer_id::uuid,
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
    AFTER UPDATE ON bulk_order
    FOR EACH ROW
    EXECUTE FUNCTION fraud_check_rapid_cancellation();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trigger_fraud_check_rapid_cancellation ON bulk_order;
DROP TRIGGER IF EXISTS trigger_fraud_check_rapid_listing_creation ON listing;
DROP TRIGGER IF EXISTS trigger_fraud_check_phone_reuse ON artisan;
DROP TRIGGER IF EXISTS trigger_fraud_check_new_buyer_large_order ON bulk_order;

DROP FUNCTION IF EXISTS fraud_check_rapid_cancellation();
DROP FUNCTION IF EXISTS fraud_check_rapid_listing_creation();
DROP FUNCTION IF EXISTS fraud_check_phone_reuse();
DROP FUNCTION IF EXISTS fraud_check_new_buyer_large_order();

DROP TABLE IF EXISTS fraud_flags;

-- +goose StatementEnd
