package wiring

import (
	"context"

	"github.com/google/uuid"
	"time"

	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/domain"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/repo"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/service"
)

// Store wraps *repo.Repo and adapts InTx from concrete *repo.Tx to service.Tx.
type Store struct {
	r *repo.Repo
}

var _ service.Store = Store{}

func NewStore(r *repo.Repo) Store {
	return Store{r: r}
}

func (s Store) InTx(ctx context.Context, fn func(service.Tx) error) error {
	return s.r.InTx(ctx, func(tx *repo.Tx) error {
		return fn(txAdapter{tx: tx})
	})
}

func (s Store) RefreshMaterializedViews(ctx context.Context) error {
	return s.r.RefreshMaterializedViews(ctx)
}

func (s Store) GetArtisansByCategory(ctx context.Context, stateCode, district *string) ([]domain.ArtisanCategoryRow, error) {
	return s.r.GetArtisansByCategory(ctx, stateCode, district)
}

func (s Store) GetListingsByCraftMonth(ctx context.Context, from, to *time.Time, craftID *uuid.UUID) ([]domain.ListingCraftMonthRow, error) {
	return s.r.GetListingsByCraftMonth(ctx, from, to, craftID)
}

func (s Store) GetEarningsByDistrict(ctx context.Context, stateCode, district *string, minBucket int32) ([]domain.EarningsDistrictRow, error) {
	return s.r.GetEarningsByDistrict(ctx, stateCode, district, minBucket)
}

func (s Store) GetDyingCrafts(ctx context.Context, limit int32) ([]domain.DyingCraftRow, error) {
	return s.r.GetDyingCrafts(ctx, limit)
}

func (s Store) GetArtisanEarningsByPeriod(ctx context.Context, artisanID uuid.UUID, start, end time.Time) (*domain.IncomeStatement, error) {
	return s.r.GetArtisanEarningsByPeriod(ctx, artisanID, start, end)
}

func (s Store) GetArtisanEarningsMonthly(ctx context.Context, artisanID uuid.UUID, start, end time.Time) ([]domain.MonthlyEarnings, error) {
	return s.r.GetArtisanEarningsMonthly(ctx, artisanID, start, end)
}

func (s Store) GetIncomeStatementByCode(ctx context.Context, code string) (*domain.IncomeStatement, error) {
	return s.r.GetIncomeStatementByCode(ctx, code)
}

func (s Store) GetArtisanIncomeStatements(ctx context.Context, artisanID uuid.UUID, limit, offset int32) ([]domain.StatementSummary, error) {
	return s.r.GetArtisanIncomeStatements(ctx, artisanID, limit, offset)
}

type txAdapter struct {
	tx *repo.Tx
}

var _ service.Tx = txAdapter{}

func (t txAdapter) CreateIncomeStatement(ctx context.Context, stmt *domain.IncomeStatement) error {
	return t.tx.CreateIncomeStatement(ctx, stmt)
}
