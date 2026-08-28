-- migrations/006_search.sql
-- +goose Up

-- One row per listing per language. Filter columns are denormalised from listing and
-- product so a candidate fetch never joins: retrieval is the latency-critical path.
CREATE TABLE listing_search (
    listing_id        uuid          NOT NULL,
    language          language_code NOT NULL,
    artisan_id        uuid          NOT NULL,
    craft_id          uuid          NOT NULL,
    cluster_id        uuid,
    listing_type      listing_type  NOT NULL,
    price_paise       bigint        NOT NULL,
    lead_time_days    integer,
    gi_certified      boolean       NOT NULL DEFAULT false,
    provenance_sealed boolean       NOT NULL DEFAULT false,
    state_code        text          NOT NULL,
    district          text,
    colours           text[]        NOT NULL DEFAULT '{}',
    materials         text[]        NOT NULL DEFAULT '{}',
    document          text          NOT NULL,
    document_tsv      tsvector      NOT NULL,
    embedding         vector(768),
    model_version     text,
    indexed_at        timestamptz   NOT NULL DEFAULT now(),
    created_at        timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT listing_search_pkey PRIMARY KEY (listing_id, language),
    CONSTRAINT listing_search_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listing (id) ON DELETE CASCADE,
    CONSTRAINT listing_search_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT listing_search_craft_id_fkey FOREIGN KEY (craft_id)
        REFERENCES craft (id) ON DELETE CASCADE,
    CONSTRAINT listing_search_cluster_id_fkey FOREIGN KEY (cluster_id)
        REFERENCES cluster (id) ON DELETE SET NULL,
    CONSTRAINT listing_search_price_paise_check CHECK (price_paise >= 0),
    -- An embedding is meaningless without the bundle that produced it; vectors from
    -- different bundles are not comparable.
    CONSTRAINT listing_search_embedding_model_check
        CHECK ((embedding IS NULL) = (model_version IS NULL))
);

-- Lexical half of hybrid retrieval.
CREATE INDEX listing_search_document_tsv_idx ON listing_search USING gin (document_tsv);
-- Vector half of hybrid retrieval; cosine because the embedder emits unit-normalised vectors.
CREATE INDEX listing_search_embedding_hnsw_idx ON listing_search
    USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);
-- Craft plus price is the most common structured narrowing before either retriever runs.
CREATE INDEX listing_search_craft_id_price_paise_idx ON listing_search (craft_id, price_paise);
-- Colour and material filters are array containment, which needs GIN.
CREATE INDEX listing_search_colours_idx ON listing_search USING gin (colours);
CREATE INDEX listing_search_materials_idx ON listing_search USING gin (materials);
-- Regional discovery ("crafts from Assam") filters on state before ranking.
CREATE INDEX listing_search_state_code_idx ON listing_search (state_code);
-- Reindex sweeps pick up rows whose projection is older than the listing.
CREATE INDEX listing_search_indexed_at_idx ON listing_search (indexed_at);

-- +goose Down

DROP TABLE IF EXISTS listing_search;
