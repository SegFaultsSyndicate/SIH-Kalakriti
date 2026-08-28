package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segfaultsyndicate/kalakriti/services/insight-svc/internal/insight/domain"
)

// Test aggregate correctness with minimum bucket size suppression.
func TestGetEarningsByDistrict_MinBucketSuppression(t *testing.T) {
	store := &mockStore{
		earningsRows: []domain.EarningsDistrictRow{
			{StateCode: "IN-UP", District: "Varanasi", ArtisanCount: 10, TotalGMVPaise: 500000},
			{StateCode: "IN-UP", District: "SmallDistrict", ArtisanCount: 3, TotalGMVPaise: 100000},
		},
	}
	svc := &Service{store: store, minBucketSize: 5}

	rows, err := svc.GetEarningsByDistrict(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Only the row with artisan_count >= 5 should be returned.
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ArtisanCount < 5 {
		t.Errorf("expected artisan_count >= 5, got %d", rows[0].ArtisanCount)
	}
}

// Test that PDF generation produces non-empty output with Devanagari text.
// Full rendering validation requires opening the PDF with a reader.
func TestGeneratePDF_NonEmpty(t *testing.T) {
	t.Skip("PDF generation requires gofpdf with embedded TTF font")
	// Acceptance: manual test confirms PDF opens and Hindi text renders correctly.
}

// Test cross-artisan access denial (handler layer test, placeholder here).
func TestCrossArtisanAccessDenied(t *testing.T) {
	t.Skip("Access control tested at handler layer with auth context")
	// Acceptance: handler returns PermissionDenied when artisan requests another artisan's statement.
}

// mockStore implements service.Store for testing.
type mockStore struct {
	earningsRows []domain.EarningsDistrictRow
}

func (m *mockStore) InTx(ctx context.Context, fn func(Tx) error) error {
	return nil
}

func (m *mockStore) RefreshMaterializedViews(ctx context.Context) error {
	return nil
}

func (m *mockStore) GetArtisansByCategory(ctx context.Context, stateCode, district *string) ([]domain.ArtisanCategoryRow, error) {
	return nil, nil
}

func (m *mockStore) GetListingsByCraftMonth(ctx context.Context, from, to *time.Time, craftID *uuid.UUID) ([]domain.ListingCraftMonthRow, error) {
	return nil, nil
}

func (m *mockStore) GetEarningsByDistrict(ctx context.Context, stateCode, district *string, minBucket int32) ([]domain.EarningsDistrictRow, error) {
	result := []domain.EarningsDistrictRow{}
	for _, row := range m.earningsRows {
		if row.ArtisanCount >= minBucket {
			result = append(result, row)
		}
	}
	return result, nil
}

func (m *mockStore) GetIncomeComparison(ctx context.Context, stateCode, district *string, minBucket int32) ([]domain.IncomeComparisonRow, error) {
	return nil, nil
}

func (m *mockStore) GetDyingCrafts(ctx context.Context, limit int32) ([]domain.DyingCraftRow, error) {
	return nil, nil
}

func (m *mockStore) GetArtisanEarningsByPeriod(ctx context.Context, artisanID uuid.UUID, start, end time.Time) (*domain.IncomeStatement, error) {
	return nil, nil
}

func (m *mockStore) GetArtisanEarningsMonthly(ctx context.Context, artisanID uuid.UUID, start, end time.Time) ([]domain.MonthlyEarnings, error) {
	return nil, nil
}

func (m *mockStore) GetIncomeStatementByCode(ctx context.Context, code string) (*domain.IncomeStatement, error) {
	return nil, nil
}

func (m *mockStore) GetArtisanIncomeStatements(ctx context.Context, artisanID uuid.UUID, limit, offset int32) ([]domain.StatementSummary, error) {
	return nil, nil
}
