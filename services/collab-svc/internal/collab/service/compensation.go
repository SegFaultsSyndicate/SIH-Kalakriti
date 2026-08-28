// services/collab-svc/internal/collab/service/compensation.go

// Compensation paths (batch 13): the buyer amendment flow raised when
// accepted lots cannot cover an order, and the two ways a lot can be given up
// on mid-fulfilment — an artisan dropping out, and a QC rework that never
// lands. Lot-offer expiry (the fourth compensation path) lives in
// RunReservationReaper (fulfilment.go) instead, since it is driven by the
// same sweep that already expires lots, not a caller-initiated action.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/collab-svc/internal/collab/domain"
)

// --- ProposeAmendment / DecideAmendment --------------------------------------

// ProposeAmendmentInput raises a buyer amendment. Exactly one of
// ProposedQuantity/ProposedRequiredBy must be set, matching Type.
type ProposeAmendmentInput struct {
	BulkOrderID        uuid.UUID
	Type               domain.AmendmentType
	ProposedQuantity   *int32
	ProposedRequiredBy *time.Time
	Reason             string
	IdempotencyKey     string
}

// ProposeAmendment raises a buyer amendment proposal — a reduced quantity or
// an extended deadline — and moves the order to AMENDMENT_PENDING. This is
// compensation path (b): called once ProposeAllocation has exhausted every
// candidate and accepted lots still fall short of the order's quantity, in
// place of leaving the order stuck in PARTIALLY_ALLOCATED indefinitely.
//
// Only one amendment may be pending at a time (migrations/021's unique
// index); a second proposal while one is outstanding is a conflict, not
// silently queued — the caller should wait for DecideAmendment first.
func (f *Fulfilment) ProposeAmendment(ctx context.Context, in ProposeAmendmentInput) (domain.BulkOrderAmendment, error) {
	if in.Reason == "" {
		return domain.BulkOrderAmendment{}, pkgdomain.InvalidInput("reason is required")
	}

	order, err := f.store.GetBulkOrder(ctx, in.BulkOrderID)
	if err != nil {
		return domain.BulkOrderAmendment{}, fmt.Errorf("loading bulk order %s: %w", in.BulkOrderID, err)
	}
	if order.State != domain.BulkOrderAllocating && order.State != domain.BulkOrderPartiallyAllocated {
		return domain.BulkOrderAmendment{}, pkgdomain.Conflict(
			"bulk order cannot take an amendment proposal in state " + string(order.State))
	}

	switch in.Type {
	case domain.AmendmentReduceQuantity:
		if in.ProposedQuantity == nil {
			return domain.BulkOrderAmendment{}, pkgdomain.InvalidInput("proposed_quantity is required for REDUCE_QUANTITY")
		}
		if *in.ProposedQuantity <= order.AllocatedQuantity || *in.ProposedQuantity >= order.Quantity {
			return domain.BulkOrderAmendment{}, pkgdomain.InvalidInput(
				"proposed_quantity must be more than what is already accepted and less than the current quantity")
		}
	case domain.AmendmentExtendDeadline:
		if in.ProposedRequiredBy == nil {
			return domain.BulkOrderAmendment{}, pkgdomain.InvalidInput("proposed_required_by is required for EXTEND_DEADLINE")
		}
		if !in.ProposedRequiredBy.After(order.RequiredBy) {
			return domain.BulkOrderAmendment{}, pkgdomain.InvalidInput("proposed_required_by must extend the current deadline")
		}
	default:
		return domain.BulkOrderAmendment{}, pkgdomain.InvalidInput("unknown amendment type " + string(in.Type))
	}

	amendmentID := ids.New()
	var created domain.BulkOrderAmendment
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		created, err = tx.CreateAmendment(ctx, amendmentID, domain.BulkOrderAmendment{
			BulkOrderID:        order.ID,
			Type:               in.Type,
			ProposedQuantity:   in.ProposedQuantity,
			ProposedRequiredBy: in.ProposedRequiredBy,
			Reason:             in.Reason,
		})
		if err != nil {
			return err
		}
		if !domain.CanTransitionBulkOrder(order.State, domain.BulkOrderAmendmentPending) {
			return pkgdomain.Conflict("bulk order cannot move to AMENDMENT_PENDING from " + string(order.State))
		}
		if _, err := tx.TransitionBulkOrder(ctx, order.ID, order.State, domain.BulkOrderAmendmentPending); err != nil {
			return err
		}
		if err := tx.RecordEvent(ctx, ids.New(), order.ID, nil, "AMENDMENT_PROPOSED", created); err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), order.ID.String(),
			topics.OrderAmendmentProposed, in.IdempotencyKey, created)
	})
	if err != nil {
		return domain.BulkOrderAmendment{}, fmt.Errorf("proposing amendment for order %s: %w", in.BulkOrderID, err)
	}
	return created, nil
}

