// services/collab-svc/internal/collab/service/compensation_test.go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/collab-svc/internal/collab/domain"
)

// --- compensation path (a): lot offer expiry ---------------------------------

// TestLotExpiryReoffersToNextCandidateWhenOneIsAvailable is compensation path
// (a)'s reoffer half: an expired OFFERED lot's reservation is released and a
// fresh lot is offered to a different candidate, driven by the same sweep
// RunReservationReaper uses (store.ReapExpiredReservations plus a
// ProposeAllocation re-run per affected order).
func TestLotExpiryReoffersToNextCandidateWhenOneIsAvailable(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)
	a1, a2 := uuid.New(), uuid.New()
	seedCandidates(store, craftID,
		domain.Candidate{ArtisanID: a1, AvailableUnits: 100, OnTimeRate: 1.0},
		domain.Candidate{ArtisanID: a2, AvailableUnits: 100, OnTimeRate: 1.0},
	)

	order, err := f.CreateBulkOrder(ctx, CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 30,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}

	result, err := f.ProposeAllocation(ctx, ProposeAllocationInput{BulkOrderID: order.ID, IdempotencyKey: "alloc-1"})
	if err != nil {
		t.Fatalf("ProposeAllocation: %v", err)
	}
	if len(result.Lots) != 1 || result.Lots[0].ArtisanID != a1 {
		t.Fatalf("expected the whole 30 units offered to the top-ranked candidate a1, got %+v", result.Lots)
	}
	firstLot := result.Lots[0]

	// Expire the reservation behind the offer.
	r := store.reservationsByLot[firstLot.ID]
	r.ExpiresAt = time.Now().UTC().Add(-time.Hour)
	store.reservationsByLot[firstLot.ID] = r

	// a1 is no longer in the pool by the time the sweep re-offers (its
	// capacity is now committed elsewhere) — only a2 remains.
	seedCandidates(store, craftID, domain.Candidate{ArtisanID: a2, AvailableUnits: 100, OnTimeRate: 1.0})

	released, orderIDs, err := store.ReapExpiredReservations(ctx, 100)
	if err != nil {
		t.Fatalf("ReapExpiredReservations: %v", err)
	}
	if released != 1 {
		t.Fatalf("released = %d, want 1", released)
	}
	if store.lots[firstLot.ID].State != domain.LotExpired {
		t.Errorf("first lot state = %s, want EXPIRED", store.lots[firstLot.ID].State)
	}
	if store.reservationsByLot[firstLot.ID].State != domain.ReservationReleased {
		t.Errorf("first reservation state = %s, want RELEASED", store.reservationsByLot[firstLot.ID].State)
	}
	assertHasEvent(t, store, order.ID, &firstLot.ID, "LOT_EXPIRED")

	if len(orderIDs) != 1 || orderIDs[0] != order.ID {
		t.Fatalf("expected the sweep to surface order %s, got %v", order.ID, orderIDs)
	}
	result2, err := f.ProposeAllocation(ctx, ProposeAllocationInput{BulkOrderID: orderIDs[0], IdempotencyKey: "reoffer:sweep"})
	if err != nil {
		t.Fatalf("re-run ProposeAllocation: %v", err)
	}
	if len(result2.Lots) != 1 || result2.Lots[0].ArtisanID != a2 {
		t.Fatalf("expected the freed 30 units re-offered to a2, got %+v", result2.Lots)
	}
	if result2.UnallocatedQuantity != 0 {
		t.Errorf("unallocated = %d, want 0", result2.UnallocatedQuantity)
	}
}

// TestLotExpiryPartiallyAllocatesWhenCandidatePoolIsExhausted is compensation
// path (a)'s other half: when the reoffer finds no eligible candidate left,
// the order moves to PARTIALLY_ALLOCATED rather than staying silently stuck.
func TestLotExpiryPartiallyAllocatesWhenCandidatePoolIsExhausted(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)
	a1 := uuid.New()
	seedCandidates(store, craftID, domain.Candidate{ArtisanID: a1, AvailableUnits: 100, OnTimeRate: 1.0})

	order, err := f.CreateBulkOrder(ctx, CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 30,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}
	result, err := f.ProposeAllocation(ctx, ProposeAllocationInput{BulkOrderID: order.ID, IdempotencyKey: "alloc-1"})
	if err != nil {
		t.Fatalf("ProposeAllocation: %v", err)
	}
	lot := result.Lots[0]
	r := store.reservationsByLot[lot.ID]
	r.ExpiresAt = time.Now().UTC().Add(-time.Hour)
	store.reservationsByLot[lot.ID] = r

	// The candidate pool is now empty — nobody left to reoffer to.
	seedCandidates(store, craftID)

	_, orderIDs, err := store.ReapExpiredReservations(ctx, 100)
	if err != nil {
		t.Fatalf("ReapExpiredReservations: %v", err)
	}
	result2, err := f.ProposeAllocation(ctx, ProposeAllocationInput{BulkOrderID: orderIDs[0], IdempotencyKey: "reoffer:sweep"})
	if err != nil {
		t.Fatalf("re-run ProposeAllocation: %v", err)
	}
	if len(result2.Lots) != 0 {
		t.Errorf("expected no lots offered, got %+v", result2.Lots)
	}
	if result2.UnallocatedQuantity != 30 {
		t.Errorf("unallocated = %d, want 30", result2.UnallocatedQuantity)
	}
	if got := store.orders[order.ID].State; got != domain.BulkOrderPartiallyAllocated {
		t.Errorf("order state = %s, want PARTIALLY_ALLOCATED", got)
	}
}

