# Database Index Audit

**Last Updated:** 2026-09-15
**Database:** PostgreSQL 18 + pgvector

This file previously described a schema that never existed in this repo
(`users`, `artisans`, `listings`, `bulk_orders`, `order_allocations`,
`otp_challenges`, plural table names throughout — none of these are real; see
`CLAUDE.md`'s migrations section for the history of this class of mistake).
It has been rewritten from the actual `CREATE INDEX` statements in
`migrations/*.sql`. There is no `users` table anywhere in this schema — buyer
identity is external, and `bulk_order.buyer_id` is an opaque `text` column.

---

## Existing indexes, by domain

### Identity & ontology (`002_identity.sql`, `003_ontology.sql`, `013_identity_extensions.sql`)
- `artisan_primary_cluster_id_idx` on `artisan (primary_cluster_id)`
- `artisan_display_name_trgm_idx` — GIN trigram on `artisan (display_name)`, for fuzzy name search
- `cluster_member_artisan_id_idx` on `cluster_member (artisan_id)`
- `shg_member_artisan_id_idx` on `shg_member (artisan_id)`
- `craft_parent_craft_id_idx` on `craft (parent_craft_id)`
- `craft_alias_lower_alias_script_key` — UNIQUE on `craft_alias (lower(alias), script)`
- `craft_alias_alias_trgm_idx` — GIN trigram on `craft_alias (alias)`
- `craft_alias_craft_id_idx` on `craft_alias (craft_id)`
- `craft_relation_to_craft_id_idx` on `craft_relation (to_craft_id, kind)`
- `artisan_craft_craft_id_idx` on `artisan_craft (craft_id)`
- `artisan_craft_one_primary_key` — UNIQUE on `artisan_craft (artisan_id) WHERE is_primary`

### Catalog (`004_catalog.sql`, `014_listing_media.sql`, `016_pipeline.sql`)
- `product_artisan_id_created_at_idx` on `product (artisan_id, created_at DESC)`
- `listing_artisan_id_state_idx` on `listing (artisan_id, state)`
- `listing_product_id_idx` on `listing (product_id)`
- `listing_published_at_idx` — partial, `listing (published_at) WHERE state = 'PUBLISHED'`
- `listing_owner_shg_id_idx` — partial, `listing (owner_shg_id) WHERE owner_shg_id IS NOT NULL`
- `listing_needs_description_idx` — partial, `listing (artisan_id) WHERE needs_description`
- `listing_media_listing_id_ordinal_key` — UNIQUE `(listing_id, ordinal)`
- `listing_media_one_primary_image_key` / `listing_media_one_process_video_key` — UNIQUE partials enforcing "at most one primary image / process video per listing"
- `listing_media_media_id_idx` on `listing_media (media_id)`

### Media (`005_media.sql`, `015_media_lifecycle.sql`)
- `media_product_id_uploaded_at_idx` on `media (product_id, uploaded_at)`
- `media_sha256_hex_idx` on `media (sha256_hex)` — dedup lookup
- `media_pending_created_at_idx` — partial, `media (created_at) WHERE state = 'PENDING'` — this is what the upload-reaper query hits
- `media_artisan_id_state_idx` on `media (artisan_id, state)`

### Search (`006_search.sql`)
- `listing_search_document_tsv_idx` — GIN on `document_tsv` (lexical full-text)
- `listing_search_embedding_hnsw_idx` — HNSW cosine on `embedding vector(768)` (semantic)
- `listing_search_craft_id_price_paise_idx` on `(craft_id, price_paise)`
- `listing_search_colours_idx` / `listing_search_materials_idx` — GIN on array columns
- `listing_search_state_code_idx`, `listing_search_indexed_at_idx`

### Orders (`007_orders.sql`, `019_bulk_order_events.sql`)
- `bulk_order_buyer_id_created_at_idx` on `bulk_order (buyer_id, created_at DESC)`
- `bulk_order_state_required_by_idx` — partial index on `(state, required_by)`
- `order_lot_artisan_id_state_idx` on `order_lot (artisan_id, state)`
- `order_lot_bulk_order_id_idx` on `order_lot (bulk_order_id)`
- `order_lot_responds_by_idx` — partial, `order_lot (responds_by) WHERE state = 'OFFERED'`
- `capacity_reservation_held_idx`
- `bulk_order_event_bulk_order_id_occurred_at_idx` on `bulk_order_event (bulk_order_id, occurred_at)`

### QC & compensation (`020_qc_results.sql`, `021_compensation.sql`)
- `qc_result_lot_id_idx` on `qc_result (lot_id)`
- `qc_defect_qc_result_id_idx` on `qc_defect (qc_result_id)`
- `bulk_order_amendment_bulk_order_id_idx`
- `bulk_order_amendment_one_pending_idx` — UNIQUE, enforces one pending amendment per order

### Payments (`008_payments.sql`)
- `payment_split_line_payee_id_created_at_idx`
- `payment_split_line_unsettled_idx` — partial on `payment_split_id`
- `escrow_milestone_lot_id_trigger_idx` on `(lot_id, trigger)`

### Social (`009_social.sql`)
- `follow_follower_id_created_at_idx` on `follow (follower_id, created_at DESC)`
- `notification_recipient_unread_idx` — partial on `(recipient_id, created_at DESC)`

### Disputes (`010_disputes.sql`) — schema exists, no service implements dispute handling yet
- `dispute_state_created_at_idx`, `dispute_bulk_order_id_idx`

### Outbox & idempotency (`011_outbox.sql`, `012_idempotency.sql`)
- `outbox_unpublished_idx` — partial, `outbox (created_at) WHERE published_at IS NULL` — this is the index the relay's poll query actually uses
- `idempotency_key_expires_at_idx` on `idempotency_key (expires_at)` — TTL cleanup

### Pricing & search analytics (`017_search_queries.sql`, `018_pricing.sql`)
- `search_query_log_normalised_trgm_idx`, `search_query_log_times_seen_idx`
- `state_minimum_wage_state_code_effective_date_idx`
- `seasonality_multiplier_craft_id_month_idx`, `seasonality_multiplier_month_idx`

### Provenance (`022_provenance.sql`)
- `provenance_record_short_code_idx` — QR lookup
- `provenance_record_artisan_id_sealed_at_idx`
- `provenance_record_listing_id_idx`

### Insight materialized views & income statements (`023_insight.sql`)
- One index per materialized view keyed on its natural grouping (state/district, craft/month, decline rate, etc.)
- `income_statement_short_code_idx`, `income_statement_artisan_id_created_at_idx`

### Audit, fraud, webhooks (`025_audit_log.sql`, `026_fraud_detection.sql`, `027_webhooks.sql`)
- `idx_audit_log_timestamp`, `idx_audit_log_actor_id` (partial), `idx_audit_log_resource`, `idx_audit_log_action`
- `idx_fraud_flags_status` (partial: pending/reviewing), `idx_fraud_flags_resource`, `idx_fraud_flags_created_at`, `idx_fraud_flags_severity` (partial: high/critical)
- `idx_webhook_subscriptions_subscriber`, `idx_webhook_subscriptions_active` (partial), `idx_webhook_deliveries_pending`, `idx_webhook_deliveries_subscription`, `idx_webhook_deliveries_event`

### Notification delivery (`029_notification_delivery.sql`)
- `notification_delivery_notification_id_idx`
- `notification_delivery_status_created_at_idx` (partial)

### B2B (`031_b2b.sql`)
- `company_type_idx`, `company_verification_status_idx`, `company_boutique_location_idx`, `company_user_id_idx`
- `company_sale_settlement_company_idx`, `company_sale_settlement_order_idx`
- `company_interest_artisan_status_idx`, `company_interest_company_idx`
- `supply_partnership_artisan_active_idx`, `supply_partnership_company_active_idx`
- `boutique_match_artisan_score_idx`

### Trends, badges, schemes (`032_trends.sql`, `033_badges.sql`, `034_schemes.sql`)
- `trend_link_feed_idx`, `trend_link_craft_idx`, `trend_link_source_idx`
- `artisan_badge_artisan_active_idx`
- `scheme_criterion_scheme_idx`, `scheme_manual_check_scheme_idx`, `government_scheme_active_idx`

---

## Adding a new index

There's no dedicated "performance indexes" migration to append to — each
migration that introduces a table also introduces its own indexes at
creation time. Add a new index in a fresh numbered migration
(`migrations/0NN_<description>.sql`), and if the table already has
meaningful row counts, use `CREATE INDEX CONCURRENTLY` inside a
`-- +goose StatementBegin` / `-- +goose StatementEnd` block with
`-- +goose NO TRANSACTION` above it (goose won't let `CONCURRENTLY` run
inside its default per-file transaction).

## Monitoring index usage

```sql
-- Find unused indexes
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND indexrelname NOT LIKE '%_pkey'
ORDER BY pg_relation_size(indexrelid) DESC;

-- Check index bloat / size
SELECT schemaname, tablename, indexname,
       pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
ORDER BY pg_relation_size(indexrelid) DESC;
```

```sql
-- After bulk inserts or deletes
VACUUM ANALYZE listing;
VACUUM ANALYZE bulk_order;

SELECT schemaname, relname, last_vacuum, last_autovacuum, last_analyze
FROM pg_stat_user_tables
ORDER BY last_autovacuum;
```

To apply pending migrations against the real stack:
```sh
make migrate-up
```