// DecideAmendmentInput carries the buyer's decision on a pending amendment.
type DecideAmendmentInput struct {
	AmendmentID    uuid.UUID
	Accept         bool
	IdempotencyKey string
}

// DecideAmendment applies the buyer's decision. Accepting re-plans: a
// REDUCE_QUANTITY amendment lowers the order's quantity (and recomputes its
// total from the pinned unit price) and returns it to PARTIALLY_ALLOCATED —
// or straight to CONFIRMED if what is already accepted now covers the
// reduced quantity; an EXTEND_DEADLINE amendment just pushes required_by out
// and returns to PARTIALLY_ALLOCATED. Either way, actually re-filling any
// remaining shortfall is a follow-up ProposeAllocation call, same as this
// package's other reoffer points — accepting an amendment does not
// implicitly re-run allocation itself. Declining releases every open
// commitment and cancels the order outright, by delegating to
// CancelBulkOrder once the amendment itself is recorded as declined.
func (f *Fulfilment) DecideAmendment(ctx context.Context, in DecideAmendmentInput) (domain.BulkOrderAmendment, error) {
	amendment, err := f.store.GetAmendment(ctx, in.AmendmentID)
	if err != nil {
		return domain.BulkOrderAmendment{}, fmt.Errorf("loading amendment %s: %w", in.AmendmentID, err)
	}
	wantStatus := domain.AmendmentDeclined
	if in.Accept {
		wantStatus = domain.AmendmentAccepted
	}
	if amendment.Status != domain.AmendmentPendingStatus {
		if amendment.Status == wantStatus {
			// Replay of an already-applied decision: idempotent no-op.
			return amendment, nil
		}
		return domain.BulkOrderAmendment{}, pkgdomain.Conflict("amendment already decided as " + string(amendment.Status))
	}

	order, err := f.store.GetBulkOrder(ctx, amendment.BulkOrderID)
	if err != nil {
		return domain.BulkOrderAmendment{}, fmt.Errorf("loading bulk order %s: %w", amendment.BulkOrderID, err)
	}

	var decided domain.BulkOrderAmendment
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		decided, err = tx.DecideAmendment(ctx, amendment.ID, wantStatus)
		if err != nil {
			return err
		}
		if err := tx.RecordEvent(ctx, ids.New(), order.ID, nil, "AMENDMENT_DECIDED", decided); err != nil {
			return err
		}
		if err := outbox.Enqueue(ctx, tx, ids.New().String(), order.ID.String(),
			topics.OrderAmendmentDecided, in.IdempotencyKey, decided); err != nil {
			return err
		}
		if !in.Accept {
			return nil
		}

		switch amendment.Type {
		case domain.AmendmentReduceQuantity:
			newQuantity := *amendment.ProposedQuantity
			newTotal := order.UnitPricePaise * int64(newQuantity)
			nextState := domain.BulkOrderPartiallyAllocated
			if order.AllocatedQuantity >= newQuantity {
				nextState = domain.BulkOrderConfirmed
			}
			updated, err := tx.ReduceBulkOrderQuantity(ctx, order.ID, newQuantity, newTotal, nextState)
			if err != nil {
				return err
			}
			return tx.RecordEvent(ctx, ids.New(), order.ID, nil, "ORDER_QUANTITY_REDUCED", updated)
		case domain.AmendmentExtendDeadline:
			updated, err := tx.ExtendBulkOrderDeadline(ctx, order.ID, *amendment.ProposedRequiredBy, domain.BulkOrderPartiallyAllocated)
			if err != nil {
				return err
			}
			return tx.RecordEvent(ctx, ids.New(), order.ID, nil, "ORDER_DEADLINE_EXTENDED", updated)
		default:
			return fmt.Errorf("amendment %s has unknown type %s", amendment.ID, amendment.Type)
		}
	})
	if err != nil {
		return domain.BulkOrderAmendment{}, fmt.Errorf("deciding amendment %s: %w", in.AmendmentID, err)
	}

	if !in.Accept {
		if _, err := f.CancelBulkOrder(ctx, CancelBulkOrderInput{
			BulkOrderID:    order.ID,
			Reason:         "amendment declined: " + amendment.Reason,
			IdempotencyKey: in.IdempotencyKey + ":cancel",
		}); err != nil {
			return domain.BulkOrderAmendment{}, fmt.Errorf("cancelling order %s after declined amendment: %w", order.ID, err)
		}
	}
	return decided, nil
}

// --- Dropout ------------------------------------------------------------------

// DropoutInput is an artisan withdrawing from a lot they had already
// accepted, before or during production.
type DropoutInput struct {
	LotID          uuid.UUID
	ArtisanID      uuid.UUID
	Reason         string
	IdempotencyKey string
}

