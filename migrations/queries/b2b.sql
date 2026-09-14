-- migrations/queries/b2b.sql

-- name: CreateCompany :one
INSERT INTO company (
    id, user_id, name, company_type, gstin, contact_name, contact_phone,
    contact_email, website, state_code, district,
    verification_status, verified, income_statement_url, commission_rate_bps,
    accepts_consignment, min_order_value_paise, preferred_craft_ids,
    store_latitude, store_longitude, store_address, store_city, store_pincode
) VALUES (
    @id, @user_id, @name, @company_type, sqlc.narg('gstin'), @contact_name,
    @contact_phone, sqlc.narg('contact_email'), sqlc.narg('website'),
    @state_code, sqlc.narg('district'),
    @verification_status, @verified, @income_statement_url, @commission_rate_bps,
    @accepts_consignment, sqlc.narg('min_order_value_paise'), @preferred_craft_ids,
    sqlc.narg('store_latitude'), sqlc.narg('store_longitude'),
    sqlc.narg('store_address'), sqlc.narg('store_city'), sqlc.narg('store_pincode')
)
RETURNING *;

-- name: GetCompany :one
SELECT * FROM company WHERE id = @id;

-- name: GetCompanyByUserID :one
SELECT * FROM company WHERE user_id = @user_id;

-- name: ListCompanies :many
SELECT * FROM company
WHERE (sqlc.narg('company_type')::company_type IS NULL OR company_type = sqlc.narg('company_type'))
  AND (sqlc.narg('state_code')::text IS NULL OR state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('verification_status')::verification_status IS NULL OR verification_status = sqlc.narg('verification_status'))
  AND (NOT @verified_only::boolean OR verified)
  AND (sqlc.narg('after')::uuid IS NULL OR id > sqlc.narg('after'))
ORDER BY id
LIMIT @page_size;

-- name: ListPendingCompanies :many
SELECT * FROM company
WHERE verification_status = 'PENDING'
ORDER BY created_at ASC
LIMIT @page_size;

-- name: VerifyCompany :one
UPDATE company
SET verification_status = @verification_status,
    verified = @verified,
    verified_by = sqlc.narg('verified_by'),
    verified_at = now(),
    rejection_reason = sqlc.narg('rejection_reason'),
    commission_rate_bps = @commission_rate_bps,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: RecordCompanySaleSettlement :one
INSERT INTO company_sale_settlement (
    id, company_id, order_id, product_name, buyer_id,
    gross_amount_paise, commission_rate_bps, platform_fee_paise, net_payout_paise, settled_at
) VALUES (
    @id, @company_id, @order_id, @product_name, @buyer_id,
    @gross_amount_paise, @commission_rate_bps, @platform_fee_paise, @net_payout_paise, now()
)
RETURNING *;

-- name: IncrementCompanySales :one
UPDATE company
SET total_sales_paise = total_sales_paise + @gross_amount_paise,
    commission_earned_paise = commission_earned_paise + @platform_fee_paise,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: GetPlatformCommissionStats :one
SELECT
    COUNT(*) AS total_companies,
    COUNT(*) FILTER (WHERE verification_status = 'PENDING') AS pending_verifications,
    COUNT(*) FILTER (WHERE verification_status = 'VERIFIED') AS verified_companies,
    COALESCE(SUM(total_sales_paise), 0)::bigint AS total_sales_paise,
    COALESCE(SUM(commission_earned_paise), 0)::bigint AS total_commission_paise
FROM company;

-- name: ListCompanySales :many
SELECT * FROM company_sale_settlement
WHERE company_id = @company_id
ORDER BY settled_at DESC
LIMIT @page_size;

-- name: CreateCompanyInterest :one
INSERT INTO company_interest (id, company_id, artisan_id, message)
VALUES (@id, @company_id, @artisan_id, @message)
RETURNING *;

-- name: GetCompanyInterest :one
SELECT * FROM company_interest WHERE id = @id;

-- name: RespondToInterest :one
UPDATE company_interest
SET status = @status, responded_at = now()
WHERE id = @id AND artisan_id = @artisan_id AND status = 'PENDING'
RETURNING *;

-- name: ListArtisanLeads :many
SELECT ci.*, c.name AS company_name, c.company_type
FROM company_interest ci
JOIN company c ON c.id = ci.company_id
WHERE ci.artisan_id = @artisan_id
  AND (sqlc.narg('status')::interest_status IS NULL OR ci.status = sqlc.narg('status'))
  AND (sqlc.narg('after')::uuid IS NULL OR ci.id > sqlc.narg('after'))
ORDER BY ci.created_at DESC
LIMIT @page_size;

-- name: CreateSupplyPartnership :one
INSERT INTO supply_partnership (id, company_id, artisan_id, craft_id, terms, renewal_date)
VALUES (@id, @company_id, @artisan_id, @craft_id, @terms, sqlc.narg('renewal_date'))
RETURNING *;

-- name: ListPartnershipsByArtisan :many
SELECT sp.*, c.name AS company_name, a.display_name AS artisan_name
FROM supply_partnership sp
JOIN company c ON c.id = sp.company_id
JOIN artisan a ON a.id = sp.artisan_id
WHERE sp.artisan_id = @artisan_id
  AND (NOT @active_only::boolean OR sp.active)
  AND (sqlc.narg('after')::uuid IS NULL OR sp.id > sqlc.narg('after'))
ORDER BY sp.created_at DESC
LIMIT @page_size;

-- name: ListPartnershipsByCompany :many
SELECT sp.*, c.name AS company_name, a.display_name AS artisan_name
FROM supply_partnership sp
JOIN company c ON c.id = sp.company_id
JOIN artisan a ON a.id = sp.artisan_id
WHERE sp.company_id = @company_id
  AND (NOT @active_only::boolean OR sp.active)
  AND (sqlc.narg('after')::uuid IS NULL OR sp.id > sqlc.narg('after'))
ORDER BY sp.created_at DESC
LIMIT @page_size;

-- name: DeactivatePartnership :execrows
UPDATE supply_partnership SET active = false, updated_at = now()
WHERE id = @id;

-- name: CreateBoutiqueMatch :one
INSERT INTO boutique_artisan_match (id, company_id, artisan_id, match_score)
VALUES (@id, @company_id, @artisan_id, @match_score)
ON CONFLICT ON CONSTRAINT boutique_artisan_match_unique
DO UPDATE SET match_score = EXCLUDED.match_score
RETURNING *;

-- name: ListBoutiqueMatchesForArtisan :many
SELECT bm.*, c.name AS boutique_name,
       c.store_latitude, c.store_longitude, c.store_address, c.store_city
FROM boutique_artisan_match bm
JOIN company c ON c.id = bm.company_id
WHERE bm.artisan_id = @artisan_id
  AND bm.status != 'DECLINED'
ORDER BY bm.match_score DESC
LIMIT @limit_val;

-- name: UpdateBoutiqueMatchStatus :execrows
UPDATE boutique_artisan_match SET status = @status
WHERE id = @id;

-- Proximity search for boutiques
-- name: ListNearbyBoutiques :many
SELECT * FROM company
WHERE company_type = 'BOUTIQUE'
  AND store_latitude IS NOT NULL
  AND store_latitude BETWEEN @lat_min AND @lat_max
  AND store_longitude BETWEEN @lng_min AND @lng_max
  AND (sqlc.narg('craft_id')::uuid IS NULL OR @craft_id = ANY(preferred_craft_ids))
ORDER BY id
LIMIT @limit_val;
