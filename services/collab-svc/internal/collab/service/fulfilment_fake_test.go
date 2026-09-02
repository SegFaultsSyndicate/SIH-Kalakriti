// services/collab-svc/internal/collab/service/fulfilment_fake_test.go
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/collab-svc/internal/collab/domain"
)

// outboxRow is what the fake records for an outbox.Enqueue call, enough to
// assert a topic and idempotency key were used without needing a real
// serialisation round trip.
type outboxRow struct {
	id, aggregateID, topic, idempotencyKey string
	payload                                []byte
}

type eventRow struct {
	id, orderID uuid.UUID
	lotID       *uuid.UUID
	eventType   string
}

// paymentLineKey is a payment_split_line's unique-conflict key, matching the
// real schema's UNIQUE(payment_split_id, lot_id, payee_id).
type paymentLineKey struct {
	splitID uuid.UUID
	lotID   uuid.UUID
	payeeID string
}

// escrowKey is an escrow_milestone's unique-conflict key, matching the real
// schema's UNIQUE NULLS NOT DISTINCT(bulk_order_id, lot_id, trigger).
type escrowKey struct {
	orderID uuid.UUID
	lotID   uuid.UUID
	trigger domain.MilestoneTrigger
}

// fakeStore is an in-memory Store + Tx. Every Tx write applies to the store
// immediately, so a later call within the SAME transaction sees earlier ones
// (matching a real DB transaction's read-your-own-writes semantics) — this
// used to buffer writes until commit instead, which let a later step in the
// same saga (e.g. TransitionBulkOrder re-reading the order) observe a stale
// value from before an earlier step's write (e.g. IncrementAllocatedQuantity)
// and clobber it back on its own write. To keep the "a returned error leaves
// the store exactly as it was before the call" guarantee the idempotency and
// resumability tests rely on, each write instead records an undo closure;
// InTx runs them in reverse on a non-nil error.
type fakeStore struct {
	mu sync.Mutex

	orders            map[uuid.UUID]domain.BulkOrder
	ordersByBuyerKey  map[[2]string]uuid.UUID
	lots              map[uuid.UUID]domain.OrderLot
	reservationsByLot map[uuid.UUID]domain.CapacityReservation
	listings          map[uuid.UUID]domain.ListingInfo
	candidates        map[uuid.UUID][]domain.Candidate // keyed by craft id
	outbox            []outboxRow
	events            []eventRow
	qcResults         []domain.QCResult

	// batch 13: compensation, payment, escrow, SHG.
	amendments         map[uuid.UUID]domain.BulkOrderAmendment
	pendingAmendmentOf map[uuid.UUID]uuid.UUID // bulk order id -> pending amendment id
	splits             map[uuid.UUID]domain.PaymentSplit
	splitByOrder       map[uuid.UUID]uuid.UUID
	lines              map[uuid.UUID]domain.PaymentSplitLine
	lineByKey          map[paymentLineKey]uuid.UUID
	milestones         map[uuid.UUID]domain.EscrowMilestone
	milestoneByKey     map[escrowKey]uuid.UUID
	shgMembers         map[uuid.UUID][]domain.SHGMemberShare // keyed by shg id
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		orders:            map[uuid.UUID]domain.BulkOrder{},
		ordersByBuyerKey:  map[[2]string]uuid.UUID{},
		lots:              map[uuid.UUID]domain.OrderLot{},
		reservationsByLot: map[uuid.UUID]domain.CapacityReservation{},
		listings:          map[uuid.UUID]domain.ListingInfo{},
		candidates:        map[uuid.UUID][]domain.Candidate{},

		amendments:         map[uuid.UUID]domain.BulkOrderAmendment{},
		pendingAmendmentOf: map[uuid.UUID]uuid.UUID{},
		splits:             map[uuid.UUID]domain.PaymentSplit{},
		splitByOrder:       map[uuid.UUID]uuid.UUID{},
		lines:              map[uuid.UUID]domain.PaymentSplitLine{},
		lineByKey:          map[paymentLineKey]uuid.UUID{},
		milestones:         map[uuid.UUID]domain.EscrowMilestone{},
		milestoneByKey:     map[escrowKey]uuid.UUID{},
		shgMembers:         map[uuid.UUID][]domain.SHGMemberShare{},
	}
}