// --- compensation path (b): amendment on shortfall ---------------------------

func TestProposeAmendmentMovesOrderToAmendmentPending(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderPartiallyAllocated, 100, 30, 10000, f.now().AddDate(0, 0, 20))

	reduced := int32(50)
	amendment, err := f.ProposeAmendment(ctx, ProposeAmendmentInput{
		BulkOrderID: order.ID, Type: domain.AmendmentReduceQuantity,
		ProposedQuantity: &reduced, Reason: "not enough capacity", IdempotencyKey: "amend-1",
	})
	if err != nil {
		t.Fatalf("ProposeAmendment: %v", err)
	}
	if amendment.Status != domain.AmendmentPendingStatus {
		t.Errorf("amendment status = %s, want PENDING", amendment.Status)
	}
	if got := store.orders[order.ID].State; got != domain.BulkOrderAmendmentPending {
		t.Errorf("order state = %s, want AMENDMENT_PENDING", got)
	}
	assertHasEvent(t, store, order.ID, nil, "AMENDMENT_PROPOSED")
	assertHasOutbox(t, store, "order.amendment.proposed")
}

func TestDecideAmendmentAcceptReplansOrder(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderPartiallyAllocated, 100, 20, 10000, f.now().AddDate(0, 0, 20))
	reduced := int32(30)
	amendment, err := f.ProposeAmendment(ctx, ProposeAmendmentInput{
		BulkOrderID: order.ID, Type: domain.AmendmentReduceQuantity,
		ProposedQuantity: &reduced, Reason: "buyer agreed to less", IdempotencyKey: "amend-1",
	})
	if err != nil {
		t.Fatalf("ProposeAmendment: %v", err)
	}

	// Between the proposal and the buyer's decision, more lots were accepted
	// and now cover the reduced quantity outright.
	withMoreAccepted := store.orders[order.ID]
	withMoreAccepted.AllocatedQuantity = 30
	store.orders[order.ID] = withMoreAccepted

	decided, err := f.DecideAmendment(ctx, DecideAmendmentInput{AmendmentID: amendment.ID, Accept: true, IdempotencyKey: "decide-1"})
	if err != nil {
		t.Fatalf("DecideAmendment: %v", err)
	}
	if decided.Status != domain.AmendmentAccepted {
		t.Errorf("amendment status = %s, want ACCEPTED", decided.Status)
	}
	updated := store.orders[order.ID]
	if updated.Quantity != 30 {
		t.Errorf("order quantity = %d, want 30", updated.Quantity)
	}
	if updated.TotalValuePaise != 30*10000 {
		t.Errorf("order total = %d, want %d", updated.TotalValuePaise, 30*10000)
	}
	// AllocatedQuantity (30) already covers the reduced quantity (30), so the
	// order goes straight to CONFIRMED rather than back to PARTIALLY_ALLOCATED.
	if updated.State != domain.BulkOrderConfirmed {
		t.Errorf("order state = %s, want CONFIRMED", updated.State)
	}
	assertHasEvent(t, store, order.ID, nil, "ORDER_QUANTITY_REDUCED")
}

