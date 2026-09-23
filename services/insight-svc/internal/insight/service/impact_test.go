package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/impact"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/domain"
)

var now = time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)

// artisan registered 200 days ago with a ₹4,500 baseline (B3K_6K midpoint)
// and ₹18,000 platform income over 90 days = ₹6,000/month, i.e. +33.3%.
func seller(category, district string, corps ...string) domain.ImpactArtisan {
	return domain.ImpactArtisan{
		ArtisanID: uuid.New(), StateCode: "IN-UP", District: district, SocialCategory: category,
		Corporations: corps, RegisteredAt: now.AddDate(0, 0, -200), BaselineBracket: impact.Bracket3To6K,
		Platform90d: 18_000_00,
	}
}

func TestSummarizeSuppressesSmallCohorts(t *testing.T) {
	four := []domain.ImpactArtisan{seller("SC", "Varanasi"), seller("SC", "Varanasi"), seller("SC", "Varanasi"), seller("SC", "Varanasi")}
	if s := summarize(four, now); !s.Suppressed || s.Beneficiaries != 0 || s.MedianUpliftPct != nil {
		t.Fatalf("4 artisans must be suppressed whole: %+v", s)
	}
	five := append(four, seller("SC", "Varanasi"))
	s := summarize(five, now)
	if s.Suppressed || s.Beneficiaries != 5 || s.UpliftSample != 5 || s.MedianUpliftPct == nil {
		t.Fatalf("5 artisans: %+v", s)
	}
	if got := *s.MedianUpliftPct; got < 33.3 || got > 33.4 {
		t.Fatalf("median uplift = %.2f, want 33.33", got)
	}
	if s.DigitalSharePct == nil || *s.DigitalSharePct != 100 {
		t.Fatalf("digital share = %v, want 100", s.DigitalSharePct)
	}
}

func TestSummarizeNeedsFiveForTheUpliftMedian(t *testing.T) {
	cohort := []domain.ImpactArtisan{seller("SC", "V"), seller("SC", "V"), seller("SC", "V"), seller("SC", "V"), seller("SC", "V")}
	cohort[0].BaselineBracket = "" // no baseline: 4 qualify
	s := summarize(cohort, now)
	if s.Suppressed || s.UpliftSample != 4 || s.MedianUpliftPct != nil {
		t.Fatalf("uplift median over 4 must be withheld: %+v", s)
	}
}

// The spec's example: 5 SC + 1 ST artisans. By category, SC shows and ST is
// suppressed -- its lone artisan's income must not be derivable.
func TestGroupBySocialCategorySuppressesTheLoneST(t *testing.T) {
	var cohort []domain.ImpactArtisan
	for range 5 {
		cohort = append(cohort, seller("SC", "Varanasi"))
	}
	cohort = append(cohort, seller("ST", "Varanasi"))
	rows := groupImpact(cohort, GroupSocialCategory, now)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	sc, st := rows[0], rows[1]
	if sc.Group != "SC" || sc.Suppressed || sc.ArtisanCount != 5 || sc.MedianBaselinePaise != 4_500_00 || sc.MedianCurrentPaise != 6_000_00 {
		t.Fatalf("SC row = %+v", sc)
	}
	if st.Group != "ST" || !st.Suppressed || st.ArtisanCount != 0 || st.PlatformIncomePaise != 0 || st.MedianUpliftPct != nil {
		t.Fatalf("ST row must carry nothing but its label: %+v", st)
	}
}

func TestGroupByCorporationCountsAnArtisanOncePerCorporation(t *testing.T) {
	var cohort []domain.ImpactArtisan
	for range 5 {
		cohort = append(cohort, seller("SC", "V", "NSFDC", "PM_AJAY"))
	}
	rows := groupImpact(cohort, GroupCorporation, now)
	if len(rows) != 2 || rows[0].ArtisanCount != 5 || rows[1].ArtisanCount != 5 {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestFinanceCoverage(t *testing.T) {
	var facts []domain.FinanceCoverageFact
	for i := range 5 {
		status := "SELF_REPORTED"
		if i < 2 {
			status = "VERIFIED"
		}
		// ₹6,000/month income against a ₹3,000 EMI: ratio 2.
		facts = append(facts, domain.FinanceCoverageFact{ArtisanID: uuid.New(), Corporation: "NSFDC", Status: status, EmiPaise: 3_000_00, Platform90d: 18_000_00})
	}
	facts = append(facts, domain.FinanceCoverageFact{ArtisanID: uuid.New(), Corporation: "NDFDC", Status: "VERIFIED", EmiPaise: 1_000_00})
	rows := financeCoverage(facts)
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	ndfdc, nsfdc := rows[0], rows[1]
	if !ndfdc.Suppressed || ndfdc.Beneficiaries != 0 {
		t.Fatalf("NDFDC (1 artisan) must be suppressed: %+v", ndfdc)
	}
	if nsfdc.Beneficiaries != 5 || nsfdc.Verified != 2 || nsfdc.SelfReported != 3 || nsfdc.MedianCoverageRatio == nil || *nsfdc.MedianCoverageRatio != 2 {
		t.Fatalf("NSFDC = %+v", nsfdc)
	}
}
