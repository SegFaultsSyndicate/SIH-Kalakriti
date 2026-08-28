// services/core-svc/internal/core/domain/pricing.go
package domain

import (
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/segfaultsyndicate/kalakriti/pkg/money"
)

// AnomalyLevel flags where a chosen price sits relative to the floor and band.
type AnomalyLevel string

// The three anomaly outcomes. OVERPRICED is informational only — the advisory
// never blocks a save over it.
const (
	AnomalyNone        AnomalyLevel = "NONE"
	AnomalyUnderpriced AnomalyLevel = "UNDERPRICED"
	AnomalyOverpriced  AnomalyLevel = "OVERPRICED"
)

// MinComparables is the smallest strict-filter sample the market band trusts;
// fewer than this widens the search to craft-only.
const MinComparables = 5

// sizeBandPct is how far a comparable's size may drift from the target's, in
// each direction, to still count as "same size".
const sizeBandPct = 30

// Driver is one number behind an advisory. Value is a client-formattable
// string; ExplanationKey is what the client looks up to render the sentence in
// the artisan's language. The service layer never returns explanatory prose.
type Driver struct {
	Name           string
	Value          string
	ExplanationKey string
}

// WageRate is the state minimum wage row the cost floor cites.
type WageRate struct {
	StateCode     string
	EffectiveDate time.Time
	PaisePerHour  int64
}

// CostFloorInput is what the artisan supplies for their own piece.
type CostFloorInput struct {
	MaterialCost money.Money
	Hours        float64
}

// MarketBand is the p25/p50/p75 of comparable listing prices.
type MarketBand struct {
	P25, P50, P75 money.Money
	SampleSize    int
	// Widened is true once the strict filter (same craft, size band, material
	// class, GI match) fell short of MinComparables and the search was
	// widened to craft-only.
	Widened bool
}

// TimingSignal is the seasonality multiplier and its named driver. Festival is
// empty when no seasonal row matched the craft and month, in which case
// Multiplier is 1.0.
type TimingSignal struct {
	Multiplier float64
	Festival   string
}

// PricingSource is the target listing's craft, size, materials, GI status and
// embedding, as read from the search projection and product table — the raw
// inputs the market band and timing signal are computed from.
type PricingSource struct {
	CraftID uuid.UUID
	// StateCode is the owning artisan's registered state — the wage floor is
	// keyed to it, and it never comes from the request.
	StateCode   string
	GICertified bool
	Materials   []string
	Embedding   []float32
	// SizeMM is the product's own length_mm; nil when the product never had
	// a dimension recorded.
	SizeMM *int32
}

// ComparablesFilter narrows the pgvector nearest-neighbour search for the
// market band. Strict applies the size/material/GI narrowing on top of craft;
// a non-strict pass is craft-only, the widened fallback.
type ComparablesFilter struct {
	CraftID          uuid.UUID
	ExcludeListingID uuid.UUID
	Embedding        []float32
	GICertified      bool
	Materials        []string
	// SizeMM is the target listing's own reference size (its product's
	// length_mm). Nil skips the size filter even when Strict is true.
	SizeMM *int32
	Strict bool
}

// Anomaly is the outcome of checking one chosen price against the floor and
// 1.5x the market p75.
type Anomaly struct {
	Level AnomalyLevel
	// ShortfallPaise is how far the chosen price sits below the floor; zero
	// unless Level is AnomalyUnderpriced.
	ShortfallPaise int64
	ExplanationKey string
}

// Advisory is the full Fair Price Advisory for one listing: three
// independently computed components (Floor, Band, Timing) folded into a
// recommended range, plus the driver list that explains every number in it.
type Advisory struct {
	RecommendedMin money.Money
	RecommendedMax money.Money
	Floor          money.Money
	Wage           WageRate
	Band           MarketBand
	Timing         TimingSignal
	Drivers        []Driver
}

// SizeBand returns the [min, max] millimetre band a comparable's size must
// fall inside to count as "same size" (±30%).
//
// ponytail: length_mm alone stands in for the listing's size, not a 3-D
// volume comparison. Upgrade to a volumetric (l*w*h) band if flat items and
// tall items keep matching each other as comparables.
func SizeBand(referenceMM int32) (min, max int32) {
	delta := int32(math.Round(float64(referenceMM) * sizeBandPct / 100))
	return referenceMM - delta, referenceMM + delta
}

