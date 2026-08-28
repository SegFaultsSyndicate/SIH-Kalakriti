# Database Index Audit & Recommendations

**Last Updated:** 2026-08-28  
**Database:** PostgreSQL 16 + pgvector

---

## Existing Indexes

Based on migrations analysis, these indexes already exist:

### Identity & Auth
- `idx_users_phone` on `users(phone_e164)` — login lookups
- `idx_artisans_user_id` on `artisans(user_id)` — profile lookups
- `idx_otp_challenges_phone` on `otp_challenges(phone_e164)` — OTP verification

### Catalog
- `idx_listings_artisan_id` on `listings(artisan_id)` — artisan's listings
- `idx_listings_status` on `listings(status)` — filter by status
- `idx_listings_craft` on `listings(craft)` — filter by craft type
- `idx_listings_created_at` on `listings(created_at DESC)` — recent listings
- `idx_listings_published_at` on `listings(published_at DESC)` — recent published

### Media
- `idx_media_artisan_id` on `media(artisan_id)` — artisan's media
- `idx_media_status` on `media(status)` — pending/uploaded filter
- `idx_listing_media_listing_id` on `listing_media(listing_id)` — listing's photos
- `idx_listing_media_media_id` on `listing_media(media_id)` — media usage

### Search
- `idx_listings_embedding_hnsw` on `listings USING hnsw(embedding vector_cosine_ops)` — vector search
- `idx_search_queries_query_text` on `search_queries(query_text)` — analytics

### Orders
- `idx_bulk_orders_buyer_id` on `bulk_orders(buyer_id)` — buyer's orders
- `idx_bulk_orders_status` on `bulk_orders(status)` — order pipeline
- `idx_order_lots_bulk_order_id` on `order_lots(bulk_order_id)` — order's lots
- `idx_order_allocations_lot_id` on `order_allocations(lot_id)` — lot allocations
- `idx_order_allocations_artisan_id` on `order_allocations(artisan_id)` — artisan's work
- `idx_qc_results_allocation_id` on `qc_results(allocation_id)` — QC lookups

### Payments
- `idx_payment_splits_bulk_order_id` on `payment_splits(bulk_order_id)` — order payments
- `idx_payment_split_lines_split_id` on `payment_split_lines(split_id)` — split details
- `idx_payment_split_lines_payee` on `payment_split_lines(payee_type, payee_id)` — payee lookup

### Social
- `idx_follows_follower_id` on `follows(follower_id)` — user's follows
- `idx_follows_artisan_id` on `follows(artisan_id)` — artisan's followers
- `idx_follows_unique` UNIQUE on `follows(follower_id, artisan_id)` — prevent duplicates

### Events & Outbox
- `idx_outbox_status` on `outbox(status)` — pending events
- `idx_outbox_created_at` on `outbox(created_at)` — event ordering
- `idx_bulk_order_events_order_id` on `bulk_order_events(bulk_order_id)` — order timeline

### Provenance
- `idx_provenance_records_listing_id` on `provenance_records(listing_id)` — product provenance
- `idx_provenance_records_code` on `provenance_records(code)` — QR lookup

### Income Statements
- `idx_income_statements_artisan` on `income_statements(artisan_id)` — artisan statements
- `idx_income_statements_code` on `income_statements(code)` — verification lookup

### Infrastructure
- `idx_idempotency_keys_key` UNIQUE on `idempotency_keys(key)` — idempotency
- `idx_idempotency_keys_expires_at` on `idempotency_keys(expires_at)` — TTL cleanup

---

## Missing Indexes (High Impact)

### 1. Composite Index for Listing Filters
**Problem:** Filtering listings by multiple criteria (status + craft + price range) does full table scan.

**Query Pattern:**
```sql
SELECT * FROM listings 
WHERE status = 'published' 
  AND craft = 'madhubani' 
  AND price_paise BETWEEN 100000 AND 500000
ORDER BY published_at DESC
LIMIT 20;
```

**Add:**
```sql
CREATE INDEX idx_listings_status_craft_price 
ON listings(status, craft, price_paise, published_at DESC);
```

**Impact:** 10-100x faster for filtered searches.

---

### 2. Artisan Search by Location
**Problem:** No index for district/state filters.

**Query Pattern:**
```sql
SELECT * FROM artisans 
WHERE state = 'Bihar' 
  AND district = 'Madhubani';
```

**Add:**
```sql
CREATE INDEX idx_artisans_location 
ON artisans(state, district);
```

**Impact:** Fast location-based artisan discovery.

