-- migrations/013_identity_extensions.sql
-- +goose Up

-- Mirrors identity.v1.ClusterMemberRole without the UNSPECIFIED member; a stored
-- membership always knows its standing. cluster_member.role was free text in 002.
CREATE TYPE cluster_member_role AS ENUM ('MEMBER', 'COORDINATOR', 'MASTER');

ALTER TABLE cluster_member
    ALTER COLUMN role DROP DEFAULT,
    ALTER COLUMN role TYPE cluster_member_role USING role::cluster_member_role,
    ALTER COLUMN role SET DEFAULT 'MEMBER';

-- Mirrors identity.v1 listing ownership. An SHG can front a listing for finance and
-- bulk acceptance while an individual artisan is still its maker, so artisan_id stays
-- NOT NULL (it is the maker, and provenance depends on it) and ownership is a separate
-- axis. A real FK on each side beats a polymorphic owner_id with no referential integrity.
CREATE TYPE listing_owner_type AS ENUM ('ARTISAN', 'SHG');

ALTER TABLE listing
    ADD COLUMN owner_type   listing_owner_type NOT NULL DEFAULT 'ARTISAN',
    ADD COLUMN owner_shg_id uuid,
    ADD CONSTRAINT listing_owner_shg_id_fkey FOREIGN KEY (owner_shg_id)
        REFERENCES shg (id) ON DELETE RESTRICT,
    -- SHG ownership requires a group; artisan ownership forbids one.
    ADD CONSTRAINT listing_owner_shg_id_check
        CHECK ((owner_type = 'SHG') = (owner_shg_id IS NOT NULL));

-- Settlement fans an SHG payout out across its roster, so the roster is on the money path.
CREATE INDEX listing_owner_shg_id_idx ON listing (owner_shg_id) WHERE owner_shg_id IS NOT NULL;

-- Payout share of group earnings, whole percent. The per-row bound is a CHECK; the
-- cross-row "must sum to 100" invariant needs the deferred constraint trigger below.
ALTER TABLE shg_member
    ADD COLUMN share_pct integer NOT NULL DEFAULT 0,
    ADD CONSTRAINT shg_member_share_pct_check CHECK (share_pct >= 0 AND share_pct <= 100);

-- The service layer validates the split before writing, but settlement divides real money
-- by these numbers, so the invariant is enforced at commit time as well, whatever code
-- path writes. DEFERRABLE INITIALLY DEFERRED is what lets a roster be replaced row by row
-- inside one transaction without transiently tripping the check.
-- +goose StatementBegin
CREATE FUNCTION shg_member_shares_sum_to_100() RETURNS trigger AS $$
DECLARE
    target_shg uuid := COALESCE(NEW.shg_id, OLD.shg_id);
    member_count integer;
    total integer;
BEGIN
    SELECT count(*), COALESCE(sum(share_pct), 0)
      INTO member_count, total
      FROM shg_member
     WHERE shg_id = target_shg;

    -- An empty roster is allowed: a group can exist before anyone joins, and
    -- replacing a roster deletes every row before inserting the new ones.
    IF member_count = 0 THEN
        RETURN NULL;
    END IF;

    IF total <> 100 THEN
        RAISE EXCEPTION 'shg % member shares sum to %, expected 100', target_shg, total
            USING ERRCODE = 'check_violation',
                  CONSTRAINT = 'shg_member_shares_sum_to_100';
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER shg_member_shares_sum_to_100
    AFTER INSERT OR UPDATE OR DELETE ON shg_member
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION shg_member_shares_sum_to_100();

-- +goose Down

DROP TRIGGER IF EXISTS shg_member_shares_sum_to_100 ON shg_member;
DROP FUNCTION IF EXISTS shg_member_shares_sum_to_100();

ALTER TABLE shg_member
    DROP CONSTRAINT IF EXISTS shg_member_share_pct_check,
    DROP COLUMN IF EXISTS share_pct;

DROP INDEX IF EXISTS listing_owner_shg_id_idx;

ALTER TABLE listing
    DROP CONSTRAINT IF EXISTS listing_owner_shg_id_check,
    DROP CONSTRAINT IF EXISTS listing_owner_shg_id_fkey,
    DROP COLUMN IF EXISTS owner_shg_id,
    DROP COLUMN IF EXISTS owner_type;

DROP TYPE IF EXISTS listing_owner_type;

ALTER TABLE cluster_member
    ALTER COLUMN role DROP DEFAULT,
    ALTER COLUMN role TYPE text USING role::text,
    ALTER COLUMN role SET DEFAULT 'MEMBER';

DROP TYPE IF EXISTS cluster_member_role;