type fakeTx struct {
	store *fakeStore
	undo  []func()
}

func (s *fakeStore) InTx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error {
	tx := &fakeTx{store: s}
	if err := fn(ctx, tx); err != nil {
		s.mu.Lock()
		for i := len(tx.undo) - 1; i >= 0; i-- {
			tx.undo[i]()
		}
		s.mu.Unlock()
		return err
	}
	return nil
}

// --- reads (Store) -------------------------------------------------------------

func (s *fakeStore) GetBulkOrder(_ context.Context, id uuid.UUID) (domain.BulkOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return domain.BulkOrder{}, fmt.Errorf("bulk order not found: %w", pkgdomain.ErrNotFound)
	}
	return o, nil
}

func (s *fakeStore) GetLot(_ context.Context, id uuid.UUID) (domain.OrderLot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.lots[id]
	if !ok {
		return domain.OrderLot{}, fmt.Errorf("lot not found: %w", pkgdomain.ErrNotFound)
	}
	return l, nil
}

func (s *fakeStore) ListLotsForOrder(_ context.Context, orderID uuid.UUID) ([]domain.OrderLot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.OrderLot
	for _, l := range s.lots {
		if l.BulkOrderID == orderID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (s *fakeStore) GetReservationForLot(_ context.Context, lotID uuid.UUID) (domain.CapacityReservation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.reservationsByLot[lotID]
	return r, ok, nil
}

func (s *fakeStore) ListingFeasibility(_ context.Context, listingID uuid.UUID) (domain.ListingInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.listings[listingID]
	if !ok {
		return domain.ListingInfo{}, fmt.Errorf("listing not found: %w", pkgdomain.ErrNotFound)
	}
	return l, nil
}

func (s *fakeStore) CandidatesForOrder(_ context.Context, in domain.CandidateQuery) ([]domain.Candidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.Candidate(nil), s.candidates[in.CraftID]...), nil
}

// ReapExpiredReservations mirrors repo.ReapExpiredReservations: it expires
// the OFFERED lot behind each released reservation and returns the distinct
// set of bulk orders touched, so RunReservationReaper's re-offer step has
// something to iterate.
func (s *fakeStore) ReapExpiredReservations(_ context.Context, batchSize int32) (int, []uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	released := 0
	seen := map[uuid.UUID]bool{}
	var orderIDs []uuid.UUID
	for lotID, r := range s.reservationsByLot {
		if int32(released) >= batchSize {
			break
		}
		if r.State != domain.ReservationHeld || r.ExpiresAt.After(now) {
			continue
		}
		r.State = domain.ReservationReleased
		s.reservationsByLot[lotID] = r
		if lot, ok := s.lots[lotID]; ok && lot.State == domain.LotOffered {
			lot.State = domain.LotExpired
			s.lots[lotID] = lot
			// Mirrors repo.ReapExpiredReservations: a LOT_EXPIRED audit row per
			// lot the sweep actually expired.
			s.events = append(s.events, eventRow{id: uuid.New(), orderID: lot.BulkOrderID, lotID: &lot.ID, eventType: "LOT_EXPIRED"})
			if !seen[lot.BulkOrderID] {
				seen[lot.BulkOrderID] = true
				orderIDs = append(orderIDs, lot.BulkOrderID)
			}
		}
		released++
	}
	return released, orderIDs, nil
}

// --- reads (Store, batch 13) ----------------------------------------------------

func (s *fakeStore) GetAmendment(_ context.Context, id uuid.UUID) (domain.BulkOrderAmendment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.amendments[id]
	if !ok {
		return domain.BulkOrderAmendment{}, fmt.Errorf("amendment not found: %w", pkgdomain.ErrNotFound)
	}
	return a, nil
}

func (s *fakeStore) CountFailedQC(_ context.Context, lotID uuid.UUID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.qcResults {
		if r.LotID == lotID && !r.Passed {
			n++
		}
	}
	return n, nil
}

func (s *fakeStore) ListSHGMemberShares(_ context.Context, shgID uuid.UUID) ([]domain.SHGMemberShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.SHGMemberShare(nil), s.shgMembers[shgID]...), nil
}

