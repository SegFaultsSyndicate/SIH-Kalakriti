-- +goose Up
-- +goose StatementBegin
CREATE TABLE income_statements (
    id UUID PRIMARY KEY,
    artisan_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    year INT NOT NULL,
    month INT NOT NULL CHECK (month >= 1 AND month <= 12),
    order_count INT NOT NULL DEFAULT 0,
    gross_paise BIGINT NOT NULL CHECK (gross_paise >= 0),
    fees_paise BIGINT NOT NULL CHECK (fees_paise >= 0),
    net_paise BIGINT NOT NULL CHECK (net_paise >= 0),
    code VARCHAR(20) NOT NULL UNIQUE,
    signature BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(artisan_id, year, month)
);

CREATE INDEX idx_income_statements_artisan ON income_statements(artisan_id);
CREATE INDEX idx_income_statements_code ON income_statements(code);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE income_statements;
-- +goose StatementEnd
