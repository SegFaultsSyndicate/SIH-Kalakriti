-- migrations/003_ontology.sql
-- +goose Up

-- Edge kinds in the craft graph. DB-only; the proto exposes only parent_craft_id.
CREATE TYPE craft_relation_kind AS ENUM (
    'PARENT_OF', 'VARIANT_OF', 'RELATED_TO', 'SHARES_TECHNIQUE'
);

CREATE TABLE craft (
    id                 uuid        NOT NULL,
    code               text        NOT NULL,
    display_name       text        NOT NULL,
    parent_craft_id    uuid,
    gi_registration_no text,
    techniques         text[]      NOT NULL DEFAULT '{}',
    materials          text[]      NOT NULL DEFAULT '{}',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT craft_pkey PRIMARY KEY (id),
    CONSTRAINT craft_code_key UNIQUE (code),
    CONSTRAINT craft_gi_registration_no_key UNIQUE (gi_registration_no),
    CONSTRAINT craft_parent_craft_id_fkey FOREIGN KEY (parent_craft_id)
        REFERENCES craft (id) ON DELETE SET NULL,
    CONSTRAINT craft_code_check CHECK (code ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT craft_parent_not_self_check CHECK (parent_craft_id IS NULL OR parent_craft_id <> id)
);

-- Walking the ontology down from a root craft is how the taxonomy page and the
-- search craft filter both expand a selection.
CREATE INDEX craft_parent_craft_id_idx ON craft (parent_craft_id);

CREATE TABLE craft_alias (
    id         uuid          NOT NULL,
    craft_id   uuid          NOT NULL,
    alias      text          NOT NULL,
    -- ISO 15924 script code, e.g. 'Latn', 'Deva', 'Beng'.
    script     text          NOT NULL,
    language   language_code NOT NULL,
    source     text          NOT NULL DEFAULT 'curator',
    created_at timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT craft_alias_pkey PRIMARY KEY (id),
    CONSTRAINT craft_alias_craft_id_fkey FOREIGN KEY (craft_id)
        REFERENCES craft (id) ON DELETE CASCADE,
    CONSTRAINT craft_alias_script_check CHECK (script ~ '^[A-Z][a-z]{3}$')
);

-- One spelling per script resolves to exactly one craft; this is the resolver's guard.
CREATE UNIQUE INDEX craft_alias_lower_alias_script_key ON craft_alias (lower(alias), script);
-- Misspelt and partially typed aliases arrive constantly from voice search.
CREATE INDEX craft_alias_alias_trgm_idx ON craft_alias USING gin (alias gin_trgm_ops);
-- Loading every alias for one craft renders the craft page's "also known as" block.
CREATE INDEX craft_alias_craft_id_idx ON craft_alias (craft_id);

CREATE TABLE craft_relation (
    id            uuid                NOT NULL,
    from_craft_id uuid                NOT NULL,
    to_craft_id   uuid                NOT NULL,
    kind          craft_relation_kind NOT NULL,
    created_at    timestamptz         NOT NULL DEFAULT now(),
    CONSTRAINT craft_relation_pkey PRIMARY KEY (id),
    CONSTRAINT craft_relation_edge_key UNIQUE (from_craft_id, to_craft_id, kind),
    CONSTRAINT craft_relation_from_craft_id_fkey FOREIGN KEY (from_craft_id)
        REFERENCES craft (id) ON DELETE CASCADE,
    CONSTRAINT craft_relation_to_craft_id_fkey FOREIGN KEY (to_craft_id)
        REFERENCES craft (id) ON DELETE CASCADE,
    CONSTRAINT craft_relation_no_self_edge_check CHECK (from_craft_id <> to_craft_id)
);

-- Traversal from the far end of an edge, for "related crafts" on a listing page.
CREATE INDEX craft_relation_to_craft_id_idx ON craft_relation (to_craft_id, kind);

CREATE TABLE artisan_craft (
    artisan_id uuid        NOT NULL,
    craft_id   uuid        NOT NULL,
    is_primary boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT artisan_craft_pkey PRIMARY KEY (artisan_id, craft_id),
    CONSTRAINT artisan_craft_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT artisan_craft_craft_id_fkey FOREIGN KEY (craft_id)
        REFERENCES craft (id) ON DELETE RESTRICT
);

-- The allocator scans "artisans who practise craft X" on every bulk order.
CREATE INDEX artisan_craft_craft_id_idx ON artisan_craft (craft_id);
-- At most one primary craft per artisan drives the default listing category.
CREATE UNIQUE INDEX artisan_craft_one_primary_key ON artisan_craft (artisan_id) WHERE is_primary;

-- +goose Down

DROP TABLE IF EXISTS artisan_craft;
DROP TABLE IF EXISTS craft_relation;
DROP TABLE IF EXISTS craft_alias;
DROP TABLE IF EXISTS craft;
DROP TYPE IF EXISTS craft_relation_kind;
