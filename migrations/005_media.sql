-- migrations/005_media.sql
-- +goose Up

-- Mirrors common.v1.MediaKind without the UNSPECIFIED member.
CREATE TYPE media_kind AS ENUM ('IMAGE', 'VIDEO', 'AUDIO', 'DOCUMENT');

CREATE TABLE media (
    id          uuid        NOT NULL,
    artisan_id  uuid        NOT NULL,
    product_id  uuid,
    bucket      text        NOT NULL,
    object_key  text        NOT NULL,
    kind        media_kind  NOT NULL,
    mime_type   text        NOT NULL,
    size_bytes  bigint      NOT NULL,
    sha256_hex  char(64)    NOT NULL,
    width_px    integer,
    height_px   integer,
    duration_ms integer,
    source      text        NOT NULL DEFAULT 'artisan-app',
    uploaded_at timestamptz NOT NULL DEFAULT now(),
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT media_pkey PRIMARY KEY (id),
    CONSTRAINT media_bucket_object_key_key UNIQUE (bucket, object_key),
    CONSTRAINT media_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT media_product_id_fkey FOREIGN KEY (product_id)
        REFERENCES product (id) ON DELETE SET NULL,
    CONSTRAINT media_size_bytes_check CHECK (size_bytes >= 0),
    CONSTRAINT media_sha256_hex_check CHECK (sha256_hex ~ '^[0-9a-f]{64}$')
);

-- Rendering a product means fetching its media in upload order.
CREATE INDEX media_product_id_uploaded_at_idx ON media (product_id, uploaded_at);
-- Re-upload of an identical file is deduped by content hash before it reaches MinIO.
CREATE INDEX media_sha256_hex_idx ON media (sha256_hex);

CREATE TABLE provenance_record (
    id                    uuid        NOT NULL,
    listing_id            uuid        NOT NULL,
    product_id            uuid        NOT NULL,
    artisan_id            uuid        NOT NULL,
    technique_claimed     text        NOT NULL,
    technique_observed    text        NOT NULL,
    technique_matches     boolean     NOT NULL,
    technique_confidence  real        NOT NULL,
    loom_is_handloom      boolean,
    loom_confidence       real,
    loom_fft_peak_ratio   real,
    content_hash          char(64)    NOT NULL,
    previous_hash         char(64),
    signature             bytea       NOT NULL,
    signature_algorithm   text        NOT NULL DEFAULT 'ed25519',
    qr_code               text        NOT NULL,
    certificate_media_id  uuid,
    model_version         text        NOT NULL,
    sealed_at             timestamptz NOT NULL DEFAULT now(),
    created_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT provenance_record_pkey PRIMARY KEY (id),
    CONSTRAINT provenance_record_listing_id_key UNIQUE (listing_id),
    CONSTRAINT provenance_record_content_hash_key UNIQUE (content_hash),
    CONSTRAINT provenance_record_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listing (id) ON DELETE CASCADE,
    CONSTRAINT provenance_record_product_id_fkey FOREIGN KEY (product_id)
        REFERENCES product (id) ON DELETE RESTRICT,
    CONSTRAINT provenance_record_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE RESTRICT,
    CONSTRAINT provenance_record_certificate_media_id_fkey FOREIGN KEY (certificate_media_id)
        REFERENCES media (id) ON DELETE SET NULL,
    CONSTRAINT provenance_record_content_hash_check CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT provenance_record_previous_hash_check
        CHECK (previous_hash IS NULL OR previous_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT provenance_record_technique_confidence_check
        CHECK (technique_confidence >= 0.0 AND technique_confidence <= 1.0),
    CONSTRAINT provenance_record_loom_confidence_check
        CHECK (loom_confidence IS NULL OR (loom_confidence >= 0.0 AND loom_confidence <= 1.0)),
    -- The loom verdict is all-or-nothing: textile crafts have every field, others none.
    CONSTRAINT provenance_record_loom_verdict_check CHECK (
        (loom_is_handloom IS NULL AND loom_confidence IS NULL AND loom_fft_peak_ratio IS NULL) OR
        (loom_is_handloom IS NOT NULL AND loom_confidence IS NOT NULL AND loom_fft_peak_ratio IS NOT NULL)
    )
);

-- Verifying an artisan's hash chain walks their records in seal order.
CREATE INDEX provenance_record_artisan_id_sealed_at_idx ON provenance_record (artisan_id, sealed_at);
-- The chain is followed link by link when a buyer audits a certificate.
CREATE INDEX provenance_record_previous_hash_idx ON provenance_record (previous_hash);

CREATE TABLE provenance_media (
    provenance_id uuid        NOT NULL,
    media_id      uuid        NOT NULL,
    ordinal       integer     NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT provenance_media_pkey PRIMARY KEY (provenance_id, media_id),
    CONSTRAINT provenance_media_provenance_id_fkey FOREIGN KEY (provenance_id)
        REFERENCES provenance_record (id) ON DELETE CASCADE,
    CONSTRAINT provenance_media_media_id_fkey FOREIGN KEY (media_id)
        REFERENCES media (id) ON DELETE RESTRICT
);

-- Deferred foreign keys: these columns point at media, which points back at artisan and product.
ALTER TABLE artisan
    ADD COLUMN photo_media_id uuid,
    ADD CONSTRAINT artisan_photo_media_id_fkey FOREIGN KEY (photo_media_id)
        REFERENCES media (id) ON DELETE SET NULL;

ALTER TABLE product
    ADD COLUMN voice_note_media_id uuid,
    ADD CONSTRAINT product_voice_note_media_id_fkey FOREIGN KEY (voice_note_media_id)
        REFERENCES media (id) ON DELETE SET NULL;

ALTER TABLE listing
    ADD CONSTRAINT listing_provenance_id_fkey FOREIGN KEY (provenance_id)
        REFERENCES provenance_record (id) ON DELETE SET NULL;

-- +goose Down

ALTER TABLE listing DROP CONSTRAINT IF EXISTS listing_provenance_id_fkey;
ALTER TABLE product DROP CONSTRAINT IF EXISTS product_voice_note_media_id_fkey;
ALTER TABLE product DROP COLUMN IF EXISTS voice_note_media_id;
ALTER TABLE artisan DROP CONSTRAINT IF EXISTS artisan_photo_media_id_fkey;
ALTER TABLE artisan DROP COLUMN IF EXISTS photo_media_id;
DROP TABLE IF EXISTS provenance_media;
DROP TABLE IF EXISTS provenance_record;
DROP TABLE IF EXISTS media;
DROP TYPE IF EXISTS media_kind;
