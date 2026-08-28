-- migrations/004_catalog.sql
-- +goose Up

-- Mirrors catalog.v1.ListingType without the UNSPECIFIED member.
CREATE TYPE listing_type AS ENUM ('MADE_TO_ORDER', 'READY_STOCK');

-- Mirrors catalog.v1.ListingState without the UNSPECIFIED member.
CREATE TYPE listing_state AS ENUM (
    'DRAFT', 'PENDING_ARTISAN_APPROVAL', 'PUBLISHED', 'SUSPENDED'
);

-- Who supplied an attribute value; drives whether the artisan still has to confirm it.
CREATE TYPE attribute_source AS ENUM ('MODEL', 'ARTISAN', 'CURATOR');

CREATE TABLE product (
    id            uuid        NOT NULL,
    artisan_id    uuid        NOT NULL,
    craft_id      uuid        NOT NULL,
    working_title text        NOT NULL,
    length_mm     integer,
    width_mm      integer,
    height_mm     integer,
    weight_g      integer,
    materials     text[]      NOT NULL DEFAULT '{}',
    techniques    text[]      NOT NULL DEFAULT '{}',
    colours       text[]      NOT NULL DEFAULT '{}',
    motifs        text[]      NOT NULL DEFAULT '{}',
    created_by    text        NOT NULL DEFAULT 'system',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT product_pkey PRIMARY KEY (id),
    CONSTRAINT product_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT product_craft_id_fkey FOREIGN KEY (craft_id)
        REFERENCES craft (id) ON DELETE RESTRICT,
    CONSTRAINT product_dimensions_check CHECK (
        (length_mm IS NULL OR length_mm > 0) AND
        (width_mm  IS NULL OR width_mm  > 0) AND
        (height_mm IS NULL OR height_mm > 0) AND
        (weight_g  IS NULL OR weight_g  > 0)
    )
);

-- The artisan's own catalogue screen pages by newest product first.
CREATE INDEX product_artisan_id_created_at_idx ON product (artisan_id, created_at DESC);

CREATE TABLE listing (
    id                                uuid          NOT NULL,
    product_id                        uuid          NOT NULL,
    artisan_id                        uuid          NOT NULL,
    type                              listing_type  NOT NULL,
    state                             listing_state NOT NULL DEFAULT 'DRAFT',
    price_paise                       bigint        NOT NULL,
    currency_code                     char(3)       NOT NULL DEFAULT 'INR',
    stock_quantity                    integer,
    min_order_quantity                integer       NOT NULL DEFAULT 1,
    lead_time_days                    integer,
    capacity_per_month                integer,
    accepting_orders                  boolean       NOT NULL DEFAULT true,
    advance_pct                       integer,
    packaging_fragile                 boolean       NOT NULL DEFAULT false,
    packaging_oversized               boolean       NOT NULL DEFAULT false,
    packaging_requires_custom_crating boolean       NOT NULL DEFAULT false,
    packed_length_mm                  integer,
    packed_width_mm                   integer,
    packed_height_mm                  integer,
    packed_weight_g                   integer,
    provenance_id                     uuid,
    gi_certified                      boolean       NOT NULL DEFAULT false,
    published_at                      timestamptz,
    suspension_reason                 text,
    created_by                        text          NOT NULL DEFAULT 'system',
    created_at                        timestamptz   NOT NULL DEFAULT now(),
    updated_at                        timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT listing_pkey PRIMARY KEY (id),
    CONSTRAINT listing_product_id_fkey FOREIGN KEY (product_id)
        REFERENCES product (id) ON DELETE CASCADE,
    CONSTRAINT listing_artisan_id_fkey FOREIGN KEY (artisan_id)
        REFERENCES artisan (id) ON DELETE CASCADE,
    CONSTRAINT listing_price_paise_check CHECK (price_paise >= 0),
    CONSTRAINT listing_currency_code_check CHECK (currency_code = 'INR'),
    CONSTRAINT listing_min_order_quantity_check CHECK (min_order_quantity > 0),
    CONSTRAINT listing_stock_quantity_check CHECK (stock_quantity IS NULL OR stock_quantity >= 0),
    CONSTRAINT listing_lead_time_days_check CHECK (lead_time_days IS NULL OR lead_time_days > 0),
    CONSTRAINT listing_capacity_per_month_check CHECK (capacity_per_month IS NULL OR capacity_per_month > 0),
    CONSTRAINT listing_advance_pct_check CHECK (advance_pct IS NULL OR advance_pct BETWEEN 0 AND 100),
    -- A made-to-order listing without a lead time cannot be quoted or allocated.
    CONSTRAINT listing_made_to_order_lead_time_check
        CHECK (type <> 'MADE_TO_ORDER' OR lead_time_days IS NOT NULL),
    -- Ready stock without a count cannot be decremented at checkout.
    CONSTRAINT listing_ready_stock_quantity_check
        CHECK (type <> 'READY_STOCK' OR stock_quantity IS NOT NULL),
    -- Suspension must record a reason; every other state must not carry one.
    CONSTRAINT listing_suspension_reason_check
        CHECK ((state = 'SUSPENDED') = (suspension_reason IS NOT NULL)),
    -- published_at is set exactly when the listing has reached PUBLISHED at least once.
    CONSTRAINT listing_published_at_check
        CHECK (state <> 'PUBLISHED' OR published_at IS NOT NULL)
);

-- The artisan dashboard filters their own listings by state.
CREATE INDEX listing_artisan_id_state_idx ON listing (artisan_id, state);
-- One listing per product is the norm, and GetListing arrives by product from the pipeline.
CREATE INDEX listing_product_id_idx ON listing (product_id);
-- search-svc backfills "everything published since T"; only published rows qualify.
CREATE INDEX listing_published_at_idx ON listing (published_at) WHERE state = 'PUBLISHED';

CREATE TABLE listing_translation (
    listing_id        uuid          NOT NULL,
    language          language_code NOT NULL,
    title             text          NOT NULL,
    description       text          NOT NULL,
    highlights        text[]        NOT NULL DEFAULT '{}',
    machine_generated boolean       NOT NULL DEFAULT true,
    edited_by         text,
    created_at        timestamptz   NOT NULL DEFAULT now(),
    updated_at        timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT listing_translation_pkey PRIMARY KEY (listing_id, language),
    CONSTRAINT listing_translation_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listing (id) ON DELETE CASCADE,
    CONSTRAINT listing_translation_title_check CHECK (length(title) > 0)
);

CREATE TABLE listing_attribute (
    id         uuid             NOT NULL,
    listing_id uuid             NOT NULL,
    name       text             NOT NULL,
    value      text             NOT NULL,
    confidence real             NOT NULL DEFAULT 1.0,
    source     attribute_source NOT NULL,
    created_at timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT listing_attribute_pkey PRIMARY KEY (id),
    CONSTRAINT listing_attribute_listing_id_name_value_key UNIQUE (listing_id, name, value),
    CONSTRAINT listing_attribute_listing_id_fkey FOREIGN KEY (listing_id)
        REFERENCES listing (id) ON DELETE CASCADE,
    CONSTRAINT listing_attribute_confidence_check CHECK (confidence >= 0.0 AND confidence <= 1.0)
);

-- +goose Down

DROP TABLE IF EXISTS listing_attribute;
DROP TABLE IF EXISTS listing_translation;
DROP TABLE IF EXISTS listing;
DROP TABLE IF EXISTS product;
DROP TYPE IF EXISTS attribute_source;
DROP TYPE IF EXISTS listing_state;
DROP TYPE IF EXISTS listing_type;
