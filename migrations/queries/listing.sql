-- migrations/queries/listing.sql

-- name: CreateProduct :one
INSERT INTO product (
    id, artisan_id, craft_id, working_title, length_mm, width_mm, height_mm, weight_g,
    materials, techniques, colours, motifs, voice_note_media_id, created_by
) VALUES (
    @id, @artisan_id, @craft_id, @working_title, sqlc.narg('length_mm'),
    sqlc.narg('width_mm'), sqlc.narg('height_mm'), sqlc.narg('weight_g'),
    @materials, @techniques, @colours, @motifs, sqlc.narg('voice_note_media_id'), @created_by
)
RETURNING *;

-- name: GetProduct :one
SELECT * FROM product WHERE id = @id;

-- name: UpdateProductAttributes :one
UPDATE product
SET materials  = @materials,
    techniques = @techniques,
    colours    = @colours,
    motifs     = @motifs,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: CreateListing :one
-- type is nullable: the artisan wizard's first POST /listings call (story
-- step) only knows craft_id/working_title, well before the wizard's pricing
-- step lets the artisan choose a type. See migrations/035_listing_draft_type.sql.
INSERT INTO listing (
    id, product_id, artisan_id, type, price_paise, stock_quantity, min_order_quantity,
    lead_time_days, capacity_per_month, accepting_orders, advance_pct,
    packaging_fragile, packaging_oversized, packaging_requires_custom_crating,
    packed_length_mm, packed_width_mm, packed_height_mm, packed_weight_g,
    gi_certified, created_by
) VALUES (
    @id, @product_id, @artisan_id, sqlc.narg('type'), @price_paise, sqlc.narg('stock_quantity'),
    @min_order_quantity, sqlc.narg('lead_time_days'), sqlc.narg('capacity_per_month'),
    @accepting_orders, sqlc.narg('advance_pct'),
    @packaging_fragile, @packaging_oversized, @packaging_requires_custom_crating,
    sqlc.narg('packed_length_mm'), sqlc.narg('packed_width_mm'),
    sqlc.narg('packed_height_mm'), sqlc.narg('packed_weight_g'),
    @gi_certified, @created_by
)
RETURNING *;

-- A suspended listing is excluded from the WHERE clause rather than rejected by a
-- constraint: editing one is a state-machine question, and the service answers it.
-- name: UpdateListing :one
UPDATE listing
SET type                              = COALESCE(sqlc.narg('type'), type),
    price_paise                       = COALESCE(sqlc.narg('price_paise'), price_paise),
    stock_quantity                    = COALESCE(sqlc.narg('stock_quantity'), stock_quantity),
    min_order_quantity                = COALESCE(sqlc.narg('min_order_quantity'), min_order_quantity),
    lead_time_days                    = COALESCE(sqlc.narg('lead_time_days'), lead_time_days),
    capacity_per_month                = COALESCE(sqlc.narg('capacity_per_month'), capacity_per_month),
    accepting_orders                  = COALESCE(sqlc.narg('accepting_orders'), accepting_orders),
    advance_pct                       = COALESCE(sqlc.narg('advance_pct'), advance_pct),
    packaging_fragile                 = COALESCE(sqlc.narg('packaging_fragile'), packaging_fragile),
    packaging_oversized               = COALESCE(sqlc.narg('packaging_oversized'), packaging_oversized),
    packaging_requires_custom_crating = COALESCE(sqlc.narg('packaging_requires_custom_crating'), packaging_requires_custom_crating),
    packed_length_mm                  = COALESCE(sqlc.narg('packed_length_mm'), packed_length_mm),
    packed_width_mm                   = COALESCE(sqlc.narg('packed_width_mm'), packed_width_mm),
    packed_height_mm                  = COALESCE(sqlc.narg('packed_height_mm'), packed_height_mm),
    packed_weight_g                   = COALESCE(sqlc.narg('packed_weight_g'), packed_weight_g),
    gi_certified                      = COALESCE(sqlc.narg('gi_certified'), gi_certified),
    updated_at                        = now()
WHERE id = @id AND state <> 'SUSPENDED'
RETURNING *;

-- Guarded transition: the caller states the state it believes the row is in, so two
-- concurrent approvals cannot both win. Zero rows means the guard failed.
-- name: TransitionListingState :one
UPDATE listing
SET state             = @next_state,
    published_at      = CASE WHEN @next_state::listing_state = 'PUBLISHED'
                             THEN COALESCE(published_at, now()) ELSE published_at END,
    suspension_reason = CASE WHEN @next_state::listing_state = 'SUSPENDED'
                             THEN sqlc.narg('suspension_reason')::text ELSE NULL END,
    updated_at        = now()
WHERE id = @id AND state = @expected_state
RETURNING *;

-- name: SetListingProvenance :one
UPDATE listing SET provenance_id = @provenance_id, updated_at = now()
WHERE id = @id
RETURNING *;

-- name: GetListing :one
SELECT * FROM listing WHERE id = @id;

-- name: SetListingNeedsDescription :one
UPDATE listing SET needs_description = @needs_description, updated_at = now()
WHERE id = @id
RETURNING *;

-- name: GetListingTranslations :many
SELECT * FROM listing_translation WHERE listing_id = @listing_id ORDER BY language;

-- name: GetListingAttributes :many
SELECT * FROM listing_attribute WHERE listing_id = @listing_id ORDER BY name, value;

