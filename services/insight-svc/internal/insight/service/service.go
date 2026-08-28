package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"

	"github.com/segfaultsyndicate/kalakriti/pkg/crypto"
	"github.com/segfaultsyndicate/kalakriti/pkg/ids"
	"github.com/segfaultsyndicate/kalakriti/pkg/money"
	"github.com/segfaultsyndicate/kalakriti/pkg/qrcode"
	"github.com/segfaultsyndicate/kalakriti/pkg/shortcode"
	"github.com/segfaultsyndicate/kalakriti/pkg/storage"
	"github.com/segfaultsyndicate/kalakriti/services/insight-svc/internal/insight/domain"
)

type Store interface {
	InTx(ctx context.Context, fn func(Tx) error) error
	RefreshMaterializedViews(ctx context.Context) error
	GetArtisansByCategory(ctx context.Context, stateCode, district *string) ([]domain.ArtisanCategoryRow, error)
	GetListingsByCraftMonth(ctx context.Context, from, to *time.Time, craftID *uuid.UUID) ([]domain.ListingCraftMonthRow, error)
	GetEarningsByDistrict(ctx context.Context, stateCode, district *string, minBucket int32) ([]domain.EarningsDistrictRow, error)
	GetIncomeComparison(ctx context.Context, stateCode, district *string, minBucket int32) ([]domain.IncomeComparisonRow, error)
	GetDyingCrafts(ctx context.Context, limit int32) ([]domain.DyingCraftRow, error)
	GetArtisanEarningsByPeriod(ctx context.Context, artisanID uuid.UUID, start, end time.Time) (*domain.IncomeStatement, error)
	GetArtisanEarningsMonthly(ctx context.Context, artisanID uuid.UUID, start, end time.Time) ([]domain.MonthlyEarnings, error)
	GetIncomeStatementByCode(ctx context.Context, code string) (*domain.IncomeStatement, error)
	GetArtisanIncomeStatements(ctx context.Context, artisanID uuid.UUID, limit, offset int32) ([]domain.StatementSummary, error)
}

type Tx interface {
	CreateIncomeStatement(ctx context.Context, stmt *domain.IncomeStatement) error
}

type Service struct {
	store          Store
	signer         *crypto.Signer
	qr             *qrcode.Generator
	codeGen        *shortcode.Generator
	s3             *storage.Client
	verifyBaseURL  string
	fontPath       string
	minBucketSize  int32
	presignExpiry  time.Duration
}

type Config struct {
	Store          Store
	Signer         *crypto.Signer
	QRGenerator    *qrcode.Generator
	CodeGenerator  *shortcode.Generator
	S3Client       *storage.Client
	VerifyBaseURL  string
	FontPath       string
	MinBucketSize  int32
	PresignExpiry  time.Duration
}

func New(cfg Config) *Service {
	return &Service{
		store:          cfg.Store,
		signer:         cfg.Signer,
		qr:             cfg.QRGenerator,
		codeGen:        cfg.CodeGenerator,
		s3:             cfg.S3Client,
		verifyBaseURL:  cfg.VerifyBaseURL,
		fontPath:       cfg.FontPath,
		minBucketSize:  cfg.MinBucketSize,
		presignExpiry:  cfg.PresignExpiry,
	}
}

func (s *Service) RefreshMaterializedViews(ctx context.Context) error {
	return s.store.RefreshMaterializedViews(ctx)
}

func (s *Service) GetArtisansByCategory(ctx context.Context, stateCode, district *string) ([]domain.ArtisanCategoryRow, error) {
	return s.store.GetArtisansByCategory(ctx, stateCode, district)
}

func (s *Service) GetListingsByCraftMonth(ctx context.Context, from, to *time.Time, craftID *uuid.UUID) ([]domain.ListingCraftMonthRow, error) {
	return s.store.GetListingsByCraftMonth(ctx, from, to, craftID)
}

func (s *Service) GetEarningsByDistrict(ctx context.Context, stateCode, district *string) ([]domain.EarningsDistrictRow, error) {
	return s.store.GetEarningsByDistrict(ctx, stateCode, district, s.minBucketSize)
}

func (s *Service) GetIncomeComparison(ctx context.Context, stateCode, district *string) ([]domain.IncomeComparisonRow, error) {
	return s.store.GetIncomeComparison(ctx, stateCode, district, s.minBucketSize)
}

