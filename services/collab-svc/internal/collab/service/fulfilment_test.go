// services/collab-svc/internal/collab/service/fulfilment_test.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"

	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/domain"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestFulfilment(store *fakeStore) *Fulfilment {
	f := NewFulfilment(store, time.Hour, testLogger())
	f.now = func() time.Time { return time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC) }
	return f
}

func seedListing(t *testing.T, store *fakeStore, listingID, craftID uuid.UUID, leadTimeDays int32, unitPricePaise int64) {
	t.Helper()
	store.listings[listingID] = domain.ListingInfo{
		ProductID:           uuid.New(),
		CraftID:             craftID,
		UnitPricePaise:      unitPricePaise,
		TypicalLeadTimeDays: leadTimeDays,
	}
}

func seedCandidates(store *fakeStore, craftID uuid.UUID, cands ...domain.Candidate) {
	store.candidates[craftID] = cands
}

// TestCreateBulkOrderRejectsInfeasibleDeadline is the acceptance criterion:
// deadline feasibility rejection.
func TestCreateBulkOrderRejectsInfeasibleDeadline(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 30, 10000) // 30-day lead time

	_, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 10,
		RequiredBy:     f.now().AddDate(0, 0, 5), // far too soon
		IdempotencyKey: "key-1",
	})
	if !errors.Is(err, pkgdomain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for an infeasible deadline, got %v", err)
	}
}

func TestCreateBulkOrderAcceptsFeasibleDeadline(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)

	order, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 10,
		RequiredBy:     f.now().AddDate(0, 0, 30),
		IdempotencyKey: "key-1",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder returned error: %v", err)
	}
	if order.State != domain.BulkOrderAllocating {
		t.Errorf("state = %s, want ALLOCATING", order.State)
	}
	if len(store.outbox) != 1 || store.outbox[0].topic != "order.bulk.requested" {
		t.Errorf("expected one order.bulk.requested outbox row, got %+v", store.outbox)
	}
}

// TestCreateBulkOrderIsIdempotent replays the same request twice with the
// same idempotency key and expects one order, not two.
func TestCreateBulkOrderIsIdempotent(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)

	in := CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 10,
		RequiredBy:     f.now().AddDate(0, 0, 30),
		IdempotencyKey: "replayed-key",
	}
	first, err := f.CreateBulkOrder(context.Background(), in)
	if err != nil {
		t.Fatalf("first call returned error: %v", err)
	}
	second, err := f.CreateBulkOrder(context.Background(), in)
	if err != nil {
		t.Fatalf("replayed call returned error: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("replay created a second order: first %s, second %s", first.ID, second.ID)
	}
	if len(store.orders) != 1 {
		t.Errorf("expected exactly one stored order, got %d", len(store.orders))
	}
	// The outbox row is only enqueued once — a real InsertOutbox would also
	// dedupe on (topic, idempotency_key), but this asserts the service layer
	// does not even attempt a second enqueue on a replay.
	if len(store.outbox) != 1 {
		t.Errorf("expected exactly one outbox row after a replay, got %d", len(store.outbox))
	}
}

// TestProposeAllocationCoversQuantityExactly is the acceptance criterion: a
// 500-unit order across artisans with capacity 30-80 allocates fully with no
// lot below the minimum, exercised through the full service (not just the
// domain algorithm), including reservations and outbox events.
func TestProposeAllocationCoversQuantityExactly(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)

	order, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 500,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}

	caps := []int32{80, 75, 70, 65, 60, 55, 50, 45, 40, 35, 30}
	var cands []domain.Candidate
	for _, c := range caps {
		cands = append(cands, domain.Candidate{ArtisanID: uuid.New(), AvailableUnits: c, OnTimeRate: 1.0})
	}
	seedCandidates(store, craftID, cands...)

	result, err := f.ProposeAllocation(context.Background(), ProposeAllocationInput{
		BulkOrderID: order.ID, IdempotencyKey: "alloc-key",
	})
	if err != nil {
		t.Fatalf("ProposeAllocation: %v", err)
	}

	var total int32
	for _, l := range result.Lots {
		if l.Quantity < domain.MinLotSize {
			t.Errorf("lot %s has quantity %d, below MinLotSize", l.ArtisanID, l.Quantity)
		}
		total += l.Quantity
	}
	if total != 500 {
		t.Errorf("allocated total = %d, want 500", total)
	}
	if result.UnallocatedQuantity != 0 {
		t.Errorf("unallocated = %d, want 0", result.UnallocatedQuantity)
	}

	// Every lot has a HELD reservation.
	for _, l := range result.Lots {
		r, ok := store.reservationsByLot[l.ID]
		if !ok || r.State != domain.ReservationHeld {
			t.Errorf("lot %s missing a HELD reservation", l.ID)
		}
	}

	// One order.lot.offered outbox row per lot, plus the one from creation.
	offered := 0
	for _, row := range store.outbox {
		if row.topic == "order.lot.offered" {
			offered++
		}
	}
	if offered != len(result.Lots) {
		t.Errorf("expected %d order.lot.offered rows, got %d", len(result.Lots), offered)
	}
}

