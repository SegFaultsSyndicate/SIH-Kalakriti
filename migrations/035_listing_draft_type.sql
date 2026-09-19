-- migrations/035_listing_draft_type.sql
-- +goose Up

-- A listing's commercial type (MADE_TO_ORDER/READY_STOCK) and its type-specific
-- required fields (lead_time_days+capacity_per_month, or stock_quantity) are
-- chosen across several steps of the artisan app's listing wizard, not all at
-- once: POST /listings (the wizard's story step) only ever sends craft_id and
-- working_title, well before the wizard's pricing step lets the artisan choose
-- a type at all. Until this migration, `type` was NOT NULL with no default and
-- the two "type-specific field required" CHECK constraints below had no DRAFT
-- exemption, so core-svc rejected that very first, deliberately incomplete
-- POST /listings call on every attempt -- no listing could ever be created.
-- See WIRING_AUDIT_PLAN.md's listing-lifecycle entry for the full trace.
--
-- A DRAFT listing (state = 'DRAFT', never visible to buyers -- see
-- listing_published_at_idx's WHERE clause) may now be assembled incrementally
-- with an unset type and incomplete type-specific fields. Anything past DRAFT
-- must be complete: SubmitForApproval enforces this in application code
-- before allowing the DRAFT -> PENDING_ARTISAN_APPROVAL transition, and these
-- CHECK constraints (extended, not removed) are the database's own backstop
-- against a bypass.
ALTER TABLE listing ALTER COLUMN type DROP NOT NULL;

ALTER TABLE listing DROP CONSTRAINT listing_made_to_order_lead_time_check;
ALTER TABLE listing ADD CONSTRAINT listing_made_to_order_lead_time_check
    CHECK (state = 'DRAFT' OR type <> 'MADE_TO_ORDER' OR lead_time_days IS NOT NULL);

ALTER TABLE listing DROP CONSTRAINT listing_ready_stock_quantity_check;
ALTER TABLE listing ADD CONSTRAINT listing_ready_stock_quantity_check
    CHECK (state = 'DRAFT' OR type <> 'READY_STOCK' OR stock_quantity IS NOT NULL);

-- type itself becoming nullable opens a gap neither existing constraint
-- covers: NULL <> 'MADE_TO_ORDER' evaluates to NULL, which a CHECK treats as
-- passing, not failing -- so without this, a still-typeless listing could
-- reach PUBLISHED. type must be chosen by the time a listing leaves DRAFT.
ALTER TABLE listing ADD CONSTRAINT listing_type_required_after_draft_check
    CHECK (state = 'DRAFT' OR type IS NOT NULL);

-- +goose Down
ALTER TABLE listing DROP CONSTRAINT listing_type_required_after_draft_check;

ALTER TABLE listing DROP CONSTRAINT listing_ready_stock_quantity_check;
ALTER TABLE listing ADD CONSTRAINT listing_ready_stock_quantity_check
    CHECK (type <> 'READY_STOCK' OR stock_quantity IS NOT NULL);

ALTER TABLE listing DROP CONSTRAINT listing_made_to_order_lead_time_check;
ALTER TABLE listing ADD CONSTRAINT listing_made_to_order_lead_time_check
    CHECK (type <> 'MADE_TO_ORDER' OR lead_time_days IS NOT NULL);

ALTER TABLE listing ALTER COLUMN type SET NOT NULL;
