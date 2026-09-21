// services/core-svc/internal/core/service/pricing.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/money"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// pricingLanguage is the search-projection row the advisory reads. Pricing
// signals (craft, size, materials, GI, embedding) do not vary by language, so
// any indexed row will do; English is guaranteed to exist per batch 9's
// buyer-language fan-out. Must be one of the language_code Postgres enum's
// values (full uppercase names, e.g. "ENGLISH") -- confirmed live that the
// ISO-style "en" this held before fails every Advise call outright with
// `invalid input value for enum language_code: "en"` (SQLSTATE 22P02),
// before ml-svc is ever reached.
const pricingLanguage = "ENGLISH"

// PricingStore is the read-only persistence port the pricing advisory needs.
// Nothing here writes: the advisory never touches a listing's stored price.
type PricingStore interface {
	// PricingSource loads the target listing's craft, size, materials, GI
	// status and embedding.
	PricingSource(ctx context.Context, listingID uuid.UUID, language string) (domain.PricingSource, error)
	// MinimumWage returns the state minimum wage in force on asOf.
	MinimumWage(ctx context.Context, stateCode string, asOf time.Time) (domain.WageRate, error)
	// Comparables returns the market band for a comparables filter.
	Comparables(ctx context.Context, filter domain.ComparablesFilter) (domain.MarketBand, error)
	// SeasonalityMultiplier returns the timing signal for a craft and month.
	SeasonalityMultiplier(ctx context.Context, craftID uuid.UUID, month time.Month) (domain.TimingSignal, error)
}

// Pricing computes the Fair Price Advisory. It is purely advisory: nothing
// here writes a listing's price, and every returned number carries a driver.
type Pricing struct {
	store PricingStore
	log   *slog.Logger
}

// NewPricing builds the pricing service.
func NewPricing(store PricingStore, log *slog.Logger) *Pricing {
	return &Pricing{store: store, log: log}
}

// AdviseInput is what the caller supplies alongside the listing being priced.
// The wage jurisdiction is deliberately not one of these fields: it is read
// from the listing's own artisan so a caller cannot shop for a lower floor.
type AdviseInput struct {
	ListingID    uuid.UUID
	MaterialCost money.Money
	Hours        float64
	// ChosenPrice, when set, runs the anomaly check against the resulting
	// floor and band.
	ChosenPrice *money.Money
	// AsOf pins the wage-effective-date and season lookups; the zero value
	// means "now". Tests set it explicitly so a fixture is not tied to the
	// calendar.
	AsOf time.Time
}

// Advise returns the Fair Price Advisory for one listing, and the anomaly
// check against ChosenPrice when one was supplied.
func (p *Pricing) Advise(ctx context.Context, in AdviseInput) (domain.Advisory, *domain.Anomaly, error) {
	if in.Hours < 0 {
		return domain.Advisory{}, nil, pkgdomain.InvalidInput("hours must not be negative")
	}
	if in.MaterialCost.Paise() < 0 {
		return domain.Advisory{}, nil, pkgdomain.InvalidInput("material_cost must not be negative")
	}
	asOf := in.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}

	src, err := p.store.PricingSource(ctx, in.ListingID, pricingLanguage)
	if err != nil {
		return domain.Advisory{}, nil, fmt.Errorf("loading pricing source for listing %s: %w", in.ListingID, err)
	}

	wage, err := p.store.MinimumWage(ctx, src.StateCode, asOf)
	if err != nil {
		return domain.Advisory{}, nil, fmt.Errorf("loading minimum wage for %s: %w", src.StateCode, err)
	}
	floor := domain.ComputeCostFloor(domain.CostFloorInput{MaterialCost: in.MaterialCost, Hours: in.Hours}, wage)

	band, err := p.marketBand(ctx, in.ListingID, src)
	if err != nil {
		return domain.Advisory{}, nil, fmt.Errorf("computing market band: %w", err)
	}

	timing, err := p.store.SeasonalityMultiplier(ctx, src.CraftID, asOf.Month())
	if err != nil {
		return domain.Advisory{}, nil, fmt.Errorf("loading seasonality multiplier: %w", err)
	}

	advisory := domain.BuildAdvisory(floor, wage, band, timing)

	var anomaly *domain.Anomaly
	if in.ChosenPrice != nil {
		a := domain.CheckAnomaly(*in.ChosenPrice, floor, band.P75)
		anomaly = &a
	}
	return advisory, anomaly, nil
}

// marketBand runs the strict comparables filter (same craft, ±30% size, same
// material class, GI matched), then widens to craft-only when the strict pass
// falls short of domain.MinComparables.
func (p *Pricing) marketBand(ctx context.Context, listingID uuid.UUID, src domain.PricingSource) (domain.MarketBand, error) {
	strict := domain.ComparablesFilter{
		CraftID:          src.CraftID,
		ExcludeListingID: listingID,
		Embedding:        src.Embedding,
		GICertified:      src.GICertified,
		Materials:        src.Materials,
		SizeMM:           src.SizeMM,
		Strict:           true,
	}
	band, err := p.store.Comparables(ctx, strict)
	if err != nil {
		return domain.MarketBand{}, err
	}
	if band.SampleSize >= domain.MinComparables {
		return band, nil
	}

	p.log.Info("market band widened to craft-only",
		"listing_id", listingID, "strict_sample_size", band.SampleSize)
	widened, err := p.store.Comparables(ctx, domain.ComparablesFilter{
		CraftID:          src.CraftID,
		ExcludeListingID: listingID,
		Embedding:        src.Embedding,
		Strict:           false,
	})
	if err != nil {
		return domain.MarketBand{}, err
	}
	widened.Widened = true
	return widened, nil
}
