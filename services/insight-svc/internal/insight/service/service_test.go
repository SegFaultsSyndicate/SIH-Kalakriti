package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/domain"
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

// Test that PDF generation produces a well-formed, non-trivial PDF with
// Devanagari text in it (the artisan name below is Devanagari, not just the
// static "आय विवरण" header gofpdf.AddUTF8FontFromBytes has to render it at
// all) — this is the embedded-font acceptance criterion that used to be
// skipped pending a real .ttf being wired in.
func TestGeneratePDF_NonEmpty(t *testing.T) {
	svc := &Service{}
	stmt := &domain.IncomeStatement{
		ArtisanID:   uuid.New(),
		ArtisanName: "लक्ष्मी देवी",
		PeriodStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		OrderCount:  5,
		GrossPaise:  500000,
		NetPaise:    450000,
		FeePaise:    50000,
		Signature:   bytes.Repeat([]byte{0xAB}, 32),
		ShortCode:   "TESTCODE1",
		MonthlyRows: []domain.MonthlyEarnings{
			{Month: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), OrderCount: 5, GrossPaise: 500000, NetPaise: 450000, FeePaise: 50000},
		},
	}

	data, err := svc.generatePDF(stmt, onePixelPNG(t))
	if err != nil {
		t.Fatalf("generatePDF: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("output does not start with the PDF magic header, got %q", data[:min(len(data), 16)])
	}
	// A PDF that actually embedded a ~150KB TTF subset is well into the tens
	// of KB; a near-empty or failed-render output would be far smaller.
	if len(data) < 5000 {
		t.Errorf("pdf output suspiciously small (%d bytes), embedded font may not have loaded", len(data))
	}
}

// onePixelPNG builds a minimal valid PNG so generatePDF's RegisterImageReader
// has real image bytes to decode, the same shape as the QR code it renders
// in production.
func onePixelPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.Black)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding test PNG: %v", err)
	}
	return buf.Bytes()
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
