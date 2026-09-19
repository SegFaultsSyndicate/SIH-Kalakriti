-- migrations/036_drop_product_source_media.sql
-- +goose Up

-- product.source_media_id (016_pipeline.sql) made the cataloguing pipeline
-- create its own product per uploaded photo, keyed 1:1 on that photo. That
-- conflicts with the real listing-creation flow (the artisan wizard), which
-- attaches several photos to one product via AttachProductMedia. The two
-- paths raced on every real upload: the wizard's manual listing, and a second,
-- orphaned, artisan-invisible listing the pipeline made from the same photo.
--
-- The pipeline no longer creates products at all -- it now enriches whichever
-- product/listing the artisan (or a cluster officer) already attached media
-- to, triggered by AttachListingMedia rather than by the raw upload. See
-- services/core-svc/internal/core/service/pipeline.go.
ALTER TABLE product
    DROP CONSTRAINT IF EXISTS product_source_media_id_fkey,
    DROP CONSTRAINT IF EXISTS product_source_media_id_key,
    DROP COLUMN IF EXISTS source_media_id;

-- +goose Down

ALTER TABLE product
    ADD COLUMN source_media_id uuid,
    ADD CONSTRAINT product_source_media_id_key UNIQUE (source_media_id),
    ADD CONSTRAINT product_source_media_id_fkey FOREIGN KEY (source_media_id)
        REFERENCES media (id) ON DELETE SET NULL;