func TestDecideAmendmentDeclineCancelsAndReleasesAllReservations(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderPartiallyAllocated, 100, 30, 10000, f.now().AddDate(0, 0, 20))
	// One still-open OFFERED lot with a live HELD reservation, standing in
	// for the shortfall this amendment is meant to resolve.
	lotID, reservationID := uuid.New(), uuid.New()
	store.lots[lotID] = domain.OrderLot{ID: lotID, BulkOrderID: order.ID, State: domain.LotOffered, Quantity: 20}
	store.reservationsByLot[lotID] = domain.CapacityReservation{ID: reservationID, LotID: lotID, State: domain.ReservationHeld}

	reduced := int32(50)
	amendment, err := f.ProposeAmendment(ctx, ProposeAmendmentInput{
		BulkOrderID: order.ID, Type: domain.AmendmentReduceQuantity,
		ProposedQuantity: &reduced, Reason: "shortfall", IdempotencyKey: "amend-1",
	})
	if err != nil {
		t.Fatalf("ProposeAmendment: %v", err)
	}

	decided, err := f.DecideAmendment(ctx, DecideAmendmentInput{AmendmentID: amendment.ID, Accept: false, IdempotencyKey: "decide-1"})
	if err != nil {
		t.Fatalf("DecideAmendment: %v", err)
	}
	if decided.Status != domain.AmendmentDeclined {
		t.Errorf("amendment status = %s, want DECLINED", decided.Status)
	}
	if got := store.orders[order.ID].State; got != domain.BulkOrderCancelled {
		t.Errorf("order state = %s, want CANCELLED", got)
	}
	if got := store.reservationsByLot[lotID].State; got != domain.ReservationReleased {
		t.Errorf("reservation state = %s, want RELEASED", got)
	}
	if got := store.lots[lotID].State; got != domain.LotDeclined {
		t.Errorf("lot state = %s, want DECLINED", got)
	}
	assertHasEvent(t, store, order.ID, nil, "ORDER_CANCELLED")
}

// --- compensation path (c): artisan dropout -----------------------------------

func TestDropoutReallocatesOnlyThatLotAndLeavesSiblingsUntouched(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderInProduction, 100, 50, 10000, f.now().AddDate(0, 0, 20))
	artisanA, artisanB := uuid.New(), uuid.New()
	lotA := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, ArtisanID: artisanA, Quantity: 20, State: domain.LotAccepted, ProgressPct: 10}
	lotB := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, ArtisanID: artisanB, Quantity: 30, State: domain.LotInProduction, ProgressPct: 40}
	store.lots[lotA.ID] = lotA
	store.lots[lotB.ID] = lotB

	updated, err := f.Dropout(ctx, DropoutInput{LotID: lotA.ID, ArtisanID: artisanA, Reason: "family emergency", IdempotencyKey: "drop-1"})
	if err != nil {
		t.Fatalf("Dropout: %v", err)
	}
	if updated.State != domain.LotReallocated {
		t.Errorf("lot A state = %s, want REALLOCATED", updated.State)
	}
	if updated.DropoutReason == nil || *updated.DropoutReason != "family emergency" {
		t.Errorf("dropout reason = %v, want it recorded", updated.DropoutReason)
	}
	if got := store.orders[order.ID].AllocatedQuantity; got != 30 {
		t.Errorf("order allocated quantity = %d, want 30 (50 - lot A's 20)", got)
	}
	assertHasEvent(t, store, order.ID, &lotA.ID, "LOT_DROPOUT")

	// Lot B: completely untouched.
	stillB := store.lots[lotB.ID]
	if stillB.State != domain.LotInProduction || stillB.Quantity != 30 || stillB.ProgressPct != 40 {
		t.Errorf("lot B was modified by lot A's dropout: %+v", stillB)
	}
}

func TestDropoutIsIdempotent(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderInProduction, 100, 20, 10000, f.now().AddDate(0, 0, 20))
	artisanA := uuid.New()
	lotA := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, ArtisanID: artisanA, Quantity: 20, State: domain.LotAccepted}
	store.lots[lotA.ID] = lotA

	in := DropoutInput{LotID: lotA.ID, ArtisanID: artisanA, Reason: "family emergency", IdempotencyKey: "drop-1"}
	if _, err := f.Dropout(ctx, in); err != nil {
		t.Fatalf("first Dropout: %v", err)
	}
	if _, err := f.Dropout(ctx, in); err != nil {
		t.Fatalf("replayed Dropout: %v", err)
	}
	if got := store.orders[order.ID].AllocatedQuantity; got != 0 {
		t.Errorf("allocated quantity decremented twice: got %d, want 0", got)
	}
	events := 0
	for _, e := range store.events {
		if e.eventType == "LOT_DROPOUT" {
			events++
		}
	}
	if events != 1 {
		t.Errorf("expected exactly one LOT_DROPOUT event across the replay, got %d", events)
	}
}