func (s *fakeStore) GetPaymentSplit(_ context.Context, orderID uuid.UUID) (domain.PaymentSplit, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	splitID, ok := s.splitByOrder[orderID]
	if !ok {
		return domain.PaymentSplit{}, false, nil
	}
	split := s.splits[splitID]
	for _, l := range s.lines {
		if l.PaymentSplitID == splitID {
			split.Lines = append(split.Lines, l)
		}
	}
	return split, true, nil
}

func (s *fakeStore) ListPendingMilestonesForLot(_ context.Context, lotID uuid.UUID) ([]domain.EscrowMilestone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.EscrowMilestone
	for _, m := range s.milestones {
		if m.LotID != nil && *m.LotID == lotID && !m.Released {
			out = append(out, m)
		}
	}
	return out, nil
}

// --- writes (Tx) -----------------------------------------------------------

func (tx *fakeTx) CreateBulkOrder(_ context.Context, id uuid.UUID, in domain.BulkOrder, idempotencyKey string) (domain.BulkOrder, bool, error) {
	tx.store.mu.Lock()
	key := [2]string{in.BuyerID, idempotencyKey}
	if existingID, ok := tx.store.ordersByBuyerKey[key]; ok {
		existing := tx.store.orders[existingID]
		tx.store.mu.Unlock()
		return existing, false, nil
	}
	in.ID = id
	in.IdempotencyKey = idempotencyKey
	in.CreatedAt, in.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	tx.store.orders[id] = in
	tx.store.ordersByBuyerKey[key] = id
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() {
		delete(tx.store.orders, id)
		delete(tx.store.ordersByBuyerKey, key)
	})
	return in, true, nil
}

func (tx *fakeTx) TransitionBulkOrder(_ context.Context, orderID uuid.UUID, from, to domain.BulkOrderState) (domain.BulkOrder, error) {
	tx.store.mu.Lock()
	o, ok := tx.store.orders[orderID]
	if !ok {
		tx.store.mu.Unlock()
		return domain.BulkOrder{}, fmt.Errorf("bulk order not found: %w", pkgdomain.ErrNotFound)
	}
	if o.State != from {
		tx.store.mu.Unlock()
		return domain.BulkOrder{}, fmt.Errorf("bulk order state is %s, not %s: %w", o.State, from, pkgdomain.ErrConflict)
	}
	prev := o
	o.State = to
	tx.store.orders[orderID] = o
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.orders[orderID] = prev })
	return o, nil
}

func (tx *fakeTx) IncrementAllocatedQuantity(_ context.Context, orderID uuid.UUID, delta int32) (domain.BulkOrder, error) {
	tx.store.mu.Lock()
	o, ok := tx.store.orders[orderID]
	if !ok {
		tx.store.mu.Unlock()
		return domain.BulkOrder{}, fmt.Errorf("bulk order not found: %w", pkgdomain.ErrNotFound)
	}
	prev := o
	o.AllocatedQuantity += delta
	tx.store.orders[orderID] = o
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.orders[orderID] = prev })
	return o, nil
}

func (tx *fakeTx) CreateLot(_ context.Context, id uuid.UUID, in domain.OrderLot) (domain.OrderLot, error) {
	in.ID = id
	in.CreatedAt, in.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	tx.store.mu.Lock()
	tx.store.lots[id] = in
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { delete(tx.store.lots, id) })
	return in, nil
}

