-- migrations/014_listing_media.sql
-- +goose Up

-- The part a media asset plays on a listing. GALLERY is the default; the other
-- two are singletons per listing and are enforced as such below.
CREATE TYPE listing_media_role AS ENUM ('GALLERY', 'PRIMARY_IMAGE', 'PROCESS_VIDEO');

CREATE TABLE listing_media (
    listing_id uuid               NOT NULL,
    media_id   uuid               NOT NULL,
    ordinal    integer            NOT NULL DEFAULT 0,
    role       listing_media_role NOT NULL DEFAULT 'GALLERY',
    created_at timestamptz        NOT NULL DEFAULT now(),
    CONSTRAINT listing_media_pkey PRIMARY KEY (listing_id, media_id),
    CONSTRAINT listing_media_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listing (id) ON DELETE CASCADE,
    CONSTRAINT listing_media_media_id_fkey FOREIGN KEY (media_id)
        REFERENCES media (id) ON DELETE RESTRICT,
    CONSTRAINT listing_media_ordinal_check CHECK (ordinal >= 0)
);

-- Display order is the only order a listing's media is ever read in.
CREATE UNIQUE INDEX listing_media_listing_id_ordinal_key ON listing_media (listing_id, ordinal);
-- Exactly one thumbnail and one process clip per listing; the buyer surface and
-- the provenance queue both assume there is no ambiguity to resolve at read time.
CREATE UNIQUE INDEX listing_media_one_primary_image_key ON listing_media (listing_id)
    WHERE role = 'PRIMARY_IMAGE';
CREATE UNIQUE INDEX listing_media_one_process_video_key ON listing_media (listing_id)
    WHERE role = 'PROCESS_VIDEO';
-- "Which listings use this asset" is asked before a media row may be deleted.
CREATE INDEX listing_media_media_id_idx ON listing_media (media_id);

-- +goose Down

DROP TABLE IF EXISTS listing_media;
DROP TYPE IF EXISTS listing_media_role;