// Dropout is compensation path (c)'s first half: the lot the artisan
// withdrew from moves straight to REALLOCATED — no state in between — and
// its own allocated quantity is given back to the order's pool. No other lot
// on the order is touched: the ones still ACCEPTED/IN_PRODUCTION/COMPLETED
// keep going exactly as they were. A best-effort ProposeAllocation call after
// commit tries to refill the gap immediately; if the deadline is too close
// for that to succeed, the order simply accumulates unallocated quantity for
// SettlePartial to act on later.
func (f *Fulfilment) Dropout(ctx context.Context, in DropoutInput) (domain.OrderLot, error) {
	if in.Reason == "" {
		return domain.OrderLot{}, pkgdomain.InvalidInput("reason is required")
	}

	lot, err := f.store.GetLot(ctx, in.LotID)
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("loading lot %s: %w", in.LotID, err)
	}
	if lot.ArtisanID != in.ArtisanID {
		return domain.OrderLot{}, pkgdomain.Forbidden("lot does not belong to this artisan")
	}
	if lot.State == domain.LotReallocated {
		// Replay of an already-applied dropout: idempotent no-op.
		return lot, nil
	}
	if lot.State != domain.LotAccepted && lot.State != domain.LotInProduction {
		return domain.OrderLot{}, pkgdomain.Conflict("lot cannot be dropped out of state " + string(lot.State))
	}

	var updated domain.OrderLot
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		updated, err = tx.TransitionLot(ctx, in.LotID, lot.State, domain.LotReallocated,
			domain.LotTransitionFields{DropoutReason: &in.Reason})
		if err != nil {
			return err
		}
		if _, err := tx.IncrementAllocatedQuantity(ctx, lot.BulkOrderID, -lot.Quantity); err != nil {
			return err
		}
		// No Kafka event here: the audit row above is the durable record, and
		// the reoffer this dropout triggers is a direct ProposeAllocation
		// call after commit (below), not something a consumer needs to pick
		// up from the outbox — order.lot.declined would be the wrong topic
		// to reuse, since a dropout is not a decline and WatchOrder
		// subscribers should not see it reported as one.
		return tx.RecordEvent(ctx, ids.New(), lot.BulkOrderID, &lot.ID, "LOT_DROPOUT", updated)
	})
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("recording dropout for lot %s: %w", in.LotID, err)
	}

	f.reofferAfterGiveUp(ctx, lot.BulkOrderID, in.IdempotencyKey)
	return updated, nil
}

// --- QC rework: resubmission and window expiry -------------------------------

// ResubmitForQCInput moves a lot the artisan has reworked back for
// re-inspection.
type ResubmitForQCInput struct {
	LotID          uuid.UUID
	ArtisanID      uuid.UUID
	IdempotencyKey string
}

// ResubmitForQC moves a QC_FAILED lot back to QC_PENDING within its rework
// window, so the inspector's next SubmitQC call is the rework's own
// re-inspection. Past the deadline this is a conflict — ExpireRework is the
// path out of a lapsed window, not a late resubmission racing it.
func (f *Fulfilment) ResubmitForQC(ctx context.Context, in ResubmitForQCInput) (domain.OrderLot, error) {
	lot, err := f.store.GetLot(ctx, in.LotID)
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("loading lot %s: %w", in.LotID, err)
	}
	if lot.ArtisanID != in.ArtisanID {
		return domain.OrderLot{}, pkgdomain.Forbidden("lot does not belong to this artisan")
	}
	if lot.State == domain.LotQCPending {
		// Replay of an already-applied resubmission: idempotent no-op.
		return lot, nil
	}
	if lot.State != domain.LotQCFailed {
		return domain.OrderLot{}, pkgdomain.Conflict("lot is not awaiting rework, state is " + string(lot.State))
	}
	if lot.ReworkDeadline == nil || !f.now().Before(*lot.ReworkDeadline) {
		return domain.OrderLot{}, pkgdomain.Conflict("rework window has lapsed, the lot must be reallocated instead")
	}

	var updated domain.OrderLot
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		updated, err = tx.TransitionLot(ctx, in.LotID, domain.LotQCFailed, domain.LotQCPending, domain.LotTransitionFields{})
		if err != nil {
			return err
		}
		return tx.RecordEvent(ctx, ids.New(), lot.BulkOrderID, &lot.ID, "LOT_REWORK_RESUBMITTED", updated)
	})
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("resubmitting lot %s for QC: %w", in.LotID, err)
	}
	return updated, nil
}

// ExpireReworkInput identifies a QC_FAILED lot whose window has lapsed.
type ExpireReworkInput struct {
	LotID          uuid.UUID
	IdempotencyKey string
}

