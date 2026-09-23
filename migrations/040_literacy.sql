-- migrations/040_literacy.sql
-- +goose Up

-- F15: digital literacy track. The lesson catalog itself is a Go constant
-- (services/core-svc/internal/core/domain/literacy.go) -- lesson content is
-- i18n-keyed and code-versioned, so only per-artisan progress is data.
CREATE TABLE literacy_progress (
    artisan_id     uuid         NOT NULL,
    -- 'PHOTO', 'STORY', 'PRICE', 'ORDERS', 'PAYMENTS', 'SAFETY', 'WHATSAPP', 'LOAN'.
    lesson_code    text         NOT NULL,
    practice_done  boolean      NOT NULL DEFAULT false,
    quiz_passed    boolean      NOT NULL DEFAULT false,
    completed_at   timestamptz,
    updated_at     timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT literacy_progress_pkey PRIMARY KEY (artisan_id, lesson_code),
    CONSTRAINT literacy_progress_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT literacy_progress_lesson_code_check CHECK (lesson_code ~ '^[A-Z_]{2,20}$')
);

-- One certificate per artisan, signed and publicly verifiable by short code
-- (same ed25519 + short-code + PDF-in-object-storage pattern as
-- income_statement, 023).
CREATE TABLE literacy_certificate (
    id             uuid         NOT NULL,
    artisan_id     uuid         NOT NULL,
    issued_at      timestamptz  NOT NULL DEFAULT now(),
    short_code     text         NOT NULL,
    signature      bytea        NOT NULL,
    public_key_id  text         NOT NULL,
    s3_key         text         NOT NULL,
    CONSTRAINT literacy_certificate_pkey PRIMARY KEY (id),
    CONSTRAINT literacy_certificate_artisan_id_key UNIQUE (artisan_id),
    CONSTRAINT literacy_certificate_short_code_key UNIQUE (short_code),
    CONSTRAINT literacy_certificate_short_code_check CHECK (length(short_code) = 10),
    CONSTRAINT literacy_certificate_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE
);

-- +goose Down

DROP TABLE IF EXISTS literacy_certificate;
DROP TABLE IF EXISTS literacy_progress;
