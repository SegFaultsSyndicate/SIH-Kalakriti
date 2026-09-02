// services/bff/internal/bff/client/order.go
package client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	fulfilmentv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/fulfilment/v1"
)

// Order is bff's view of fulfilment-svc's bulk order lifecycle.
type Order struct {
	fulfilment fulfilmentv1.FulfilmentServiceClient
}

// NewOrder builds the order client.
func NewOrder(conn grpc.ClientConnInterface) *Order {
	return &Order{fulfilment: fulfilmentv1.NewFulfilmentServiceClient(conn)}
}

// CreateBulkOrder registers a buyer's requirement. fields carries listing_id
// (required), quantity (required), required_by (RFC3339, required),
// customisations (optional object of strings) and notes (optional string).
// Splitting the order into per-artisan lots is a separate, asynchronous step
// (ProposeAllocation) — there's no lots input here.
func (o *Order) CreateBulkOrder(ctx context.Context, buyerID, idempotencyKey string, fields map[string]any) (string, error) {
	listingID, _ := fields["listing_id"].(string)
	if listingID == "" {
		return "", domain.InvalidInput("listing_id: is required")
	}
	quantity, ok := fields["quantity"].(float64)
	if !ok {
		return "", domain.InvalidInput("quantity: is required")
	}
	requiredByStr, _ := fields["required_by"].(string)
	requiredBy, err := time.Parse(time.RFC3339, requiredByStr)
	if err != nil {
		return "", domain.InvalidInput("required_by: must be RFC3339")
	}
	customisations, err := stringMap(fields["customisations"])
	if err != nil {
		return "", err
	}

	req := &fulfilmentv1.CreateBulkOrderRequest{
		BuyerId:        buyerID,
		ListingId:      listingID,
		Quantity:       int32(quantity),
		RequiredBy:     timestamppb.New(requiredBy),
		Customisations: customisations,
		IdempotencyKey: idempotencyKey,
	}
	if v, ok := fields["notes"].(string); ok {
		req.Notes = &v
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := o.fulfilment.CreateBulkOrder(ctx, req)
	if err != nil {
		return "", grpcErr(err)
	}
	return resp.GetOrder().GetId(), nil
}

// GetOrder is not wired: fulfilment-svc has no RPC that fetches one bulk
// order by id — only WatchOrder's event stream and CreateBulkOrder's own
// response return a BulkOrder. Needs a new RPC (and a service-layer method;
// only the store layer has GetBulkOrder today), not a guessed adapter.
func (o *Order) GetOrder(ctx context.Context, orderID string) (map[string]any, error) {
	return nil, domain.Unavailable("fetching a bulk order by id is not supported: fulfilment-svc has no such RPC yet")
}

// RespondToLot carries an artisan's accept or decline. fields carries
// promised_ship_date (RFC3339, required when accept) or decline_reason
// (required when declining).
func (o *Order) RespondToLot(ctx context.Context, lotID, artisanID, idempotencyKey string, accept bool, fields map[string]any) error {
	req := &fulfilmentv1.RespondToLotRequest{
		LotId:          lotID,
		ArtisanId:      artisanID,
		Accept:         accept,
		IdempotencyKey: idempotencyKey,
	}
	if accept {
		shipDateStr, _ := fields["promised_ship_date"].(string)
		shipDate, err := time.Parse(time.RFC3339, shipDateStr)
		if err != nil {
			return domain.InvalidInput("promised_ship_date: must be RFC3339, required when accepting")
		}
		req.PromisedShipDate = timestamppb.New(shipDate)
	} else {
		reason, _ := fields["decline_reason"].(string)
		if reason == "" {
			return domain.InvalidInput("decline_reason: is required when declining")
		}
		req.DeclineReason = &reason
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	if _, err := o.fulfilment.RespondToLot(ctx, req); err != nil {
		return grpcErr(err)
	}
	return nil
}

// WatchOrder streams one order's events until the caller disconnects
// (ctx.Done()) or the backend closes the stream. Uses withAuth, not
// withTimeout: a stream is meant to outlive callTimeout's 10 seconds.
func (o *Order) WatchOrder(ctx context.Context, orderID string) (<-chan map[string]any, error) {
	ctx = withAuth(ctx)
	stream, err := o.fulfilment.WatchOrder(ctx, &fulfilmentv1.WatchOrderRequest{BulkOrderId: orderID})
	if err != nil {
		return nil, grpcErr(err)
	}

	events := make(chan map[string]any)
	go func() {
		defer close(events)
		for {
			resp, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case events <- orderEventToMap(resp.GetEvent()):
			case <-ctx.Done():
				return
			}
		}
	}()
	return events, nil
}

func orderEventToMap(e *fulfilmentv1.OrderEvent) map[string]any {
	out := map[string]any{
		"event_id":      e.GetEventId(),
		"bulk_order_id": e.GetBulkOrderId(),
	}
	switch p := e.GetPayload().(type) {
	case *fulfilmentv1.OrderEvent_LotOffered:
		out["type"], out["lot"] = "lot_offered", orderLotToMap(p.LotOffered)
	case *fulfilmentv1.OrderEvent_LotAccepted:
		out["type"], out["lot"] = "lot_accepted", orderLotToMap(p.LotAccepted)
	case *fulfilmentv1.OrderEvent_LotDeclined:
		out["type"], out["lot"] = "lot_declined", orderLotToMap(p.LotDeclined)
	case *fulfilmentv1.OrderEvent_LotExpired:
		out["type"], out["lot"] = "lot_expired", orderLotToMap(p.LotExpired)
	case *fulfilmentv1.OrderEvent_LotProgressed:
		out["type"], out["lot"] = "lot_progressed", orderLotToMap(p.LotProgressed)
	case *fulfilmentv1.OrderEvent_QcRecorded:
		out["type"] = "qc_recorded"
		out["passed"] = p.QcRecorded.GetPassed()
	case *fulfilmentv1.OrderEvent_OrderStateChanged:
		out["type"] = "order_state_changed"
		out["state"] = trimEnumPrefix(p.OrderStateChanged.String(), "BULK_ORDER_STATE_")
	case *fulfilmentv1.OrderEvent_PaymentSettled:
		out["type"] = "payment_settled"
	}
	return out
}

func orderLotToMap(l *fulfilmentv1.OrderLot) map[string]any {
	return map[string]any{
		"id":         l.GetId(),
		"artisan_id": l.GetArtisanId(),
		"quantity":   l.GetQuantity(),
		"state":      trimEnumPrefix(l.GetState().String(), "LOT_STATE_"),
	}
}

// stringMap converts a JSON-decoded object into map[string]string, rejecting
// any non-string value rather than silently dropping it.
func stringMap(raw any) (map[string]string, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		if raw != nil {
			return nil, domain.InvalidInput("customisations: must be an object of strings")
		}
		return nil, nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			return nil, domain.InvalidInput("customisations: values must be strings")
		}
		out[k] = s
	}
	return out, nil
}
