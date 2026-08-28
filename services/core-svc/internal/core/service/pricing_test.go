// services/core-svc/internal/core/service/pricing_test.go
package service

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/pkg/money"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// fakePricingStore is a hand-configured PricingStore; each field a test needs
// is set directly rather than reused from the catalog/media fakes, since
// pricing reads nothing those fakes model.
type fakePricingStore struct {
	source domain.PricingSource
	wage   domain.WageRate
	timing domain.TimingSignal

	// strictBand/widenedBand let a test give the strict and the craft-only
	// widened pass different sample sizes; comparableCalls records how many
	// times, and with what Strict value, Comparables was called.
	strictBand      domain.MarketBand
	widenedBand     domain.MarketBand
	comparableCalls []bool
}

func (f *fakePricingStore) PricingSource(context.Context, uuid.UUID, string) (domain.PricingSource, error) {
	return f.source, nil
}

func (f *fakePricingStore) MinimumWage(context.Context, string, time.Time) (domain.WageRate, error) {
	return f.wage, nil
}

func (f *fakePricingStore) SeasonalityMultiplier(context.Context, uuid.UUID, time.Month) (domain.TimingSignal, error) {
	return f.timing, nil
}

func (f *fakePricingStore) Comparables(_ context.Context, filter domain.ComparablesFilter) (domain.MarketBand, error) {
	f.comparableCalls = append(f.comparableCalls, filter.Strict)
	if filter.Strict {
		return f.strictBand, nil
	}
	return f.widenedBand, nil
}

func newTestPricing(store *fakePricingStore) *Pricing {
	return NewPricing(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func baseAdviseInput(listingID uuid.UUID) AdviseInput {
	return AdviseInput{
		ListingID:    listingID,
		MaterialCost: money.New(50000), // ₹500
		Hours:        4,
		AsOf:         time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
	}
}

// TestRecommendedMinNeverBelowFloor is the acceptance-criteria invariant,
// exercised across bands that sit below, at, and above the floor, widened or
// not, with and without a seasonal multiplier.
func TestRecommendedMinNeverBelowFloor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		wageRate   int64
		band       domain.MarketBand
		widened    bool
		multiplier float64
	}{
		{"band far below floor", 6800, domain.MarketBand{P25: money.New(1000), P50: money.New(1200), P75: money.New(1500), SampleSize: 8}, false, 1.0},
		{"band above floor", 6800, domain.MarketBand{P25: money.New(900000), P50: money.New(950000), P75: money.New(1000000), SampleSize: 8}, false, 1.0},
		{"band exactly at floor", 6800, domain.MarketBand{P25: money.New(77200), P50: money.New(77200), P75: money.New(77200), SampleSize: 8}, false, 1.0},
		{"zero comparables even after widening", 6800, domain.MarketBand{}, true, 1.0},
		{"seasonal multiplier pushes band up", 6800, domain.MarketBand{P25: money.New(900000), P50: money.New(950000), P75: money.New(1000000), SampleSize: 5}, false, 1.3},
		{"seasonal multiplier still below floor", 6800, domain.MarketBand{P25: money.New(100), P50: money.New(150), P75: money.New(200), SampleSize: 5}, false, 1.3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			craftID := uuid.Must(uuid.NewRandom())
			listingID := uuid.Must(uuid.NewRandom())
			store := &fakePricingStore{
				source: domain.PricingSource{CraftID: craftID, StateCode: "IN-UP"},
				wage:   domain.WageRate{StateCode: "IN-UP", PaisePerHour: tc.wageRate},
				timing: domain.TimingSignal{Multiplier: tc.multiplier, Festival: "Diwali"},
			}
			tc.band.Widened = tc.widened
			// domain.MinComparables is 5; route the band through whichever
			// pass the sample size implies so BuildAdvisory sees it either way.
			if tc.band.SampleSize >= domain.MinComparables && !tc.widened {
				store.strictBand = tc.band
			} else {
				store.strictBand = domain.MarketBand{SampleSize: 0}
				store.widenedBand = tc.band
			}

			advisory, _, err := newTestPricing(store).Advise(context.Background(), baseAdviseInput(listingID))
			require.NoError(t, err)
			require.True(t, advisory.RecommendedMin.Paise() >= advisory.Floor.Paise(),
				"recommended_min %d must be >= cost_floor %d", advisory.RecommendedMin.Paise(), advisory.Floor.Paise())
			require.True(t, advisory.RecommendedMax.Paise() >= advisory.RecommendedMin.Paise())
		})
	}
}

