// services/core-svc/internal/core/domain/impact.go

package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/impact"
)

// IncomeBaseline is an artisan's self-reported income from craft before
// joining. Every figure derived from it is labelled "self-reported" in the UI.
type IncomeBaseline struct {
	MonthlyBracket    string
	MonthlyPaise      *int64
	FairsPerYear      *int32
	FairIncomeBracket *string
	CapturedAt        time.Time
	Source            string // SELF | AGENT
}

var incomeBrackets = map[string]bool{
	impact.BracketLT3K: true, impact.Bracket3To6K: true, impact.Bracket6To10K: true,
	impact.Bracket10To15K: true, impact.BracketGT15K: true, impact.BracketPreferNotToSay: true,
}

// Validate checks a baseline before it is stored.
func (b IncomeBaseline) Validate() error {
	if !incomeBrackets[b.MonthlyBracket] {
		return pkgdomain.InvalidInput(fmt.Sprintf("monthly_bracket %q is not a known bracket", b.MonthlyBracket))
	}
	if b.FairIncomeBracket != nil && !incomeBrackets[*b.FairIncomeBracket] {
		return pkgdomain.InvalidInput(fmt.Sprintf("fair_income_bracket %q is not a known bracket", *b.FairIncomeBracket))
	}
	if b.MonthlyPaise != nil && (*b.MonthlyPaise < 0 || *b.MonthlyPaise > 100_000_000) {
		return pkgdomain.InvalidInput("monthly_paise must be between 0 and ₹10,00,000")
	}
	if b.FairsPerYear != nil && (*b.FairsPerYear < 0 || *b.FairsPerYear > 60) {
		return pkgdomain.InvalidInput("fairs_per_year must be between 0 and 60")
	}
	return nil
}

// Offline sale channels (offline_sale_channel enum).
var offlineChannels = map[string]bool{"FAIR": true, "LOCAL_MARKET": true, "DIRECT": true, "OTHER": true}

// OfflineSale is a sale the artisan made outside the platform and logged.
type OfflineSale struct {
	ID          uuid.UUID
	Channel     string
	EventName   *string
	AmountPaise int64
	SoldOn      time.Time
	CreatedAt   time.Time
}

// ValidateOfflineSale checks a logged sale; today is the artisan's "now".
func ValidateOfflineSale(channel string, eventName *string, amountPaise int64, soldOn, today time.Time) error {
	if !offlineChannels[channel] {
		return pkgdomain.InvalidInput(fmt.Sprintf("channel %q must be FAIR, LOCAL_MARKET, DIRECT or OTHER", channel))
	}
	if amountPaise <= 0 || amountPaise > 100_000_000 {
		return pkgdomain.InvalidInput("amount_paise must be positive and at most ₹10,00,000")
	}
	if eventName != nil && len([]rune(strings.TrimSpace(*eventName))) > 120 {
		return pkgdomain.InvalidInput("event_name must be at most 120 characters")
	}
	// A day of slack either way absorbs a phone in another timezone.
	if soldOn.After(today.AddDate(0, 0, 1)) {
		return pkgdomain.InvalidInput("sold_on cannot be in the future")
	}
	if soldOn.Before(today.AddDate(-2, 0, 0)) {
		return pkgdomain.InvalidInput("sold_on must be within the last two years")
	}
	return nil
}

// MonthIncome is one calendar month of an artisan's income.
type MonthIncome struct {
	Month         time.Time // first of the month
	PlatformPaise int64
	OfflinePaise  int64
}

// IncomeFacts is the raw per-artisan input to the income summary.
type IncomeFacts struct {
	RegisteredAt         time.Time
	PlatformPaise90d     int64
	PlatformPendingPaise int64
	OfflinePaise90d      int64
	FairPaise90d         int64
}

// IncomeSummary is an artisan's own income picture, computed only from data.
type IncomeSummary struct {
	Baseline             *IncomeBaseline
	BaselineMonthlyPaise int64
	Months               []MonthIncome
	Facts                IncomeFacts
	CurrentMonthlyPaise  int64
	UpliftPct            *float64
	InsufficientData     string
	DigitalSharePct      *float64
}

// BuildIncomeSummary assembles the summary from stored facts, with the
// last three calendar months (oldest first, zero-filled) ending at now.
func BuildIncomeSummary(baseline *IncomeBaseline, facts IncomeFacts, monthly []MonthIncome, now time.Time) IncomeSummary {
	s := IncomeSummary{Baseline: baseline, Facts: facts}
	s.CurrentMonthlyPaise = impact.CurrentMonthlyPaise(facts.PlatformPaise90d, facts.OfflinePaise90d)

	f := impact.Facts{RegisteredAt: facts.RegisteredAt, Platform90d: facts.PlatformPaise90d, Offline90d: facts.OfflinePaise90d}
	if baseline != nil {
		f.BaselineBracket, f.BaselineExact = baseline.MonthlyBracket, baseline.MonthlyPaise
		if v, ok := impact.BaselineMonthlyPaise(baseline.MonthlyBracket, baseline.MonthlyPaise); ok {
			s.BaselineMonthlyPaise = v
		}
	}
	if pct, reason := impact.Uplift(f, now); reason == "" {
		s.UpliftPct = &pct
	} else {
		s.InsufficientData = reason
	}
	if total := facts.PlatformPaise90d + facts.OfflinePaise90d; total > 0 {
		share := float64(facts.PlatformPaise90d) / float64(total) * 100
		s.DigitalSharePct = &share
	}

	byMonth := make(map[string]MonthIncome, len(monthly))
	for _, m := range monthly {
		byMonth[m.Month.Format("2006-01")] = m
	}
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := 2; i >= 0; i-- {
		m := first.AddDate(0, -i, 0)
		got := byMonth[m.Format("2006-01")]
		got.Month = m
		s.Months = append(s.Months, got)
	}
	return s
}