// TransitionLot mirrors repo.Tx.TransitionLot's guard (current state must
// match from) and field application, without replicating its per-target
// query dispatch — for the fake, one generic state-machine check plus
// applying whichever fields are set is enough to exercise the service layer.
func (tx *fakeTx) TransitionLot(_ context.Context, lotID uuid.UUID, from, to domain.LotState, fields domain.LotTransitionFields) (domain.OrderLot, error) {
	tx.store.mu.Lock()
	l, ok := tx.store.lots[lotID]
	if !ok {
		tx.store.mu.Unlock()
		return domain.OrderLot{}, fmt.Errorf("lot not found: %w", pkgdomain.ErrNotFound)
	}
	if l.State != from {
		tx.store.mu.Unlock()
		return domain.OrderLot{}, fmt.Errorf("lot state is %s, not %s: %w", l.State, from, pkgdomain.ErrConflict)
	}
	prev := l
	l.State = to
	if fields.AcceptedAt != nil {
		l.AcceptedAt = fields.AcceptedAt
	}
	if fields.PromisedShipDate != nil {
		l.PromisedShipDate = fields.PromisedShipDate
	}
	if fields.DeclineReason != nil {
		l.DeclineReason = fields.DeclineReason
	}
	if fields.ProgressPct != nil {
		l.ProgressPct = *fields.ProgressPct
	}
	if fields.DropoutReason != nil {
		l.DropoutReason = fields.DropoutReason
	}
	if fields.ReworkDeadline != nil {
		l.ReworkDeadline = fields.ReworkDeadline
	}
	tx.store.lots[lotID] = l
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.lots[lotID] = prev })
	return l, nil
}

func (tx *fakeTx) CreateReservation(_ context.Context, id uuid.UUID, in domain.CapacityReservation) (domain.CapacityReservation, error) {
	in.ID = id
	tx.store.mu.Lock()
	tx.store.reservationsByLot[in.LotID] = in
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { delete(tx.store.reservationsByLot, in.LotID) })
	return in, nil
}

func (tx *fakeTx) ReleaseReservation(_ context.Context, reservationID uuid.UUID) error {
	tx.store.mu.Lock()
	defer tx.store.mu.Unlock()
	for lotID, r := range tx.store.reservationsByLot {
		if r.ID == reservationID {
			prev := r
			r.State = domain.ReservationReleased
			tx.store.reservationsByLot[lotID] = r
			tx.undo = append(tx.undo, func() { tx.store.reservationsByLot[lotID] = prev })
			return nil
		}
	}
	return nil
}

func (tx *fakeTx) ConsumeReservation(_ context.Context, reservationID uuid.UUID) error {
	tx.store.mu.Lock()
	defer tx.store.mu.Unlock()
	for lotID, r := range tx.store.reservationsByLot {
		if r.ID == reservationID {
			prev := r
			r.State = domain.ReservationConsumed
			tx.store.reservationsByLot[lotID] = r
			tx.undo = append(tx.undo, func() { tx.store.reservationsByLot[lotID] = prev })
			return nil
		}
	}
	return nil
}

func (tx *fakeTx) RecordEvent(_ context.Context, id, orderID uuid.UUID, lotID *uuid.UUID, eventType string, _ any) error {
	tx.store.mu.Lock()
	n := len(tx.store.events)
	tx.store.events = append(tx.store.events, eventRow{id: id, orderID: orderID, lotID: lotID, eventType: eventType})
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.events = tx.store.events[:n] })
	return nil
}

func (tx *fakeTx) CreateQCResult(_ context.Context, result domain.QCResult) error {
	tx.store.mu.Lock()
	n := len(tx.store.qcResults)
	tx.store.qcResults = append(tx.store.qcResults, result)
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.qcResults = tx.store.qcResults[:n] })
	return nil
}

func (tx *fakeTx) InsertOutbox(_ context.Context, id, aggregateID, topic, idempotencyKey string, payload []byte) error {
	tx.store.mu.Lock()
	n := len(tx.store.outbox)
	tx.store.outbox = append(tx.store.outbox, outboxRow{
		id: id, aggregateID: aggregateID, topic: topic, idempotencyKey: idempotencyKey, payload: payload,
	})
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.outbox = tx.store.outbox[:n] })
	return nil
}

// --- writes (Tx, batch 13: amendments) -------------------------------------

func (tx *fakeTx) CreateAmendment(_ context.Context, id uuid.UUID, in domain.BulkOrderAmendment) (domain.BulkOrderAmendment, error) {
	tx.store.mu.Lock()
	if _, pending := tx.store.pendingAmendmentOf[in.BulkOrderID]; pending {
		tx.store.mu.Unlock()
		return domain.BulkOrderAmendment{}, fmt.Errorf("amendment already pending: %w", pkgdomain.ErrConflict)
	}
	in.ID = id
	in.Status = domain.AmendmentPendingStatus
	in.CreatedAt, in.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	tx.store.amendments[id] = in
	tx.store.pendingAmendmentOf[in.BulkOrderID] = id
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() {
		delete(tx.store.amendments, id)
		delete(tx.store.pendingAmendmentOf, in.BulkOrderID)
	})
	return in, nil
}