// ExpireRework is compensation path (d)'s timeout half: a lot stuck in
// QC_FAILED past its rework deadline with no resubmission goes to
// REALLOCATED, withholding only that lot's own payment split (see
// RequestPaymentSplit) and reoffering its units to a fresh candidate — the
// same terminal move a second QC failure makes inside SubmitQC.
//
// ponytail: called on demand (an operator or dispute-handling workflow
// invoking it once the deadline has visibly passed), not from an automatic
// sweep — add one alongside RunReservationReaper if a lot regularly outlives
// its rework window unnoticed.
func (f *Fulfilment) ExpireRework(ctx context.Context, in ExpireReworkInput) (domain.OrderLot, error) {
	lot, err := f.store.GetLot(ctx, in.LotID)
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("loading lot %s: %w", in.LotID, err)
	}
	if lot.State == domain.LotReallocated {
		// Replay of an already-applied expiry: idempotent no-op.
		return lot, nil
	}
	if lot.State != domain.LotQCFailed {
		return domain.OrderLot{}, pkgdomain.Conflict("lot is not in rework, state is " + string(lot.State))
	}
	if lot.ReworkDeadline == nil || f.now().Before(*lot.ReworkDeadline) {
		return domain.OrderLot{}, pkgdomain.Conflict("rework window has not lapsed yet")
	}

	var updated domain.OrderLot
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		updated, err = tx.TransitionLot(ctx, in.LotID, domain.LotQCFailed, domain.LotReallocated, domain.LotTransitionFields{})
		if err != nil {
			return err
		}
		return tx.RecordEvent(ctx, ids.New(), lot.BulkOrderID, &lot.ID, "LOT_REWORK_EXPIRED", updated)
	})
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("expiring rework for lot %s: %w", in.LotID, err)
	}

	f.reofferAfterGiveUp(ctx, lot.BulkOrderID, in.IdempotencyKey)
	return updated, nil
}

// --- SettlePartial ------------------------------------------------------------

// SettlePartialInput closes out a bulk order on whatever it actually
// delivered.
type SettlePartialInput struct {
	BulkOrderID    uuid.UUID
	Reason         string
	IdempotencyKey string
}

// SettlePartial is compensation path (c)'s second half: when a dropped-out
// lot cannot be refilled before the deadline (ProposeAllocation keeps
// reporting unallocated quantity and RequiredBy has passed), the buyer is
// offered partial fulfilment — this call accepts that outcome and moves the
// order straight to COMPLETED, provided every lot still open has already
// resolved one way or the other (COMPLETED or REALLOCATED; nothing may be
// OFFERED, ACCEPTED, IN_PRODUCTION, QC_PENDING or QC_FAILED). Settlement
// (RequestPaymentSplit) then pays only the lots that actually delivered,
// exactly as it always does — this call does not touch money itself.
func (f *Fulfilment) SettlePartial(ctx context.Context, in SettlePartialInput) (domain.BulkOrder, error) {
	if in.Reason == "" {
		return domain.BulkOrder{}, pkgdomain.InvalidInput("reason is required")
	}

	order, err := f.store.GetBulkOrder(ctx, in.BulkOrderID)
	if err != nil {
		return domain.BulkOrder{}, fmt.Errorf("loading bulk order %s: %w", in.BulkOrderID, err)
	}
	if order.State == domain.BulkOrderCompleted {
		// Replay of an already-applied settlement: idempotent no-op.
		return order, nil
	}
	if order.State != domain.BulkOrderInProduction {
		return domain.BulkOrder{}, pkgdomain.Conflict("bulk order cannot be partially settled from state " + string(order.State))
	}

	lots, err := f.store.ListLotsForOrder(ctx, in.BulkOrderID)
	if err != nil {
		return domain.BulkOrder{}, fmt.Errorf("listing lots for order %s: %w", in.BulkOrderID, err)
	}
	if !allLotsSettled(lots, uuid.Nil, "") {
		return domain.BulkOrder{}, pkgdomain.Conflict("bulk order still has an open lot, cannot settle partially yet")
	}

	var settled domain.BulkOrder
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		settled, err = tx.TransitionBulkOrder(ctx, order.ID, domain.BulkOrderInProduction, domain.BulkOrderCompleted)
		if err != nil {
			return err
		}
		if err := tx.RecordEvent(ctx, ids.New(), order.ID, nil, "ORDER_PARTIALLY_SETTLED", bulkOrderEventPayload{
			BulkOrderID: settled.ID.String(), BuyerID: settled.BuyerID, ListingID: settled.ListingID.String(),
			Quantity: settled.Quantity, State: string(settled.State),
		}); err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), settled.ID.String(),
			topics.OrderFulfilmentCompleted, in.IdempotencyKey, orderConfirmedPayload(settled))
	})
	if err != nil {
		return domain.BulkOrder{}, fmt.Errorf("settling order %s partially: %w", in.BulkOrderID, err)
	}
	return settled, nil
}
