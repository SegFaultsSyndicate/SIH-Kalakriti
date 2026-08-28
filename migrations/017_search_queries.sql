-- migrations/017_search_queries.sql
-- +goose Up

-- Successful queries, for type-ahead. Only the normalised form is kept: it is
-- what the suggester matches on, and it is not tied to a buyer.
CREATE TABLE search_query_log (
    normalised   text          NOT NULL,
    language     language_code NOT NULL,
    hit_count    integer       NOT NULL DEFAULT 0,
    times_seen   integer       NOT NULL DEFAULT 1,
    last_seen_at timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT search_query_log_pkey PRIMARY KEY (normalised, language),
    CONSTRAINT search_query_log_normalised_check CHECK (length(normalised) BETWEEN 2 AND 120)
);

-- Suggest is a prefix match with a trigram fallback, and must stay under 50ms.
CREATE INDEX search_query_log_normalised_trgm_idx ON search_query_log
    USING gin (normalised gin_trgm_ops);
-- Popularity is the tiebreak, so it is the sort key.
CREATE INDEX search_query_log_times_seen_idx ON search_query_log (times_seen DESC);

-- +goose Down

DROP TABLE IF EXISTS search_query_log;
