-- migrations/queries/search.sql

-- name: UpsertListingSearch :exec
INSERT INTO listing_search (
    listing_id, language, artisan_id, craft_id, cluster_id, listing_type, price_paise,
    lead_time_days, gi_certified, provenance_sealed, state_code, district, colours,
    materials, document, document_tsv, embedding, model_version, indexed_at
) VALUES (
    @listing_id, @language, @artisan_id, @craft_id, sqlc.narg('cluster_id'), @listing_type,
    @price_paise, sqlc.narg('lead_time_days'), @gi_certified, @provenance_sealed,
    @state_code, sqlc.narg('district'), @colours, @materials, @document,
    to_tsvector('simple', unaccent(@document::text)),
    sqlc.narg('embedding')::vector, sqlc.narg('model_version'), now()
)
ON CONFLICT ON CONSTRAINT listing_search_pkey DO UPDATE
SET artisan_id        = EXCLUDED.artisan_id,
    craft_id          = EXCLUDED.craft_id,
    cluster_id        = EXCLUDED.cluster_id,
    listing_type      = EXCLUDED.listing_type,
    price_paise       = EXCLUDED.price_paise,
    lead_time_days    = EXCLUDED.lead_time_days,
    gi_certified      = EXCLUDED.gi_certified,
    provenance_sealed = EXCLUDED.provenance_sealed,
    state_code        = EXCLUDED.state_code,
    district          = EXCLUDED.district,
    colours           = EXCLUDED.colours,
    materials         = EXCLUDED.materials,
    document          = EXCLUDED.document,
    document_tsv      = EXCLUDED.document_tsv,
    embedding         = COALESCE(EXCLUDED.embedding, listing_search.embedding),
    model_version     = COALESCE(EXCLUDED.model_version, listing_search.model_version),
    indexed_at        = now();

-- name: DeleteListingSearch :execrows
DELETE FROM listing_search WHERE listing_id = @listing_id;

-- Lexical half of hybrid retrieval. ts_rank_cd, not true BM25 — Postgres has no BM25,
-- and the ranks are only used for the fusion ordering, which is rank-based anyway.
-- name: SearchLexicalCandidates :many
SELECT s.listing_id, s.language, s.artisan_id, s.craft_id, s.price_paise,
       ts_rank_cd(s.document_tsv, q.query) AS score
FROM listing_search s,
     plainto_tsquery('simple', unaccent(@query::text)) AS q(query)
WHERE s.document_tsv @@ q.query
  AND s.language = @language
  AND (cardinality(@craft_ids::uuid[]) = 0 OR s.craft_id = ANY(@craft_ids))
  AND (cardinality(@colours::text[]) = 0 OR s.colours && @colours)
  AND (cardinality(@materials::text[]) = 0 OR s.materials && @materials)
  AND (@min_price_paise::bigint IS NULL OR s.price_paise >= @min_price_paise)
  AND (@max_price_paise::bigint IS NULL OR s.price_paise <= @max_price_paise)
  AND (@state_code::text IS NULL OR s.state_code = @state_code)
  AND (@listing_type::listing_type IS NULL OR s.listing_type = @listing_type)
  AND (NOT @gi_only::boolean OR s.gi_certified)
  AND (NOT @sealed_only::boolean OR s.provenance_sealed)
  AND (@max_lead_time_days::integer IS NULL OR s.lead_time_days <= @max_lead_time_days)
ORDER BY score DESC, s.listing_id
LIMIT @page_size;

-- Vector half of hybrid retrieval. Cosine distance, matching the HNSW opclass.
-- Fusion of the two candidate lists happens in Go, not here.
-- name: SearchVectorCandidates :many
SELECT s.listing_id, s.language, s.artisan_id, s.craft_id, s.price_paise,
       1 - (s.embedding <=> @embedding::vector) AS score
FROM listing_search s
WHERE s.embedding IS NOT NULL
  AND s.language = @language
  AND (cardinality(@craft_ids::uuid[]) = 0 OR s.craft_id = ANY(@craft_ids))
  AND (cardinality(@colours::text[]) = 0 OR s.colours && @colours)
  AND (cardinality(@materials::text[]) = 0 OR s.materials && @materials)
  AND (@min_price_paise::bigint IS NULL OR s.price_paise >= @min_price_paise)
  AND (@max_price_paise::bigint IS NULL OR s.price_paise <= @max_price_paise)
  AND (@state_code::text IS NULL OR s.state_code = @state_code)
  AND (@listing_type::listing_type IS NULL OR s.listing_type = @listing_type)
  AND (NOT @gi_only::boolean OR s.gi_certified)
  AND (NOT @sealed_only::boolean OR s.provenance_sealed)
  AND (@max_lead_time_days::integer IS NULL OR s.lead_time_days <= @max_lead_time_days)
