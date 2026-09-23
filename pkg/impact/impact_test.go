// pkg/impact/impact_test.go
package impact

import (
	"math"
	"testing"
	"time"
)

func TestBracketMidpointPaise(t *testing.T) {
	t.Parallel()
	cases := map[string]int64{
		BracketLT3K:    150_000,
		Bracket3To6K:   450_000,
		Bracket6To10K:  800_000,
		Bracket10To15K: 1_250_000,
		BracketGT15K:   2_000_000,
	}
	for bracket, want := range cases {
		got, ok := BracketMidpointPaise(bracket)
		if !ok || got != want {
			t.Errorf("BracketMidpointPaise(%s) = %d, %v; want %d, true", bracket, got, ok, want)
		}
	}
	for _, excluded := range []string{BracketPreferNotToSay, "", "NONSENSE"} {
		if _, ok := BracketMidpointPaise(excluded); ok {
			t.Errorf("BracketMidpointPaise(%q) should not resolve", excluded)
		}
	}
}

func TestBaselineExactWinsButPreferNotToSayNeverResolves(t *testing.T) {
	t.Parallel()
	exact := int64(700_000)
	if got, ok := BaselineMonthlyPaise(Bracket6To10K, &exact); !ok || got != exact {
		t.Errorf("exact baseline = %d, %v; want %d", got, ok, exact)
	}
	if _, ok := BaselineMonthlyPaise(BracketPreferNotToSay, &exact); ok {
		t.Error("PREFER_NOT_TO_SAY must exclude the artisan even with an exact figure")
	}
}

func TestUplift(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	old := now.Add(-120 * 24 * time.Hour)

	// Baseline ₹4,500/month; 90-day income ₹18,000 = ₹6,000/month -> +33.33%.
	pct, reason := Uplift(Facts{RegisteredAt: old, BaselineBracket: Bracket3To6K,
		Platform90d: 1_200_000, Offline90d: 600_000}, now)
	if reason != "" || math.Abs(pct-33.333) > 0.01 {
		t.Errorf("uplift = %.3f (%q), want 33.333", pct, reason)
	}

	// Income fell: negative uplift is reported honestly, not clamped.
	pct, _ = Uplift(Facts{RegisteredAt: old, BaselineBracket: Bracket6To10K, Platform90d: 1_200_000}, now)
	if math.Abs(pct-(-50)) > 0.01 {
		t.Errorf("uplift = %.3f, want -50", pct)
	}

	cases := []struct {
		name string
		f    Facts
		want string
	}{
		{"no baseline", Facts{RegisteredAt: old, Platform90d: 1}, ReasonNoBaseline},
		{"prefer not to say", Facts{RegisteredAt: old, BaselineBracket: BracketPreferNotToSay, Platform90d: 1}, ReasonNoBaseline},
		{"too new", Facts{RegisteredAt: now.Add(-30 * 24 * time.Hour), BaselineBracket: BracketLT3K, Platform90d: 1}, ReasonTooNew},
		{"no sales", Facts{RegisteredAt: old, BaselineBracket: BracketLT3K}, ReasonNoSales},
	}
	for _, c := range cases {
		if _, reason := Uplift(c.f, now); reason != c.want {
			t.Errorf("%s: reason = %q, want %q", c.name, reason, c.want)
		}
	}
}

func TestSuppressedAtFewerThanFive(t *testing.T) {
	t.Parallel()
	for n := 0; n < 5; n++ {
		if !Suppressed(n) {
			t.Errorf("a group of %d must be suppressed", n)
		}
	}
	if Suppressed(5) {
		t.Error("a group of 5 must be shown")
	}
}

func TestMedian(t *testing.T) {
	t.Parallel()
	if _, ok := Median([]int64{}); ok {
		t.Error("median of nothing should not resolve")
	}
	in := []int64{9, 1, 5}
	if m, _ := Median(in); m != 5 {
		t.Errorf("median = %d, want 5", m)
	}
	if in[0] != 9 {
		t.Error("Median must not reorder its input")
	}
	if m, _ := Median([]float64{1, 2, 3, 10}); m != 2.5 {
		t.Errorf("median = %v, want 2.5", m)
	}
}
