// services/insight-svc/internal/insight/repo/repo.go
package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pkgpostgres "github.com/ZoroNewbie00/kalakriti/pkg/postgres"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/domain"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/repo/db"
)

// Repo implements the insight service's Store port.
type Repo struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// New builds a Repo over an existing pool.
func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool, q: db.New(pool)} }

// Tx is what a transaction can do: create an income statement.
type Tx struct {
	q *db.Queries
}

// CreateIncomeStatement inserts a new income statement, filling in ID and CreatedAt.
func (t *Tx) CreateIncomeStatement(ctx context.Context, stmt *domain.IncomeStatement) error {
	row, err := t.q.CreateIncomeStatement(ctx, db.CreateIncomeStatementParams{
		ID: uuid.New(), ArtisanID: stmt.ArtisanID, PeriodStart: stmt.PeriodStart, PeriodEnd: stmt.PeriodEnd,
		OrderCount: int32(stmt.OrderCount), GrossPaise: stmt.GrossPaise, NetPaise: stmt.NetPaise, FeePaise: stmt.FeePaise,
		Signature: stmt.Signature, SignatureAlgo: stmt.SignatureAlgo, PublicKeyID: stmt.PublicKeyID,
		ShortCode: stmt.ShortCode, S3Key: stmt.S3Key,
	})
	if err != nil {
		return fmt.Errorf("creating income statement: %w", err)
	}
	stmt.ID = row.ID
	stmt.CreatedAt = row.CreatedAt
	return nil
}

// InTx runs fn inside a transaction.
func (r *Repo) InTx(ctx context.Context, fn func(tx *Tx) error) error {
	return pkgpostgres.RunInTx(ctx, r.pool, func(ctx context.Context, pgtx pgx.Tx) error {
		return fn(&Tx{q: db.New(pgtx)})
	})
}

func (r *Repo) RefreshMaterializedViews(ctx context.Context) error {
	return r.q.RefreshMaterializedViews(ctx)
}

// ShortCodeExists checks if an income statement short code is already taken.
func (r *Repo) ShortCodeExists(ctx context.Context, code string) (bool, error) {
	return r.q.IncomeStatementShortCodeExists(ctx, code)
}

func (r *Repo) GetArtisansByCategory(ctx context.Context, stateCode, district *string) ([]domain.ArtisanCategoryRow, error) {
	rows, err := r.q.GetArtisansByCategory(ctx, db.GetArtisansByCategoryParams{
		Column1: derefString(stateCode), Column2: derefString(district),
	})
	if err != nil {
		return nil, fmt.Errorf("artisans by category: %w", err)
	}
	out := make([]domain.ArtisanCategoryRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ArtisanCategoryRow{
			StateCode: row.StateCode, District: derefString(row.District), SocialCategory: derefString(row.SocialCategory),
			ArtisanCount: int32(row.ArtisanCount), VerifiedCount: int32(row.VerifiedCount),
		})
	}
	return out, nil
}

func (r *Repo) GetListingsByCraftMonth(ctx context.Context, from, to *time.Time, craftID *uuid.UUID) ([]domain.ListingCraftMonthRow, error) {
	rows, err := r.q.GetListingsByCraftMonth(ctx, db.GetListingsByCraftMonthParams{
		Column1: derefTime(from), Column2: derefTime(to), Column3: derefUUID(craftID),
	})
	if err != nil {
		return nil, fmt.Errorf("listings by craft month: %w", err)
	}
	out := make([]domain.ListingCraftMonthRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ListingCraftMonthRow{
			CraftID: row.CraftID.String(), CraftName: row.CraftName, Month: row.Month,
			ListingCount: int32(row.ListingCount), ArtisanCount: int32(row.ArtisanCount),
		})
	}
	return out, nil
}

func (r *Repo) GetEarningsByDistrict(ctx context.Context, stateCode, district *string, minBucket int32) ([]domain.EarningsDistrictRow, error) {
	rows, err := r.q.GetEarningsByDistrict(ctx, db.GetEarningsByDistrictParams{
		Column1: derefString(stateCode), Column2: derefString(district), ArtisanCount: int64(minBucket),
	})
	if err != nil {
		return nil, fmt.Errorf("earnings by district: %w", err)
	}
	out := make([]domain.EarningsDistrictRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.EarningsDistrictRow{
			StateCode: row.StateCode, District: derefString(row.District),
			TotalGMVPaise: row.TotalGmvPaise, TotalNetPaise: row.TotalNetPaise,
			ArtisanCount: int32(row.ArtisanCount), AvgEarnings: int64(row.AvgEarningsPaise),
		})
	}
	return out, nil
}

