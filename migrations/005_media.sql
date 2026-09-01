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

-- Deferred foreign keys: these columns point at media, which points back at artisan and product.
ALTER TABLE artisan
    ADD COLUMN photo_media_id uuid,
    ADD CONSTRAINT artisan_photo_media_id_fkey FOREIGN KEY (photo_media_id)
        REFERENCES media (id) ON DELETE SET NULL;

ALTER TABLE product
    ADD COLUMN voice_note_media_id uuid,
    ADD CONSTRAINT product_voice_note_media_id_fkey FOREIGN KEY (voice_note_media_id)
        REFERENCES media (id) ON DELETE SET NULL;

-- listing.provenance_id's FK is added in migration 022, which is where
-- provenance_record is actually defined; an earlier draft of that table
-- briefly lived here and was superseded before this ever shipped.

-- +goose Down

ALTER TABLE product DROP CONSTRAINT IF EXISTS product_voice_note_media_id_fkey;
ALTER TABLE product DROP COLUMN IF EXISTS voice_note_media_id;
ALTER TABLE artisan DROP CONSTRAINT IF EXISTS artisan_photo_media_id_fkey;
ALTER TABLE artisan DROP COLUMN IF EXISTS photo_media_id;
DROP TABLE IF EXISTS media;
DROP TYPE IF EXISTS media_kind;
