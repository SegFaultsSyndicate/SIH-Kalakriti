// services/bff/internal/bff/client/order_test.go
package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	fulfilmentv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/fulfilment/v1"
)

type fakeFulfilmentService struct {
	fulfilmentv1.FulfilmentServiceClient
	createBulkOrder func(ctx context.Context, in *fulfilmentv1.CreateBulkOrderRequest, opts ...grpc.CallOption) (*fulfilmentv1.CreateBulkOrderResponse, error)
	getOrder        func(ctx context.Context, in *fulfilmentv1.GetOrderRequest, opts ...grpc.CallOption) (*fulfilmentv1.GetOrderResponse, error)
	respondToLot    func(ctx context.Context, in *fulfilmentv1.RespondToLotRequest, opts ...grpc.CallOption) (*fulfilmentv1.RespondToLotResponse, error)
	watchOrder      func(ctx context.Context, in *fulfilmentv1.WatchOrderRequest, opts ...grpc.CallOption) (fulfilmentv1.FulfilmentService_WatchOrderClient, error)
}

func (f *fakeFulfilmentService) GetOrder(ctx context.Context, in *fulfilmentv1.GetOrderRequest, opts ...grpc.CallOption) (*fulfilmentv1.GetOrderResponse, error) {
	return f.getOrder(ctx, in, opts...)
}

func (f *fakeFulfilmentService) CreateBulkOrder(ctx context.Context, in *fulfilmentv1.CreateBulkOrderRequest, opts ...grpc.CallOption) (*fulfilmentv1.CreateBulkOrderResponse, error) {
	return f.createBulkOrder(ctx, in, opts...)
}

func (f *fakeFulfilmentService) RespondToLot(ctx context.Context, in *fulfilmentv1.RespondToLotRequest, opts ...grpc.CallOption) (*fulfilmentv1.RespondToLotResponse, error) {
	return f.respondToLot(ctx, in, opts...)
}

func (f *fakeFulfilmentService) WatchOrder(ctx context.Context, in *fulfilmentv1.WatchOrderRequest, opts ...grpc.CallOption) (fulfilmentv1.FulfilmentService_WatchOrderClient, error) {
	return f.watchOrder(ctx, in, opts...)
}

// fakeWatchOrderStream implements FulfilmentService_WatchOrderClient by
// draining a fixed slice of responses, then returning io.EOF-equivalent.
type fakeWatchOrderStream struct {
	grpc.ClientStream
	responses []*fulfilmentv1.WatchOrderResponse
	i         int
}

func (s *fakeWatchOrderStream) Recv() (*fulfilmentv1.WatchOrderResponse, error) {
	if s.i >= len(s.responses) {
		return nil, errors.New("stream closed")
	}
	resp := s.responses[s.i]
	s.i++
	return resp, nil
}

