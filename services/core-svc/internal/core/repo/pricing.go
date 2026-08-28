// services/core-svc/internal/core/repo/pricing.go
package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/money"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// candidatesPerLeg bounds how many nearest-embedding rows the market band
// draws from before taking percentiles over them.
const candidatesPerLeg = 50

// PricingSource loads the target listing's craft, size, materials, GI status
// and embedding from the search projection joined to its product.
func (r *Repo) PricingSource(ctx context.Context, listingID uuid.UUID, language string) (domain.PricingSource, error) {
	row, err := r.q.GetListingPricingSource(ctx, db.GetListingPricingSourceParams{
		ListingID: listingID,
		Language:  db.LanguageCode(language),
	})
	if err != nil {
		return domain.PricingSource{}, translate(err, "listing pricing source")
	}
	if row.Embedding == nil {
		return domain.PricingSource{}, pkgdomain.Unavailable("listing has not been indexed for search yet")
	}
	return domain.PricingSource{
		CraftID:     row.CraftID,
		StateCode:   row.StateCode,
		GICertified: row.GiCertified,
		Materials:   row.Materials,
		Embedding:   row.Embedding.Slice(),
		SizeMM:      row.LengthMm,
	}, nil
}

// MinimumWage returns the state minimum wage row in force on asOf: the latest
// effective_date at or before it.
func (r *Repo) MinimumWage(ctx context.Context, stateCode string, asOf time.Time) (domain.WageRate, error) {
	row, err := r.q.GetMinimumWage(ctx, db.GetMinimumWageParams{
		StateCode: stateCode,
		AsOf:      asOf,
	})
	if err != nil {
		return domain.WageRate{}, translate(err, "minimum wage for "+stateCode)
	}
	return domain.WageRate{
		StateCode:     row.StateCode,
		EffectiveDate: row.EffectiveDate,
		PaisePerHour:  row.WagePaisePerHour,
	}, nil
}

// SeasonalityMultiplier picks the timing signal for a craft and month: a
// craft-specific row beats a crafts-wide one for the same month, and among
// ties the higher multiplier wins (concurrent festivals compound demand, they
// do not cancel it out). No match at all is 1.0 with no named driver.
func (r *Repo) SeasonalityMultiplier(ctx context.Context, craftID uuid.UUID, month time.Month) (domain.TimingSignal, error) {
	rows, err := r.q.ListSeasonalityMultipliers(ctx, db.ListSeasonalityMultipliersParams{
		CraftID: craftID,
		Month:   int16(month),
	})
	if err != nil {
		return domain.TimingSignal{}, translate(err, "seasonality multiplier")
	}

	best := domain.TimingSignal{Multiplier: 1.0}
	haveCraftSpecific := false
	for _, row := range rows {
		isCraftSpecific := row.CraftID != nil
		switch {
		case isCraftSpecific && !haveCraftSpecific:
			haveCraftSpecific = true
			best = domain.TimingSignal{Multiplier: float64(row.Multiplier), Festival: row.Festival}
		case isCraftSpecific == haveCraftSpecific && float64(row.Multiplier) > best.Multiplier:
			best = domain.TimingSignal{Multiplier: float64(row.Multiplier), Festival: row.Festival}
		}
	}
	return best, nil
}

// Comparables returns the market band for one pgvector nearest-neighbour
// pass. filter.Strict applies the size/material-class/GI narrowing on top of
// craft; a non-strict filter is craft-only, the caller's widened fallback.
func (r *Repo) Comparables(ctx context.Context, filter domain.ComparablesFilter) (domain.MarketBand, error) {
	params := db.PricingComparableStatsParams{
		CraftID:          filter.CraftID,
		ExcludeListingID: filter.ExcludeListingID,
		Embedding:        pgvector.NewVector(filter.Embedding),
		CandidateLimit:   candidatesPerLeg,
		RequireGi:        filter.Strict,
		GiCertified:      filter.GICertified,
		RequireMaterials: filter.Strict && len(filter.Materials) > 0,
		Materials:        orEmpty(filter.Materials),
		RequireSize:      filter.Strict && filter.SizeMM != nil,
	}
	if params.RequireSize {
		min, max := domain.SizeBand(*filter.SizeMM)
		params.SizeMinMm, params.SizeMaxMm = min, max
	}

	row, err := r.q.PricingComparableStats(ctx, params)
	if err != nil {
		return domain.MarketBand{}, translate(err, "comparable listings")
	}
	return domain.MarketBand{
		P25:        money.New(row.P25Paise),
		P50:        money.New(row.P50Paise),
		P75:        money.New(row.P75Paise),
		SampleSize: int(row.SampleSize),
	}, nil
}
