// services/core-svc/internal/core/repo/impact.go

package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

func (r *Repo) UpsertIncomeBaseline(ctx context.Context, artisanID uuid.UUID, b domain.IncomeBaseline) (domain.IncomeBaseline, error) {
	var fairs *int16
	if b.FairsPerYear != nil {
		v := int16(*b.FairsPerYear)
		fairs = &v
	}
	var fairBracket *db.IncomeBracket
	if b.FairIncomeBracket != nil {
		v := db.IncomeBracket(*b.FairIncomeBracket)
		fairBracket = &v
	}
	row, err := r.q.UpsertIncomeBaseline(ctx, db.UpsertIncomeBaselineParams{
		ArtisanID: artisanID, MonthlyBracket: db.IncomeBracket(b.MonthlyBracket), MonthlyPaise: b.MonthlyPaise,
		FairsPerYear: fairs, FairIncomeBracket: fairBracket, Source: b.Source,
	})
	if err != nil {
		return domain.IncomeBaseline{}, translate(err, "income baseline")
	}
	return baselineFromRow(row), nil
}

// GetIncomeBaseline returns nil (no error) when the artisan never gave one.
func (r *Repo) GetIncomeBaseline(ctx context.Context, artisanID uuid.UUID) (*domain.IncomeBaseline, error) {
	row, err := r.q.GetIncomeBaseline(ctx, artisanID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, translate(err, "income baseline")
	}
	b := baselineFromRow(row)
	return &b, nil
}

// InsertOfflineSale is idempotent on the client-generated id: a retried
// outbox write returns the already-stored row instead of a duplicate.
func (r *Repo) InsertOfflineSale(ctx context.Context, artisanID uuid.UUID, s domain.OfflineSale) (domain.OfflineSale, error) {
	row, err := r.q.InsertOfflineSale(ctx, db.InsertOfflineSaleParams{
		ID: s.ID, ArtisanID: artisanID, Channel: db.OfflineSaleChannel(s.Channel), EventName: s.EventName,
		AmountPaise: s.AmountPaise, SoldOn: toDate(s.SoldOn),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		existing, gerr := r.q.GetOfflineSale(ctx, s.ID)
		if gerr != nil {
			return domain.OfflineSale{}, translate(gerr, "offline sale")
		}
		if existing.ArtisanID != artisanID {
			return domain.OfflineSale{}, translate(pgx.ErrNoRows, "offline sale")
		}
		return offlineSaleFromRow(existing), nil
	}
	if err != nil {
		return domain.OfflineSale{}, translate(err, "offline sale")
	}
	return offlineSaleFromRow(row), nil
}

func (r *Repo) ListOfflineSales(ctx context.Context, artisanID uuid.UUID, limit int32) ([]domain.OfflineSale, error) {
	rows, err := r.q.ListOfflineSales(ctx, db.ListOfflineSalesParams{ArtisanID: artisanID, PageSize: limit})
	if err != nil {
		return nil, translate(err, "offline sales")
	}
	out := make([]domain.OfflineSale, len(rows))
	for i, row := range rows {
		out[i] = offlineSaleFromRow(row)
	}
	return out, nil
}

func (r *Repo) DeleteOfflineSale(ctx context.Context, id, artisanID uuid.UUID) (bool, error) {
	n, err := r.q.DeleteOfflineSale(ctx, db.DeleteOfflineSaleParams{ID: id, ArtisanID: artisanID})
	if err != nil {
		return false, translate(err, "offline sale")
	}
	return n > 0, nil
}

func (r *Repo) GetArtisanIncomeFacts(ctx context.Context, artisanID uuid.UUID) (domain.IncomeFacts, error) {
	row, err := r.q.GetArtisanIncomeFacts(ctx, artisanID)
	if err != nil {
		return domain.IncomeFacts{}, translate(err, "artisan")
	}
	return domain.IncomeFacts{
		RegisteredAt: row.RegisteredAt, PlatformPaise90d: row.PlatformPaise90d,
		PlatformPendingPaise: row.PlatformPendingPaise, OfflinePaise90d: row.OfflinePaise90d,
		FairPaise90d: row.FairPaise90d,
	}, nil
}

func (r *Repo) GetArtisanMonthlyIncome(ctx context.Context, artisanID uuid.UUID, since time.Time) ([]domain.MonthIncome, error) {
	rows, err := r.q.GetArtisanMonthlyIncome(ctx, db.GetArtisanMonthlyIncomeParams{ArtisanID: artisanID, Since: toDate(since)})
	if err != nil {
		return nil, translate(err, "monthly income")
	}
	out := make([]domain.MonthIncome, len(rows))
	for i, row := range rows {
		out[i] = domain.MonthIncome{Month: row.Month.Time, PlatformPaise: row.PlatformPaise, OfflinePaise: row.OfflinePaise}
	}
	return out, nil
}

func baselineFromRow(row db.ArtisanIncomeBaseline) domain.IncomeBaseline {
	b := domain.IncomeBaseline{
		MonthlyBracket: string(row.MonthlyBracket), MonthlyPaise: row.MonthlyPaise,
		CapturedAt: row.CapturedAt, Source: row.Source,
	}
	if row.FairsPerYear != nil {
		v := int32(*row.FairsPerYear)
		b.FairsPerYear = &v
	}
	if row.FairIncomeBracket != nil {
		v := string(*row.FairIncomeBracket)
		b.FairIncomeBracket = &v
	}
	return b
}

func offlineSaleFromRow(row db.OfflineSale) domain.OfflineSale {
	return domain.OfflineSale{
		ID: row.ID, Channel: string(row.Channel), EventName: row.EventName, AmountPaise: row.AmountPaise,
		SoldOn: row.SoldOn.Time, CreatedAt: row.CreatedAt,
	}
}

func toDate(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

func toOptDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return toDate(*t)
}

func fromOptDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	return &d.Time
}