func TestOrderCreateBulkOrderSendsListingQuantityAndRequiredBy(t *testing.T) {
	var sawReq *fulfilmentv1.CreateBulkOrderRequest
	o := &Order{fulfilment: &fakeFulfilmentService{
		createBulkOrder: func(ctx context.Context, in *fulfilmentv1.CreateBulkOrderRequest, opts ...grpc.CallOption) (*fulfilmentv1.CreateBulkOrderResponse, error) {
			sawReq = in
			return &fulfilmentv1.CreateBulkOrderResponse{Order: &fulfilmentv1.BulkOrder{Id: "order-1"}}, nil
		},
	}}

	id, err := o.CreateBulkOrder(context.Background(), "buyer-1", "idem-1", map[string]any{
		"listing_id":  "lst-1",
		"quantity":    float64(50),
		"required_by": "2025-01-31T00:00:00Z",
		"customisations": map[string]any{
			"border_colour": "red",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "order-1", id)

	require.NotNil(t, sawReq)
	assert.Equal(t, "buyer-1", sawReq.GetBuyerId())
	assert.Equal(t, "lst-1", sawReq.GetListingId())
	assert.Equal(t, int32(50), sawReq.GetQuantity())
	assert.Equal(t, "red", sawReq.GetCustomisations()["border_colour"])
	assert.Equal(t, "idem-1", sawReq.GetIdempotencyKey())
	require.NotNil(t, sawReq.GetRequiredBy())
	assert.Equal(t, 2025, sawReq.GetRequiredBy().AsTime().Year())
}

func TestOrderCreateBulkOrderRejectsMissingListingID(t *testing.T) {
	o := &Order{}
	_, err := o.CreateBulkOrder(context.Background(), "buyer-1", "idem-1", map[string]any{
		"quantity":    float64(50),
		"required_by": "2025-01-31T00:00:00Z",
	})
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestOrderCreateBulkOrderRejectsUnparseableRequiredBy(t *testing.T) {
	o := &Order{}
	_, err := o.CreateBulkOrder(context.Background(), "buyer-1", "idem-1", map[string]any{
		"listing_id":  "lst-1",
		"quantity":    float64(50),
		"required_by": "not-a-date",
	})
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestOrderRespondToLotAcceptSendsPromisedShipDate(t *testing.T) {
	var sawReq *fulfilmentv1.RespondToLotRequest
	o := &Order{fulfilment: &fakeFulfilmentService{
		respondToLot: func(ctx context.Context, in *fulfilmentv1.RespondToLotRequest, opts ...grpc.CallOption) (*fulfilmentv1.RespondToLotResponse, error) {
			sawReq = in
			return &fulfilmentv1.RespondToLotResponse{Lot: &fulfilmentv1.OrderLot{Id: "lot-1"}}, nil
		},
	}}

	err := o.RespondToLot(context.Background(), "lot-1", "art-1", "idem-2", true, map[string]any{
		"promised_ship_date": "2025-02-01T00:00:00Z",
	})
	require.NoError(t, err)
	require.NotNil(t, sawReq)
	assert.True(t, sawReq.GetAccept())
	require.NotNil(t, sawReq.PromisedShipDate)
	assert.Nil(t, sawReq.DeclineReason)
}

func TestOrderRespondToLotDeclineSendsReason(t *testing.T) {
	var sawReq *fulfilmentv1.RespondToLotRequest
	o := &Order{fulfilment: &fakeFulfilmentService{
		respondToLot: func(ctx context.Context, in *fulfilmentv1.RespondToLotRequest, opts ...grpc.CallOption) (*fulfilmentv1.RespondToLotResponse, error) {
			sawReq = in
			return &fulfilmentv1.RespondToLotResponse{Lot: &fulfilmentv1.OrderLot{Id: "lot-1"}}, nil
		},
	}}

	err := o.RespondToLot(context.Background(), "lot-1", "art-1", "idem-2", false, map[string]any{
		"decline_reason": "capacity full this month",
	})
	require.NoError(t, err)
	require.NotNil(t, sawReq)
	assert.False(t, sawReq.GetAccept())
	require.NotNil(t, sawReq.DeclineReason)
	assert.Equal(t, "capacity full this month", *sawReq.DeclineReason)
	assert.Nil(t, sawReq.PromisedShipDate)
}

func TestOrderRespondToLotRejectsAcceptWithNoShipDate(t *testing.T) {
	o := &Order{}
	err := o.RespondToLot(context.Background(), "lot-1", "art-1", "idem-2", true, map[string]any{})
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestOrderRespondToLotRejectsDeclineWithNoReason(t *testing.T) {
	o := &Order{}
	err := o.RespondToLot(context.Background(), "lot-1", "art-1", "idem-2", false, map[string]any{})
	assert.Equal(t, 400, domain.HTTPStatus(err))
}

func TestOrderGetOrderReturnsOrderWithLots(t *testing.T) {
	var sawReq *fulfilmentv1.GetOrderRequest
	unitPrice := &commonv1.Money{AmountPaise: 5000, CurrencyCode: "INR"}
	o := &Order{fulfilment: &fakeFulfilmentService{
		getOrder: func(ctx context.Context, in *fulfilmentv1.GetOrderRequest, opts ...grpc.CallOption) (*fulfilmentv1.GetOrderResponse, error) {
			sawReq = in
			return &fulfilmentv1.GetOrderResponse{Order: &fulfilmentv1.BulkOrder{
				Id:       "order-1",
				BuyerId:  "buyer-1",
				Quantity: 50,
				State:    fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_CONFIRMED,
				UnitPrice: unitPrice,
				Lots:     []*fulfilmentv1.OrderLot{{Id: "lot-1", ArtisanId: "art-1"}},
			}}, nil
		},
	}}

	order, err := o.GetOrder(context.Background(), "order-1")
	require.NoError(t, err)
	require.NotNil(t, sawReq)
	assert.Equal(t, "order-1", sawReq.GetBulkOrderId())

	assert.Equal(t, "order-1", order["id"])
	assert.Equal(t, "buyer-1", order["buyer_id"])
	assert.Equal(t, "CONFIRMED", order["state"])
	lots, ok := order["lots"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, lots, 1)
	assert.Equal(t, "lot-1", lots[0]["id"])
}

func TestOrderGetOrderPropagatesGRPCError(t *testing.T) {
	o := &Order{fulfilment: &fakeFulfilmentService{
		getOrder: func(ctx context.Context, in *fulfilmentv1.GetOrderRequest, opts ...grpc.CallOption) (*fulfilmentv1.GetOrderResponse, error) {
			return nil, errors.New("not found")
		},
	}}
	_, err := o.GetOrder(context.Background(), "order-1")
	assert.Error(t, err)
}

func TestOrderWatchOrderStreamsMappedEvents(t *testing.T) {
	stream := &fakeWatchOrderStream{responses: []*fulfilmentv1.WatchOrderResponse{
		{Event: &fulfilmentv1.OrderEvent{
			EventId: "evt-1", BulkOrderId: "order-1",
			Payload: &fulfilmentv1.OrderEvent_LotOffered{LotOffered: &fulfilmentv1.OrderLot{Id: "lot-1", ArtisanId: "art-1"}},
		}},
		{Event: &fulfilmentv1.OrderEvent{
			EventId: "evt-2", BulkOrderId: "order-1",
			Payload: &fulfilmentv1.OrderEvent_OrderStateChanged{OrderStateChanged: fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_ALLOCATING},
		}},
	}}
	o := &Order{fulfilment: &fakeFulfilmentService{
		watchOrder: func(ctx context.Context, in *fulfilmentv1.WatchOrderRequest, opts ...grpc.CallOption) (fulfilmentv1.FulfilmentService_WatchOrderClient, error) {
			require.Equal(t, "order-1", in.GetBulkOrderId())
			return stream, nil
		},
	}}

	events, err := o.WatchOrder(context.Background(), "order-1")
	require.NoError(t, err)

	select {
	case e := <-events:
		assert.Equal(t, "lot_offered", e["type"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first event")
	}

	select {
	case e := <-events:
		assert.Equal(t, "order_state_changed", e["type"])
		assert.Equal(t, "ALLOCATING", e["state"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second event")
	}

	select {
	case _, ok := <-events:
		assert.False(t, ok, "channel should close once the fake stream is drained")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}
