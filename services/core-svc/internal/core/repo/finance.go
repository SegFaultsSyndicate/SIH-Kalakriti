// services/core-svc/internal/core/repo/finance.go

package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// InsertFinanceLink stores a link. It takes the already-hashed reference:
// this layer never sees the plaintext.
func (r *Repo) InsertFinanceLink(ctx context.Context, l domain.FinanceLink, refHash []byte, t domain.FinanceTerms) (domain.FinanceLink, error) {
	row, err := r.q.InsertFinanceLink(ctx, db.InsertFinanceLinkParams{
		ID: l.ID, ArtisanID: l.ArtisanID, Corporation: db.FinanceCorporation(l.Corporation),
		ChannelizingAgency: t.ChannelizingAgency, ReferenceLast4: l.ReferenceLast4, ReferenceHash: refHash,
		SanctionedPaise: t.SanctionedPaise, EmiPaise: t.EmiPaise, EmiDayOfMonth: toInt16(t.EmiDayOfMonth),
		RepaymentStart: toOptDate(t.RepaymentStart), ConsentVersion: l.ConsentVersion,
	})
	if err != nil {
		return domain.FinanceLink{}, translate(err, "finance link")
	}
	return financeLinkFromRow(row), nil
}

func (r *Repo) GetFinanceLink(ctx context.Context, id uuid.UUID) (domain.FinanceLink, error) {
	row, err := r.q.GetFinanceLink(ctx, id)
	if err != nil {
		return domain.FinanceLink{}, translate(err, "finance link")
	}
	return financeLinkFromRow(row), nil
}

func (r *Repo) ListFinanceLinks(ctx context.Context, artisanID uuid.UUID) ([]domain.FinanceLink, error) {
	rows, err := r.q.ListFinanceLinks(ctx, artisanID)
	if err != nil {
		return nil, translate(err, "finance links")
	}
	out := make([]domain.FinanceLink, len(rows))
	for i, row := range rows {
		out[i] = financeLinkFromRow(row)
	}
	return out, nil
}

func (r *Repo) UpdateFinanceLink(ctx context.Context, id, artisanID uuid.UUID, t domain.FinanceTerms) (domain.FinanceLink, error) {
	row, err := r.q.UpdateFinanceLink(ctx, db.UpdateFinanceLinkParams{
		ID: id, ArtisanID: artisanID, ChannelizingAgency: t.ChannelizingAgency, SanctionedPaise: t.SanctionedPaise,
		EmiPaise: t.EmiPaise, EmiDayOfMonth: toInt16(t.EmiDayOfMonth), RepaymentStart: toOptDate(t.RepaymentStart),
	})
	if err != nil {
		return domain.FinanceLink{}, translate(err, "finance link")
	}
	return financeLinkFromRow(row), nil
}

func (r *Repo) DeleteFinanceLink(ctx context.Context, id, artisanID uuid.UUID) (bool, error) {
	n, err := r.q.DeleteFinanceLink(ctx, db.DeleteFinanceLinkParams{ID: id, ArtisanID: artisanID})
	if err != nil {
		return false, translate(err, "finance link")
	}
	return n > 0, nil
}

func (r *Repo) SetFinanceLinkStatus(ctx context.Context, id uuid.UUID, status, reviewer string, reason *string) (domain.FinanceLink, error) {
	row, err := r.q.SetFinanceLinkStatus(ctx, db.SetFinanceLinkStatusParams{
		ID: id, Status: db.FinanceLinkStatus(status), Reviewer: reviewer, RejectReason: reason,
	})
	if err != nil {
		return domain.FinanceLink{}, translate(err, "finance link")
	}
	return financeLinkFromRow(row), nil
}

