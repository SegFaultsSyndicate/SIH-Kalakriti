-- migrations/020_qc_results.sql
-- +goose Up

-- qc_result is one inspection outcome for one lot. Kept separate from
-- bulk_order_event (which still gets a QC_RECORDED audit row for every
-- inspection) because this table is queried on its own shape  -  "every
-- CRITICAL defect this month", "an artisan's QC pass rate"  -  not just
-- replayed chronologically.
CREATE TABLE qc_result (
    id            uuid        NOT NULL,
    lot_id        uuid        NOT NULL,
    inspector_id  text        NOT NULL,
    passed        boolean     NOT NULL,
    notes         text,
    media_ids     uuid[]      NOT NULL DEFAULT '{}',
    inspected_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT qc_result_pkey PRIMARY KEY (id),
    CONSTRAINT qc_result_lot_id_fkey FOREIGN KEY (lot_id)
        REFERENCES order_lot (id) ON DELETE CASCADE
);

CREATE INDEX qc_result_lot_id_idx ON qc_result (lot_id);

CREATE TYPE defect_severity AS ENUM ('MINOR', 'MAJOR', 'CRITICAL');

-- qc_defect is one fault found during one inspection. A CRITICAL defect is
-- what triggers reallocation (order_lot.state -> REALLOCATED, handled in the
-- service layer) rather than QC_FAILED-and-stop, per DefectSeverity's own
-- proto doc comment ("Unsaleable; the lot is reallocated.").
CREATE TABLE qc_defect (
    id             uuid            NOT NULL,
    qc_result_id   uuid            NOT NULL,
    code           text            NOT NULL,
    description    text            NOT NULL,
    severity       defect_severity NOT NULL,
    media_ids      uuid[]          NOT NULL DEFAULT '{}',
    affected_units integer         NOT NULL,
    CONSTRAINT qc_defect_pkey PRIMARY KEY (id),
    CONSTRAINT qc_defect_qc_result_id_fkey FOREIGN KEY (qc_result_id)
        REFERENCES qc_result (id) ON DELETE CASCADE,
    CONSTRAINT qc_defect_affected_units_check CHECK (affected_units >= 0)
);

CREATE INDEX qc_defect_qc_result_id_idx ON qc_defect (qc_result_id);

-- +goose Down

DROP TABLE IF EXISTS qc_defect;
DROP TYPE IF EXISTS defect_severity;
DROP TABLE IF EXISTS qc_result;
