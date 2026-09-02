// services/bff/internal/bff/client/listing_test.go
package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
)

func validListingFields() map[string]any {
	return map[string]any{
		"craft_id":           "craft-1",
		"working_title":      "Blue dhurrie",
		"type":               "READY_STOCK",
		"price":              map[string]any{"amount_paise": float64(250000)},
		"stock_quantity":     float64(5),
		"min_order_quantity": float64(1),
		"translations": []any{
			map[string]any{"language": "ENGLISH", "title": "Blue dhurrie", "description": "Hand-woven"},
		},
	}
}

func TestListingCreateListingChainsCreateProductThenUpsertListingWithSuffixedIdempotencyKeys(t *testing.T) {
	var sawProductReq *catalogv1.CreateProductRequest
	var sawListingReq *catalogv1.UpsertListingRequest
	l := &Listing{catalog: &fakeCatalogService{
		createProduct: func(ctx context.Context, in *catalogv1.CreateProductRequest, opts ...grpc.CallOption) (*catalogv1.CreateProductResponse, error) {
			sawProductReq = in
			return &catalogv1.CreateProductResponse{Product: &catalogv1.Product{Id: "prod-1"}}, nil
		},
		upsertListing: func(ctx context.Context, in *catalogv1.UpsertListingRequest, opts ...grpc.CallOption) (*catalogv1.UpsertListingResponse, error) {
			sawListingReq = in
			return &catalogv1.UpsertListingResponse{Listing: &catalogv1.Listing{Id: "lst-1"}}, nil
		},
	}}

	id, err := l.CreateListing(context.Background(), "art-1", "idem-1", validListingFields())
	require.NoError(t, err)
	assert.Equal(t, "lst-1", id)

	require.NotNil(t, sawProductReq)
	assert.Equal(t, "art-1", sawProductReq.GetArtisanId())
	assert.Equal(t, "craft-1", sawProductReq.GetCraftId())
	assert.Equal(t, "idem-1:product", sawProductReq.GetIdempotencyKey())

	require.NotNil(t, sawListingReq)
	assert.Equal(t, "prod-1", sawListingReq.GetProductId())
	assert.Equal(t, catalogv1.ListingType_LISTING_TYPE_READY_STOCK, sawListingReq.GetType())
	assert.Equal(t, int64(250000), sawListingReq.GetPrice().GetAmountPaise())
	assert.Equal(t, "INR", sawListingReq.GetPrice().GetCurrencyCode())
	assert.Equal(t, int32(5), sawListingReq.GetStockQuantity())
	assert.Equal(t, "idem-1:listing", sawListingReq.GetIdempotencyKey())
}

func TestListingCreateListingRejectsMissingCraftID(t *testing.T) {
	l := &Listing{}
	fields := validListingFields()
	delete(fields, "craft_id")
	_, err := l.CreateListing(context.Background(), "art-1", "idem-1", fields)
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestListingUpdateListingBackfillsFieldsNotInUpdatesFromTheExistingListing(t *testing.T) {
	existing := &catalogv1.Listing{
		Id:               "lst-1",
		ProductId:        "prod-1",
		Type:             catalogv1.ListingType_LISTING_TYPE_READY_STOCK,
		Price:            &commonv1.Money{AmountPaise: 100000, CurrencyCode: "INR"},
		MinOrderQuantity: 2,
		Packaging:        &catalogv1.PackagingMeta{Fragile: true},
		Translations: []*catalogv1.ListingTranslation{
			{Language: commonv1.Language_LANGUAGE_ENGLISH, Title: "Old title"},
		},
	}

	var sawReq *catalogv1.UpsertListingRequest
	l := &Listing{catalog: &fakeCatalogService{
		getListing: func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
			return &catalogv1.GetListingResponse{Listing: existing}, nil
		},
		upsertListing: func(ctx context.Context, in *catalogv1.UpsertListingRequest, opts ...grpc.CallOption) (*catalogv1.UpsertListingResponse, error) {
			sawReq = in
			return &catalogv1.UpsertListingResponse{Listing: existing}, nil
		},
	}}

	// Only touches stock_quantity; everything else must come back from `existing`.
	err := l.UpdateListing(context.Background(), "lst-1", "idem-2", map[string]any{
		"stock_quantity": float64(9),
	})
	require.NoError(t, err)

	require.NotNil(t, sawReq)
	assert.Equal(t, "prod-1", sawReq.GetProductId())
	require.NotNil(t, sawReq.ListingId)
	assert.Equal(t, "lst-1", *sawReq.ListingId)
	assert.Equal(t, catalogv1.ListingType_LISTING_TYPE_READY_STOCK, sawReq.GetType())
	assert.Equal(t, int64(100000), sawReq.GetPrice().GetAmountPaise())
	assert.Equal(t, int32(2), sawReq.GetMinOrderQuantity())
	assert.True(t, sawReq.GetPackaging().GetFragile())
	require.Len(t, sawReq.GetTranslations(), 1)
	assert.Equal(t, "Old title", sawReq.GetTranslations()[0].GetTitle())
	require.NotNil(t, sawReq.StockQuantity)
	assert.Equal(t, int32(9), *sawReq.StockQuantity)
}