// TestRespondToLotDeclineReleasesReservation is the acceptance criterion:
// reservations released on decline.
func TestRespondToLotDeclineReleasesReservation(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)

	order, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 100,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}

	artisanA := uuid.New()
	seedCandidates(store, craftID,
		domain.Candidate{ArtisanID: artisanA, AvailableUnits: 100, OnTimeRate: 1.0},
	)
	allocResult, err := f.ProposeAllocation(context.Background(), ProposeAllocationInput{
		BulkOrderID: order.ID, IdempotencyKey: "alloc-1",
	})
	if err != nil || len(allocResult.Lots) != 1 {
		t.Fatalf("ProposeAllocation: %v, lots=%+v", err, allocResult.Lots)
	}
	lot := allocResult.Lots[0]
	reservationID := store.reservationsByLot[lot.ID].ID

	declineReason := "cannot meet the timeline"
	declined, err := f.RespondToLot(context.Background(), RespondToLotInput{
		LotID: lot.ID, ArtisanID: artisanA, Accept: false,
		DeclineReason: &declineReason, IdempotencyKey: "respond-1",
	})
	if err != nil {
		t.Fatalf("RespondToLot decline: %v", err)
	}
	if declined.State != domain.LotDeclined {
		t.Errorf("lot state = %s, want DECLINED", declined.State)
	}
	if got := store.reservationsByLot[reservationID].State; got != domain.ReservationReleased {
		t.Errorf("reservation state = %s, want RELEASED", got)
	}
}

// TestRespondToLotDeclineReoffersToNextCandidate confirms a decline's
// automatic re-offer pass picks up a previously-unused candidate for the
// freed capacity.
func TestRespondToLotDeclineReoffersToNextCandidate(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)

	order, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 100,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}

	artisanA, artisanB := uuid.New(), uuid.New()
	// Only artisan A is visible for the first allocation pass.
	seedCandidates(store, craftID, domain.Candidate{ArtisanID: artisanA, AvailableUnits: 100, OnTimeRate: 1.0})
	allocResult, err := f.ProposeAllocation(context.Background(), ProposeAllocationInput{
		BulkOrderID: order.ID, IdempotencyKey: "alloc-1",
	})
	if err != nil || len(allocResult.Lots) != 1 {
		t.Fatalf("ProposeAllocation: %v", err)
	}
	lot := allocResult.Lots[0]

	// Now B becomes available too (e.g. joined the ontology allowlist between
	// passes); the reoffer triggered by A's decline should find them.
	seedCandidates(store, craftID,
		domain.Candidate{ArtisanID: artisanB, AvailableUnits: 100, OnTimeRate: 1.0},
	)

	declineReason := "double-booked"
	_, err = f.RespondToLot(context.Background(), RespondToLotInput{
		LotID: lot.ID, ArtisanID: artisanA, Accept: false,
		DeclineReason: &declineReason, IdempotencyKey: "respond-1",
	})
	if err != nil {
		t.Fatalf("RespondToLot decline: %v", err)
	}

	var offeredToB bool
	for _, l := range store.lots {
		if l.ArtisanID == artisanB && l.State == domain.LotOffered {
			offeredToB = true
		}
	}
	if !offeredToB {
		t.Error("expected the decline's re-offer pass to have offered a lot to artisan B")
	}
}

