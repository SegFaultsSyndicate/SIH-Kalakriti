-- migrations/015_media_lifecycle.sql
-- +goose Up

-- Where an asset is between "the app asked for an upload URL" and "the buyer can
-- see it". PENDING rows exist before their bytes do, which is what the reaper cleans up.
CREATE TYPE media_state AS ENUM ('PENDING', 'UPLOADED', 'PROCESSING', 'READY', 'FAILED');

ALTER TABLE media
    ADD COLUMN state               media_state NOT NULL DEFAULT 'PENDING',
    -- The enhancement pipeline writes a second object rather than overwriting the
    -- artisan's original, which provenance later hashes.
    ADD COLUMN enhanced_object_key text,
    ADD COLUMN model_version       text,
    ADD COLUMN failure_reason      text,
    ADD COLUMN confirmed_at        timestamptz;

-- A PENDING row has no bytes yet, so it cannot have a content hash; everything
-- past PENDING keeps the format check it always had.
ALTER TABLE media ALTER COLUMN sha256_hex DROP NOT NULL;
ALTER TABLE media DROP CONSTRAINT IF EXISTS media_sha256_hex_check;
ALTER TABLE media ADD CONSTRAINT media_sha256_hex_check
    CHECK (sha256_hex IS NULL OR sha256_hex ~ '^[0-9a-f]{64}$');

-- FAILED must say why; nothing else may carry a reason.
ALTER TABLE media ADD CONSTRAINT media_failure_reason_check
    CHECK ((state = 'FAILED') = (failure_reason IS NOT NULL));
-- READY means an enhanced object exists to serve, or the original was accepted as-is.
ALTER TABLE media ADD CONSTRAINT media_confirmed_at_check
    CHECK (state = 'PENDING' OR confirmed_at IS NOT NULL);

-- The reaper's only read: the oldest rows still waiting for bytes. Confirmed rows
-- fall out of the index, so it stays small however large the table grows.
CREATE INDEX media_pending_created_at_idx ON media (created_at) WHERE state = 'PENDING';
-- The dedupe lookup is per artisan and only ever interested in usable assets.
CREATE INDEX media_artisan_id_state_idx ON media (artisan_id, state);

-- Existing rows predate the lifecycle and already have their bytes.
UPDATE media SET state = 'READY', confirmed_at = uploaded_at WHERE state = 'PENDING';

-- +goose Down

DROP INDEX IF EXISTS media_artisan_id_state_idx;
DROP INDEX IF EXISTS media_pending_created_at_idx;
ALTER TABLE media DROP CONSTRAINT IF EXISTS media_confirmed_at_check;
ALTER TABLE media DROP CONSTRAINT IF EXISTS media_failure_reason_check;
ALTER TABLE media DROP CONSTRAINT IF EXISTS media_sha256_hex_check;
ALTER TABLE media ADD CONSTRAINT media_sha256_hex_check CHECK (sha256_hex ~ '^[0-9a-f]{64}$');
ALTER TABLE media
    DROP COLUMN IF EXISTS confirmed_at,
    DROP COLUMN IF EXISTS failure_reason,
    DROP COLUMN IF EXISTS model_version,
    DROP COLUMN IF EXISTS enhanced_object_key,
    DROP COLUMN IF EXISTS state;
DROP TYPE IF EXISTS media_state;