func TestListingApproveListingSendsEditedTranslations(t *testing.T) {
	var sawReq *catalogv1.ApproveListingRequest
	l := &Listing{catalog: &fakeCatalogService{
		approveListing: func(ctx context.Context, in *catalogv1.ApproveListingRequest, opts ...grpc.CallOption) (*catalogv1.ApproveListingResponse, error) {
			sawReq = in
			return &catalogv1.ApproveListingResponse{Listing: &catalogv1.Listing{Id: "lst-1"}}, nil
		},
	}}

	err := l.ApproveListing(context.Background(), "lst-1", "art-1", "idem-3", []map[string]any{
		{"language": "ENGLISH", "title": "Edited title", "description": "Edited body"},
	})
	require.NoError(t, err)

	require.NotNil(t, sawReq)
	assert.Equal(t, "lst-1", sawReq.GetListingId())
	assert.Equal(t, "art-1", sawReq.GetArtisanId())
	assert.Equal(t, "idem-3", sawReq.GetIdempotencyKey())
	require.Len(t, sawReq.GetEditedTranslations(), 1)
	assert.Equal(t, "Edited title", sawReq.GetEditedTranslations()[0].GetTitle())
}

func TestListingApproveListingWithNoEditsSendsNoTranslations(t *testing.T) {
	var sawReq *catalogv1.ApproveListingRequest
	l := &Listing{catalog: &fakeCatalogService{
		approveListing: func(ctx context.Context, in *catalogv1.ApproveListingRequest, opts ...grpc.CallOption) (*catalogv1.ApproveListingResponse, error) {
			sawReq = in
			return &catalogv1.ApproveListingResponse{Listing: &catalogv1.Listing{Id: "lst-1"}}, nil
		},
	}}

	err := l.ApproveListing(context.Background(), "lst-1", "art-1", "idem-3", nil)
	require.NoError(t, err)
	require.NotNil(t, sawReq)
	assert.Empty(t, sawReq.GetEditedTranslations())
}

func TestListingGetListingMapsToAMap(t *testing.T) {
	stock := int32(3)
	l := &Listing{catalog: &fakeCatalogService{
		getListing: func(ctx context.Context, in *catalogv1.GetListingRequest, opts ...grpc.CallOption) (*catalogv1.GetListingResponse, error) {
			return &catalogv1.GetListingResponse{Listing: &catalogv1.Listing{
				Id: "lst-1", ProductId: "prod-1", ArtisanId: "art-1",
				Type:          catalogv1.ListingType_LISTING_TYPE_READY_STOCK,
				State:         catalogv1.ListingState_LISTING_STATE_PUBLISHED,
				Price:         &commonv1.Money{AmountPaise: 100000, CurrencyCode: "INR"},
				StockQuantity: &stock,
			}}, nil
		},
	}}

	got, err := l.GetListing(context.Background(), "lst-1")
	require.NoError(t, err)
	assert.Equal(t, "lst-1", got["id"])
	assert.Equal(t, "READY_STOCK", got["type"])
	assert.Equal(t, "PUBLISHED", got["state"])
	assert.Equal(t, int32(3), got["stock_quantity"])
}

func TestListingListListingsAppliesFilters(t *testing.T) {
	var sawReq *catalogv1.ListListingsRequest
	l := &Listing{catalog: &fakeCatalogService{
		listListings: func(ctx context.Context, in *catalogv1.ListListingsRequest, opts ...grpc.CallOption) (*catalogv1.ListListingsResponse, error) {
			sawReq = in
			return &catalogv1.ListListingsResponse{}, nil
		},
	}}

	_, err := l.ListListings(context.Background(), map[string]any{"craft_id": "craft-1", "state": "PUBLISHED"})
	require.NoError(t, err)
	require.NotNil(t, sawReq.CraftId)
	assert.Equal(t, "craft-1", *sawReq.CraftId)
	assert.Equal(t, catalogv1.ListingState_LISTING_STATE_PUBLISHED, sawReq.GetState())
}

func TestMoneyFromAnyDefaultsCurrencyToINR(t *testing.T) {
	m, err := moneyFromAny(map[string]any{"amount_paise": float64(500)})
	require.NoError(t, err)
	assert.Equal(t, int64(500), m.GetAmountPaise())
	assert.Equal(t, "INR", m.GetCurrencyCode())
}

func TestMoneyFromAnyRejectsMissingAmount(t *testing.T) {
	_, err := moneyFromAny(map[string]any{})
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestTranslationsFromAnyRejectsMissingTitle(t *testing.T) {
	_, err := translationsFromAny([]any{
		map[string]any{"language": "ENGLISH"},
	})
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestTranslationsFromAnyCapsLength(t *testing.T) {
	list := make([]any, maxTranslations+1)
	for i := range list {
		list[i] = map[string]any{"language": "ENGLISH", "title": "t"}
	}
	_, err := translationsFromAny(list)
	assert.Equal(t, 400, domain.HTTPStatus(err))
}
