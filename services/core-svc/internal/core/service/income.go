// services/core-svc/internal/core/service/income.go

package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// IncomeStore is the persistence surface of the income service (F13).
type IncomeStore interface {
	UpsertIncomeBaseline(ctx context.Context, artisanID uuid.UUID, b domain.IncomeBaseline) (domain.IncomeBaseline, error)
	GetIncomeBaseline(ctx context.Context, artisanID uuid.UUID) (*domain.IncomeBaseline, error)
	InsertOfflineSale(ctx context.Context, artisanID uuid.UUID, s domain.OfflineSale) (domain.OfflineSale, error)
	ListOfflineSales(ctx context.Context, artisanID uuid.UUID, limit int32) ([]domain.OfflineSale, error)
	DeleteOfflineSale(ctx context.Context, id, artisanID uuid.UUID) (bool, error)
	GetArtisanIncomeFacts(ctx context.Context, artisanID uuid.UUID) (domain.IncomeFacts, error)
	GetArtisanMonthlyIncome(ctx context.Context, artisanID uuid.UUID, since time.Time) ([]domain.MonthIncome, error)
}

// Income is the artisan side of impact measurement. Every method acts on
// the caller only (or the artisan an agent is acting for -- the bff swaps
// that artisan in as the principal, with Actor set).
type Income struct {
	store IncomeStore
	log   *slog.Logger
	now   func() time.Time
}

// NewIncome builds the income service.
func NewIncome(store IncomeStore, log *slog.Logger) *Income {
	if log == nil {
		log = slog.Default()
	}
	return &Income{store: store, log: log, now: time.Now}
}

// artisanSelf returns the calling artisan's id. Staff tokens are refused:
// staff reach an artisan's data only through assisted mode, which arrives
// here as an ARTISAN principal with Actor set.
func artisanSelf(ctx context.Context) (uuid.UUID, auth.Principal, error) {
	p, err := auth.RequireRole(ctx, auth.RoleArtisan)
	if err != nil {
		return uuid.Nil, auth.Principal{}, err
	}
	id, err := uuid.Parse(p.Subject)
	if err != nil {
		return uuid.Nil, auth.Principal{}, pkgdomain.Forbidden("finish registering before using this")
	}
	return id, p, nil
}

func (s *Income) SetIncomeBaseline(ctx context.Context, b domain.IncomeBaseline) (domain.IncomeBaseline, error) {
	artisanID, p, err := artisanSelf(ctx)
	if err != nil {
		return domain.IncomeBaseline{}, err
	}
	if err := b.Validate(); err != nil {
		return domain.IncomeBaseline{}, err
	}
	b.Source = "SELF"
	if p.Actor != "" {
		b.Source = "AGENT"
	}
	return s.store.UpsertIncomeBaseline(ctx, artisanID, b)
}

func (s *Income) GetIncomeBaseline(ctx context.Context) (*domain.IncomeBaseline, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.GetIncomeBaseline(ctx, artisanID)
}

func (s *Income) LogOfflineSale(ctx context.Context, sale domain.OfflineSale) (domain.OfflineSale, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return domain.OfflineSale{}, err
	}
	if sale.ID == uuid.Nil {
		return domain.OfflineSale{}, pkgdomain.InvalidInput("id is required (client-generated, for safe retries)")
	}
	if err := domain.ValidateOfflineSale(sale.Channel, sale.EventName, sale.AmountPaise, sale.SoldOn, s.now()); err != nil {
		return domain.OfflineSale{}, err
	}
	return s.store.InsertOfflineSale(ctx, artisanID, sale)
}

func (s *Income) ListOfflineSales(ctx context.Context, limit int32) ([]domain.OfflineSale, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.store.ListOfflineSales(ctx, artisanID, limit)
}

func (s *Income) DeleteOfflineSale(ctx context.Context, id uuid.UUID) (bool, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return false, err
	}
	return s.store.DeleteOfflineSale(ctx, id, artisanID)
}

// GetMyIncomeSummary computes the caller's income picture live (not from a
// materialized view), so a sale logged a moment ago shows immediately.
func (s *Income) GetMyIncomeSummary(ctx context.Context) (domain.IncomeSummary, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return domain.IncomeSummary{}, err
	}
	now := s.now().UTC()
	baseline, err := s.store.GetIncomeBaseline(ctx, artisanID)
	if err != nil {
		return domain.IncomeSummary{}, err
	}
	facts, err := s.store.GetArtisanIncomeFacts(ctx, artisanID)
	if err != nil {
		return domain.IncomeSummary{}, err
	}
	since := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -2, 0)
	monthly, err := s.store.GetArtisanMonthlyIncome(ctx, artisanID, since)
	if err != nil {
		return domain.IncomeSummary{}, err
	}
	return domain.BuildIncomeSummary(baseline, facts, monthly, now), nil
}
