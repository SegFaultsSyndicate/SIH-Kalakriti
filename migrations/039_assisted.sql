-- migrations/039_assisted.sql
-- +goose Up

-- Staff accounts: the only way a FIELD_AGENT, CLUSTER_OFFICER or MINISTRY
-- token is ever minted outside dev. VerifyOtp checks this table first; a
-- phone that matches an active row logs in as that staff role, everything
-- else falls through to the artisan flow. The first MINISTRY row is created
-- by services/core-svc/cmd/create-staff; the rest through the admin /staff page.
CREATE TYPE staff_role AS ENUM ('FIELD_AGENT', 'CLUSTER_OFFICER', 'MINISTRY');

CREATE TABLE staff_account (
    id            uuid         NOT NULL,
    phone_e164    text         NOT NULL,
    display_name  text         NOT NULL,
    role          staff_role   NOT NULL,
    state_code    text,
    district      text,
    cluster_id    uuid,
    csc_id        text,
    active        boolean      NOT NULL DEFAULT true,
    created_by    text         NOT NULL,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT staff_account_pkey PRIMARY KEY (id),
    CONSTRAINT staff_account_phone_key UNIQUE (phone_e164),
    CONSTRAINT staff_account_phone_e164_check CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
    CONSTRAINT staff_account_display_name_check CHECK (length(display_name) > 0),
    -- A district scope without a state is meaningless.
    CONSTRAINT staff_account_scope_check CHECK (district IS NULL OR state_code IS NOT NULL),
    CONSTRAINT staff_account_cluster_id_fkey FOREIGN KEY (cluster_id)
        REFERENCES cluster (id) ON DELETE SET NULL
);

-- An agent may act for an artisan only through an unrevoked link the
-- artisan consented to, by OTP read out from their own phone or, with no
-- signal, a recorded voice consent (flagged for officer review).
CREATE TABLE assisted_link (
    id              uuid         NOT NULL,
    agent_id        uuid         NOT NULL,
    artisan_id      uuid         NOT NULL,
    consent_method  text         NOT NULL,
    consent_ref     text         NOT NULL,
    consent_at      timestamptz  NOT NULL,
    needs_review    boolean      NOT NULL DEFAULT false,
    reviewed_by     text,
    reviewed_at     timestamptz,
    revoked_at      timestamptz,
    created_at      timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT assisted_link_pkey PRIMARY KEY (id),
    CONSTRAINT assisted_link_consent_method_check
        CHECK (consent_method IN ('ARTISAN_OTP', 'VOICE_RECORDING')),
    CONSTRAINT assisted_link_active_unique UNIQUE (agent_id, artisan_id),
    CONSTRAINT assisted_link_agent_id_fkey FOREIGN KEY (agent_id)
        REFERENCES staff_account (id),
    CONSTRAINT assisted_link_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE
);

CREATE INDEX assisted_link_artisan_idx ON assisted_link (artisan_id) WHERE revoked_at IS NULL;

-- audit_log (025) has no subject column; assisted-mode writes record the
-- artisan acted for here so "listed with help from" and agent productivity
-- are one indexed lookup rather than a jsonb scan.
ALTER TABLE audit_log ADD COLUMN subject_id uuid;
CREATE INDEX idx_audit_log_subject_id ON audit_log (subject_id) WHERE subject_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_audit_log_subject_id;
ALTER TABLE audit_log DROP COLUMN IF EXISTS subject_id;
DROP TABLE IF EXISTS assisted_link;
DROP TABLE IF EXISTS staff_account;
DROP TYPE IF EXISTS staff_role;
