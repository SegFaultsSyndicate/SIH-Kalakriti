// services/bff/internal/bff/client/order.go
package client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
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

// GetOrder fetches one bulk order by id, with its lots.
func (o *Order) GetOrder(ctx context.Context, orderID string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := o.fulfilment.GetOrder(ctx, &fulfilmentv1.GetOrderRequest{BulkOrderId: orderID})
	if err != nil {
		return nil, grpcErr(err)
	}
	return bulkOrderToMap(resp.GetOrder()), nil
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

// ReportProgress records production progress against an accepted (or, for a
// rework resubmission, QC_FAILED) lot. fields carries progress_pct (number,
// required), media (array of confirmed media ids, optional) and note
// (string, optional).
func (o *Order) ReportProgress(ctx context.Context, lotID, artisanID, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	pct, ok := fields["progress_pct"].(float64)
	if !ok {
		return nil, domain.InvalidInput("progress_pct: is required")
	}
	mediaIDs, err := stringSlice(fields, "media")
	if err != nil {
		return nil, err
	}
	media := make([]*commonv1.MediaRef, len(mediaIDs))
	for i, id := range mediaIDs {
		media[i] = &commonv1.MediaRef{Id: id}
	}
	req := &fulfilmentv1.ReportProgressRequest{
		LotId:          lotID,
		ArtisanId:      artisanID,
		ProgressPct:    int32(pct),
		Media:          media,
		IdempotencyKey: idempotencyKey,
	}
	if note, ok := fields["note"].(string); ok && note != "" {
		req.Note = &note
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := o.fulfilment.ReportProgress(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}
	return orderLotToMap(resp.GetLot()), nil
}

// RequestReallocation gives up an artisan's own accepted lot they cannot
// complete, so its units go back out to another artisan instead of quietly
// missing the ship date. fields carries reason (string, required).
func (o *Order) RequestReallocation(ctx context.Context, lotID, artisanID, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	reason, _ := fields["reason"].(string)
	if reason == "" {
		return nil, domain.InvalidInput("reason: is required")
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := o.fulfilment.RequestReallocation(ctx, &fulfilmentv1.RequestReallocationRequest{
		LotId: lotID, ArtisanId: artisanID, Reason: reason, IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	return orderLotToMap(resp.GetLot()), nil
}

// WatchOrder streams one order's events until the caller disconnects
// (ctx.Done()) or the backend closes the stream. Uses withAuth, not
// withTimeout: a stream is meant to outlive callTimeout's 10 seconds. since,
// when non-nil, replays events that occurred after it before streaming live
// ones -- the reconnect/backfill path (see handler.WatchOrder, which derives
// it from the SSE Last-Event-ID).
func (o *Order) WatchOrder(ctx context.Context, orderID string, since *time.Time) (<-chan map[string]any, error) {
	ctx = withAuth(ctx)
	req := &fulfilmentv1.WatchOrderRequest{BulkOrderId: orderID}
	if since != nil {
		req.Since = timestamppb.New(*since)
	}
	stream, err := o.fulfilment.WatchOrder(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	events := make(chan map[string]any)
	go func() {
		defer close(events)
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("order watch stream panicked", "panic", fmt.Sprintf("%v", rec))
			}
		}()
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
		"occurred_at":   e.GetOccurredAt().AsTime().Format(time.RFC3339Nano),
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
	case *fulfilmentv1.OrderEvent_LotGaveUp:
		out["type"], out["lot"] = "lot_gave_up", orderLotToMap(p.LotGaveUp)
	case *fulfilmentv1.OrderEvent_QcRecorded:
		out["type"] = "qc_recorded"
		out["qc_result"] = qcResultToMap(p.QcRecorded)
	case *fulfilmentv1.OrderEvent_OrderStateChanged:
		out["type"] = "order_state_changed"
		out["state"] = trimEnumPrefix(p.OrderStateChanged.String(), "BULK_ORDER_STATE_")
	case *fulfilmentv1.OrderEvent_PaymentSettled:
		out["type"] = "payment_settled"
		out["payment_split"] = paymentSplitToMap(p.PaymentSettled)
	}
	return out
}

func qcResultToMap(r *fulfilmentv1.QCResult) map[string]any {
	defects := make([]map[string]any, 0, len(r.GetDefects()))
	for _, d := range r.GetDefects() {
		defects = append(defects, map[string]any{
			"code":           d.GetCode(),
			"description":    d.GetDescription(),
			"severity":       trimEnumPrefix(d.GetSeverity().String(), "DEFECT_SEVERITY_"),
			"affected_units": d.GetAffectedUnits(),
		})
	}
	m := map[string]any{
		"lot_id":       r.GetLotId(),
		"passed":       r.GetPassed(),
		"defects":      defects,
		"inspected_at": r.GetInspectedAt().AsTime().Format(time.RFC3339),
	}
	if r.Notes != nil {
		m["notes"] = *r.Notes
	}
	return m
}

func paymentSplitToMap(s *fulfilmentv1.PaymentSplit) map[string]any {
	lines := make([]map[string]any, 0, len(s.GetLines()))
	for _, l := range s.GetLines() {
		lines = append(lines, map[string]any{
			"payee_id":     l.GetPayeeId(),
			"lot_id":       l.GetLotId(),
			"gross_amount": moneyMap(l.GetGrossAmount()),
			"net_amount":   moneyMap(l.GetNetAmount()),
		})
	}
	return map[string]any{
		"gross_total":      moneyMap(s.GetGrossTotal()),
		"commission_total": moneyMap(s.GetCommissionTotal()),
		"net_total":        moneyMap(s.GetNetTotal()),
		"lines":            lines,
	}
}

func bulkOrderToMap(o *fulfilmentv1.BulkOrder) map[string]any {
	m := map[string]any{
		"id":                 o.GetId(),
		"buyer_id":           o.GetBuyerId(),
		"listing_id":         o.GetListingId(),
		"product_id":         o.GetProductId(),
		"quantity":           o.GetQuantity(),
		"unit_price":         moneyMap(o.GetUnitPrice()),
		"total_value":        moneyMap(o.GetTotalValue()),
		"required_by":        o.GetRequiredBy().AsTime().Format(time.RFC3339),
		"state":              trimEnumPrefix(o.GetState().String(), "BULK_ORDER_STATE_"),
		"allocated_quantity": o.GetAllocatedQuantity(),
		"customisations":     o.GetCustomisations(),
	}
	if o.Notes != nil {
		m["notes"] = *o.Notes
	}
	lots := make([]map[string]any, 0, len(o.GetLots()))
	for _, l := range o.GetLots() {
		lots = append(lots, orderLotToMap(l))
	}
	m["lots"] = lots
	return m
}

func orderLotToMap(l *fulfilmentv1.OrderLot) map[string]any {
	m := map[string]any{
		"id":            l.GetId(),
		"bulk_order_id": l.GetBulkOrderId(),
		"artisan_id":    l.GetArtisanId(),
		"quantity":      l.GetQuantity(),
		"unit_price":    moneyMap(l.GetUnitPrice()),
		"lot_value":     moneyMap(l.GetLotValue()),
		"state":         trimEnumPrefix(l.GetState().String(), "LOT_STATE_"),
		"offered_at":    l.GetOfferedAt().AsTime().Format(time.RFC3339),
		"responds_by":   l.GetRespondsBy().AsTime().Format(time.RFC3339),
		"progress_pct":  l.GetProgressPct(),
	}
	if l.ClusterId != nil {
		m["cluster_id"] = *l.ClusterId
	}
	if l.AcceptedAt != nil {
		m["accepted_at"] = l.GetAcceptedAt().AsTime().Format(time.RFC3339)
	}
	if l.PromisedShipDate != nil {
		m["promised_ship_date"] = l.GetPromisedShipDate().AsTime().Format(time.RFC3339)
	}
	if l.DeclineReason != nil {
		m["decline_reason"] = *l.DeclineReason
	}
	if l.ReallocatedFromLotId != nil {
		m["reallocated_from_lot_id"] = *l.ReallocatedFromLotId
	}
	return m
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
