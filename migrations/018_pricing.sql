-- migrations/018_pricing.sql
-- +goose Up

-- State minimum wage for skilled work, the cost floor's wage input. Keyed by
-- state and effective date so a rate revision is a new row, never an update  - 
-- a floor computed last month must still cite the rate that was live then.
-- effective_date is timestamptz, not date, so this table follows the same
-- "all timestamps TIMESTAMPTZ UTC" rule as everything else and the repo layer
-- never needs a second time-handling type for one column.
CREATE TABLE state_minimum_wage (
    state_code          text        NOT NULL,
    effective_date       timestamptz NOT NULL,
    wage_paise_per_hour  bigint      NOT NULL,
    source               text        NOT NULL DEFAULT 'seed',
    created_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT state_minimum_wage_pkey PRIMARY KEY (state_code, effective_date),
    CONSTRAINT state_minimum_wage_amount_check CHECK (wage_paise_per_hour > 0)
);

-- The floor query wants "the latest rate on or before this date" per state.
CREATE INDEX state_minimum_wage_state_code_effective_date_idx
    ON state_minimum_wage (state_code, effective_date DESC);

-- ponytail: illustrative skilled-wage figures for the prototype, not sourced from
-- a gazette notification. Swap for real per-state Schedule VB rates before this
-- advisory ships to a real artisan; the query shape (latest row <= as-of date)
-- does not change when the numbers do.
INSERT INTO state_minimum_wage (state_code, effective_date, wage_paise_per_hour) VALUES
    ('IN-UP', '2026-04-01', 6800),
    ('IN-WB', '2026-04-01', 6200),
    ('IN-RJ', '2026-04-01', 6500),
    ('IN-GJ', '2026-04-01', 7400),
    ('IN-TN', '2026-04-01', 7800),
    ('IN-KL', '2026-04-01', 8600),
    ('IN-OR', '2026-04-01', 6100),
    ('IN-AS', '2026-04-01', 6300),
    ('IN-MP', '2026-04-01', 6400),
    ('IN-BR', '2026-04-01', 6000),
    ('IN-MH', '2026-04-01', 7600),
    ('IN-KA', '2026-04-01', 7500);

-- Seasonality multiplier table: (craft, month, festival) -> demand multiplier.
-- craft_id NULL means the row applies across every craft (the festival drives
-- demand regardless of what is being made); a craft-specific row, when present,
-- takes precedence over a generic one for the same month in the lookup query.
-- multiplier is `real`, not `numeric`, matching listing_attribute.confidence:
-- one less pgtype.Numeric conversion in the repo layer for a value that is a
-- business ratio, never money.
CREATE TABLE seasonality_multiplier (
    id         uuid        NOT NULL,
    craft_id   uuid,
    month      smallint    NOT NULL,
    festival   text        NOT NULL,
    multiplier real        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT seasonality_multiplier_pkey PRIMARY KEY (id),
    CONSTRAINT seasonality_multiplier_craft_id_fkey FOREIGN KEY (craft_id)
        REFERENCES craft (id) ON DELETE CASCADE,
    CONSTRAINT seasonality_multiplier_month_check CHECK (month BETWEEN 1 AND 12),
    CONSTRAINT seasonality_multiplier_multiplier_check CHECK (multiplier > 0)
);

CREATE INDEX seasonality_multiplier_craft_id_month_idx
    ON seasonality_multiplier (craft_id, month);
CREATE INDEX seasonality_multiplier_month_idx
    ON seasonality_multiplier (month) WHERE craft_id IS NULL;

-- Diwali's calendar date drifts between October and November; both months are
-- seeded so the lookup does not miss it in either kind of year.
INSERT INTO seasonality_multiplier (id, craft_id, month, festival, multiplier) VALUES
    (gen_random_uuid(), NULL, 10, 'Diwali', 1.25),
    (gen_random_uuid(), NULL, 11, 'Diwali', 1.30),
    (gen_random_uuid(), NULL, 9,  'Durga Puja', 1.20),
    (gen_random_uuid(), NULL, 10, 'Durga Puja', 1.20),
    (gen_random_uuid(), NULL, 8,  'Onam', 1.15),
    (gen_random_uuid(), NULL, 9,  'Onam', 1.15),
    (gen_random_uuid(), NULL, 1,  'Pongal', 1.15),
    (gen_random_uuid(), NULL, 11, 'Wedding Season', 1.20),
    (gen_random_uuid(), NULL, 12, 'Wedding Season', 1.20),
    (gen_random_uuid(), NULL, 1,  'Wedding Season', 1.20),
    (gen_random_uuid(), NULL, 2,  'Wedding Season', 1.15),
    (gen_random_uuid(), NULL, 4,  'Wedding Season', 1.10),
    (gen_random_uuid(), NULL, 3,  'Regional Fair Season', 1.10);

-- +goose Down

DROP TABLE IF EXISTS seasonality_multiplier;
DROP TABLE IF EXISTS state_minimum_wage;
