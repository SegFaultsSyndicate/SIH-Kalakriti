// services/core-svc/internal/core/service/pricing.go
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
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
	// GetListing, GetProduct and GetArtisan back unindexedSource's fallback
	// path -- reading the listing/product/artisan rows directly when
	// PricingSource has nothing to join against yet. All three are already
	// implemented on *repo.Repo for PipelineStore's sake (see pipeline.go),
	// so wiring.PricingStore's embedded *repo.Repo satisfies these for free.
	GetListing(ctx context.Context, id uuid.UUID) (domain.Listing, error)
	GetProduct(ctx context.Context, id uuid.UUID) (domain.Product, error)
	GetArtisan(ctx context.Context, id uuid.UUID) (domain.Artisan, error)
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
	unindexed := false
	if err != nil {
		// Neither case is a real failure: a listing that has never been
		// published has no listing_search row at all (ErrNotFound, the
		// common case -- pricing is offered during the artisan's own
		// creation wizard, well before publish), and a published-but-not-
		// yet-embedded one has a row with a null embedding (ErrUnavailable).
		// Either way there is no comparable-listing data to draw from yet;
		// unindexedSource below reads the craft/state/GI/materials the
		// advisory still needs straight off the listing/product/artisan
		// rows instead, and marketBand degrades to a synthetic estimate.
		if !errors.Is(err, pkgdomain.ErrNotFound) && !errors.Is(err, pkgdomain.ErrUnavailable) {
			return domain.Advisory{}, nil, fmt.Errorf("loading pricing source for listing %s: %w", in.ListingID, err)
		}
		src, err = p.unindexedSource(ctx, in.ListingID)
		if err != nil {
			return domain.Advisory{}, nil, fmt.Errorf("loading unindexed pricing source for listing %s: %w", in.ListingID, err)
		}
		unindexed = true
	}

	wage, err := p.store.MinimumWage(ctx, src.StateCode, asOf)
	if err != nil {
		return domain.Advisory{}, nil, fmt.Errorf("loading minimum wage for %s: %w", src.StateCode, err)
	}
	floor := domain.ComputeCostFloor(domain.CostFloorInput{MaterialCost: in.MaterialCost, Hours: in.Hours}, wage)

	var band domain.MarketBand
	if unindexed {
		band = syntheticMarketBand(floor)
	} else {
		band, err = p.marketBand(ctx, in.ListingID, src)
		if err != nil {
			return domain.Advisory{}, nil, fmt.Errorf("computing market band: %w", err)
		}
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

// unindexedSource builds a PricingSource straight from the listing/product/
// artisan rows, for a listing PricingSource itself could not resolve (see its
// caller in Advise). GICertified and Materials/SizeMM come from the listing's
// and product's own columns -- the same values search-svc would otherwise
// have copied into listing_search -- only the pgvector Embedding is genuinely
// unavailable pre-index, which is exactly what tells marketBand to fall back
// to syntheticMarketBand instead of a nearest-neighbour search with nothing
// to search.
func (p *Pricing) unindexedSource(ctx context.Context, listingID uuid.UUID) (domain.PricingSource, error) {
	listing, err := p.store.GetListing(ctx, listingID)
	if err != nil {
		return domain.PricingSource{}, fmt.Errorf("loading listing: %w", err)
	}
	product, err := p.store.GetProduct(ctx, listing.ProductID)
	if err != nil {
		return domain.PricingSource{}, fmt.Errorf("loading product: %w", err)
	}
	artisan, err := p.store.GetArtisan(ctx, listing.ArtisanID)
	if err != nil {
		return domain.PricingSource{}, fmt.Errorf("loading artisan: %w", err)
	}
	return domain.PricingSource{
		CraftID:     product.CraftID,
		StateCode:   artisan.Region.StateCode,
		GICertified: listing.GICertified,
		Materials:   product.Materials,
		SizeMM:      product.Dimensions.LengthMM,
	}, nil
}

// syntheticMarketBandMinPct/MaxPct bound the random markup over the cost
// floor a synthetic band's median is drawn from -- wide enough that it reads
// as a rough estimate, not a precise figure, since it is one.
const (
	syntheticMarketBandMinPct = 140
	syntheticMarketBandMaxPct = 220
)

// syntheticMarketBand estimates a market band directly from the artisan's own
// cost floor when there is no comparable-listing data to draw from at all
// (see unindexedSource): a listing still being drafted through the wizard has
// never been published, so search-svc has never projected it into
// listing_search, and neither has anything else in the same craft that also
// hasn't published yet. Recommending exactly the floor with no markup would
// be technically safe but useless as an "advisory" the artisan is meant to
// react to, so this picks a random markup in a plausible artisan-margin range
// instead of a fixed one -- every accompanying driver names the band
// Synthetic (BuildAdvisory's market_synthetic driver) so it is never mistaken
// for a real market read.
func syntheticMarketBand(floor money.Money) domain.MarketBand {
	midPct := syntheticMarketBandMinPct + rand.Intn(syntheticMarketBandMaxPct-syntheticMarketBandMinPct+1)
	return domain.MarketBand{
		P25:        floor.MulPct(midPct - 15),
		P50:        floor.MulPct(midPct),
		P75:        floor.MulPct(midPct + 25),
		SampleSize: 0,
		Synthetic:  true,
	}
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
