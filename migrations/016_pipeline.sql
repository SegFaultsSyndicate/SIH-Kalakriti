-- migrations/016_pipeline.sql
-- +goose Up

-- The pipeline creates one product per uploaded photograph. This column is the
-- idempotency key for that step: a redelivered media.uploaded hits the unique
-- index and loads the product it made the first time instead of making a second.
ALTER TABLE product
    ADD COLUMN source_media_id uuid,
    ADD CONSTRAINT product_source_media_id_key UNIQUE (source_media_id),
    ADD CONSTRAINT product_source_media_id_fkey FOREIGN KEY (source_media_id)
        REFERENCES media (id) ON DELETE SET NULL;

-- Set when copy generation failed but the attributes survived: the listing stays
-- a usable DRAFT and the artisan is asked to write the description themselves.
ALTER TABLE listing
    ADD COLUMN needs_description boolean NOT NULL DEFAULT false;

-- The artisan's "what needs my attention" screen reads exactly this.
CREATE INDEX listing_needs_description_idx ON listing (artisan_id) WHERE needs_description;

-- +goose Down

DROP INDEX IF EXISTS listing_needs_description_idx;
ALTER TABLE listing DROP COLUMN IF EXISTS needs_description;
ALTER TABLE product
    DROP CONSTRAINT IF EXISTS product_source_media_id_fkey,
    DROP CONSTRAINT IF EXISTS product_source_media_id_key,
    DROP COLUMN IF EXISTS source_media_id;
