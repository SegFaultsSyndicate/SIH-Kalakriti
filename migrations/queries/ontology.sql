-- migrations/queries/ontology.sql

-- The ontology is small and read on every catalogue request, so it is loaded whole and cached.
-- name: ListCrafts :many
SELECT * FROM craft ORDER BY code;

-- name: GetCraft :one
SELECT * FROM craft WHERE id = @id;

-- name: GetCraftByCode :one
SELECT * FROM craft WHERE code = @code;

-- The other half of an index rebuild: every alias in one round trip. Ordered so
-- two rebuilds of the same data produce the same content hash.
-- name: ListAllCraftAliases :many
SELECT * FROM craft_alias ORDER BY craft_id, script, alias;

-- Exact alias resolution. Case-insensitive via the unique index on (lower(alias), script);
-- script-insensitive when @script is NULL, which is the voice-search path.
-- name: ResolveCraftAlias :one
SELECT c.*
FROM craft_alias a
JOIN craft c ON c.id = a.craft_id
WHERE lower(a.alias) = lower(@alias::text)
  AND (@script::text IS NULL OR a.script = @script)
ORDER BY a.created_at
LIMIT 1;

-- Fuzzy fallback when exact resolution misses: unaccent folds diacritics, trigram
-- similarity absorbs transliteration drift.
-- name: SearchCraftAliases :many
SELECT c.id, c.code, c.display_name, a.alias, a.script, a.language,
       similarity(unaccent(a.alias), unaccent(@alias::text)) AS score
FROM craft_alias a
JOIN craft c ON c.id = a.craft_id
WHERE unaccent(a.alias) % unaccent(@alias::text)
ORDER BY score DESC, c.code
LIMIT @page_size;

-- name: ListCraftAliases :many
SELECT * FROM craft_alias WHERE craft_id = @craft_id ORDER BY script, alias;

-- Conflict target is the code, so a re-seed updates the row and keeps its id.
-- name: UpsertCraft :one
INSERT INTO craft (id, code, display_name, parent_craft_id, gi_registration_no, techniques, materials)
VALUES (@id, @code, @display_name, sqlc.narg('parent_craft_id'),
        sqlc.narg('gi_registration_no'), @techniques, @materials)
ON CONFLICT ON CONSTRAINT craft_code_key DO UPDATE
SET display_name       = EXCLUDED.display_name,
    parent_craft_id    = EXCLUDED.parent_craft_id,
    gi_registration_no = EXCLUDED.gi_registration_no,
    techniques         = EXCLUDED.techniques,
    materials          = EXCLUDED.materials,
    updated_at         = now()
RETURNING *;

-- Conflict target is the expression index, so a re-seed updates rather than duplicates.
-- name: UpsertCraftAlias :one
INSERT INTO craft_alias (id, craft_id, alias, script, language, source)
VALUES (@id, @craft_id, @alias, @script, @language, @source)
ON CONFLICT (lower(alias), script) DO UPDATE
SET craft_id = EXCLUDED.craft_id,
    language = EXCLUDED.language,
    source   = EXCLUDED.source
RETURNING *;

-- name: UpsertCraftRelation :exec
INSERT INTO craft_relation (id, from_craft_id, to_craft_id, kind)
VALUES (@id, @from_craft_id, @to_craft_id, @kind)
ON CONFLICT ON CONSTRAINT craft_relation_edge_key DO NOTHING;
