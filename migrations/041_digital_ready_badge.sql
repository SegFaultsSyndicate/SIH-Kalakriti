-- migrations/041_digital_ready_badge.sql
-- +goose NO TRANSACTION
-- +goose Up

-- A new enum value cannot be used in the transaction that adds it, so this
-- migration runs outside one: the ALTER commits before the INSERT reads it.
ALTER TYPE badge_metric ADD VALUE IF NOT EXISTS 'LESSONS_COMPLETED';

INSERT INTO badge (id, code, kind, tier, icon_name, metric, threshold, sort_order)
SELECT gen_random_uuid(), 'digital_ready', 'EARNED', NULL, 'badge-milestone', 'LESSONS_COMPLETED', 8, 100
WHERE NOT EXISTS (SELECT 1 FROM badge WHERE code = 'digital_ready');

-- +goose Down

DELETE FROM badge WHERE code = 'digital_ready';
-- badge_metric's LESSONS_COMPLETED value stays: Postgres cannot drop an enum value.
