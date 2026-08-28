// services/collab-svc/internal/collab/handler/broker.go
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	segmentio "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/types/known/timestamppb"

	fulfilmentv1 "github.com/segfaultsyndicate/kalakriti/pkg/pb/fulfilment/v1"
)

// envelope is the outbox envelope every Kalakriti event is wrapped in.
type envelope struct {
	Header struct {
		EventID string `json:"event_id"`
	} `json:"header"`
	Payload json.RawMessage `json:"payload"`
}

// lotEventPayload mirrors service's private lotEventPayload JSON shape. The
// two are independent by package boundary (service must not import handler
// or protobuf) and kept in sync by convention, the same as every other
// domain<->wire mapping in this codebase.
type lotEventPayload struct {
	LotID       string `json:"lot_id"`
	BulkOrderID string `json:"bulk_order_id"`
	ArtisanID   string `json:"artisan_id"`
	Quantity    int32  `json:"quantity"`
	State       string `json:"state"`
	ProgressPct int32  `json:"progress_pct"`
}

func (p lotEventPayload) toProtoLot() *fulfilmentv1.OrderLot {
	return &fulfilmentv1.OrderLot{
		Id:          p.LotID,
		BulkOrderId: p.BulkOrderID,
		ArtisanId:   p.ArtisanID,
		Quantity:    p.Quantity,
		State:       fulfilmentv1.LotState(fulfilmentv1.LotState_value["LOT_STATE_"+p.State]),
		ProgressPct: p.ProgressPct,
	}
}

// subscriberBufferSize bounds how many events a slow WatchOrder client can
// fall behind by before further events are dropped for it, rather than
// blocking the Kafka consumer goroutine indefinitely.
const subscriberBufferSize = 32

// Broker fans saga events out to per-order WatchOrder subscriber channels. It
// holds no saga state itself — every event it forwards was already durably
// committed by the saga's own transaction and published through the outbox;
// the broker only distributes copies to whichever WatchOrder callers happen
// to be connected right now. A client connected before an event publishes
// gets it live; one that connects after relies on WatchOrder's `since`-bounded
// history replay (see repo.ListBulkOrderEventsSince) instead.
type Broker struct {
	mu   sync.Mutex
	subs map[string]map[chan *fulfilmentv1.OrderEvent]struct{}
}

// NewBroker builds an empty broker.
func NewBroker() *Broker {
	return &Broker{subs: map[string]map[chan *fulfilmentv1.OrderEvent]struct{}{}}
}

// Subscribe satisfies handler.EventBroker.
func (b *Broker) Subscribe(_ context.Context, orderID string) (<-chan *fulfilmentv1.OrderEvent, func()) {
	ch := make(chan *fulfilmentv1.OrderEvent, subscriberBufferSize)

	b.mu.Lock()
	if b.subs[orderID] == nil {
		b.subs[orderID] = map[chan *fulfilmentv1.OrderEvent]struct{}{}
	}
	b.subs[orderID][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs[orderID], ch)
			if len(b.subs[orderID]) == 0 {
				delete(b.subs, orderID)
			}
			b.mu.Unlock()
			close(ch)
		})
	}
	return ch, unsubscribe
}

// publish delivers ev to every live subscriber of its order, dropping it for
// any subscriber whose buffer is full rather than blocking — a slow client
// falls behind, it does not stall delivery to every other order's watchers.
func (b *Broker) publish(orderID string, ev *fulfilmentv1.OrderEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[orderID] {
		select {
		case ch <- ev:
		default:
		}
	}
}

// decodeLotEvent unmarshals one order.lot.* message's envelope into the
// common lot payload shape every one of those topics shares.
func decodeLotEvent(msg []byte) (lotEventPayload, string, error) {
	var env envelope
	if err := json.Unmarshal(msg, &env); err != nil {
		return lotEventPayload{}, "", fmt.Errorf("decoding envelope: %w", err)
	}
	var p lotEventPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return lotEventPayload{}, "", fmt.Errorf("decoding lot event payload of %s: %w", env.Header.EventID, err)
	}
	return p, env.Header.EventID, nil
}