func (r *Repo) GetIncomeComparison(ctx context.Context, stateCode, district *string, minBucket int32) ([]domain.IncomeComparisonRow, error) {
	rows, err := r.q.GetIncomeComparison(ctx, db.GetIncomeComparisonParams{
		Column1: derefString(stateCode), Column2: derefString(district), ArtisanCount: int64(minBucket),
	})
	if err != nil {
		return nil, fmt.Errorf("income comparison: %w", err)
	}
	out := make([]domain.IncomeComparisonRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.IncomeComparisonRow{
			StateCode: row.StateCode, District: derefString(row.District),
			MedianBeforePaise: int64(row.MedianBeforePaise), MedianAfterPaise: int64(row.MedianAfterPaise),
			ArtisanCount: int32(row.ArtisanCount),
		})
	}
	return out, nil
}

func (r *Repo) GetDyingCrafts(ctx context.Context, limit int32) ([]domain.DyingCraftRow, error) {
	rows, err := r.q.GetDyingCrafts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("dying crafts: %w", err)
	}
	out := make([]domain.DyingCraftRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.DyingCraftRow{
			CraftID: row.CraftID.String(), CraftName: row.CraftName,
			DeclineRate: row.DeclineRate, PeakArtisans: row.PeakArtisans, CurrentArtisans: row.CurrentArtisans,
		})
	}
	return out, nil
}

func (r *Repo) GetArtisanEarningsByPeriod(ctx context.Context, artisanID uuid.UUID, start, end time.Time) (*domain.IncomeStatement, error) {
	row, err := r.q.GetArtisanEarningsByPeriod(ctx, db.GetArtisanEarningsByPeriodParams{
		ID: artisanID, SettledAt: &start, SettledAt_2: &end,
	})
	if err != nil {
		return nil, fmt.Errorf("artisan earnings by period: %w", err)
	}
	return &domain.IncomeStatement{
		ArtisanID: row.ID, ArtisanName: row.DisplayName, PeriodStart: start, PeriodEnd: end,
		OrderCount: row.OrderCount, GrossPaise: row.GrossPaise, NetPaise: row.NetPaise, FeePaise: row.FeePaise,
	}, nil
}

func (r *Repo) GetArtisanEarningsMonthly(ctx context.Context, artisanID uuid.UUID, start, end time.Time) ([]domain.MonthlyEarnings, error) {
	rows, err := r.q.GetArtisanEarningsMonthly(ctx, db.GetArtisanEarningsMonthlyParams{
		ArtisanID: artisanID, SettledAt: &start, SettledAt_2: &end,
	})
	if err != nil {
		return nil, fmt.Errorf("artisan earnings monthly: %w", err)
	}
	out := make([]domain.MonthlyEarnings, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.MonthlyEarnings{
			Month: row.Month, OrderCount: row.OrderCount,
			GrossPaise: row.GrossPaise, NetPaise: row.NetPaise, FeePaise: row.FeePaise,
		})
	}
	return out, nil
}

func (r *Repo) GetIncomeStatementByCode(ctx context.Context, code string) (*domain.IncomeStatement, error) {
	row, err := r.q.GetIncomeStatementByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("income statement by code: %w", err)
	}
	return &domain.IncomeStatement{
		ID: row.ID, ArtisanID: row.ArtisanID, ArtisanName: row.ArtisanName,
		PeriodStart: row.PeriodStart, PeriodEnd: row.PeriodEnd, OrderCount: int64(row.OrderCount),
		GrossPaise: row.GrossPaise, NetPaise: row.NetPaise, FeePaise: row.FeePaise,
		Signature: row.Signature, SignatureAlgo: row.SignatureAlgo, PublicKeyID: row.PublicKeyID,
		ShortCode: row.ShortCode, S3Key: row.S3Key, CreatedAt: row.CreatedAt,
	}, nil
}

func (r *Repo) GetArtisanIncomeStatements(ctx context.Context, artisanID uuid.UUID, limit, offset int32) ([]domain.StatementSummary, error) {
	rows, err := r.q.GetArtisanIncomeStatements(ctx, db.GetArtisanIncomeStatementsParams{
		ArtisanID: artisanID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("artisan income statements: %w", err)
	}
	out := make([]domain.StatementSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.StatementSummary{
			ID: row.ID, PeriodStart: row.PeriodStart, PeriodEnd: row.PeriodEnd, OrderCount: int64(row.OrderCount),
			GrossPaise: row.GrossPaise, NetPaise: row.NetPaise, FeePaise: row.FeePaise,
			ShortCode: row.ShortCode, S3Key: row.S3Key, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func derefUUID(u *uuid.UUID) uuid.UUID {
	if u == nil {
		return uuid.Nil
	}
	return *u
}
