// services/insight-svc/internal/insight/domain/statement.go
package domain

import (
	"time"

	"github.com/google/uuid"
)

// IncomeStatement is a verifiable, signed income statement for an artisan.
type IncomeStatement struct {
	ID            uuid.UUID
	ArtisanID     uuid.UUID
	ArtisanName   string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	OrderCount    int64
	GrossPaise    int64
	NetPaise      int64
	FeePaise      int64
	Signature     []byte
	SignatureAlgo string
	PublicKeyID   string
	ShortCode     string
	S3Key         string
	CreatedAt     time.Time
	MonthlyRows   []MonthlyEarnings
}

// MonthlyEarnings is one month's breakdown within a statement's period.
type MonthlyEarnings struct {
	Month      time.Time
	OrderCount int64
	GrossPaise int64
	NetPaise   int64
	FeePaise   int64
}

// StatementSummary is one row in an artisan's statement history.
type StatementSummary struct {
	ID          uuid.UUID
	PeriodStart time.Time
	PeriodEnd   time.Time
	OrderCount  int64
	GrossPaise  int64
	NetPaise    int64
	FeePaise    int64
	ShortCode   string
	S3Key       string
	CreatedAt   time.Time
}
