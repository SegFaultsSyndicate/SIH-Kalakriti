-- migrations/queries/finance.sql
-- F12: finance-corporation links, repayment coverage and EMI reminders.
-- Only reference_last4 / reference_hash are ever stored -- never the full ref.

-- name: InsertFinanceLink :one
INSERT INTO artisan_finance_link (
    id, artisan_id, corporation, channelizing_agency, reference_last4, reference_hash,
    sanctioned_paise, emi_paise, emi_day_of_month, repayment_start, consent_at, consent_version
) VALUES (
    @id, @artisan_id, @corporation, sqlc.narg('channelizing_agency'), @reference_last4, @reference_hash,
    sqlc.narg('sanctioned_paise'), sqlc.narg('emi_paise'), sqlc.narg('emi_day_of_month'),
    sqlc.narg('repayment_start'), now(), @consent_version
)
RETURNING *;

-- name: GetFinanceLink :one
SELECT * FROM artisan_finance_link WHERE id = @id;

-- name: ListFinanceLinks :many
SELECT * FROM artisan_finance_link WHERE artisan_id = @artisan_id ORDER BY created_at;

-- name: UpdateFinanceLink :one
-- An artisan's edit of their own loan terms. Editing re-opens verification:
-- an officer verified the old terms, not these.
UPDATE artisan_finance_link SET
    channelizing_agency = sqlc.narg('channelizing_agency'),
    sanctioned_paise    = sqlc.narg('sanctioned_paise'),
    emi_paise           = sqlc.narg('emi_paise'),
    emi_day_of_month    = sqlc.narg('emi_day_of_month'),
    repayment_start     = sqlc.narg('repayment_start'),
    status              = 'SELF_REPORTED',
    verified_by         = NULL,
    verified_at         = NULL,
    reject_reason       = NULL,
    updated_at          = now()
WHERE id = @id AND artisan_id = @artisan_id
RETURNING *;

-- name: DeleteFinanceLink :execrows
-- Hard delete: consent withdrawal (DPDP Act 2023) removes the record.
DELETE FROM artisan_finance_link WHERE id = @id AND artisan_id = @artisan_id;

-- name: SetFinanceLinkStatus :one
UPDATE artisan_finance_link SET
    status        = @status,
    verified_by   = CASE WHEN @status = 'VERIFIED'::finance_link_status THEN @reviewer::text ELSE verified_by END,
    verified_at   = CASE WHEN @status = 'VERIFIED'::finance_link_status THEN now() ELSE NULL END,
    reject_reason = sqlc.narg('reject_reason'),
    updated_at    = now()
WHERE id = @id
RETURNING *;

-- name: ListFinanceLinksForReview :many
SELECT f.id, f.artisan_id, a.display_name AS artisan_name, a.state_code, a.district,
       f.corporation, f.channelizing_agency, f.reference_last4, f.sanctioned_paise, f.emi_paise,
       f.emi_day_of_month, f.status, f.created_at
FROM artisan_finance_link f
JOIN artisan a ON a.id = f.artisan_id
WHERE (sqlc.narg('status')::finance_link_status IS NULL OR f.status = sqlc.narg('status'))
  AND (sqlc.narg('state_code')::text IS NULL OR a.state_code = sqlc.narg('state_code'))
  AND (sqlc.narg('district')::text IS NULL OR a.district = sqlc.narg('district'))
ORDER BY f.created_at DESC
LIMIT 200;

-- name: GetEarnedInRange :one
-- Settled platform net + logged offline sales in [@from, @to), plus platform
-- lines not yet settled ("on the way") -- shown, never counted.
SELECT
    COALESCE((SELECT SUM(psl.net_amount_paise) FROM payment_split_line psl
              JOIN order_lot ol ON ol.id = psl.lot_id
              WHERE ol.artisan_id = @artisan_id AND psl.payee_type = 'ARTISAN'
                AND psl.settled_at >= @from_date::date AND psl.settled_at < @to_date::date), 0)::bigint AS platform_paise,
    COALESCE((SELECT SUM(os.amount_paise) FROM offline_sale os
              WHERE os.artisan_id = @artisan_id
                AND os.sold_on >= @from_date::date AND os.sold_on < @to_date::date), 0)::bigint AS offline_paise,
    COALESCE((SELECT SUM(psl.net_amount_paise) FROM payment_split_line psl
              JOIN order_lot ol ON ol.id = psl.lot_id
              WHERE ol.artisan_id = @artisan_id AND psl.payee_type = 'ARTISAN'
                AND psl.settled_at IS NULL), 0)::bigint AS pending_paise;

-- name: ListLinksDueForReminder :many
-- Links whose EMI falls on @emi_day this month and that have not yet been
-- reminded for @due_month. Rejected links are never reminded.
SELECT f.id, f.artisan_id, f.corporation, f.emi_paise, f.emi_day_of_month,
       COALESCE(a.languages[1]::text, 'ENGLISH')::text AS language
FROM artisan_finance_link f
JOIN artisan a ON a.id = f.artisan_id
WHERE f.emi_day_of_month = @emi_day
  AND f.emi_paise IS NOT NULL
  AND f.status <> 'REJECTED'
  AND NOT EXISTS (SELECT 1 FROM emi_reminder_sent r WHERE r.link_id = f.id AND r.due_month = @due_month);

-- name: ClaimEmiReminder :execrows
-- The (link, month) primary key makes this the de-duplication point: a
-- replica that loses the race inserts nothing and sends nothing.
INSERT INTO emi_reminder_sent (link_id, due_month) VALUES (@link_id, @due_month)
ON CONFLICT DO NOTHING;

-- name: InsertNotification :exec
INSERT INTO notification (id, recipient_id, kind, language, title, body, payload)
VALUES (@id, @recipient_id, @kind, @language, @title, @body, @payload);
