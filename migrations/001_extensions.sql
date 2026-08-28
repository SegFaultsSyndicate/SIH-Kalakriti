-- migrations/001_extensions.sql
-- +goose Up

-- pgvector: embedding storage and HNSW indexes for search-svc.
CREATE EXTENSION IF NOT EXISTS vector;
-- pg_trgm: fuzzy alias matching on craft_alias and artisan names.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- unaccent: fold diacritics so transliterated craft names match.
CREATE EXTENSION IF NOT EXISTS unaccent;
-- pgcrypto: digest() for content hashes computed in-database.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- +goose Down

DROP EXTENSION IF EXISTS pgcrypto;
DROP EXTENSION IF EXISTS unaccent;
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS vector;