-- One round trip for the listing detail page: the listing plus its translations and
-- attributes as JSON arrays, decoded in the repo wrapper.
-- name: GetListingDetail :one
SELECT
    l.*,
    COALESCE((SELECT json_agg(to_jsonb(t) ORDER BY t.language)
              FROM listing_translation t WHERE t.listing_id = l.id), '[]'::json) AS translations,
    COALESCE((SELECT json_agg(to_jsonb(a) ORDER BY a.name, a.value)
              FROM listing_attribute a WHERE a.listing_id = l.id), '[]'::json) AS attributes
FROM listing l
WHERE l.id = @id;

-- name: ListListingsByArtisan :many
SELECT * FROM listing
WHERE artisan_id = @artisan_id
  AND (@state::listing_state IS NULL OR state = @state)
  AND (@after::uuid IS NULL OR id > @after)
ORDER BY id
LIMIT @page_size;

-- The catalogue browse query. Craft comes from the product and cluster from the
-- artisan, so both filters are joins rather than denormalised columns.
-- name: ListListings :many
SELECT l.* FROM listing l
JOIN product p ON p.id = l.product_id
JOIN artisan a ON a.id = l.artisan_id
WHERE (sqlc.narg('artisan_id')::uuid IS NULL OR l.artisan_id = sqlc.narg('artisan_id'))
  AND (sqlc.narg('cluster_id')::uuid IS NULL OR a.primary_cluster_id = sqlc.narg('cluster_id'))
  AND (sqlc.narg('craft_id')::uuid IS NULL OR p.craft_id = sqlc.narg('craft_id'))
  AND (sqlc.narg('state')::listing_state IS NULL OR l.state = sqlc.narg('state'))
  AND (sqlc.narg('type')::listing_type IS NULL OR l.type = sqlc.narg('type'))
  AND (sqlc.narg('after')::uuid IS NULL OR l.id > sqlc.narg('after'))
ORDER BY l.id
LIMIT @page_size;

-- search-svc backfill: everything that reached PUBLISHED after a watermark.
-- name: ListPublishedSince :many
SELECT * FROM listing
WHERE state = 'PUBLISHED' AND published_at > @since
ORDER BY published_at, id
LIMIT @page_size;

-- name: UpsertListingTranslation :one
INSERT INTO listing_translation (listing_id, language, title, description, highlights, machine_generated, edited_by)
VALUES (@listing_id, @language, @title, @description, @highlights, @machine_generated, sqlc.narg('edited_by'))
ON CONFLICT ON CONSTRAINT listing_translation_pkey DO UPDATE
SET title             = EXCLUDED.title,
    description       = EXCLUDED.description,
    highlights        = EXCLUDED.highlights,
    machine_generated = EXCLUDED.machine_generated,
    edited_by         = EXCLUDED.edited_by,
    updated_at        = now()
RETURNING *;

-- The model's write, guarded in SQL as well as in the service: the INSERT selects
-- no row when the attribute name already carries an artisan or curator value, and
-- the DO UPDATE refuses to touch anything that is not the model's own. Artisan
-- values therefore survive a concurrent model write, not just a sequential one.
-- name: UpsertModelAttribute :execrows
INSERT INTO listing_attribute (id, listing_id, name, value, confidence, source)
SELECT @id, @listing_id, @name, @value, @confidence, 'MODEL'::attribute_source
WHERE NOT EXISTS (
    SELECT 1 FROM listing_attribute held
    WHERE held.listing_id = @listing_id AND held.name = @name AND held.source <> 'MODEL'
)
ON CONFLICT ON CONSTRAINT listing_attribute_listing_id_name_value_key DO UPDATE
SET confidence = EXCLUDED.confidence
WHERE listing_attribute.source = 'MODEL';

-- The artisan's or curator's write, which always lands.
-- name: UpsertAuthoritativeAttribute :exec
INSERT INTO listing_attribute (id, listing_id, name, value, confidence, source)
VALUES (@id, @listing_id, @name, @value, @confidence, @source)
ON CONFLICT ON CONSTRAINT listing_attribute_listing_id_name_value_key DO UPDATE
SET confidence = EXCLUDED.confidence,
    source     = EXCLUDED.source;

-- Clears the model's guesses for one attribute name, which is what makes room for
-- the artisan's answer without disturbing their other answers.
-- name: DeleteListingAttributesByName :execrows
DELETE FROM listing_attribute
WHERE listing_id = @listing_id AND name = @name AND source = 'MODEL';

-- name: DeleteListingAttributesBySource :execrows
DELETE FROM listing_attribute WHERE listing_id = @listing_id AND source = @source;

-- name: DeleteListingMedia :execrows
DELETE FROM listing_media WHERE listing_id = @listing_id;

-- name: InsertListingMedia :exec
INSERT INTO listing_media (listing_id, media_id, ordinal, role)
VALUES (@listing_id, @media_id, @ordinal, @role)
ON CONFLICT ON CONSTRAINT listing_media_pkey DO UPDATE
SET ordinal = EXCLUDED.ordinal,
    role    = EXCLUDED.role;

-- name: ListListingMedia :many
SELECT lm.listing_id, lm.media_id, lm.ordinal, lm.role, m.kind
FROM listing_media lm
JOIN media m ON m.id = lm.media_id
WHERE lm.listing_id = @listing_id
ORDER BY lm.ordinal, lm.media_id;