---

### 3. Order Allocation Status
**Problem:** Dashboard queries "orders in production" scan all allocations.

**Query Pattern:**
```sql
SELECT * FROM order_allocations 
WHERE status = 'in_production' 
  AND deadline < NOW() + INTERVAL '7 days';
```

**Add:**
```sql
CREATE INDEX idx_order_allocations_status_deadline 
ON order_allocations(status, deadline);
```

**Impact:** Fast dashboard for overdue/urgent orders.

---

### 4. Media Pending Cleanup
**Problem:** Reaper job scans all media to find expired pending uploads.

**Query Pattern:**
```sql
DELETE FROM media 
WHERE status = 'pending' 
  AND created_at < NOW() - INTERVAL '1 hour';
```

**Add:**
```sql
CREATE INDEX idx_media_status_created_at 
ON media(status, created_at) 
WHERE status = 'pending';
```

**Impact:** Partial index makes reaper job instant.

---

### 5. Outbox Relay Performance
**Problem:** Outbox relay polls for pending events every second.

**Query Pattern:**
```sql
SELECT * FROM outbox 
WHERE status = 'pending' 
ORDER BY created_at 
LIMIT 100 
FOR UPDATE SKIP LOCKED;
```

**Existing:** `idx_outbox_status` + `idx_outbox_created_at` (separate indexes)

**Add:**
```sql
CREATE INDEX idx_outbox_pending_fifo 
ON outbox(created_at) 
WHERE status = 'pending';
```

**Impact:** Partial index optimized for the common case (pending events).

---

### 6. Feed Generation Performance
**Problem:** Generating user feed scans all followed artisans' listings.

**Query Pattern:**
```sql
SELECT l.* FROM listings l
JOIN follows f ON l.artisan_id = f.artisan_id
WHERE f.follower_id = '...' 
  AND l.status = 'published'
ORDER BY l.published_at DESC
LIMIT 20;
```

**Add:**
```sql
CREATE INDEX idx_listings_published_recent 
ON listings(artisan_id, published_at DESC) 
WHERE status = 'published';
```

**Impact:** Fast personalized feed queries.

---

### 7. Payment Split Lookup by Artisan
**Problem:** "How much do I earn this month?" queries are slow.

**Query Pattern:**
```sql
SELECT SUM(psl.amount_paise) 
FROM payment_split_lines psl
JOIN payment_splits ps ON psl.split_id = ps.id
WHERE psl.payee_type = 'artisan' 
  AND psl.payee_id = '...'
  AND ps.created_at >= '2026-08-01'
  AND ps.created_at < '2026-09-01';
```

**Add:**
```sql
CREATE INDEX idx_payment_splits_created_at 
ON payment_splits(created_at);
```

**Impact:** Fast income aggregation queries.

---

## Performance Testing

Run these queries before/after adding indexes:

```sql
-- Enable timing
\timing on

-- Test 1: Filtered listing search
EXPLAIN ANALYZE
SELECT * FROM listings 
WHERE status = 'published' 
  AND craft = 'madhubani' 
  AND price_paise BETWEEN 100000 AND 500000
ORDER BY published_at DESC
LIMIT 20;

-- Test 2: Artisan location search
EXPLAIN ANALYZE
SELECT * FROM artisans 
WHERE state = 'Bihar';

-- Test 3: Urgent orders dashboard
EXPLAIN ANALYZE
SELECT * FROM order_allocations 
WHERE status = 'in_production' 
  AND deadline < NOW() + INTERVAL '7 days';

-- Test 4: User feed
EXPLAIN ANALYZE
SELECT l.* FROM listings l
JOIN follows f ON l.artisan_id = f.artisan_id
WHERE f.follower_id = (SELECT id FROM users LIMIT 1)
  AND l.status = 'published'
ORDER BY l.published_at DESC
LIMIT 20;

-- Test 5: Outbox relay
EXPLAIN ANALYZE
SELECT * FROM outbox 
WHERE status = 'pending' 
ORDER BY created_at 
LIMIT 100;
```

**Before indexes:** Look for "Seq Scan" in EXPLAIN output.  
**After indexes:** Should see "Index Scan" or "Index Only Scan".

---

## Migration File

Create `migrations/024_performance_indexes.sql`:

```sql
-- +goose Up
-- +goose StatementBegin

-- 1. Composite index for listing filters
CREATE INDEX CONCURRENTLY idx_listings_status_craft_price 
ON listings(status, craft, price_paise, published_at DESC);

-- 2. Artisan location search
CREATE INDEX CONCURRENTLY idx_artisans_location 
ON artisans(state, district);

-- 3. Order allocation status + deadline
CREATE INDEX CONCURRENTLY idx_order_allocations_status_deadline 
ON order_allocations(status, deadline);

-- 4. Media pending cleanup (partial index)
CREATE INDEX CONCURRENTLY idx_media_status_created_at 
ON media(status, created_at) 
WHERE status = 'pending';

-- 5. Outbox relay optimization (partial index)
CREATE INDEX CONCURRENTLY idx_outbox_pending_fifo 
ON outbox(created_at) 
WHERE status = 'pending';

-- 6. Published listings feed (partial index)
CREATE INDEX CONCURRENTLY idx_listings_published_recent 
ON listings(artisan_id, published_at DESC) 
WHERE status = 'published';

-- 7. Payment split time range queries
CREATE INDEX CONCURRENTLY idx_payment_splits_created_at 
ON payment_splits(created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX CONCURRENTLY IF EXISTS idx_listings_status_craft_price;
DROP INDEX CONCURRENTLY IF EXISTS idx_artisans_location;
DROP INDEX CONCURRENTLY IF EXISTS idx_order_allocations_status_deadline;
DROP INDEX CONCURRENTLY IF EXISTS idx_media_status_created_at;
DROP INDEX CONCURRENTLY IF EXISTS idx_outbox_pending_fifo;
DROP INDEX CONCURRENTLY IF EXISTS idx_listings_published_recent;
DROP INDEX CONCURRENTLY IF EXISTS idx_payment_splits_created_at;

-- +goose StatementEnd
```

**Note:** `CONCURRENTLY` allows creating indexes without blocking writes (safe for production).

---

## Index Maintenance

### Monitor Index Usage
```sql
-- Find unused indexes
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND indexrelname NOT LIKE '%_pkey'
ORDER BY pg_relation_size(indexrelid) DESC;

-- Find duplicate indexes
SELECT pg_size_pretty(SUM(pg_relation_size(idx))::BIGINT) AS size,
       (array_agg(idx))[1] AS idx1, (array_agg(idx))[2] AS idx2
FROM (
    SELECT indexrelid::regclass AS idx, indrelid::regclass AS tbl,
           (indrelid::text ||E'\n'|| indclass::text ||E'\n'|| indkey::text ||E'\n'||
            COALESCE(indexprs::text,'')||E'\n' || COALESCE(indpred::text,'')) AS key
    FROM pg_index
) sub
GROUP BY key, tbl
HAVING COUNT(*) > 1;
```

### Rebuild Bloated Indexes
```sql
-- Check index bloat
SELECT schemaname, tablename, indexname,
       pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
ORDER BY pg_relation_size(indexrelid) DESC;

-- Rebuild if needed
REINDEX INDEX CONCURRENTLY idx_listings_status_craft_price;
```

### VACUUM & ANALYZE
```sql
-- After bulk inserts or deletes
VACUUM ANALYZE listings;
VACUUM ANALYZE order_allocations;

-- Check last vacuum time
SELECT schemaname, relname, last_vacuum, last_autovacuum, last_analyze
FROM pg_stat_user_tables
ORDER BY last_autovacuum;
```

---

## Future Considerations

### 1. Partial Indexes for Soft Deletes
If tables add `deleted_at` column:
```sql
CREATE INDEX idx_listings_active 
ON listings(id, status, published_at) 
WHERE deleted_at IS NULL;
```

### 2. GIN Indexes for Full-Text Search
If adding description search:
```sql
CREATE INDEX idx_listings_description_fts 
ON listings USING gin(to_tsvector('english', description));
```

### 3. BRIN Indexes for Time-Series
For very large tables (>10M rows) with time-based queries:
```sql
CREATE INDEX idx_outbox_created_at_brin 
ON outbox USING brin(created_at);
```

### 4. Covering Indexes
For index-only scans:
```sql
CREATE INDEX idx_listings_covering 
ON listings(status, craft) 
INCLUDE (title, price_paise, published_at);
```

---

## Summary

**Indexes Added:** 7 new indexes (migration 024)  
**Expected Performance Gain:** 10-100x on filtered queries  
**Disk Space Impact:** ~50-100MB (negligible for current data size)  
**Maintenance:** Auto-updated, no manual work needed

**Apply now:**
```bash
cd /c/projects/kalakritibatch13/kalakriti
make migrate-up
```

**Verify:**
```bash
make psql
\di  # List all indexes
```