func (tx *fakeTx) DecideAmendment(_ context.Context, amendmentID uuid.UUID, status domain.AmendmentStatus) (domain.BulkOrderAmendment, error) {
	tx.store.mu.Lock()
	a, ok := tx.store.amendments[amendmentID]
	if !ok {
		tx.store.mu.Unlock()
		return domain.BulkOrderAmendment{}, fmt.Errorf("amendment not found: %w", pkgdomain.ErrNotFound)
	}
	if a.Status != domain.AmendmentPendingStatus {
		tx.store.mu.Unlock()
		return domain.BulkOrderAmendment{}, fmt.Errorf("amendment already decided: %w", pkgdomain.ErrConflict)
	}
	prev := a
	now := time.Now().UTC()
	a.Status = status
	a.DecidedAt = &now
	tx.store.amendments[amendmentID] = a
	delete(tx.store.pendingAmendmentOf, a.BulkOrderID)
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() {
		tx.store.amendments[amendmentID] = prev
		tx.store.pendingAmendmentOf[prev.BulkOrderID] = amendmentID
	})
	return a, nil
}

func (tx *fakeTx) ReduceBulkOrderQuantity(_ context.Context, orderID uuid.UUID, quantity int32, totalValuePaise int64, state domain.BulkOrderState) (domain.BulkOrder, error) {
	tx.store.mu.Lock()
	o, ok := tx.store.orders[orderID]
	if !ok {
		tx.store.mu.Unlock()
		return domain.BulkOrder{}, fmt.Errorf("bulk order not found: %w", pkgdomain.ErrNotFound)
	}
	prev := o
	o.Quantity, o.TotalValuePaise, o.State = quantity, totalValuePaise, state
	tx.store.orders[orderID] = o
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.orders[orderID] = prev })
	return o, nil
}

func (tx *fakeTx) ExtendBulkOrderDeadline(_ context.Context, orderID uuid.UUID, requiredBy time.Time, state domain.BulkOrderState) (domain.BulkOrder, error) {
	tx.store.mu.Lock()
	o, ok := tx.store.orders[orderID]
	if !ok {
		tx.store.mu.Unlock()
		return domain.BulkOrder{}, fmt.Errorf("bulk order not found: %w", pkgdomain.ErrNotFound)
	}
	prev := o
	o.RequiredBy, o.State = requiredBy, state
	tx.store.orders[orderID] = o
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.orders[orderID] = prev })
	return o, nil
}

// --- writes (Tx, batch 13: payment split) -----------------------------------

func (tx *fakeTx) CreatePaymentSplit(_ context.Context, id, orderID uuid.UUID, grossTotal, commissionTotal, netTotal int64) (domain.PaymentSplit, bool, error) {
	tx.store.mu.Lock()
	if existingID, ok := tx.store.splitByOrder[orderID]; ok {
		existing := tx.store.splits[existingID]
		tx.store.mu.Unlock()
		return existing, false, nil
	}
	split := domain.PaymentSplit{
		ID: id, BulkOrderID: orderID,
		GrossTotalPaise: grossTotal, CommissionTotalPaise: commissionTotal, NetTotalPaise: netTotal,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	tx.store.splits[id] = split
	tx.store.splitByOrder[orderID] = id
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() {
		delete(tx.store.splits, id)
		delete(tx.store.splitByOrder, orderID)
	})
	return split, true, nil
}