// TestEveryReturnedNumberHasADriver is the second acceptance-criteria
// invariant: cost_floor, wage_rate, market p25/p50/p75, sample_size, timing
// multiplier and the two recommended bounds must each appear in Drivers.
func TestEveryReturnedNumberHasADriver(t *testing.T) {
	t.Parallel()
	craftID := uuid.Must(uuid.NewRandom())
	listingID := uuid.Must(uuid.NewRandom())
	store := &fakePricingStore{
		source: domain.PricingSource{CraftID: craftID, StateCode: "IN-UP"},
		wage:   domain.WageRate{StateCode: "IN-KL", PaisePerHour: 8600},
		timing: domain.TimingSignal{Multiplier: 1.25, Festival: "Diwali"},
		strictBand: domain.MarketBand{
			P25: money.New(400000), P50: money.New(450000), P75: money.New(500000), SampleSize: 9,
		},
	}

	advisory, _, err := newTestPricing(store).Advise(context.Background(), baseAdviseInput(listingID))
	require.NoError(t, err)

	values := make(map[string]bool, len(advisory.Drivers))
	for _, d := range advisory.Drivers {
		values[d.Value] = true
		require.NotEmpty(t, d.ExplanationKey, "driver %q has no explanation key", d.Name)
	}

	want := []string{
		strconv.FormatInt(advisory.Floor.Paise(), 10),
		strconv.FormatInt(store.wage.PaisePerHour, 10),
		strconv.FormatInt(store.strictBand.P25.Paise(), 10),
		strconv.FormatInt(store.strictBand.P50.Paise(), 10),
		strconv.FormatInt(store.strictBand.P75.Paise(), 10),
		strconv.Itoa(store.strictBand.SampleSize),
		strconv.FormatInt(advisory.RecommendedMin.Paise(), 10),
		strconv.FormatInt(advisory.RecommendedMax.Paise(), 10),
	}
	for _, v := range want {
		require.True(t, values[v], "no driver carries value %q; drivers=%+v", v, advisory.Drivers)
	}
}

// TestFewerThanFiveComparablesWidens exercises deliverable 1.b directly: a
// strict sample under domain.MinComparables triggers the craft-only widened
// pass, and the widened result is reported as such.
func TestFewerThanFiveComparablesWidens(t *testing.T) {
	t.Parallel()
	craftID := uuid.Must(uuid.NewRandom())
	listingID := uuid.Must(uuid.NewRandom())
	store := &fakePricingStore{
		source:      domain.PricingSource{CraftID: craftID, StateCode: "IN-UP"},
		wage:        domain.WageRate{StateCode: "IN-UP", PaisePerHour: 6800},
		timing:      domain.TimingSignal{Multiplier: 1.0},
		strictBand:  domain.MarketBand{SampleSize: 3, P25: money.New(1), P50: money.New(2), P75: money.New(3)},
		widenedBand: domain.MarketBand{SampleSize: 40, P25: money.New(100), P50: money.New(200), P75: money.New(300)},
	}

	advisory, _, err := newTestPricing(store).Advise(context.Background(), baseAdviseInput(listingID))
	require.NoError(t, err)
	require.Equal(t, []bool{true, false}, store.comparableCalls, "strict pass must run before the widened pass")
	require.True(t, advisory.Band.Widened)
	require.Equal(t, 40, advisory.Band.SampleSize)

	found := false
	for _, d := range advisory.Drivers {
		if d.Name == "market_widened" {
			found = true
		}
	}
	require.True(t, found, "a widened band must carry a market_widened driver explaining why")
}