ORDER BY s.embedding <=> @embedding::vector
LIMIT @page_size;

-- Rows whose projection predates the listing's last edit, for the reindex sweeper.
-- name: ListStaleSearchRows :many
SELECT s.listing_id, s.language FROM listing_search s
JOIN listing l ON l.id = s.listing_id
WHERE l.updated_at > s.indexed_at
ORDER BY s.indexed_at
LIMIT @page_size;

-- Everything one publish needs to build its searchable text: the listing, the
-- craft it belongs to with every alias in every script, and the artisan's region.
-- One row per language the listing has copy in.
--
-- The aliases are the cross-lingual trick: an English query and a Hindi listing
-- meet in one document, so neither has to be translated at query time.
-- name: LoadIndexSource :many
SELECT
    l.id AS listing_id, t.language, l.artisan_id, p.craft_id, a.primary_cluster_id AS cluster_id,
    l.type AS listing_type, l.price_paise, l.lead_time_days, l.gi_certified,
    (l.provenance_id IS NOT NULL) AS provenance_sealed,
    a.state_code, a.district,
    p.colours, p.materials, p.techniques, p.motifs,
    t.title, t.description, t.highlights,
    c.display_name AS craft_name,
    COALESCE((SELECT array_agg(al.alias ORDER BY al.script, al.alias)
              FROM craft_alias al WHERE al.craft_id = c.id), '{}') AS craft_aliases
FROM listing l
JOIN product p ON p.id = l.product_id
JOIN artisan a ON a.id = l.artisan_id
JOIN craft c ON c.id = p.craft_id
JOIN listing_translation t ON t.listing_id = l.id
WHERE l.id = @listing_id AND l.state = 'PUBLISHED';

-- The hit's own copy, for rendering labels in the requester's language. Falls
-- back to whatever copy exists when the listing has none in that language, so a
-- result is never rendered blank.
-- name: HydrateSearchHits :many
SELECT DISTINCT ON (l.id)
    l.id AS listing_id, l.product_id, l.artisan_id, p.craft_id,
    l.gi_certified, l.type AS listing_type, t.title, t.description
FROM listing l
JOIN product p ON p.id = l.product_id
LEFT JOIN listing_translation t ON t.listing_id = l.id
WHERE l.id = ANY(@listing_ids::uuid[])
ORDER BY l.id, (t.language = @language::language_code) DESC NULLS LAST, t.language;

-- Crafts adjacent to the ones a query pointed at, for the zero-result fallback.
-- Both directions of the edge, because "shares technique" is symmetric even
-- though the row is not.
-- name: ListSiblingCrafts :many
SELECT DISTINCT c.id AS craft_id, c.code, c.display_name, r.kind
FROM craft_relation r
JOIN craft c ON c.id = CASE WHEN r.from_craft_id = ANY(@craft_ids::uuid[])
                            THEN r.to_craft_id ELSE r.from_craft_id END
WHERE (r.from_craft_id = ANY(@craft_ids) OR r.to_craft_id = ANY(@craft_ids))
  AND NOT c.id = ANY(@craft_ids)
ORDER BY c.display_name
LIMIT @page_size;

-- Type-ahead over craft aliases: prefix first, then trigram similarity for the
-- misspellings voice search produces. The gin_trgm index on craft_alias.alias is
-- what keeps this under the latency budget.
-- name: SuggestCraftAliases :many
SELECT c.id AS craft_id, c.display_name, al.alias,
       CASE WHEN unaccent(lower(al.alias)) LIKE unaccent(lower(@prefix::text)) || '%'
            THEN 1.0 ELSE similarity(unaccent(al.alias), unaccent(@prefix)) END AS score
FROM craft_alias al
JOIN craft c ON c.id = al.craft_id
WHERE unaccent(lower(al.alias)) LIKE unaccent(lower(@prefix)) || '%'
   OR unaccent(al.alias) % unaccent(@prefix)
ORDER BY score DESC, c.display_name
LIMIT @page_size;

-- Type-ahead over what other buyers have successfully searched for.
-- name: SuggestPopularQueries :many
SELECT normalised, times_seen
FROM search_query_log
WHERE hit_count > 0
  AND (normalised LIKE @prefix::text || '%' OR normalised % @prefix)
ORDER BY (normalised LIKE @prefix || '%') DESC, times_seen DESC
LIMIT @page_size;

-- name: RecordSearchQuery :exec
INSERT INTO search_query_log (normalised, language, hit_count, times_seen, last_seen_at)
VALUES (@normalised, @language, @hit_count, 1, now())
ON CONFLICT ON CONSTRAINT search_query_log_pkey DO UPDATE
SET hit_count    = EXCLUDED.hit_count,
    times_seen   = search_query_log.times_seen + 1,
    last_seen_at = now();
