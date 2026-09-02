// services/bff/internal/bff/client/pricing_test.go
package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	pricingv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/pricing/v1"
)

type fakePricingService struct {
	pricingv1.PricingServiceClient
	getAdvisory func(ctx context.Context, in *pricingv1.GetAdvisoryRequest, opts ...grpc.CallOption) (*pricingv1.GetAdvisoryResponse, error)
}

func (f *fakePricingService) GetAdvisory(ctx context.Context, in *pricingv1.GetAdvisoryRequest, opts ...grpc.CallOption) (*pricingv1.GetAdvisoryResponse, error) {
	return f.getAdvisory(ctx, in, opts...)
}

func TestPricingAdvisePricingSendsListingMaterialCostAndHours(t *testing.T) {
	var sawReq *pricingv1.GetAdvisoryRequest
	p := &Pricing{pricing: &fakePricingService{
		getAdvisory: func(ctx context.Context, in *pricingv1.GetAdvisoryRequest, opts ...grpc.CallOption) (*pricingv1.GetAdvisoryResponse, error) {
			sawReq = in
			return &pricingv1.GetAdvisoryResponse{
				RecommendedMin: &commonv1.Money{AmountPaise: 1000, CurrencyCode: "INR"},
				RecommendedMax: &commonv1.Money{AmountPaise: 2000, CurrencyCode: "INR"},
			}, nil
		},
	}}

	got, err := p.AdvisePricing(context.Background(), "lst-1", map[string]any{
		"material_cost": map[string]any{"amount_paise": float64(500)},
		"hours":         float64(3),
	})
	require.NoError(t, err)

	require.NotNil(t, sawReq)
	assert.Equal(t, "lst-1", sawReq.GetListingId())
	assert.Equal(t, int64(500), sawReq.GetMaterialCost().GetAmountPaise())
	assert.Equal(t, 3.0, sawReq.GetHours())
	assert.Nil(t, sawReq.ChosenPrice)

	min, ok := got["recommended_min"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, int64(1000), min["amount_paise"])
}

func TestPricingAdvisePricingRunsAnomalyCheckWhenChosenPriceGiven(t *testing.T) {
	var sawReq *pricingv1.GetAdvisoryRequest
	p := &Pricing{pricing: &fakePricingService{
		getAdvisory: func(ctx context.Context, in *pricingv1.GetAdvisoryRequest, opts ...grpc.CallOption) (*pricingv1.GetAdvisoryResponse, error) {
			sawReq = in
			return &pricingv1.GetAdvisoryResponse{
				Anomaly: &pricingv1.Anomaly{
					Level:          pricingv1.AnomalyLevel_ANOMALY_LEVEL_UNDERPRICED,
					ShortfallPaise: 200,
				},
			}, nil
		},
	}}

	got, err := p.AdvisePricing(context.Background(), "lst-1", map[string]any{
		"material_cost": map[string]any{"amount_paise": float64(500)},
		"hours":         float64(3),
		"chosen_price":  map[string]any{"amount_paise": float64(400)},
	})
	require.NoError(t, err)

	require.NotNil(t, sawReq.ChosenPrice)
	assert.Equal(t, int64(400), sawReq.ChosenPrice.GetAmountPaise())

	anomaly, ok := got["anomaly"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "UNDERPRICED", anomaly["level"])
	assert.Equal(t, int64(200), anomaly["shortfall_paise"])
}

func TestPricingAdvisePricingRejectsMissingHours(t *testing.T) {
	p := &Pricing{}
	_, err := p.AdvisePricing(context.Background(), "lst-1", map[string]any{
		"material_cost": map[string]any{"amount_paise": float64(500)},
	})
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestPricingAdvisePricingRejectsMissingMaterialCost(t *testing.T) {
	p := &Pricing{}
	_, err := p.AdvisePricing(context.Background(), "lst-1", map[string]any{
		"hours": float64(3),
	})
	assert.Equal(t, 400, domain.HTTPStatus(err))
}
