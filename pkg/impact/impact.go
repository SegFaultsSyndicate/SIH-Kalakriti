// pkg/impact/impact.go

// Package impact holds the few rules every impact figure in Kalakriti is
// computed with, so an artisan's own "vs your baseline" card (core-svc) and
// the ministry dashboard (insight-svc) can never disagree:
//
//   - the income bracket -> representative monthly rupee value mapping,
//   - the uplift formula and when it is allowed to be computed at all,
//   - the k-anonymity threshold below which a group is suppressed.
package impact

import (
	"slices"
	"time"
)

// Income brackets (income_bracket enum, migrations/038_impact.sql).
const (
	BracketLT3K           = "LT_3K"
	Bracket3To6K          = "B3K_6K"
	Bracket6To10K         = "B6K_10K"
	Bracket10To15K        = "B10K_15K"
	BracketGT15K          = "GT_15K"
	BracketPreferNotToSay = "PREFER_NOT_TO_SAY"
)

// MinCohort is the smallest group an aggregate may be shown for. A group of
// fewer artisans is suppressed ("<5") so no individual can be re-identified.
const MinCohort = 5

// MinTenure is how long an artisan must have been registered before their
// uplift is computed: the trailing window has to be entirely post-join.
const MinTenure = 90 * 24 * time.Hour

// BracketMidpointPaise is the representative monthly income for a bracket,
// in paise. The open-ended top bracket uses ₹20,000 -- deliberately
// conservative, so it can only understate uplift for that group, never
// inflate it. ok is false for PREFER_NOT_TO_SAY or an unknown value; such an
// artisan is excluded from uplift statistics.
func BracketMidpointPaise(bracket string) (paise int64, ok bool) {
	const rupee = 100
	switch bracket {
	case BracketLT3K:
		return 1_500 * rupee, true
	case Bracket3To6K:
		return 4_500 * rupee, true
	case Bracket6To10K:
		return 8_000 * rupee, true
	case Bracket10To15K:
		return 12_500 * rupee, true
	case BracketGT15K:
		return 20_000 * rupee, true
	default:
		return 0, false
	}
}

// BaselineMonthlyPaise resolves an artisan's baseline: an exact figure when
// they gave one, else their bracket's midpoint.
func BaselineMonthlyPaise(bracket string, exact *int64) (int64, bool) {
	if bracket == BracketPreferNotToSay {
		return 0, false
	}
	if exact != nil && *exact > 0 {
		return *exact, true
	}
	return BracketMidpointPaise(bracket)
}

// Reasons uplift cannot be computed (the income summary's
// insufficient_data field).
const (
	ReasonNoBaseline = "NO_BASELINE"
	ReasonTooNew     = "TOO_NEW"
	ReasonNoSales    = "NO_SALES"
)

// Facts is what an uplift calculation needs about one artisan.
type Facts struct {
	RegisteredAt    time.Time
	BaselineBracket string // "" when no baseline was captured
	BaselineExact   *int64
	// Trailing-90-day totals, paise.
	Platform90d int64
	Offline90d  int64
}

// CurrentMonthlyPaise is the trailing-90-day average monthly income.
func CurrentMonthlyPaise(platform90d, offline90d int64) int64 {
	return (platform90d + offline90d) / 3
}

// Uplift returns the percentage change of trailing-90-day average monthly
// income over the self-reported baseline, or the reason it cannot be
// computed. Only artisans registered for at least MinTenure with a usable
// baseline and some sale in the window qualify.
func Uplift(f Facts, now time.Time) (pct float64, reason string) {
	baseline, ok := BaselineMonthlyPaise(f.BaselineBracket, f.BaselineExact)
	if f.BaselineBracket == "" || !ok || baseline <= 0 {
		return 0, ReasonNoBaseline
	}
	if now.Sub(f.RegisteredAt) < MinTenure {
		return 0, ReasonTooNew
	}
	if f.Platform90d+f.Offline90d <= 0 {
		return 0, ReasonNoSales
	}
	current := CurrentMonthlyPaise(f.Platform90d, f.Offline90d)
	return float64(current-baseline) / float64(baseline) * 100, ""
}

// Suppressed reports whether a group of n artisans must be hidden.
func Suppressed(n int) bool { return n < MinCohort }

// Median returns the median of vs (mean of the middle two for an even
// count); ok is false for an empty slice. vs is not modified.
func Median[T int64 | float64](vs []T) (T, bool) {
	if len(vs) == 0 {
		return 0, false
	}
	s := slices.Clone(vs)
	slices.Sort(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid], true
	}
	return (s[mid-1] + s[mid]) / 2, true
}