func (tx *fakeTx) InsertPaymentSplitLine(_ context.Context, id, splitID uuid.UUID, payeeID string, payeeType domain.PayeeType, lotID uuid.UUID, gross, commission, net int64, payoutRef string) (domain.PaymentSplitLine, bool, error) {
	key := paymentLineKey{splitID: splitID, lotID: lotID, payeeID: payeeID}
	tx.store.mu.Lock()
	if existingID, ok := tx.store.lineByKey[key]; ok {
		existing := tx.store.lines[existingID]
		tx.store.mu.Unlock()
		return existing, false, nil
	}
	line := domain.PaymentSplitLine{
		ID: id, PaymentSplitID: splitID, PayeeID: payeeID, PayeeType: payeeType, LotID: lotID,
		GrossAmountPaise: gross, CommissionPaise: commission, NetAmountPaise: net, PayoutRef: payoutRef,
	}
	tx.store.lines[id] = line
	tx.store.lineByKey[key] = id
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() {
		delete(tx.store.lines, id)
		delete(tx.store.lineByKey, key)
	})
	return line, true, nil
}

func (tx *fakeTx) MarkSplitLineSettled(_ context.Context, lineID uuid.UUID, settlementRef string) (domain.PaymentSplitLine, error) {
	tx.store.mu.Lock()
	l, ok := tx.store.lines[lineID]
	if !ok || l.SettledAt != nil {
		tx.store.mu.Unlock()
		return domain.PaymentSplitLine{}, fmt.Errorf("payment split line not found or already settled: %w", pkgdomain.ErrNotFound)
	}
	prev := l
	now := time.Now().UTC()
	l.SettledAt = &now
	l.SettlementRef = &settlementRef
	tx.store.lines[lineID] = l
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.lines[lineID] = prev })
	return l, nil
}

func (tx *fakeTx) MarkPaymentSplitSettled(_ context.Context, splitID uuid.UUID) (domain.PaymentSplit, bool, error) {
	tx.store.mu.Lock()
	split, ok := tx.store.splits[splitID]
	if !ok {
		tx.store.mu.Unlock()
		return domain.PaymentSplit{}, false, fmt.Errorf("payment split not found: %w", pkgdomain.ErrNotFound)
	}
	if split.SettledAt != nil {
		tx.store.mu.Unlock()
		return domain.PaymentSplit{}, false, nil
	}
	for _, l := range tx.store.lines {
		if l.PaymentSplitID == splitID && l.SettledAt == nil {
			tx.store.mu.Unlock()
			return domain.PaymentSplit{}, false, nil
		}
	}
	prev := split
	now := time.Now().UTC()
	split.SettledAt = &now
	tx.store.splits[splitID] = split
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.splits[splitID] = prev })
	return split, true, nil
}

// --- writes (Tx, batch 13: escrow) -------------------------------------------

func (tx *fakeTx) CreateEscrowMilestone(_ context.Context, id, orderID uuid.UUID, lotID *uuid.UUID, trigger domain.MilestoneTrigger, amountPaise int64) (domain.EscrowMilestone, bool, error) {
	var lot uuid.UUID
	if lotID != nil {
		lot = *lotID
	}
	key := escrowKey{orderID: orderID, lotID: lot, trigger: trigger}
	tx.store.mu.Lock()
	if existingID, ok := tx.store.milestoneByKey[key]; ok {
		existing := tx.store.milestones[existingID]
		tx.store.mu.Unlock()
		return existing, false, nil
	}
	m := domain.EscrowMilestone{ID: id, BulkOrderID: orderID, LotID: lotID, Trigger: trigger, AmountPaise: amountPaise}
	tx.store.milestones[id] = m
	tx.store.milestoneByKey[key] = id
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() {
		delete(tx.store.milestones, id)
		delete(tx.store.milestoneByKey, key)
	})
	return m, true, nil
}

func (tx *fakeTx) ReleaseEscrowMilestone(_ context.Context, id uuid.UUID, releaseRef string) (domain.EscrowMilestone, error) {
	tx.store.mu.Lock()
	m, ok := tx.store.milestones[id]
	if !ok || m.Released {
		tx.store.mu.Unlock()
		return domain.EscrowMilestone{}, fmt.Errorf("escrow milestone not found or already released: %w", pkgdomain.ErrNotFound)
	}
	prev := m
	now := time.Now().UTC()
	m.Released, m.ReleasedAt, m.ReleaseRef = true, &now, &releaseRef
	tx.store.milestones[id] = m
	tx.store.mu.Unlock()
	tx.undo = append(tx.undo, func() { tx.store.milestones[id] = prev })
	return m, nil
}