// HandleLotOffered decodes an order.lot.offered message and fans it out.
func (b *Broker) HandleLotOffered(_ context.Context, msg segmentio.Message) error {
	p, eventID, err := decodeLotEvent(msg.Value)
	if err != nil {
		return err
	}
	b.publish(p.BulkOrderID, &fulfilmentv1.OrderEvent{
		EventId: eventID, BulkOrderId: p.BulkOrderID, OccurredAt: timestamppb.Now(),
		Payload: &fulfilmentv1.OrderEvent_LotOffered{LotOffered: p.toProtoLot()},
	})
	return nil
}

// HandleLotAccepted decodes an order.lot.accepted message and fans it out.
func (b *Broker) HandleLotAccepted(_ context.Context, msg segmentio.Message) error {
	p, eventID, err := decodeLotEvent(msg.Value)
	if err != nil {
		return err
	}
	b.publish(p.BulkOrderID, &fulfilmentv1.OrderEvent{
		EventId: eventID, BulkOrderId: p.BulkOrderID, OccurredAt: timestamppb.Now(),
		Payload: &fulfilmentv1.OrderEvent_LotAccepted{LotAccepted: p.toProtoLot()},
	})
	return nil
}

// HandleLotDeclined decodes an order.lot.declined message and fans it out.
func (b *Broker) HandleLotDeclined(_ context.Context, msg segmentio.Message) error {
	p, eventID, err := decodeLotEvent(msg.Value)
	if err != nil {
		return err
	}
	b.publish(p.BulkOrderID, &fulfilmentv1.OrderEvent{
		EventId: eventID, BulkOrderId: p.BulkOrderID, OccurredAt: timestamppb.Now(),
		Payload: &fulfilmentv1.OrderEvent_LotDeclined{LotDeclined: p.toProtoLot()},
	})
	return nil
}

// HandleLotProgressed decodes an order.lot.progressed message and fans it out.
func (b *Broker) HandleLotProgressed(_ context.Context, msg segmentio.Message) error {
	p, eventID, err := decodeLotEvent(msg.Value)
	if err != nil {
		return err
	}
	b.publish(p.BulkOrderID, &fulfilmentv1.OrderEvent{
		EventId: eventID, BulkOrderId: p.BulkOrderID, OccurredAt: timestamppb.Now(),
		Payload: &fulfilmentv1.OrderEvent_LotProgressed{LotProgressed: p.toProtoLot()},
	})
	return nil
}

// bulkOrderEventPayload mirrors service's private bulkOrderEventPayload.
type bulkOrderEventPayload struct {
	BulkOrderID string `json:"bulk_order_id"`
	State       string `json:"state"`
}

// HandleOrderConfirmed decodes an order.fulfilment.completed message (used
// for both the CONFIRMED and COMPLETED order-level transitions — see
// service.RespondToLot and service.SubmitQC) and fans out an
// order_state_changed event.
func (b *Broker) HandleOrderConfirmed(_ context.Context, msg segmentio.Message) error {
	return b.handleOrderStateEvent(msg)
}

// HandleOrderCancelled decodes an order.bulk.cancelled message and fans out
// an order_state_changed event.
func (b *Broker) HandleOrderCancelled(_ context.Context, msg segmentio.Message) error {
	return b.handleOrderStateEvent(msg)
}

func (b *Broker) handleOrderStateEvent(msg segmentio.Message) error {
	var env envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		return fmt.Errorf("decoding envelope: %w", err)
	}
	var p bulkOrderEventPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("decoding order event payload of %s: %w", env.Header.EventID, err)
	}
	state := fulfilmentv1.BulkOrderState(fulfilmentv1.BulkOrderState_value["BULK_ORDER_STATE_"+p.State])
	b.publish(p.BulkOrderID, &fulfilmentv1.OrderEvent{
		EventId: env.Header.EventID, BulkOrderId: p.BulkOrderID, OccurredAt: timestamppb.Now(),
		Payload: &fulfilmentv1.OrderEvent_OrderStateChanged{OrderStateChanged: state},
	})
	return nil
}
