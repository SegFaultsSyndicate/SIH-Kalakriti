// services/collab-svc/internal/collab/handler/fulfilment.go
package handler

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
	"github.com/segfaultsyndicate/kalakriti/pkg/ids"
	commonv1 "github.com/segfaultsyndicate/kalakriti/pkg/pb/common/v1"
	fulfilmentv1 "github.com/segfaultsyndicate/kalakriti/pkg/pb/fulfilment/v1"

	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/domain"
	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/service"
)

// EventBroker is what WatchOrder needs to subscribe to one order's live
// event stream. It is implemented by the Kafka fan-in wired in main.go, kept
// as a narrow interface here so the handler has no direct Kafka dependency.
type EventBroker interface {
	// Subscribe registers ch to receive every OrderEvent for orderID until
	// unsubscribe is called or ctx is cancelled, whichever comes first.
	// unsubscribe is always safe to call more than once.
	Subscribe(ctx context.Context, orderID string) (ch <-chan *fulfilmentv1.OrderEvent, unsubscribe func())
}

// EventHistory is what WatchOrder needs to replay events that occurred
// before the caller connected, when the request supplies `since`.
type EventHistory interface {
	ListEventsSince(ctx context.Context, orderID string, since *time.Time) ([]*fulfilmentv1.OrderEvent, error)
}

// Fulfilment implements fulfilment.v1.FulfilmentService.
type Fulfilment struct {
	fulfilmentv1.UnimplementedFulfilmentServiceServer
	svc     *service.Fulfilment
	broker  EventBroker
	history EventHistory
}

// NewFulfilment builds the fulfilment handler.
func NewFulfilment(svc *service.Fulfilment, broker EventBroker, history EventHistory) *Fulfilment {
	return &Fulfilment{svc: svc, broker: broker, history: history}
}

