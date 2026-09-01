// services/insight-svc/internal/insight/service/statement.go
package service

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/domain"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
)

// StatementRepo persists income statements.
type StatementRepo interface {
	GetOrdersByArtisan(ctx context.Context, artisanID uuid.UUID, from, to time.Time) ([]domain.OrderSummary, error)
	SaveStatement(ctx context.Context, stmt *domain.IncomeStatement) error
	GetStatementByCode(ctx context.Context, code string) (*domain.IncomeStatement, error)
}

// StorageClient uploads PDFs.
type StorageClient interface {
	PutObject(ctx context.Context, key string, data []byte, contentType string) error
	GetPresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// StatementService generates verifiable income statements.
type StatementService struct {
	repo       StatementRepo
	storage    StorageClient
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewStatementService creates the service.
func NewStatementService(repo StatementRepo, storage StorageClient, privateKey ed25519.PrivateKey) *StatementService {
	return &StatementService{
		repo:       repo,
		storage:    storage,
		privateKey: privateKey,
		publicKey:  privateKey.Public().(ed25519.PublicKey),
	}
}

// GenerateStatement creates a signed PDF income statement.
func (s *StatementService) GenerateStatement(ctx context.Context, artisanID uuid.UUID, year, month int) (pdfURL, code string, err error) {
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	orders, err := s.repo.GetOrdersByArtisan(ctx, artisanID, from, to)
	if err != nil {
		return "", "", fmt.Errorf("fetching orders: %w", err)
	}

	grossPaise := int64(0)
	for _, o := range orders {
		grossPaise += o.AmountPaise
	}

	// No platform fees below earnings floor (spec says zero below floor)
	feesPaise := int64(0)
	netPaise := grossPaise - feesPaise

	stmt := &domain.IncomeStatement{
		ID:         uuid.New(),
		ArtisanID:  artisanID,
		Year:       year,
		Month:      month,
		OrderCount: len(orders),
		GrossPaise: grossPaise,
		FeesPaise:  feesPaise,
		NetPaise:   netPaise,
		CreatedAt:  time.Now().UTC(),
	}

	// Generate short code
	stmt.Code = s.generateCode(stmt)

	// Sign canonical JSON
	canonical, err := s.canonicalJSON(stmt)
	if err != nil {
		return "", "", fmt.Errorf("canonical JSON: %w", err)
	}
	hash := sha256.Sum256(canonical)
	stmt.Signature = ed25519.Sign(s.privateKey, hash[:])

	// Generate PDF
	pdfData, err := s.renderPDF(stmt, orders)
	if err != nil {
		return "", "", fmt.Errorf("rendering PDF: %w", err)
	}

	// Upload to storage
	key := fmt.Sprintf("statements/%s/%s.pdf", artisanID, stmt.ID)
	if err := s.storage.PutObject(ctx, key, pdfData, "application/pdf"); err != nil {
		return "", "", fmt.Errorf("uploading PDF: %w", err)
	}

	// Save statement record
	if err := s.repo.SaveStatement(ctx, stmt); err != nil {
		return "", "", fmt.Errorf("saving statement: %w", err)
	}

	// Get presigned URL
	url, err := s.storage.GetPresignedURL(ctx, key, 24*time.Hour)
	if err != nil {
		return "", "", fmt.Errorf("presigned URL: %w", err)
	}

	return url, stmt.Code, nil
}

// VerifyStatement checks signature and returns statement details.
func (s *StatementService) VerifyStatement(ctx context.Context, code string) (*domain.IncomeStatement, bool, error) {
	stmt, err := s.repo.GetStatementByCode(ctx, code)
	if err == domain.ErrNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("fetching statement: %w", err)
	}

	canonical, err := s.canonicalJSON(stmt)
	if err != nil {
		return nil, false, fmt.Errorf("canonical JSON: %w", err)
	}
	hash := sha256.Sum256(canonical)

	valid := ed25519.Verify(s.publicKey, hash[:], stmt.Signature)
	return stmt, valid, nil
}

func (s *StatementService) generateCode(stmt *domain.IncomeStatement) string {
	data := fmt.Sprintf("%s-%d-%02d", stmt.ArtisanID, stmt.Year, stmt.Month)
	hash := sha256.Sum256([]byte(data))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash[:])
	return encoded[:10] // 10 chars, collision-checked in repo
}

func (s *StatementService) canonicalJSON(stmt *domain.IncomeStatement) ([]byte, error) {
	// Deterministic JSON: sorted keys, no whitespace
	canonical := map[string]interface{}{
		"artisan_id":  stmt.ArtisanID.String(),
		"created_at":  stmt.CreatedAt.Format(time.RFC3339),
		"fees_paise":  stmt.FeesPaise,
		"gross_paise": stmt.GrossPaise,
		"id":          stmt.ID.String(),
		"month":       stmt.Month,
		"net_paise":   stmt.NetPaise,
		"order_count": stmt.OrderCount,
		"year":        stmt.Year,
	}
	return json.Marshal(canonical)
}

func (s *StatementService) renderPDF(stmt *domain.IncomeStatement, orders []domain.OrderSummary) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// ponytail: Using default font Arial for now, Noto Sans Devanagari requires TTF file embed
	// Add when Indic rendering is needed: pdf.AddUTF8Font("NotoSans", "", "NotoSans-Regular.ttf")
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Income Statement")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 6, fmt.Sprintf("Artisan ID: %s", stmt.ArtisanID))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Period: %d-%02d", stmt.Year, stmt.Month))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Generated: %s", stmt.CreatedAt.Format("2006-01-02")))
	pdf.Ln(10)

	// Summary
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 6, "Summary")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 11)
	pdf.Cell(80, 6, "Order Count:")
	pdf.Cell(0, 6, fmt.Sprintf("%d", stmt.OrderCount))
	pdf.Ln(6)
	pdf.Cell(80, 6, "Gross Earnings:")
	pdf.Cell(0, 6, formatPaise(stmt.GrossPaise))
	pdf.Ln(6)
	pdf.Cell(80, 6, "Platform Fees:")
	pdf.Cell(0, 6, formatPaise(stmt.FeesPaise))
	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(80, 6, "Net Earnings:")
	pdf.Cell(0, 6, formatPaise(stmt.NetPaise))
	pdf.Ln(12)

	// Order table
	if len(orders) > 0 {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(0, 6, "Order Details")
		pdf.Ln(8)

		pdf.SetFont("Arial", "", 9)
		pdf.Cell(30, 6, "Date")
		pdf.Cell(50, 6, "Order ID")
		pdf.Cell(0, 6, "Amount")
		pdf.Ln(6)

		for _, o := range orders {
			pdf.Cell(30, 6, o.Date.Format("2006-01-02"))
			pdf.Cell(50, 6, o.OrderID.String()[:8]+"...")
			pdf.Cell(0, 6, formatPaise(o.AmountPaise))
			pdf.Ln(5)
		}
	}

	pdf.Ln(10)
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 6, fmt.Sprintf("Verification Code: %s", stmt.Code))
	pdf.Ln(5)
	pdf.Cell(0, 6, "Verify at: https://kalakriti.in/v/statement/"+stmt.Code)

	var buf []byte
	pdfBuf := pdf.Output(func(data string, length int) error {
		buf = []byte(data)
		return nil
	})
	if pdfBuf != nil {
		return nil, pdfBuf
	}
	return buf, nil
}

func formatPaise(paise int64) string {
	rupees := paise / 100
	paiseRem := paise % 100
	return fmt.Sprintf("₹%d.%02d", rupees, paiseRem)
}