// TestFiveComparablesDoesNotWiden is the boundary: exactly MinComparables
// must not trigger the widened pass at all.
func TestFiveComparablesDoesNotWiden(t *testing.T) {
	t.Parallel()
	craftID := uuid.Must(uuid.NewRandom())
	listingID := uuid.Must(uuid.NewRandom())
	store := &fakePricingStore{
		source:     domain.PricingSource{CraftID: craftID, StateCode: "IN-UP"},
		wage:       domain.WageRate{StateCode: "IN-UP", PaisePerHour: 6800},
		timing:     domain.TimingSignal{Multiplier: 1.0},
		strictBand: domain.MarketBand{SampleSize: domain.MinComparables, P25: money.New(100), P50: money.New(200), P75: money.New(300)},
	}

	advisory, _, err := newTestPricing(store).Advise(context.Background(), baseAdviseInput(listingID))
	require.NoError(t, err)
	require.Equal(t, []bool{true}, store.comparableCalls, "a strict sample at the minimum must not widen")
	require.False(t, advisory.Band.Widened)
}

// TestAnomalyBoundaries covers deliverable 3 at the exact thresholds: below
// the floor and above 1.5x p75 flag; exactly at either boundary does not.
func TestAnomalyBoundaries(t *testing.T) {
	t.Parallel()
	floor := money.New(10000)
	p75 := money.New(20000)
	ceiling := p75.MulPct(150) // 30000

	tests := []struct {
		name      string
		chosen    money.Money
		wantLevel domain.AnomalyLevel
		wantShort int64
	}{
		{"exactly at floor is not underpriced", floor, domain.AnomalyNone, 0},
		{"one paisa below floor is underpriced", money.New(9999), domain.AnomalyUnderpriced, 1},
		{"one rupee below floor reports exact shortfall", money.New(9000), domain.AnomalyUnderpriced, 1000},
		{"exactly at 1.5x p75 is not overpriced", ceiling, domain.AnomalyNone, 0},
		{"one paisa above 1.5x p75 is overpriced", money.New(ceiling.Paise() + 1), domain.AnomalyOverpriced, 0},
		{"mid-band is neither", money.New(15000), domain.AnomalyNone, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := domain.CheckAnomaly(tc.chosen, floor, p75)
			require.Equal(t, tc.wantLevel, got.Level)
			require.Equal(t, tc.wantShort, got.ShortfallPaise)
			require.NotEmpty(t, got.ExplanationKey)
		})
	}
}

// TestAdviseOmitsAnomalyWithoutAChosenPrice confirms the anomaly check only
// runs when the caller actually supplies a price to check.
func TestAdviseOmitsAnomalyWithoutAChosenPrice(t *testing.T) {
	t.Parallel()
	craftID := uuid.Must(uuid.NewRandom())
	listingID := uuid.Must(uuid.NewRandom())
	store := &fakePricingStore{
		source:     domain.PricingSource{CraftID: craftID, StateCode: "IN-UP"},
		wage:       domain.WageRate{StateCode: "IN-UP", PaisePerHour: 6800},
		timing:     domain.TimingSignal{Multiplier: 1.0},
		strictBand: domain.MarketBand{SampleSize: 5, P25: money.New(100), P50: money.New(200), P75: money.New(300)},
	}

	in := baseAdviseInput(listingID)
	_, anomaly, err := newTestPricing(store).Advise(context.Background(), in)
	require.NoError(t, err)
	require.Nil(t, anomaly)

	chosen := money.New(1)
	in.ChosenPrice = &chosen
	_, anomaly, err = newTestPricing(store).Advise(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, anomaly)
	require.Equal(t, domain.AnomalyUnderpriced, anomaly.Level)
}

// TestAdviseRejectsInvalidInput checks the trust-boundary validation: negative
// hours and negative material cost are both rejected before any store call.
func TestAdviseRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	listingID := uuid.Must(uuid.NewRandom())
	store := &fakePricingStore{}

	negHours := baseAdviseInput(listingID)
	negHours.Hours = -1
	_, _, err := newTestPricing(store).Advise(context.Background(), negHours)
	require.Error(t, err)

	negCost := baseAdviseInput(listingID)
	negCost.MaterialCost = money.New(-1)
	_, _, err = newTestPricing(store).Advise(context.Background(), negCost)
	require.Error(t, err)

	require.Empty(t, store.comparableCalls, "an invalid request must not reach the store")
}