func TestSettlePartialSettlesOnlyDeliveredLots(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderInProduction, 100, 50, 10000, f.now().AddDate(0, 0, 20))
	delivered := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, Quantity: 30, State: domain.LotCompleted}
	gaveUp := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, Quantity: 20, State: domain.LotReallocated}
	store.lots[delivered.ID] = delivered
	store.lots[gaveUp.ID] = gaveUp

	settled, err := f.SettlePartial(ctx, SettlePartialInput{BulkOrderID: order.ID, Reason: "deadline passed, no capacity left", IdempotencyKey: "settle-1"})
	if err != nil {
		t.Fatalf("SettlePartial: %v", err)
	}
	if settled.State != domain.BulkOrderCompleted {
		t.Errorf("order state = %s, want COMPLETED", settled.State)
	}
	assertHasEvent(t, store, order.ID, nil, "ORDER_PARTIALLY_SETTLED")
	assertHasOutbox(t, store, "order.fulfilment.completed")
}

func TestSettlePartialRejectsWhileALotIsStillOpen(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderInProduction, 100, 50, 10000, f.now().AddDate(0, 0, 20))
	openLotID := uuid.New()
	store.lots[openLotID] = domain.OrderLot{ID: openLotID, BulkOrderID: order.ID, Quantity: 20, State: domain.LotAccepted}

	_, err := f.SettlePartial(ctx, SettlePartialInput{BulkOrderID: order.ID, Reason: "too soon", IdempotencyKey: "settle-1"})
	if !errors.Is(err, pkgdomain.ErrConflict) {
		t.Fatalf("expected ErrConflict with an open lot, got %v", err)
	}
}

// --- compensation path (d): QC rework -----------------------------------------

func TestSubmitQCFailureOpensReworkWindow(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderInProduction, 100, 20, 10000, f.now().AddDate(0, 0, 20))
	lot := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, Quantity: 20, State: domain.LotQCPending}
	store.lots[lot.ID] = lot

	updated, err := f.SubmitQC(ctx, SubmitQCInput{
		LotID: lot.ID, InspectorID: "inspector-1", Passed: false,
		Defects:        []domain.QCDefect{{Code: "stitch", Severity: domain.DefectMajor, AffectedUnits: 2}},
		IdempotencyKey: "qc-1",
	})
	if err != nil {
		t.Fatalf("SubmitQC: %v", err)
	}
	if updated.State != domain.LotQCFailed {
		t.Fatalf("lot state = %s, want QC_FAILED", updated.State)
	}
	wantDeadline := f.now().Add(defaultReworkWindow)
	if updated.ReworkDeadline == nil || !updated.ReworkDeadline.Equal(wantDeadline) {
		t.Errorf("rework deadline = %v, want %v", updated.ReworkDeadline, wantDeadline)
	}
	assertHasEvent(t, store, order.ID, &lot.ID, "QC_RECORDED")
}

func TestSubmitQCSecondFailureReallocatesAndWithholdsOnlyThatLotsSplit(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	listingID := uuid.New()
	store.listings[listingID] = domain.ListingInfo{UnitPricePaise: 10000, Owner: domain.ListingOwner{OwnerType: domain.ListingOwnerArtisan}}
	order := domain.BulkOrder{ID: uuid.New(), BuyerID: "buyer-1", ListingID: listingID, Quantity: 50, State: domain.BulkOrderInProduction}
	store.orders[order.ID] = order

	troubled := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, ArtisanID: uuid.New(), Quantity: 20, UnitPricePaise: 10000, LotValuePaise: 200000, State: domain.LotQCPending}
	sibling := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, ArtisanID: uuid.New(), Quantity: 30, UnitPricePaise: 10000, LotValuePaise: 300000, State: domain.LotCompleted}
	store.lots[troubled.ID] = troubled
	store.lots[sibling.ID] = sibling

	// First failure: non-critical, opens the rework window.
	if _, err := f.SubmitQC(ctx, SubmitQCInput{LotID: troubled.ID, InspectorID: "i1", Passed: false,
		Defects: []domain.QCDefect{{Code: "colour", Severity: domain.DefectMinor}}, IdempotencyKey: "qc-1"}); err != nil {
		t.Fatalf("first SubmitQC: %v", err)
	}
	if _, err := f.ResubmitForQC(ctx, ResubmitForQCInput{LotID: troubled.ID, ArtisanID: troubled.ArtisanID, IdempotencyKey: "resub-1"}); err != nil {
		t.Fatalf("ResubmitForQC: %v", err)
	}

	// Second failure: reallocated even though this defect alone is minor.
	updated, err := f.SubmitQC(ctx, SubmitQCInput{LotID: troubled.ID, InspectorID: "i1", Passed: false,
		Defects: []domain.QCDefect{{Code: "colour", Severity: domain.DefectMinor}}, IdempotencyKey: "qc-2"})
	if err != nil {
		t.Fatalf("second SubmitQC: %v", err)
	}
	if updated.State != domain.LotReallocated {
		t.Fatalf("lot state = %s, want REALLOCATED after a second failure", updated.State)
	}
	if got := store.lots[sibling.ID].State; got != domain.LotCompleted {
		t.Errorf("sibling lot state = %s, want untouched COMPLETED", got)
	}

	// The reallocated lot's split is withheld: only the sibling gets paid.
	split, err := f.RequestPaymentSplit(ctx, RequestPaymentSplitInput{BulkOrderID: order.ID, CommissionPct: 10, IdempotencyKey: "pay-1"})
	if err != nil {
		t.Fatalf("RequestPaymentSplit: %v", err)
	}
	if len(split.Lines) != 1 || split.Lines[0].LotID != sibling.ID {
		t.Fatalf("expected exactly one payment line for the sibling lot, got %+v", split.Lines)
	}
	if split.GrossTotalPaise != sibling.LotValuePaise {
		t.Errorf("gross total = %d, want %d (the troubled lot's value must not appear)", split.GrossTotalPaise, sibling.LotValuePaise)
	}
}

