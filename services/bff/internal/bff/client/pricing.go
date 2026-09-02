// services/bff/internal/bff/client/pricing.go
package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	pricingv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/pricing/v1"
)

// Pricing is bff's view of core-svc's Fair Price Advisory RPC. PricingService
// is registered on the same grpc.Server as CatalogService/IdentityService, so
// this shares the core-svc connection like every other client here.
type Pricing struct {
	pricing pricingv1.PricingServiceClient
}

// NewPricing builds the pricing client, sharing conn with core-svc's other services.
func NewPricing(conn grpc.ClientConnInterface) *Pricing {
	return &Pricing{pricing: pricingv1.NewPricingServiceClient(conn)}
}

// AdvisePricing returns the cost floor, market band and timing signal for one
// listing. inputs carries material_cost ({amount_paise, currency_code?},
// required), hours (number, required) and the optional chosen_price
// ({amount_paise, currency_code?}) — supplying it also runs the anomaly check.
func (p *Pricing) AdvisePricing(ctx context.Context, listingID string, inputs map[string]any) (map[string]any, error) {
	materialCost, err := moneyFromAny(inputs["material_cost"])
	if err != nil {
		return nil, err
	}
	hours, ok := inputs["hours"].(float64)
	if !ok {
		return nil, domain.InvalidInput("hours: is required")
	}

	req := &pricingv1.GetAdvisoryRequest{
		ListingId:    listingID,
		MaterialCost: materialCost,
		Hours:        hours,
	}
	if raw, ok := inputs["chosen_price"]; ok {
		chosenPrice, err := moneyFromAny(raw)
		if err != nil {
			return nil, err
		}
		req.ChosenPrice = chosenPrice
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := p.pricing.GetAdvisory(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	out := map[string]any{
		"recommended_min": moneyMap(resp.GetRecommendedMin()),
		"recommended_max": moneyMap(resp.GetRecommendedMax()),
		"drivers":         driversToAny(resp.GetDrivers()),
	}
	if a := resp.GetAnomaly(); a != nil {
		out["anomaly"] = map[string]any{
			"level":           trimEnumPrefix(a.GetLevel().String(), "ANOMALY_LEVEL_"),
			"shortfall_paise": a.GetShortfallPaise(),
			"explanation_key": a.GetExplanationKey(),
		}
	}
	return out, nil
}

func driversToAny(drivers []*pricingv1.Driver) []map[string]any {
	out := make([]map[string]any, 0, len(drivers))
	for _, d := range drivers {
		out = append(out, map[string]any{
			"name":            d.GetName(),
			"value":           d.GetValue(),
			"explanation_key": d.GetExplanationKey(),
		})
	}
	return out
}