// TestRespondToLotDoubleAcceptIsIdempotent is the acceptance criterion:
// double-accept is idempotent.
func TestRespondToLotDoubleAcceptIsIdempotent(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)

	order, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 50,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}

	artisanA := uuid.New()
	seedCandidates(store, craftID, domain.Candidate{ArtisanID: artisanA, AvailableUnits: 50, OnTimeRate: 1.0})
	allocResult, err := f.ProposeAllocation(context.Background(), ProposeAllocationInput{
		BulkOrderID: order.ID, IdempotencyKey: "alloc-1",
	})
	if err != nil || len(allocResult.Lots) != 1 {
		t.Fatalf("ProposeAllocation: %v", err)
	}
	lot := allocResult.Lots[0]
	shipDate := f.now().AddDate(0, 0, 20)

	first, err := f.RespondToLot(context.Background(), RespondToLotInput{
		LotID: lot.ID, ArtisanID: artisanA, Accept: true,
		PromisedShipDate: &shipDate, IdempotencyKey: "respond-1",
	})
	if err != nil {
		t.Fatalf("first accept: %v", err)
	}
	if first.State != domain.LotAccepted {
		t.Fatalf("state = %s, want ACCEPTED", first.State)
	}

	second, err := f.RespondToLot(context.Background(), RespondToLotInput{
		LotID: lot.ID, ArtisanID: artisanA, Accept: true,
		PromisedShipDate: &shipDate, IdempotencyKey: "respond-1-retry",
	})
	if err != nil {
		t.Fatalf("second (replayed) accept returned an error instead of being idempotent: %v", err)
	}
	if second.State != domain.LotAccepted {
		t.Errorf("replayed accept state = %s, want ACCEPTED", second.State)
	}

	// The order's allocated_quantity must not have been double-incremented.
	updatedOrder := store.orders[order.ID]
	if updatedOrder.AllocatedQuantity != 50 {
		t.Errorf("allocated_quantity = %d, want 50 (double-accept must not double-count)", updatedOrder.AllocatedQuantity)
	}
	if updatedOrder.State != domain.BulkOrderConfirmed {
		t.Errorf("order state = %s, want CONFIRMED once its only lot is accepted", updatedOrder.State)
	}
}

func TestRespondToLotRejectsWrongArtisan(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)
	order, _ := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 20,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	artisanA, artisanB := uuid.New(), uuid.New()
	seedCandidates(store, craftID, domain.Candidate{ArtisanID: artisanA, AvailableUnits: 20, OnTimeRate: 1.0})
	allocResult, _ := f.ProposeAllocation(context.Background(), ProposeAllocationInput{
		BulkOrderID: order.ID, IdempotencyKey: "alloc-1",
	})
	lot := allocResult.Lots[0]
	shipDate := f.now().AddDate(0, 0, 20)

	_, err := f.RespondToLot(context.Background(), RespondToLotInput{
		LotID: lot.ID, ArtisanID: artisanB, Accept: true,
		PromisedShipDate: &shipDate, IdempotencyKey: "respond-1",
	})
	if !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden when a different artisan responds, got %v", err)
	}
}

func TestReapExpiredReservationsExpiresLotAndReleasesCapacity(t *testing.T) {
	t.Parallel()
	store := newFakeStore()

	lotID, reservationID := uuid.New(), uuid.New()
	store.lots[lotID] = domain.OrderLot{ID: lotID, State: domain.LotOffered}
	store.reservationsByLot[lotID] = domain.CapacityReservation{
		ID: reservationID, LotID: lotID, State: domain.ReservationHeld,
		ExpiresAt: time.Now().UTC().Add(-time.Hour), // already expired
	}

	released, _, err := store.ReapExpiredReservations(context.Background(), 100)
	if err != nil {
		t.Fatalf("ReapExpiredReservations: %v", err)
	}
	if released != 1 {
		t.Fatalf("released = %d, want 1", released)
	}
	if got := store.reservationsByLot[lotID].State; got != domain.ReservationReleased {
		t.Errorf("reservation state = %s, want RELEASED", got)
	}
	if got := store.lots[lotID].State; got != domain.LotExpired {
		t.Errorf("lot state = %s, want EXPIRED", got)
	}
}

func TestReapExpiredReservationsLeavesLiveReservationsAlone(t *testing.T) {
	t.Parallel()
	store := newFakeStore()

	lotID, reservationID := uuid.New(), uuid.New()
	store.lots[lotID] = domain.OrderLot{ID: lotID, State: domain.LotOffered}
	store.reservationsByLot[lotID] = domain.CapacityReservation{
		ID: reservationID, LotID: lotID, State: domain.ReservationHeld,
		ExpiresAt: time.Now().UTC().Add(time.Hour), // not yet expired
	}

	released, _, err := store.ReapExpiredReservations(context.Background(), 100)
	if err != nil {
		t.Fatalf("ReapExpiredReservations: %v", err)
	}
	if released != 0 {
		t.Errorf("released = %d, want 0", released)
	}
	if got := store.reservationsByLot[lotID].State; got != domain.ReservationHeld {
		t.Errorf("reservation state = %s, want still HELD", got)
	}
}

