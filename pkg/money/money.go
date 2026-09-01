// pkg/money/money.go
package money

import (
	"fmt"

	pb "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
)

// Money represents an amount in paise (1/100 of rupee).
type Money int64

// NewMoney creates Money from paise.
func NewMoney(paise int64) Money {
	return Money(paise)
}

// Paise returns the raw paise value.
func (m Money) Paise() int64 {
	return int64(m)
}

// Add returns m + other.
func (m Money) Add(other Money) Money {
	return Money(int64(m) + int64(other))
}

// Sub returns m - other.
func (m Money) Sub(other Money) Money {
	return Money(int64(m) - int64(other))
}

// MulPct returns m * (pct / 100), rounded down.
func (m Money) MulPct(pct int) Money {
	return Money((int64(m) * int64(pct)) / 100)
}

// Split divides m into n parts, distributing remainder to first parts so no paise is lost.
// Split(1000, 3) = [334, 333, 333].
func (m Money) Split(n int) []Money {
	if n <= 0 {
		return nil
	}
	base := int64(m) / int64(n)
	remainder := int64(m) % int64(n)

	parts := make([]Money, n)
	for i := 0; i < n; i++ {
		parts[i] = Money(base)
		if i < int(remainder) {
			parts[i]++
		}
	}
	return parts
}

// Format returns a display string in rupees with 2 decimal places.
func (m Money) Format() string {
	rupees := int64(m) / 100
	paise := int64(m) % 100
	return fmt.Sprintf("₹%d.%02d", rupees, paise)
}

// ToProto converts to protobuf Money message.
func (m Money) ToProto() *pb.Money {
	return &pb.Money{
		AmountPaise:  int64(m),
		CurrencyCode: "INR",
	}
}

// FromProto converts from protobuf Money message.
func FromProto(pm *pb.Money) Money {
	if pm == nil {
		return 0
	}
	return Money(pm.AmountPaise)
}
