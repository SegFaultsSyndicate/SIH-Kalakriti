-- migrations/002_identity.sql
-- +goose Up

-- Mirrors common.v1.Language; LANGUAGE_UNSPECIFIED is deliberately absent
-- because a stored row always knows its language.
CREATE TYPE language_code AS ENUM (
    'ASSAMESE', 'BENGALI', 'BODO', 'DOGRI', 'GUJARATI', 'HINDI', 'KANNADA',
    'KASHMIRI', 'KONKANI', 'MAITHILI', 'MALAYALAM', 'MANIPURI', 'MARATHI',
    'NEPALI', 'ODIA', 'PUNJABI', 'SANSKRIT', 'SANTALI', 'SINDHI', 'TAMIL',
    'TELUGU', 'URDU', 'ENGLISH'
);

CREATE TABLE cluster (
    id                      uuid        NOT NULL,
    name                    text        NOT NULL,
    state_code              text        NOT NULL,
    district                text,
    block                   text,
    village                 text,
    pincode                 text,
    coordinator_phone_e164  text,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT cluster_pkey PRIMARY KEY (id),
    CONSTRAINT cluster_pincode_check CHECK (pincode IS NULL OR pincode ~ '^[1-9][0-9]{5}$')
);

CREATE TABLE artisan (
    id                  uuid          NOT NULL,
    user_id             text,
    display_name        text          NOT NULL,
    phone_e164          text          NOT NULL,
    -- Pehchan artisan card issued by the Ministry of Textiles.
    pehchan_id          text,
    -- PM Vishwakarma beneficiary id.
    pm_vishwakarma_id   text,
    primary_cluster_id  uuid,
    state_code          text          NOT NULL,
    district            text,
    block               text,
    village             text,
    pincode             text,
    languages           language_code[] NOT NULL DEFAULT '{}',
    years_of_experience integer,
    bio                 text,
    verified            boolean       NOT NULL DEFAULT false,
    created_by          text          NOT NULL DEFAULT 'system',
    created_at          timestamptz   NOT NULL DEFAULT now(),
    updated_at          timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT artisan_pkey PRIMARY KEY (id),
    CONSTRAINT artisan_phone_e164_key UNIQUE (phone_e164),
    CONSTRAINT artisan_user_id_key UNIQUE (user_id),
    CONSTRAINT artisan_pehchan_id_key UNIQUE (pehchan_id),
    CONSTRAINT artisan_pm_vishwakarma_id_key UNIQUE (pm_vishwakarma_id),
    CONSTRAINT artisan_primary_cluster_id_fkey FOREIGN KEY (primary_cluster_id)
        REFERENCES cluster (id) ON DELETE SET NULL,
    CONSTRAINT artisan_phone_e164_check CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
    CONSTRAINT artisan_years_of_experience_check CHECK (years_of_experience IS NULL OR years_of_experience >= 0)
);

-- Artisan lookup by cluster is the artisan-app home screen and the allocator's candidate scan.
CREATE INDEX artisan_primary_cluster_id_idx ON artisan (primary_cluster_id);
-- Field staff search artisans by partial name; trigram beats a leading-wildcard LIKE scan.
CREATE INDEX artisan_display_name_trgm_idx ON artisan USING gin (display_name gin_trgm_ops);

CREATE TABLE cluster_member (
    cluster_id uuid        NOT NULL,
    artisan_id uuid        NOT NULL,
    role       text        NOT NULL DEFAULT 'MEMBER',
    joined_at  timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT cluster_member_pkey PRIMARY KEY (cluster_id, artisan_id),
    CONSTRAINT cluster_member_cluster_id_fkey FOREIGN KEY (cluster_id)
        REFERENCES cluster (id) ON DELETE CASCADE,
    CONSTRAINT cluster_member_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE
);

-- Reverse lookup: every cluster an artisan belongs to, for the profile screen.
CREATE INDEX cluster_member_artisan_id_idx ON cluster_member (artisan_id);

CREATE TABLE shg (
    id                   uuid        NOT NULL,
    name                 text        NOT NULL,
    registration_no      text        NOT NULL,
    cluster_id           uuid,
    signatory_artisan_id uuid,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT shg_pkey PRIMARY KEY (id),
    CONSTRAINT shg_registration_no_key UNIQUE (registration_no),
    CONSTRAINT shg_cluster_id_fkey FOREIGN KEY (cluster_id)
        REFERENCES cluster (id) ON DELETE SET NULL,
    CONSTRAINT shg_signatory_artisan_id_fkey FOREIGN KEY (signatory_artisan_id)
        REFERENCES artisan (id) ON DELETE SET NULL
);

CREATE TABLE shg_member (
    shg_id     uuid        NOT NULL,
    artisan_id uuid        NOT NULL,
    joined_at  timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT shg_member_pkey PRIMARY KEY (shg_id, artisan_id),
    CONSTRAINT shg_member_shg_id_fkey FOREIGN KEY (shg_id)
        REFERENCES shg (id) ON DELETE CASCADE,
    CONSTRAINT shg_member_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE
);

-- Payment splits resolve artisan -> SHG payee, so the reverse edge is on the hot path.
CREATE INDEX shg_member_artisan_id_idx ON shg_member (artisan_id);

-- +goose Down

DROP TABLE IF EXISTS shg_member;
DROP TABLE IF EXISTS shg;
DROP TABLE IF EXISTS cluster_member;
DROP TABLE IF EXISTS artisan;
DROP TABLE IF EXISTS cluster;
DROP TYPE IF EXISTS language_code;
