-- migrations/032_trends.sql
-- +goose Up

-- Source types mirror trends.v1.TrendSourceType without the UNSPECIFIED member.
CREATE TYPE trend_source_type AS ENUM (
    'INSTAGRAM', 'PINTEREST', 'BLOG', 'NEWS', 'YOUTUBE', 'OTHER'
);

CREATE TABLE trend_link (
    id                      uuid              NOT NULL,
    title                   text              NOT NULL,
    description             text              NOT NULL DEFAULT '',
    url                     text              NOT NULL,
    source_type             trend_source_type NOT NULL,
    -- Optional association to a craft; null for general trends.
    craft_id                uuid,
    -- Thumbnail URL, either from oEmbed or user-supplied.
    thumbnail_url           text,
    -- oEmbed HTML for inline embedding (Instagram, YouTube, etc.).
    embed_html              text,
    -- Full oEmbed JSON response for frontend flexibility.
    embed_metadata          jsonb,
    -- Pinned links sort to the top of the feed (admin privilege).
    pinned                  boolean           NOT NULL DEFAULT false,
    -- User who created the link (admin or artisan).
    created_by              text              NOT NULL,
    curator_role            text              NOT NULL DEFAULT 'MINISTRY',
    curator_name            text              NOT NULL DEFAULT '',
    -- Optional expiry; null means perpetual.
    expires_at              timestamptz,
    created_at              timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT trend_link_pkey PRIMARY KEY (id),
    CONSTRAINT trend_link_craft_id_fkey FOREIGN KEY (craft_id)
        REFERENCES craft (id) ON DELETE SET NULL,
    CONSTRAINT trend_link_url_check CHECK (url ~ '^https?://')
);

-- Feed reads: pinned first, then newest.
CREATE INDEX trend_link_feed_idx ON trend_link (pinned DESC, created_at DESC);
-- Filter by craft.
CREATE INDEX trend_link_craft_idx ON trend_link (craft_id, created_at DESC)
    WHERE craft_id IS NOT NULL;
-- Filter by source.
CREATE INDEX trend_link_source_idx ON trend_link (source_type, created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS trend_link;
DROP TYPE IF EXISTS trend_source_type;
