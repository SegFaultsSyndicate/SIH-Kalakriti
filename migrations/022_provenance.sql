-- migrations/022_provenance.sql
-- +goose Up

-- Provenance records freeze evidence and form a hash chain per artisan.
CREATE TABLE provenance_record (
    id               uuid        NOT NULL,
    listing_id       uuid        NOT NULL,
    artisan_id       uuid        NOT NULL,
    craft_id         uuid        NOT NULL,
    content_hash     text        NOT NULL,
    previous_hash    text,
    signature        bytea       NOT NULL,
    signature_algo   text        NOT NULL DEFAULT 'ed25519',
    public_key_id    text        NOT NULL,
    short_code       text        NOT NULL,
    technique_matched boolean    NOT NULL,
    media_hashes     text[]      NOT NULL DEFAULT '{}',
    sealed_at        timestamptz NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT provenance_record_pkey PRIMARY KEY (id),
    CONSTRAINT provenance_record_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listing (id) ON DELETE RESTRICT,
    CONSTRAINT provenance_record_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE RESTRICT,
    CONSTRAINT provenance_record_craft_id_fkey FOREIGN KEY (craft_id)
        REFERENCES craft (id) ON DELETE RESTRICT,
    CONSTRAINT provenance_record_short_code_key UNIQUE (short_code),
    CONSTRAINT provenance_record_content_hash_check CHECK (length(content_hash) = 64),
    CONSTRAINT provenance_record_previous_hash_check CHECK (previous_hash IS NULL OR length(previous_hash) = 64),
    CONSTRAINT provenance_record_short_code_check CHECK (length(short_code) = 10)
);

-- Public verification lookups by short code.
CREATE INDEX provenance_record_short_code_idx ON provenance_record (short_code);

-- Artisan's chain of records, newest first.
CREATE INDEX provenance_record_artisan_id_sealed_at_idx ON provenance_record (artisan_id, sealed_at DESC);

-- Listing provenance lookups.
CREATE INDEX provenance_record_listing_id_idx ON provenance_record (listing_id);

-- +goose Down

DROP TABLE IF EXISTS provenance_record;
