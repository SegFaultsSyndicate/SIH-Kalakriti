// services/core-svc/internal/core/handler/pricing.go
package handler

import (
	"context"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/money"
	pricingv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/pricing/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Pricing implements pricing.v1.PricingService: the Fair Price Advisory. It
// never sets or enforces a price; every RPC here is read-only advice.
type Pricing struct {
	pricingv1.UnimplementedPricingServiceServer
	svc *service.Pricing
}

// NewPricing builds the pricing handler.
func NewPricing(svc *service.Pricing) *Pricing { return &Pricing{svc: svc} }

// GetAdvisory returns the cost floor, market band and timing signal for one
// listing, plus an anomaly check when the caller supplies a chosen price.
func (h *Pricing) GetAdvisory(
	ctx context.Context,
	req *pricingv1.GetAdvisoryRequest,
) (*pricingv1.GetAdvisoryResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	in := service.AdviseInput{
		ListingID:    listingID,
		MaterialCost: money.FromProto(req.GetMaterialCost()),
		Hours:        req.GetHours(),
	}
	if cp := req.GetChosenPrice(); cp != nil {
		m := money.FromProto(cp)
		in.ChosenPrice = &m
	}

	advisory, anomaly, err := h.svc.Advise(ctx, in)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	out := &pricingv1.GetAdvisoryResponse{
		RecommendedMin: advisory.RecommendedMin.ToProto(),
		RecommendedMax: advisory.RecommendedMax.ToProto(),
		Drivers:        driversToProto(advisory.Drivers),
	}
	if anomaly != nil {
		out.Anomaly = anomalyToProto(*anomaly)
	}
	return out, nil
}

func driversToProto(drivers []domain.Driver) []*pricingv1.Driver {
	out := make([]*pricingv1.Driver, 0, len(drivers))
	for _, d := range drivers {
		out = append(out, &pricingv1.Driver{
			Name: d.Name, Value: d.Value, ExplanationKey: d.ExplanationKey,
		})
	}
	return out
}

func anomalyToProto(a domain.Anomaly) *pricingv1.Anomaly {
	return &pricingv1.Anomaly{
		Level:          anomalyLevelToProto(a.Level),
		ShortfallPaise: a.ShortfallPaise,
		ExplanationKey: a.ExplanationKey,
	}
}

func anomalyLevelToProto(l domain.AnomalyLevel) pricingv1.AnomalyLevel {
	switch l {
	case domain.AnomalyUnderpriced:
		return pricingv1.AnomalyLevel_ANOMALY_LEVEL_UNDERPRICED
	case domain.AnomalyOverpriced:
		return pricingv1.AnomalyLevel_ANOMALY_LEVEL_OVERPRICED
	case domain.AnomalyNone:
		return pricingv1.AnomalyLevel_ANOMALY_LEVEL_NONE
	default:
		return pricingv1.AnomalyLevel_ANOMALY_LEVEL_UNSPECIFIED
	}
}