func TestExpireReworkReallocatesLapsedLotAndLeavesSiblingsUntouched(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderInProduction, 100, 50, 10000, f.now().AddDate(0, 0, 20))
	lapsed := f.now().Add(-time.Hour)
	troubled := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, Quantity: 20, State: domain.LotQCFailed, ReworkDeadline: &lapsed}
	sibling := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, Quantity: 30, State: domain.LotInProduction}
	store.lots[troubled.ID] = troubled
	store.lots[sibling.ID] = sibling

	updated, err := f.ExpireRework(ctx, ExpireReworkInput{LotID: troubled.ID, IdempotencyKey: "expire-1"})
	if err != nil {
		t.Fatalf("ExpireRework: %v", err)
	}
	if updated.State != domain.LotReallocated {
		t.Errorf("lot state = %s, want REALLOCATED", updated.State)
	}
	assertHasEvent(t, store, order.ID, &troubled.ID, "LOT_REWORK_EXPIRED")
	if got := store.lots[sibling.ID].State; got != domain.LotInProduction {
		t.Errorf("sibling lot state = %s, want untouched IN_PRODUCTION", got)
	}
}

func TestExpireReworkRejectsBeforeDeadline(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	order := seedBulkOrder(store, domain.BulkOrderInProduction, 100, 20, 10000, f.now().AddDate(0, 0, 20))
	future := f.now().Add(time.Hour)
	lot := domain.OrderLot{ID: uuid.New(), BulkOrderID: order.ID, Quantity: 20, State: domain.LotQCFailed, ReworkDeadline: &future}
	store.lots[lot.ID] = lot

	_, err := f.ExpireRework(ctx, ExpireReworkInput{LotID: lot.ID, IdempotencyKey: "expire-1"})
	if !errors.Is(err, pkgdomain.ErrConflict) {
		t.Fatalf("expected ErrConflict before the window lapses, got %v", err)
	}
}

// --- test helpers --------------------------------------------------------------

func seedBulkOrder(store *fakeStore, state domain.BulkOrderState, quantity, allocatedQuantity int32, unitPricePaise int64, requiredBy time.Time) domain.BulkOrder {
	order := domain.BulkOrder{
		ID: uuid.New(), BuyerID: "buyer-1", ListingID: uuid.New(),
		Quantity: quantity, AllocatedQuantity: allocatedQuantity, UnitPricePaise: unitPricePaise,
		TotalValuePaise: unitPricePaise * int64(quantity), RequiredBy: requiredBy, State: state,
	}
	store.orders[order.ID] = order
	return order
}

func assertHasEvent(t *testing.T, store *fakeStore, orderID uuid.UUID, lotID *uuid.UUID, eventType string) {
	t.Helper()
	for _, e := range store.events {
		if e.orderID != orderID || e.eventType != eventType {
			continue
		}
		if lotID == nil {
			return
		}
		if e.lotID != nil && *e.lotID == *lotID {
			return
		}
	}
	t.Errorf("expected an event %s for order %s, found none in %+v", eventType, orderID, store.events)
}

func assertHasOutbox(t *testing.T, store *fakeStore, topic string) {
	t.Helper()
	for _, row := range store.outbox {
		if row.topic == topic {
			return
		}
	}
	t.Errorf("expected an outbox row on topic %s, found none in %+v", topic, store.outbox)
}