// CreateBulkOrder registers a buyer's bulk requirement.
func (h *Fulfilment) CreateBulkOrder(ctx context.Context, req *fulfilmentv1.CreateBulkOrderRequest) (*fulfilmentv1.CreateBulkOrderResponse, error) {
	listingID, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	requiredBy := req.GetRequiredBy().AsTime()

	order, err := h.svc.CreateBulkOrder(ctx, service.CreateBulkOrderInput{
		BuyerID:        req.GetBuyerId(),
		ListingID:      listingID,
		Quantity:       req.GetQuantity(),
		RequiredBy:     requiredBy,
		Customisations: req.GetCustomisations(),
		Notes:          req.Notes,
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &fulfilmentv1.CreateBulkOrderResponse{Order: bulkOrderToProto(order, nil)}, nil
}

// ProposeAllocation ranks candidates, decomposes the order and (unless
// DryRun) offers the resulting lots.
func (h *Fulfilment) ProposeAllocation(ctx context.Context, req *fulfilmentv1.ProposeAllocationRequest) (*fulfilmentv1.ProposeAllocationResponse, error) {
	orderID, err := parseUUID("bulk_order_id", req.GetBulkOrderId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	preferred, err := parseUUIDs("preferred_artisan_ids", req.GetPreferredArtisanIds())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	var window time.Duration
	if h := req.ResponseWindowHours; h != nil {
		window = time.Duration(*h) * time.Hour
	}

	result, err := h.svc.ProposeAllocation(ctx, service.ProposeAllocationInput{
		BulkOrderID:         orderID,
		PreferredArtisanIDs: preferred,
		MaxUnitsPerArtisan:  req.GetMaxUnitsPerArtisan(),
		ResponseWindow:      window,
		DryRun:              req.GetDryRun(),
		IdempotencyKey:      req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	out := &fulfilmentv1.ProposeAllocationResponse{
		Lots:                lotsToProto(result.Lots),
		UnallocatedQuantity: result.UnallocatedQuantity,
	}
	if result.ShortfallReason != "" {
		out.ShortfallReason = &result.ShortfallReason
	}
	return out, nil
}

// RespondToLot records an artisan's accept or decline.
func (h *Fulfilment) RespondToLot(ctx context.Context, req *fulfilmentv1.RespondToLotRequest) (*fulfilmentv1.RespondToLotResponse, error) {
	lotID, err := parseUUID("lot_id", req.GetLotId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	in := service.RespondToLotInput{
		LotID:          lotID,
		ArtisanID:      artisanID,
		Accept:         req.GetAccept(),
		DeclineReason:  req.DeclineReason,
		IdempotencyKey: req.GetIdempotencyKey(),
	}
	if ts := req.PromisedShipDate; ts != nil {
		t := ts.AsTime()
		in.PromisedShipDate = &t
	}

	lot, err := h.svc.RespondToLot(ctx, in)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &fulfilmentv1.RespondToLotResponse{Lot: lotToProto(lot)}, nil
}

// ReportProgress records production progress against an accepted lot.
func (h *Fulfilment) ReportProgress(ctx context.Context, req *fulfilmentv1.ReportProgressRequest) (*fulfilmentv1.ReportProgressResponse, error) {
	lotID, err := parseUUID("lot_id", req.GetLotId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	lot, err := h.svc.ReportProgress(ctx, service.ReportProgressInput{
		LotID:          lotID,
		ArtisanID:      artisanID,
		ProgressPct:    req.GetProgressPct(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &fulfilmentv1.ReportProgressResponse{Lot: lotToProto(lot)}, nil
}

// SubmitQC records a quality inspection outcome. Defects and media are
// accepted but not yet persisted — the QC result row itself is a later
// batch's scope (see fulfilment.go's SubmitQC doc comment); this RPC only
// drives the lot and order state machines through the happy path.
func (h *Fulfilment) SubmitQC(ctx context.Context, req *fulfilmentv1.SubmitQCRequest) (*fulfilmentv1.SubmitQCResponse, error) {
	lotID, err := parseUUID("lot_id", req.GetLotId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	mediaIDs, err := parseUUIDs("media", mediaRefIDs(req.GetMedia()))
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	defects, err := defectsFromProto(req.GetDefects())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	lot, err := h.svc.SubmitQC(ctx, service.SubmitQCInput{
		LotID:          lotID,
		InspectorID:    req.GetInspectorId(),
		Passed:         req.GetPassed(),
		Notes:          req.Notes,
		MediaIDs:       mediaIDs,
		Defects:        defects,
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &fulfilmentv1.SubmitQCResponse{
		Result: &fulfilmentv1.QCResult{
			Id:          ids.New().String(),
			LotId:       req.GetLotId(),
			Passed:      req.GetPassed(),
			InspectorId: req.GetInspectorId(),
			Defects:     req.GetDefects(),
			Notes:       req.Notes,
			InspectedAt: timestamppb.Now(),
		},
		Lot: lotToProto(lot),
	}, nil
}

// mediaRefIDs extracts the media row id from each ref; SubmitQC only needs
// the id to persist (see qc_result.media_ids / qc_defect.media_ids), not the
// bucket/key the media pipeline already owns.
func mediaRefIDs(refs []*commonv1.MediaRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.GetId())
	}
	return out
}

// defectsFromProto converts wire QCDefect inputs to the domain type.
func defectsFromProto(in []*fulfilmentv1.QCDefect) ([]domain.QCDefect, error) {
	out := make([]domain.QCDefect, 0, len(in))
	for i, d := range in {
		mediaIDs, err := parseUUIDs(fmt.Sprintf("defects[%d].media", i), mediaRefIDs(d.GetMedia()))
		if err != nil {
			return nil, err
		}
		severity, err := defectSeverityFromProto(d.GetSeverity())
		if err != nil {
			return nil, fmt.Errorf("defects[%d].severity: %w", i, err)
		}
		out = append(out, domain.QCDefect{
			Code:          d.GetCode(),
			Description:   d.GetDescription(),
			Severity:      severity,
			MediaIDs:      mediaIDs,
			AffectedUnits: d.GetAffectedUnits(),
		})
	}
	return out, nil
}

func defectSeverityFromProto(s fulfilmentv1.DefectSeverity) (domain.DefectSeverity, error) {
	switch s {
	case fulfilmentv1.DefectSeverity_DEFECT_SEVERITY_MINOR:
		return domain.DefectMinor, nil
	case fulfilmentv1.DefectSeverity_DEFECT_SEVERITY_MAJOR:
		return domain.DefectMajor, nil
	case fulfilmentv1.DefectSeverity_DEFECT_SEVERITY_CRITICAL:
		return domain.DefectCritical, nil
	default:
		return "", pkgdomain.InvalidInput("severity must be specified for every defect")
	}
}

// CancelBulkOrder cancels a bulk order before production has started.
func (h *Fulfilment) CancelBulkOrder(ctx context.Context, req *fulfilmentv1.CancelBulkOrderRequest) (*fulfilmentv1.CancelBulkOrderResponse, error) {
	orderID, err := parseUUID("bulk_order_id", req.GetBulkOrderId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	result, err := h.svc.CancelBulkOrder(ctx, service.CancelBulkOrderInput{
		BulkOrderID:    orderID,
		Reason:         req.GetReason(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	releasedIDs := make([]string, 0, len(result.ReleasedLotIDs))
	for _, id := range result.ReleasedLotIDs {
		releasedIDs = append(releasedIDs, id.String())
	}
	return &fulfilmentv1.CancelBulkOrderResponse{
		Order:          bulkOrderToProto(result.Order, nil),
		ReleasedLotIds: releasedIDs,
	}, nil
}

// WatchOrder streams one bulk order's events until the caller disconnects. It
// replays persisted history first (bounded by `since` when given), then
// forwards live events from the Kafka-backed broker, subscribing before the
// replay finishes so no event in between is missed — a fan-in mechanism, not
// a poll, per the "hand-rolled saga" spec: the coordinator's state lives in
// Postgres and this RPC merely observes the outbox-published trail of it.
func (h *Fulfilment) WatchOrder(req *fulfilmentv1.WatchOrderRequest, stream fulfilmentv1.FulfilmentService_WatchOrderServer) error {
	ctx := stream.Context()
	orderID := req.GetBulkOrderId()
	if _, err := parseUUID("bulk_order_id", orderID); err != nil {
		return pkgdomain.GRPCError(err)
	}

	// Subscribe before replaying so a live event that lands during replay is
	// buffered on ch rather than lost between the history read and the
	// subscription starting.
	ch, unsubscribe := h.broker.Subscribe(ctx, orderID)
	defer unsubscribe()

	var since *time.Time
	if ts := req.Since; ts != nil {
		t := ts.AsTime()
		since = &t
	}
	history, err := h.history.ListEventsSince(ctx, orderID, since)
	if err != nil {
		return pkgdomain.GRPCError(fmt.Errorf("loading event history for order %s: %w", orderID, err))
	}
	for _, ev := range history {
		if err := stream.Send(&fulfilmentv1.WatchOrderResponse{Event: ev}); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(&fulfilmentv1.WatchOrderResponse{Event: ev}); err != nil {
				return err
			}
		}
	}
}

// --- domain <-> proto ----------------------------------------------------------

func bulkOrderToProto(o domain.BulkOrder, lots []domain.OrderLot) *fulfilmentv1.BulkOrder {
	out := &fulfilmentv1.BulkOrder{
		Id:                o.ID.String(),
		BuyerId:           o.BuyerID,
		ListingId:         o.ListingID.String(),
		ProductId:         o.ProductID.String(),
		Quantity:          o.Quantity,
		UnitPrice:         moneyToProto(o.UnitPricePaise),
		TotalValue:        moneyToProto(o.TotalValuePaise),
		RequiredBy:        timestamppb.New(o.RequiredBy),
		State:             bulkOrderStateToProto(o.State),
		AllocatedQuantity: o.AllocatedQuantity,
		Customisations:    o.Customisations,
		Notes:             o.Notes,
		Lots:              lotsToProto(lots),
	}
	return out
}

func lotsToProto(lots []domain.OrderLot) []*fulfilmentv1.OrderLot {
	out := make([]*fulfilmentv1.OrderLot, 0, len(lots))
	for _, l := range lots {
		out = append(out, lotToProto(l))
	}
	return out
}

func lotToProto(l domain.OrderLot) *fulfilmentv1.OrderLot {
	out := &fulfilmentv1.OrderLot{
		Id:          l.ID.String(),
		BulkOrderId: l.BulkOrderID.String(),
		ArtisanId:   l.ArtisanID.String(),
		Quantity:    l.Quantity,
		UnitPrice:   moneyToProto(l.UnitPricePaise),
		LotValue:    moneyToProto(l.LotValuePaise),
		State:       lotStateToProto(l.State),
		OfferedAt:   timestamppb.New(l.OfferedAt),
		RespondsBy:  timestamppb.New(l.RespondsBy),
		ProgressPct: l.ProgressPct,
	}
	if l.ClusterID != nil {
		id := l.ClusterID.String()
		out.ClusterId = &id
	}
	if l.AcceptedAt != nil {
		out.AcceptedAt = timestamppb.New(*l.AcceptedAt)
	}
	if l.PromisedShipDate != nil {
		out.PromisedShipDate = timestamppb.New(*l.PromisedShipDate)
	}
	if l.CapacityReservationID != nil {
		id := l.CapacityReservationID.String()
		out.CapacityReservationId = &id
	}
	out.DeclineReason = l.DeclineReason
	if l.ReallocatedFromLotID != nil {
		id := l.ReallocatedFromLotID.String()
		out.ReallocatedFromLotId = &id
	}
	return out
}

func bulkOrderStateToProto(s domain.BulkOrderState) fulfilmentv1.BulkOrderState {
	switch s {
	case domain.BulkOrderAllocating:
		return fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_ALLOCATING
	case domain.BulkOrderPartiallyAllocated:
		return fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_PARTIALLY_ALLOCATED
	case domain.BulkOrderConfirmed:
		return fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_CONFIRMED
	case domain.BulkOrderInProduction:
		return fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_IN_PRODUCTION
	case domain.BulkOrderAmendmentPending:
		return fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_AMENDMENT_PENDING
	case domain.BulkOrderCompleted:
		return fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_COMPLETED
	case domain.BulkOrderCancelled:
		return fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_CANCELLED
	default:
		return fulfilmentv1.BulkOrderState_BULK_ORDER_STATE_UNSPECIFIED
	}
}

func lotStateToProto(s domain.LotState) fulfilmentv1.LotState {
	switch s {
	case domain.LotOffered:
		return fulfilmentv1.LotState_LOT_STATE_OFFERED
	case domain.LotAccepted:
		return fulfilmentv1.LotState_LOT_STATE_ACCEPTED
	case domain.LotDeclined:
		return fulfilmentv1.LotState_LOT_STATE_DECLINED
	case domain.LotExpired:
		return fulfilmentv1.LotState_LOT_STATE_EXPIRED
	case domain.LotInProduction:
		return fulfilmentv1.LotState_LOT_STATE_IN_PRODUCTION
	case domain.LotQCPending:
		return fulfilmentv1.LotState_LOT_STATE_QC_PENDING
	case domain.LotQCFailed:
		return fulfilmentv1.LotState_LOT_STATE_QC_FAILED
	case domain.LotCompleted:
		return fulfilmentv1.LotState_LOT_STATE_COMPLETED
	case domain.LotReallocated:
		return fulfilmentv1.LotState_LOT_STATE_REALLOCATED
	default:
		return fulfilmentv1.LotState_LOT_STATE_UNSPECIFIED
	}
}