// TestCancelBulkOrderReleasesOpenLots covers cancellation releasing both an
// OFFERED lot's reservation (and moving the lot to DECLINED) and an ACCEPTED
// lot's reservation, while leaving the order CANCELLED.
func TestCancelBulkOrderReleasesOpenLots(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)
	order, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 100,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}

	artisanA, artisanB := uuid.New(), uuid.New()
	seedCandidates(store, craftID,
		domain.Candidate{ArtisanID: artisanA, AvailableUnits: 50, OnTimeRate: 1.0},
		domain.Candidate{ArtisanID: artisanB, AvailableUnits: 50, OnTimeRate: 1.0},
	)
	allocResult, err := f.ProposeAllocation(context.Background(), ProposeAllocationInput{
		BulkOrderID: order.ID, IdempotencyKey: "alloc-1",
	})
	if err != nil || len(allocResult.Lots) != 2 {
		t.Fatalf("ProposeAllocation: %v, lots=%+v", err, allocResult.Lots)
	}

	// Accept one lot, leave the other OFFERED.
	var acceptedLot, offeredLot domain.OrderLot
	for _, l := range allocResult.Lots {
		if l.ArtisanID == artisanA {
			acceptedLot = l
		} else {
			offeredLot = l
		}
	}
	shipDate := f.now().AddDate(0, 0, 20)
	if _, err := f.RespondToLot(context.Background(), RespondToLotInput{
		LotID: acceptedLot.ID, ArtisanID: artisanA, Accept: true,
		PromisedShipDate: &shipDate, IdempotencyKey: "respond-1",
	}); err != nil {
		t.Fatalf("RespondToLot accept: %v", err)
	}

	result, err := f.CancelBulkOrder(context.Background(), CancelBulkOrderInput{
		BulkOrderID: order.ID, Reason: "buyer changed their mind", IdempotencyKey: "cancel-1",
	})
	if err != nil {
		t.Fatalf("CancelBulkOrder: %v", err)
	}
	if result.Order.State != domain.BulkOrderCancelled {
		t.Errorf("order state = %s, want CANCELLED", result.Order.State)
	}
	if len(result.ReleasedLotIDs) != 2 {
		t.Errorf("released lot count = %d, want 2", len(result.ReleasedLotIDs))
	}
	if got := store.lots[offeredLot.ID].State; got != domain.LotDeclined {
		t.Errorf("offered lot state = %s, want DECLINED", got)
	}
	if got := store.reservationsByLot[offeredLot.ID].State; got != domain.ReservationReleased {
		t.Errorf("offered lot reservation state = %s, want RELEASED", got)
	}
	if got := store.reservationsByLot[acceptedLot.ID].State; got != domain.ReservationReleased {
		t.Errorf("accepted lot reservation state = %s, want RELEASED", got)
	}
}

// TestCancelBulkOrderIsIdempotent replays a cancellation and expects a
// no-op, not a second set of release events.
func TestCancelBulkOrderIsIdempotent(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)
	order, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 20,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}

	first, err := f.CancelBulkOrder(context.Background(), CancelBulkOrderInput{
		BulkOrderID: order.ID, Reason: "test", IdempotencyKey: "cancel-1",
	})
	if err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	second, err := f.CancelBulkOrder(context.Background(), CancelBulkOrderInput{
		BulkOrderID: order.ID, Reason: "test", IdempotencyKey: "cancel-2",
	})
	if err != nil {
		t.Fatalf("replayed cancel returned an error instead of being idempotent: %v", err)
	}
	if second.Order.State != domain.BulkOrderCancelled || first.Order.State != domain.BulkOrderCancelled {
		t.Errorf("expected both calls to report CANCELLED, got %s and %s", first.Order.State, second.Order.State)
	}
}

func TestCancelBulkOrderRejectsInProduction(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	orderID := uuid.New()
	store.orders[orderID] = domain.BulkOrder{ID: orderID, BuyerID: "buyer-1", State: domain.BulkOrderInProduction}

	_, err := f.CancelBulkOrder(context.Background(), CancelBulkOrderInput{
		BulkOrderID: orderID, Reason: "too late", IdempotencyKey: "cancel-1",
	})
	if !errors.Is(err, pkgdomain.ErrConflict) {
		t.Fatalf("expected ErrConflict once production has started, got %v", err)
	}
}

