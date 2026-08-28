-- migrations/queries/provenance.sql

-- name: InsertProvenanceRecord :one
INSERT INTO provenance_record (
    id, listing_id, artisan_id, craft_id, content_hash, previous_hash,
    signature, signature_algo, public_key_id, short_code,
    technique_matched, media_hashes, sealed_at
) VALUES (
    @id, @listing_id, @artisan_id, @craft_id, @content_hash, sqlc.narg('previous_hash'),
    @signature, @signature_algo, @public_key_id, @short_code,
    @technique_matched, @media_hashes, @sealed_at
)
RETURNING *;

-- name: GetProvenanceRecord :one
SELECT * FROM provenance_record WHERE id = @id;

-- name: GetProvenanceByShortCode :one
SELECT * FROM provenance_record WHERE short_code = @short_code;

-- name: GetProvenanceByListing :one
SELECT * FROM provenance_record WHERE listing_id = @listing_id ORDER BY sealed_at DESC LIMIT 1;

-- name: GetLatestProvenanceForArtisan :one
SELECT * FROM provenance_record
WHERE artisan_id = @artisan_id
ORDER BY sealed_at DESC
LIMIT 1;

-- name: ListProvenanceByArtisan :many
SELECT * FROM provenance_record
WHERE artisan_id = @artisan_id
ORDER BY sealed_at DESC
LIMIT @limit;

-- name: ShortCodeExists :one
SELECT EXISTS(SELECT 1 FROM provenance_record WHERE short_code = @short_code) AS exists;