// ComputeCostFloor is material_cost + hours * the cited wage rate. The
// advisory never recommends below this.
func ComputeCostFloor(in CostFloorInput, wage WageRate) money.Money {
	labourPaise := int64(math.Round(in.Hours * float64(wage.PaisePerHour)))
	floor, _ := in.MaterialCost.Add(money.New(labourPaise)) // both default to INR
	return floor
}

// BuildAdvisory folds the three independently computed components into a
// recommended range: the timing multiplier scales the market band, and the
// result is clamped up to the cost floor so recommended_min never drops
// below it. This is a fixed formula — it never optimises for margin or
// demand, only combines numbers the artisan and the market already gave it.
func BuildAdvisory(floor money.Money, wage WageRate, band MarketBand, timing TimingSignal) Advisory {
	adjustedP25 := scale(band.P25, timing.Multiplier)
	adjustedP75 := scale(band.P75, timing.Multiplier)

	recMin := floor
	if adjustedP25.Paise() > recMin.Paise() {
		recMin = adjustedP25
	}
	recMax := recMin
	if adjustedP75.Paise() > recMax.Paise() {
		recMax = adjustedP75
	}

	drivers := []Driver{
		{Name: "cost_floor", Value: paiseStr(floor), ExplanationKey: "pricing.driver.cost_floor"},
		{Name: "wage_rate", Value: paiseStr(money.New(wage.PaisePerHour)), ExplanationKey: "pricing.driver.wage_rate"},
		{Name: "wage_state", Value: wage.StateCode, ExplanationKey: "pricing.driver.wage_state"},
		{Name: "market_p25", Value: paiseStr(band.P25), ExplanationKey: "pricing.driver.market_p25"},
		{Name: "market_p50", Value: paiseStr(band.P50), ExplanationKey: "pricing.driver.market_p50"},
		{Name: "market_p75", Value: paiseStr(band.P75), ExplanationKey: "pricing.driver.market_p75"},
		{Name: "market_sample_size", Value: intStr(band.SampleSize), ExplanationKey: "pricing.driver.market_sample_size"},
		{Name: "timing_multiplier", Value: floatStr(timing.Multiplier), ExplanationKey: "pricing.driver.timing_multiplier"},
		{Name: "recommended_min", Value: paiseStr(recMin), ExplanationKey: "pricing.driver.recommended_min"},
		{Name: "recommended_max", Value: paiseStr(recMax), ExplanationKey: "pricing.driver.recommended_max"},
	}
	if band.Widened {
		drivers = append(drivers, Driver{
			Name: "market_widened", Value: "craft_only",
			ExplanationKey: "pricing.driver.market_widened",
		})
	}
	if timing.Festival != "" {
		drivers = append(drivers, Driver{
			Name: "timing_festival", Value: timing.Festival,
			ExplanationKey: "pricing.driver.timing_festival",
		})
	}

	return Advisory{
		RecommendedMin: recMin,
		RecommendedMax: recMax,
		Floor:          floor,
		Wage:           wage,
		Band:           band,
		Timing:         timing,
		Drivers:        drivers,
	}
}

// CheckAnomaly flags a chosen price against the floor and 1.5x the market
// p75. Only strictly below the floor or strictly above the ceiling counts —
// a price exactly at either boundary is not flagged.
func CheckAnomaly(chosen, floor, p75 money.Money) Anomaly {
	if chosen.Paise() < floor.Paise() {
		shortfall, _ := floor.Sub(chosen)
		return Anomaly{Level: AnomalyUnderpriced, ShortfallPaise: shortfall.Paise(), ExplanationKey: "pricing.anomaly.underpriced"}
	}
	if ceiling := p75.MulPct(150); chosen.Paise() > ceiling.Paise() {
		return Anomaly{Level: AnomalyOverpriced, ExplanationKey: "pricing.anomaly.overpriced"}
	}
	return Anomaly{Level: AnomalyNone, ExplanationKey: "pricing.anomaly.none"}
}

func scale(m money.Money, ratio float64) money.Money {
	return money.New(int64(math.Round(float64(m.Paise()) * ratio)))
}

func paiseStr(m money.Money) string { return strconv.FormatInt(m.Paise(), 10) }
func intStr(n int) string           { return strconv.FormatInt(int64(n), 10) }
func floatStr(f float64) string     { return strconv.FormatFloat(f, 'f', 4, 64) }
