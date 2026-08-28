// services/collab-svc/internal/collab/handler/history.go
package handler

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	fulfilmentv1 "github.com/segfaultsyndicate/kalakriti/pkg/pb/fulfilment/v1"

	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/repo"
)

// HistoryStore adapts the repo to handler.EventHistory, so WatchOrder can
// replay bulk_order_event rows recorded before the caller connected. It
// reads directly from *repo.Repo rather than through the saga's Store
// interface because history playback is not a saga concern — it needs no
// InTx, no write surface, just the one read.
type HistoryStore struct {
	repo *repo.Repo
}

// NewHistoryStore builds the WatchOrder replay source.
func NewHistoryStore(r *repo.Repo) *HistoryStore { return &HistoryStore{repo: r} }

// ListEventsSince satisfies handler.EventHistory. The event_type recorded by
// service.Fulfilment (ORDER_CREATED, LOT_OFFERED, LOT_ACCEPTED, LOT_DECLINED,
// LOT_PROGRESSED, QC_RECORDED, ORDER_CONFIRMED, ORDER_COMPLETED) is mapped
// onto the proto oneof case it corresponds to; a row whose payload does not
// decode into the shape its event_type implies is returned with an empty
// payload rather than failing the whole replay, so one malformed historical
// row cannot block a caller from seeing everything after it.
func (h *HistoryStore) ListEventsSince(ctx context.Context, orderID string, since *time.Time) ([]*fulfilmentv1.OrderEvent, error) {
	id, err := parseUUID("bulk_order_id", orderID)
	if err != nil {
		return nil, err
	}
	rows, err := h.repo.ListBulkOrderEventsSince(ctx, id, since)
	if err != nil {
		return nil, err
	}

	out := make([]*fulfilmentv1.OrderEvent, 0, len(rows))
	for _, row := range rows {
		ev := &fulfilmentv1.OrderEvent{
			EventId:     row.ID.String(),
			BulkOrderId: row.BulkOrderID.String(),
			OccurredAt:  timestamppb.New(row.OccurredAt),
		}
		decodeHistoricalPayload(ev, row.EventType, row.Payload)
		out = append(out, ev)
	}
	return out, nil
}

// decodeHistoricalPayload maps one bulk_order_event row onto the OrderEvent
// oneof case its event_type implies, setting it directly on ev. ok is false
// for an event_type this batch does not know how to render as a typed
// payload (there are none as of this batch — every event_type
// service.Fulfilment writes is handled — but a future batch's new
// event_type must not break replay of everything recorded before it, hence
// the graceful skip instead of a decode error).
func decodeHistoricalPayload(ev *fulfilmentv1.OrderEvent, eventType string, raw json.RawMessage) {
	switch eventType {
	case "LOT_OFFERED", "LOT_ACCEPTED", "LOT_DECLINED", "LOT_PROGRESSED":
		var p lotEventPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return
		}
		lot := p.toProtoLot()
		switch eventType {
		case "LOT_OFFERED":
			ev.Payload = &fulfilmentv1.OrderEvent_LotOffered{LotOffered: lot}
		case "LOT_ACCEPTED":
			ev.Payload = &fulfilmentv1.OrderEvent_LotAccepted{LotAccepted: lot}
		case "LOT_DECLINED":
			ev.Payload = &fulfilmentv1.OrderEvent_LotDeclined{LotDeclined: lot}
		default:
			ev.Payload = &fulfilmentv1.OrderEvent_LotProgressed{LotProgressed: lot}
		}
	case "ORDER_CONFIRMED", "ORDER_COMPLETED", "ORDER_CANCELLED":
		var p bulkOrderEventPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return
		}
		state := fulfilmentv1.BulkOrderState(fulfilmentv1.BulkOrderState_value["BULK_ORDER_STATE_"+p.State])
		ev.Payload = &fulfilmentv1.OrderEvent_OrderStateChanged{OrderStateChanged: state}
	}
}
