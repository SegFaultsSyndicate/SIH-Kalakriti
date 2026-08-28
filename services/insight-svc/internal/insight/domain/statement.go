// services/insight-svc/internal/insight/domain/statement.go
package domain

import (
	"time"

	"github.com/google/uuid"
)

// IncomeStatement represents a verifiable income statement for an artisan.
type IncomeStatement struct {
	ID         uuid.UUID
	ArtisanID  uuid.UUID
	Year       int
	Month      int
	OrderCount int
	GrossPaise int64
	FeesPaise  int64
	NetPaise   int64
	CreatedAt  time.Time
	Code       string // Short verification code
	Signature  []byte
}

// OrderSummary is one row in the statement PDF.
type OrderSummary struct {
	Date       time.Time
	OrderID    uuid.UUID
	BuyerName  string
	AmountPaise int64
}