func (s *Service) GetDyingCrafts(ctx context.Context, limit int32) ([]domain.DyingCraftRow, error) {
	return s.store.GetDyingCrafts(ctx, limit)
}

func (s *Service) GenerateIncomeStatement(ctx context.Context, artisanID uuid.UUID, start, end time.Time) (*domain.IncomeStatement, error) {
	// Fetch earnings data.
	stmt, err := s.store.GetArtisanEarningsByPeriod(ctx, artisanID, start, end)
	if err != nil {
		return nil, fmt.Errorf("fetching earnings: %w", err)
	}
	monthly, err := s.store.GetArtisanEarningsMonthly(ctx, artisanID, start, end)
	if err != nil {
		return nil, fmt.Errorf("fetching monthly breakdown: %w", err)
	}
	stmt.MonthlyRows = monthly

	// Generate short code.
	code, err := s.codeGen.Generate(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating short code: %w", err)
	}
	stmt.ShortCode = code

	// Generate QR code for verification.
	qrPNG, err := s.qr.GeneratePNG(code, 256)
	if err != nil {
		return nil, fmt.Errorf("generating QR code: %w", err)
	}

	// Generate PDF.
	pdfBytes, err := s.generatePDF(stmt, qrPNG)
	if err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}

	// Sign the PDF content.
	pdfHash := sha256.Sum256(pdfBytes)
	signature := s.signer.Sign(pdfHash[:])
	stmt.Signature = signature
	stmt.SignatureAlgo = "ed25519"
	stmt.PublicKeyID = s.signer.KeyID()

	// Upload to S3.
	stmt.ID = ids.New()
	stmt.S3Key = fmt.Sprintf("income-statements/%s/%s.pdf", artisanID.String(), stmt.ID.String())

	putURL, err := s.s3.PresignedPutURL(ctx, stmt.S3Key, "application/pdf", 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("generating S3 put URL: %w", err)
	}

	// ponytail: direct HTTP PUT with presigned URL instead of SDK upload
	if err := uploadToPresignedURL(ctx, putURL, pdfBytes, "application/pdf"); err != nil {
		return nil, fmt.Errorf("uploading PDF: %w", err)
	}

	// Store statement record.
	stmt.CreatedAt = time.Now().UTC()
	if err := s.store.InTx(ctx, func(tx Tx) error {
		return tx.CreateIncomeStatement(ctx, stmt)
	}); err != nil {
		return nil, fmt.Errorf("storing statement: %w", err)
	}

	return stmt, nil
}

func (s *Service) GetIncomeStatements(ctx context.Context, artisanID uuid.UUID, limit, offset int32) ([]domain.StatementSummary, error) {
	return s.store.GetArtisanIncomeStatements(ctx, artisanID, limit, offset)
}

func (s *Service) VerifyIncomeStatement(ctx context.Context, code string) (*domain.IncomeStatement, bool, error) {
	stmt, err := s.store.GetIncomeStatementByCode(ctx, code)
	if err != nil {
		return nil, false, fmt.Errorf("fetching statement: %w", err)
	}

	// Download PDF from S3 to verify signature.
	getURL, err := s.s3.PresignedGetURL(ctx, stmt.S3Key, s.presignExpiry)
	if err != nil {
		return stmt, false, fmt.Errorf("generating S3 get URL: %w", err)
	}

	pdfBytes, err := downloadFromPresignedURL(ctx, getURL)
	if err != nil {
		return stmt, false, fmt.Errorf("downloading PDF: %w", err)
	}

	// Verify signature.
	pdfHash := sha256.Sum256(pdfBytes)
	publicKey, err := hex.DecodeString(s.signer.PublicKeyHex())
	if err != nil {
		return stmt, false, fmt.Errorf("decoding public key: %w", err)
	}
	valid := crypto.Verify(publicKey, pdfHash[:], stmt.Signature)

	return stmt, valid, nil
}

func (s *Service) GetStatementDownloadURL(ctx context.Context, s3Key string) (string, error) {
	return s.s3.PresignedGetURL(ctx, s3Key, s.presignExpiry)
}