// TestHandleLotDeclinedReoffersCapacity is the durability fix's own test:
// the decline event handler (not RespondToLot itself) is what performs the
// reoffer.
func TestHandleLotDeclinedReoffersCapacity(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)
	order, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 100,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}

	artisanA, artisanB := uuid.New(), uuid.New()
	seedCandidates(store, craftID, domain.Candidate{ArtisanID: artisanA, AvailableUnits: 100, OnTimeRate: 1.0})
	allocResult, err := f.ProposeAllocation(context.Background(), ProposeAllocationInput{
		BulkOrderID: order.ID, IdempotencyKey: "alloc-1",
	})
	if err != nil || len(allocResult.Lots) != 1 {
		t.Fatalf("ProposeAllocation: %v", err)
	}
	lot := allocResult.Lots[0]

	seedCandidates(store, craftID, domain.Candidate{ArtisanID: artisanB, AvailableUnits: 100, OnTimeRate: 1.0})

	declineReason := "unavailable"
	if _, err := f.RespondToLot(context.Background(), RespondToLotInput{
		LotID: lot.ID, ArtisanID: artisanA, Accept: false,
		DeclineReason: &declineReason, IdempotencyKey: "respond-1",
	}); err != nil {
		t.Fatalf("RespondToLot decline: %v", err)
	}

	// Nothing should have been re-offered yet — RespondToLot no longer does
	// this inline (see the durability fix's own comment on RespondToLot).
	offeredToB := false
	for _, l := range store.lots {
		if l.ArtisanID == artisanB && l.State == domain.LotOffered {
			offeredToB = true
		}
	}
	if offeredToB {
		t.Fatal("RespondToLot must not reoffer inline; that is HandleLotDeclined's job")
	}

	// Simulate the outbox-published event's payload reaching the consumer.
	payload, err := json.Marshal(map[string]string{"bulk_order_id": order.ID.String()})
	if err != nil {
		t.Fatalf("marshalling test payload: %v", err)
	}
	if err := f.HandleLotDeclined(context.Background(), payload); err != nil {
		t.Fatalf("HandleLotDeclined: %v", err)
	}

	offeredToB = false
	for _, l := range store.lots {
		if l.ArtisanID == artisanB && l.State == domain.LotOffered {
			offeredToB = true
		}
	}
	if !offeredToB {
		t.Error("expected HandleLotDeclined to have offered a lot to artisan B")
	}
}

// TestHandleLotDeclinedIsSafeOnRedelivery simulates Kafka's at-least-once
// delivery calling the handler twice for the same decline and expects the
// second call to be a rejected no-op, not a duplicate offer.
func TestHandleLotDeclinedIsSafeOnRedelivery(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)

	listingID, craftID := uuid.New(), uuid.New()
	seedListing(t, store, listingID, craftID, 14, 10000)
	order, err := f.CreateBulkOrder(context.Background(), CreateBulkOrderInput{
		BuyerID: "buyer-1", ListingID: listingID, Quantity: 50,
		RequiredBy: f.now().AddDate(0, 0, 30), IdempotencyKey: "order-key",
	})
	if err != nil {
		t.Fatalf("CreateBulkOrder: %v", err)
	}
	artisanA, artisanB := uuid.New(), uuid.New()
	seedCandidates(store, craftID, domain.Candidate{ArtisanID: artisanA, AvailableUnits: 50, OnTimeRate: 1.0})
	allocResult, err := f.ProposeAllocation(context.Background(), ProposeAllocationInput{
		BulkOrderID: order.ID, IdempotencyKey: "alloc-1",
	})
	if err != nil || len(allocResult.Lots) != 1 {
		t.Fatalf("ProposeAllocation: %v", err)
	}
	lot := allocResult.Lots[0]
	seedCandidates(store, craftID, domain.Candidate{ArtisanID: artisanB, AvailableUnits: 50, OnTimeRate: 1.0})

	declineReason := "unavailable"
	if _, err := f.RespondToLot(context.Background(), RespondToLotInput{
		LotID: lot.ID, ArtisanID: artisanA, Accept: false,
		DeclineReason: &declineReason, IdempotencyKey: "respond-1",
	}); err != nil {
		t.Fatalf("RespondToLot decline: %v", err)
	}

	payload, _ := json.Marshal(map[string]string{"bulk_order_id": order.ID.String()})
	if err := f.HandleLotDeclined(context.Background(), payload); err != nil {
		t.Fatalf("first HandleLotDeclined: %v", err)
	}
	if err := f.HandleLotDeclined(context.Background(), payload); err != nil {
		t.Fatalf("redelivered HandleLotDeclined must not error: %v", err)
	}

	var offeredToBCount int
	for _, l := range store.lots {
		if l.ArtisanID == artisanB && l.State == domain.LotOffered {
			offeredToBCount++
		}
	}
	if offeredToBCount != 1 {
		t.Errorf("expected exactly one lot offered to artisan B after redelivery, got %d", offeredToBCount)
	}
}
