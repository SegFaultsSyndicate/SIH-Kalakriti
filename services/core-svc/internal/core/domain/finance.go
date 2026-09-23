// services/core-svc/internal/core/domain/finance.go

package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// FinanceCorporations are the MoSJE finance corporations and schemes an
// artisan can link (finance_corporation enum).
var FinanceCorporations = map[string]bool{
	"NSFDC": true, "NBCFDC": true, "NSKFDC": true, "NDFDC": true,
	"PM_DAKSH": true, "PM_AJAY": true, "OTHER": true,
}

// Finance link statuses (finance_link_status enum).
const (
	FinanceSelfReported = "SELF_REPORTED"
	FinanceVerified     = "VERIFIED"
	FinanceRejected     = "REJECTED"
)

// FinanceConsentVersion is the consent text version the app currently shows.
// Bump it whenever the consent wording changes; older links keep the
// version the artisan actually agreed to.
const FinanceConsentVersion = "finance-consent-2026-09"

// FinanceLink is an artisan's linked loan / scheme record. It never holds the
// full reference -- only its last four characters.
type FinanceLink struct {
	ID                 uuid.UUID
	ArtisanID          uuid.UUID
	Corporation        string
	ChannelizingAgency *string
	ReferenceLast4     string
	SanctionedPaise    *int64
	EmiPaise           *int64
	EmiDayOfMonth      *int32
	RepaymentStart     *time.Time
	Status             string
	VerifiedAt         *time.Time
	RejectReason       *string
	ConsentAt          time.Time
	ConsentVersion     string
	CreatedAt          time.Time
}

// FinanceTerms are the editable loan terms.
type FinanceTerms struct {
	ChannelizingAgency *string
	SanctionedPaise    *int64
	EmiPaise           *int64
	EmiDayOfMonth      *int32
	RepaymentStart     *time.Time
}

// Validate checks loan terms against the table's constraints.
func (t FinanceTerms) Validate() error {
	if t.SanctionedPaise != nil && (*t.SanctionedPaise < 0 || *t.SanctionedPaise > 100_000_000_00) {
		return pkgdomain.InvalidInput("sanctioned_paise is out of range")
	}
	if t.EmiPaise != nil && (*t.EmiPaise < 0 || *t.EmiPaise > 10_000_000_00) {
		return pkgdomain.InvalidInput("emi_paise is out of range")
	}
	if t.EmiDayOfMonth != nil && (*t.EmiDayOfMonth < 1 || *t.EmiDayOfMonth > 28) {
		return pkgdomain.InvalidInput("emi_day_of_month must be between 1 and 28")
	}
	if t.ChannelizingAgency != nil && len([]rune(*t.ChannelizingAgency)) > 120 {
		return pkgdomain.InvalidInput("channelizing_agency must be at most 120 characters")
	}
	return nil
}

// NormalizeReference canonicalises a loan / beneficiary reference so the
// same number typed with spaces, hyphens or lower case hashes identically.
func NormalizeReference(ref string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(ref) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// HashReference returns the stored form of a reference: its last four
// characters (for display as XXXX1234) and HMAC-SHA256(salt, normalized)
// (for de-duplication). The plaintext goes nowhere else.
func HashReference(salt []byte, ref string) (last4 string, hash []byte, err error) {
	norm := NormalizeReference(ref)
	if len(norm) < 4 || len(norm) > 40 {
		return "", nil, pkgdomain.InvalidInput("reference must have between 4 and 40 letters or digits")
	}
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(norm))
	return norm[len(norm)-4:], mac.Sum(nil), nil
}

// Coverage statuses for the repayment card.
const (
	CoverageCovered = "COVERED"
	CoverageAlmost  = "ALMOST"
	CoverageNotYet  = "NOT_YET"
	CoverageNoEMI   = "NO_EMI"
)

// RepaymentCoverage is how far a month's income covers the artisan's EMIs.
type RepaymentCoverage struct {
	Month               time.Time
	EmiPaise            int64
	EarnedPlatformPaise int64
	EarnedOfflinePaise  int64
	PendingPaise        int64
	CoverageRatio       float64
	DaysToEmi           int32
	Status              string
	AnyVerified         bool
}

// ComputeCoverage turns EMI terms and earnings into the repayment card.
// "Almost" is three-quarters of the EMI: close enough that one more sale
// could cover it. Only settled platform income and logged offline sales
// count; pending payouts are reported but never counted.
func ComputeCoverage(links []FinanceLink, month time.Time, platform, offline, pending int64, today time.Time) RepaymentCoverage {
	c := RepaymentCoverage{Month: month, EarnedPlatformPaise: platform, EarnedOfflinePaise: offline,
		PendingPaise: pending, DaysToEmi: -1}
	for _, l := range links {
		if l.Status == FinanceRejected {
			continue
		}
		if l.Status == FinanceVerified {
			c.AnyVerified = true
		}
		if l.EmiPaise != nil {
			c.EmiPaise += *l.EmiPaise
		}
		if l.EmiDayOfMonth != nil {
			if d := DaysUntilDay(today, int(*l.EmiDayOfMonth)); c.DaysToEmi < 0 || d < c.DaysToEmi {
				c.DaysToEmi = d
			}
		}
	}
	if c.EmiPaise <= 0 {
		c.Status = CoverageNoEMI
		return c
	}
	c.CoverageRatio = float64(platform+offline) / float64(c.EmiPaise)
	switch {
	case c.CoverageRatio >= 1:
		c.Status = CoverageCovered
	case c.CoverageRatio >= 0.75:
		c.Status = CoverageAlmost
	default:
		c.Status = CoverageNotYet
	}
	return c
}

// DaysUntilDay is the number of days from today until the next occurrence of
// day-of-month (1..28), counting today as 0.
func DaysUntilDay(today time.Time, day int) int32 {
	t := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	next := time.Date(t.Year(), t.Month(), day, 0, 0, 0, 0, time.UTC)
	if next.Before(t) {
		next = next.AddDate(0, 1, 0)
	}
	return int32(next.Sub(t).Hours() / 24)
}

// ParseMonth reads "YYYY-MM" (empty = today's month) as the first of that month, UTC.
func ParseMonth(s string, today time.Time) (time.Time, error) {
	if s == "" {
		return time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC), nil
	}
	m, err := time.Parse("2006-01", s)
	if err != nil {
		return time.Time{}, pkgdomain.InvalidInput(fmt.Sprintf("month %q must be YYYY-MM", s))
	}
	return m, nil
}

// FinanceLinkForReview is a link as an officer's review queue shows it.
type FinanceLinkForReview struct {
	FinanceLink
	ArtisanName string
	StateCode   string
	District    *string
}

// EMIReminder is one finance link whose EMI reminder is due this cycle.
type EMIReminder struct {
	LinkID      uuid.UUID
	ArtisanID   uuid.UUID
	Corporation string
	EmiPaise    int64
	EmiDay      int32
	Language    string
}
