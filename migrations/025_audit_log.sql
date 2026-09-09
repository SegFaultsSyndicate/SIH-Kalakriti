-- migrations/025_audit_log.sql
-- +goose Up
-- +goose StatementBegin

-- Audit log for compliance: immutable record of sensitive operations
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_id UUID,  -- user or service account performing the action
    actor_type TEXT NOT NULL CHECK (actor_type IN ('user', 'service', 'system')),
    action TEXT NOT NULL,  -- e.g., 'payment_split_created', 'order_amended', 'qc_failed'
    resource_type TEXT NOT NULL,  -- e.g., 'payment_split', 'bulk_order', 'qc_result'
    resource_id UUID NOT NULL,
    changes JSONB,  -- before/after values for updates
    ip_address INET,
    user_agent TEXT,
    metadata JSONB  -- additional context
);

-- Indexes for common audit queries
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp DESC);
CREATE INDEX idx_audit_log_actor_id ON audit_log(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_log_resource ON audit_log(resource_type, resource_id);
CREATE INDEX idx_audit_log_action ON audit_log(action);

-- Prevent updates and deletes (append-only)
CREATE RULE audit_log_no_update AS ON UPDATE TO audit_log DO INSTEAD NOTHING;
CREATE RULE audit_log_no_delete AS ON DELETE TO audit_log DO INSTEAD NOTHING;

-- Function to log payment split creation
CREATE OR REPLACE FUNCTION audit_payment_split_created()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO audit_log (
        actor_type,
        action,
        resource_type,
        resource_id,
        changes
    ) VALUES (
        'system',
        'payment_split_created',
        'payment_split',
        NEW.id,
        jsonb_build_object(
            'bulk_order_id', NEW.bulk_order_id,
            'gross_total_paise', NEW.gross_total_paise,
            'net_total_paise', NEW.net_total_paise
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_audit_payment_split_created
    AFTER INSERT ON payment_split
    FOR EACH ROW
    EXECUTE FUNCTION audit_payment_split_created();

-- Function to log order amendments
CREATE OR REPLACE FUNCTION audit_order_status_changed()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.state IS DISTINCT FROM NEW.state THEN
        INSERT INTO audit_log (
            actor_type,
            action,
            resource_type,
            resource_id,
            changes
        ) VALUES (
            'system',
            'order_status_changed',
            'bulk_order',
            NEW.id,
            jsonb_build_object(
                'old_status', OLD.state,
                'new_status', NEW.state,
                'changed_at', NOW()
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_audit_order_status_changed
    AFTER UPDATE ON bulk_order
    FOR EACH ROW
    EXECUTE FUNCTION audit_order_status_changed();

-- Function to log QC results
CREATE OR REPLACE FUNCTION audit_qc_result_recorded()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO audit_log (
        actor_type,
        action,
        resource_type,
        resource_id,
        changes
    ) VALUES (
        'system',
        'qc_result_recorded',
        'qc_result',
        NEW.id,
        jsonb_build_object(
            'lot_id', NEW.lot_id,
            'inspector_id', NEW.inspector_id,
            'passed', NEW.passed,
            'notes', NEW.notes
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_audit_qc_result_recorded
    AFTER INSERT ON qc_result
    FOR EACH ROW
    EXECUTE FUNCTION audit_qc_result_recorded();

-- Function to log artisan verification
CREATE OR REPLACE FUNCTION audit_artisan_verified()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.verified IS FALSE AND NEW.verified IS TRUE THEN
        INSERT INTO audit_log (
            actor_type,
            action,
            resource_type,
            resource_id,
            changes
        ) VALUES (
            'system',
            'artisan_verified',
            'artisan',
            NEW.id,
            jsonb_build_object(
                'pehchan_id', NEW.pehchan_id,
                'verified_at', NOW()
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_audit_artisan_verified
    AFTER UPDATE ON artisan
    FOR EACH ROW
    EXECUTE FUNCTION audit_artisan_verified();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trigger_audit_artisan_verified ON artisan;
DROP TRIGGER IF EXISTS trigger_audit_qc_result_recorded ON qc_result;
DROP TRIGGER IF EXISTS trigger_audit_order_status_changed ON bulk_order;
DROP TRIGGER IF EXISTS trigger_audit_payment_split_created ON payment_split;

DROP FUNCTION IF EXISTS audit_artisan_verified();
DROP FUNCTION IF EXISTS audit_qc_result_recorded();
DROP FUNCTION IF EXISTS audit_order_status_changed();
DROP FUNCTION IF EXISTS audit_payment_split_created();

DROP TABLE IF EXISTS audit_log;

-- +goose StatementEnd
