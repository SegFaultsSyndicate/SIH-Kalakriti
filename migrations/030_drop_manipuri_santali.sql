-- migrations/030_drop_manipuri_santali.sql
-- +goose Up

-- Dropping Manipuri and Santali from the supported language list: both use
-- scripts (Meitei Mayek, Ol Chiki) with the weakest font/input support of the
-- 22 scheduled languages, and neither has any real translated content in the
-- frontend catalogue (packages/i18n's `coverage: 'fallback'`) -- so this
-- removes zero real translation work.
--
-- Postgres cannot drop enum values in place, so this recreates the type
-- without them and repoints every column that uses it. Any row still using
-- MANIPURI/SANTALI is defensively cleared first (expected to be none in
-- practice -- this is demo/dev data with no real translation pipeline output
-- yet in either language).

DELETE FROM search_query_log WHERE language IN ('MANIPURI', 'SANTALI');
DELETE FROM listing_search WHERE language IN ('MANIPURI', 'SANTALI');
DELETE FROM listing_translation WHERE language IN ('MANIPURI', 'SANTALI');
DELETE FROM craft_alias WHERE language IN ('MANIPURI', 'SANTALI');
DELETE FROM notification WHERE language IN ('MANIPURI', 'SANTALI');
UPDATE artisan SET languages = array_remove(array_remove(languages, 'MANIPURI'), 'SANTALI')
    WHERE 'MANIPURI' = ANY(languages) OR 'SANTALI' = ANY(languages);

CREATE TYPE language_code_new AS ENUM (
    'ASSAMESE', 'BENGALI', 'BODO', 'DOGRI', 'GUJARATI', 'HINDI', 'KANNADA',
    'KASHMIRI', 'KONKANI', 'MAITHILI', 'MALAYALAM', 'MARATHI',
    'NEPALI', 'ODIA', 'PUNJABI', 'SANSKRIT', 'SINDHI', 'TAMIL',
    'TELUGU', 'URDU', 'ENGLISH'
);

ALTER TABLE artisan
    ALTER COLUMN languages DROP DEFAULT,
    ALTER COLUMN languages TYPE language_code_new[] USING languages::text[]::language_code_new[],
    ALTER COLUMN languages SET DEFAULT '{}';

ALTER TABLE craft_alias
    ALTER COLUMN language TYPE language_code_new USING language::text::language_code_new;

ALTER TABLE listing_translation
    ALTER COLUMN language TYPE language_code_new USING language::text::language_code_new;

ALTER TABLE listing_search
    ALTER COLUMN language TYPE language_code_new USING language::text::language_code_new;

ALTER TABLE search_query_log
    ALTER COLUMN language TYPE language_code_new USING language::text::language_code_new;

ALTER TABLE notification
    ALTER COLUMN language DROP DEFAULT,
    ALTER COLUMN language TYPE language_code_new USING language::text::language_code_new,
    ALTER COLUMN language SET DEFAULT 'ENGLISH'::language_code_new;

DROP TYPE language_code;
ALTER TYPE language_code_new RENAME TO language_code;

-- +goose Down

-- Data for MANIPURI/SANTALI deleted by the Up migration is not recoverable;
-- this only restores the two enum values for new rows going forward.
CREATE TYPE language_code_old AS ENUM (
    'ASSAMESE', 'BENGALI', 'BODO', 'DOGRI', 'GUJARATI', 'HINDI', 'KANNADA',
    'KASHMIRI', 'KONKANI', 'MAITHILI', 'MALAYALAM', 'MANIPURI', 'MARATHI',
    'NEPALI', 'ODIA', 'PUNJABI', 'SANSKRIT', 'SANTALI', 'SINDHI', 'TAMIL',
    'TELUGU', 'URDU', 'ENGLISH'
);

ALTER TABLE artisan
    ALTER COLUMN languages DROP DEFAULT,
    ALTER COLUMN languages TYPE language_code_old[] USING languages::text[]::language_code_old[],
    ALTER COLUMN languages SET DEFAULT '{}';

ALTER TABLE craft_alias
    ALTER COLUMN language TYPE language_code_old USING language::text::language_code_old;

ALTER TABLE listing_translation
    ALTER COLUMN language TYPE language_code_old USING language::text::language_code_old;

ALTER TABLE listing_search
    ALTER COLUMN language TYPE language_code_old USING language::text::language_code_old;

ALTER TABLE search_query_log
    ALTER COLUMN language TYPE language_code_old USING language::text::language_code_old;

ALTER TABLE notification
    ALTER COLUMN language DROP DEFAULT,
    ALTER COLUMN language TYPE language_code_old USING language::text::language_code_old,
    ALTER COLUMN language SET DEFAULT 'ENGLISH'::language_code_old;

DROP TYPE language_code;
ALTER TYPE language_code_old RENAME TO language_code;