func (r *Repo) ListFinanceLinksForReview(ctx context.Context, status, stateCode, district *string) ([]domain.FinanceLinkForReview, error) {
	var st *db.FinanceLinkStatus
	if status != nil {
		v := db.FinanceLinkStatus(*status)
		st = &v
	}
	rows, err := r.q.ListFinanceLinksForReview(ctx, db.ListFinanceLinksForReviewParams{Status: st, StateCode: stateCode, District: district})
	if err != nil {
		return nil, translate(err, "finance links")
	}
	out := make([]domain.FinanceLinkForReview, len(rows))
	for i, row := range rows {
		out[i] = domain.FinanceLinkForReview{
			FinanceLink: domain.FinanceLink{
				ID: row.ID, ArtisanID: row.ArtisanID, Corporation: string(row.Corporation),
				ChannelizingAgency: row.ChannelizingAgency, ReferenceLast4: row.ReferenceLast4,
				SanctionedPaise: row.SanctionedPaise, EmiPaise: row.EmiPaise, EmiDayOfMonth: fromInt16(row.EmiDayOfMonth),
				Status: string(row.Status), CreatedAt: row.CreatedAt,
			},
			ArtisanName: row.ArtisanName, StateCode: row.StateCode, District: row.District,
		}
	}
	return out, nil
}

// GetEarnedInRange returns settled platform, offline and pending paise for
// an artisan over [from, to).
func (r *Repo) GetEarnedInRange(ctx context.Context, artisanID uuid.UUID, from, to time.Time) (platform, offline, pending int64, err error) {
	row, err := r.q.GetEarnedInRange(ctx, db.GetEarnedInRangeParams{ArtisanID: artisanID, FromDate: toDate(from), ToDate: toDate(to)})
	if err != nil {
		return 0, 0, 0, translate(err, "earnings")
	}
	return row.PlatformPaise, row.OfflinePaise, row.PendingPaise, nil
}

func (r *Repo) ListDueReminders(ctx context.Context, emiDay int32, dueMonth time.Time) ([]domain.EMIReminder, error) {
	d := int16(emiDay)
	rows, err := r.q.ListLinksDueForReminder(ctx, db.ListLinksDueForReminderParams{EmiDay: &d, DueMonth: toDate(dueMonth)})
	if err != nil {
		return nil, translate(err, "emi reminders")
	}
	out := make([]domain.EMIReminder, 0, len(rows))
	for _, row := range rows {
		due := domain.EMIReminder{LinkID: row.ID, ArtisanID: row.ArtisanID, Corporation: string(row.Corporation), EmiDay: emiDay, Language: row.Language}
		if row.EmiPaise != nil {
			due.EmiPaise = *row.EmiPaise
		}
		out = append(out, due)
	}
	return out, nil
}

// SendEMIReminder claims (link, month) and, only if this replica won the
// claim, writes the notification -- in one transaction, so a claim is never
// recorded without its notification or vice versa.
func (r *Repo) SendEMIReminder(ctx context.Context, due domain.EMIReminder, dueMonth time.Time, title, body string, payload map[string]any) (bool, error) {
	sent := false
	err := r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		n, err := tx.q.ClaimEmiReminder(ctx, db.ClaimEmiReminderParams{LinkID: due.LinkID, DueMonth: toDate(dueMonth)})
		if err != nil || n == 0 {
			return err
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if err := tx.q.InsertNotification(ctx, db.InsertNotificationParams{
			ID: ids.New(), RecipientID: due.ArtisanID.String(), Kind: db.NotificationKind("EMI_REMINDER"),
			Language: db.LanguageCode(due.Language), Title: title, Body: body, Payload: raw,
		}); err != nil {
			return err
		}
		sent = true
		return nil
	})
	return sent, translate(err, "emi reminder")
}

func financeLinkFromRow(row db.ArtisanFinanceLink) domain.FinanceLink {
	return domain.FinanceLink{
		ID: row.ID, ArtisanID: row.ArtisanID, Corporation: string(row.Corporation),
		ChannelizingAgency: row.ChannelizingAgency, ReferenceLast4: row.ReferenceLast4,
		SanctionedPaise: row.SanctionedPaise, EmiPaise: row.EmiPaise, EmiDayOfMonth: fromInt16(row.EmiDayOfMonth),
		RepaymentStart: fromOptDate(row.RepaymentStart), Status: string(row.Status), VerifiedAt: row.VerifiedAt,
		RejectReason: row.RejectReason, ConsentAt: row.ConsentAt, ConsentVersion: row.ConsentVersion,
		CreatedAt: row.CreatedAt,
	}
}

func toInt16(v *int32) *int16 {
	if v == nil {
		return nil
	}
	x := int16(*v)
	return &x
}

func fromInt16(v *int16) *int32 {
	if v == nil {
		return nil
	}
	x := int32(*v)
	return &x
}
