// services/core-svc/internal/core/domain/impact_test.go
package domain

import (
	"testing"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/impact"
)

func TestBuildIncomeSummaryNewArtisanIsHonestlyEmpty(t *testing.T) {
	now := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	s := BuildIncomeSummary(nil, IncomeFacts{RegisteredAt: now.AddDate(0, 0, -3)}, nil, now)
	if s.UpliftPct != nil || s.InsufficientData != impact.ReasonNoBaseline {
		t.Errorf("uplift = %v reason = %q, want nil NO_BASELINE", s.UpliftPct, s.InsufficientData)
	}
	if s.DigitalSharePct != nil {
		t.Error("no income must mean no digital share, not 0%")
	}
	if len(s.Months) != 3 || s.Months[0].Month.Format("2006-01") != "2026-07" || s.Months[2].Month.Format("2006-01") != "2026-09" {
		t.Errorf("months = %+v, want Jul..Sep 2026 zero-filled", s.Months)
	}
}

func TestBuildIncomeSummaryComputesUpliftAndShare(t *testing.T) {
	now := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	baseline := &IncomeBaseline{MonthlyBracket: impact.Bracket3To6K}
	facts := IncomeFacts{RegisteredAt: now.AddDate(0, -6, 0), PlatformPaise90d: 1_200_000, OfflinePaise90d: 600_000}
	aug := MonthIncome{Month: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), PlatformPaise: 500_000}

	s := BuildIncomeSummary(baseline, facts, []MonthIncome{aug}, now)
	if s.UpliftPct == nil || *s.UpliftPct < 33.3 || *s.UpliftPct > 33.4 {
		t.Fatalf("uplift = %v, want ~33.3", s.UpliftPct)
	}
	if s.BaselineMonthlyPaise != 450_000 || s.CurrentMonthlyPaise != 600_000 {
		t.Errorf("baseline/current = %d/%d, want 450000/600000", s.BaselineMonthlyPaise, s.CurrentMonthlyPaise)
	}
	if s.DigitalSharePct == nil || *s.DigitalSharePct < 66.6 || *s.DigitalSharePct > 66.7 {
		t.Errorf("digital share = %v, want ~66.7", s.DigitalSharePct)
	}
	if s.Months[1].PlatformPaise != 500_000 {
		t.Errorf("August platform = %d, want 500000", s.Months[1].PlatformPaise)
	}
}