func (s *Service) generatePDF(stmt *domain.IncomeStatement, qrPNG []byte) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Add Unicode font for Devanagari rendering. Must use embedded TTF, NOT core fonts.
	// ponytail: assumes fontPath points to NotoSansDevanagari-Regular.ttf
	pdf.AddUTF8Font("NotoSans", "", s.fontPath)
	pdf.SetFont("NotoSans", "", 12)

	pdf.AddPage()

	// Header.
	pdf.SetFont("NotoSans", "", 18)
	pdf.CellFormat(0, 10, "Income Statement / आय विवरण", "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Artisan details.
	pdf.SetFont("NotoSans", "", 12)
	pdf.CellFormat(50, 8, "Artisan Name:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, stmt.ArtisanName, "", 1, "L", false, 0, "")
	pdf.CellFormat(50, 8, "Artisan ID:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, stmt.ArtisanID.String(), "", 1, "L", false, 0, "")
	pdf.CellFormat(50, 8, "Period:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, fmt.Sprintf("%s to %s", stmt.PeriodStart.Format("2006-01-02"), stmt.PeriodEnd.Format("2006-01-02")), "", 1, "L", false, 0, "")
	pdf.CellFormat(50, 8, "Verification Code:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, stmt.ShortCode, "", 1, "L", false, 0, "")
	pdf.Ln(5)

	// Summary.
	pdf.SetFont("NotoSans", "", 14)
	pdf.CellFormat(0, 10, "Summary", "", 1, "L", false, 0, "")
	pdf.SetFont("NotoSans", "", 12)
	pdf.CellFormat(50, 8, "Orders Fulfilled:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, fmt.Sprintf("%d", stmt.OrderCount), "", 1, "L", false, 0, "")
	pdf.CellFormat(50, 8, "Gross Earnings:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, money.New(stmt.GrossPaise).Format(), "", 1, "L", false, 0, "")
	pdf.CellFormat(50, 8, "Platform Fee:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, money.New(stmt.FeePaise).Format(), "", 1, "L", false, 0, "")
	pdf.CellFormat(50, 8, "Net Earnings:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, money.New(stmt.NetPaise).Format(), "", 1, "L", false, 0, "")
	pdf.Ln(5)

	// Monthly breakdown table.
	pdf.SetFont("NotoSans", "", 14)
	pdf.CellFormat(0, 10, "Monthly Breakdown", "", 1, "L", false, 0, "")
	pdf.SetFont("NotoSans", "", 10)
	pdf.CellFormat(40, 7, "Month", "1", 0, "C", false, 0, "")
	pdf.CellFormat(25, 7, "Orders", "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, 7, "Gross", "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, 7, "Fee", "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, 7, "Net", "1", 1, "C", false, 0, "")

	for _, row := range stmt.MonthlyRows {
		pdf.CellFormat(40, 7, row.Month.Format("Jan 2006"), "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 7, fmt.Sprintf("%d", row.OrderCount), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 7, money.New(row.GrossPaise).Format(), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 7, money.New(row.FeePaise).Format(), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 7, money.New(row.NetPaise).Format(), "1", 1, "R", false, 0, "")
	}
	pdf.Ln(5)

	// QR code for verification.
	pdf.SetFont("NotoSans", "", 10)
	pdf.CellFormat(0, 8, "Scan to verify:", "", 1, "L", false, 0, "")

	// Write QR PNG to temp variable and embed.
	qrPath := fmt.Sprintf("qr_%s.png", stmt.ShortCode)
	pdf.RegisterImageReader(qrPath, "PNG", bytes.NewReader(qrPNG))
	pdf.Image(qrPath, 10, pdf.GetY(), 40, 40, false, "", 0, "")
	pdf.Ln(42)

	pdf.SetFont("NotoSans", "", 8)
	pdf.CellFormat(0, 5, fmt.Sprintf("Issued: %s", time.Now().UTC().Format("2006-01-02 15:04:05 UTC")), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("Signature: %s", hex.EncodeToString(stmt.Signature[:16])+"..."), "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("outputting PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// ponytail: stdlib HTTP PUT/GET for presigned URLs, no SDK overhead
func uploadToPresignedURL(ctx context.Context, url string, data []byte, contentType string) error {
	req, err := httpNewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed: %s", resp.Status)
	}
	return nil
}

func downloadFromPresignedURL(ctx context.Context, url string) ([]byte, error) {
	req, err := httpNewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
