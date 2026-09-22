// services/collab-svc/internal/collab/service/fulfilment.go

// Package service implements the collective-fulfilment saga: an orchestrated,
// hand-rolled coordinator whose only state is the rows in bulk_order,
// order_lot and capacity_reservation. Every step re-derives what to do next
// from what is persisted, so killing the process at any point and restarting
// it resumes correctly — there is no in-memory saga state anywhere in this
// package.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/money"
	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/collab-svc/internal/collab/domain"
)

// defaultReservationTTL is how long a capacity hold survives before the
// reaper reclaims it, when the caller does not override it.
const defaultReservationTTL = 48 * time.Hour

// allocationBufferDays covers the time ProposeAllocation, OfferLots and
// RespondToLot need before production can even start, on top of the
// listing's own typical production lead time.
const allocationBufferDays = 3

// Escrow milestone split, whole percent of one lot's value, summing to 100.
//
// ponytail: a fixed platform-wide split, not configurable per order or
// craft — promote to config if buyers ever need to negotiate their own
// tranche sizes.
const (
	escrowAdvancePct         = 20
	escrowProductionStartPct = 20
	escrowQCPassedPct        = 60
)

// defaultReworkWindow is how long an artisan has to fix a non-critical QC
// failure before a second failure (or a timeout — see ExpireRework) sends
// the lot to REALLOCATED instead.
const defaultReworkWindow = 7 * 24 * time.Hour

// Tx is the transactional surface the saga writes through. It embeds
// outbox.Enqueuer so a state change and the event announcing it commit
// together, and it always writes a bulk_order_event row alongside — the
// audit trail is not optional infrastructure bolted on afterward, every
// transition writes one in the same transaction as the state change itself.
type Tx interface {
	outbox.Enqueuer

	CreateBulkOrder(ctx context.Context, id uuid.UUID, in domain.BulkOrder, idempotencyKey string) (order domain.BulkOrder, created bool, err error)
	TransitionBulkOrder(ctx context.Context, orderID uuid.UUID, from, to domain.BulkOrderState) (domain.BulkOrder, error)
	IncrementAllocatedQuantity(ctx context.Context, orderID uuid.UUID, delta int32) (domain.BulkOrder, error)

	CreateLot(ctx context.Context, id uuid.UUID, in domain.OrderLot) (domain.OrderLot, error)
	TransitionLot(ctx context.Context, lotID uuid.UUID, from, to domain.LotState, fields domain.LotTransitionFields) (domain.OrderLot, error)

	CreateReservation(ctx context.Context, id uuid.UUID, in domain.CapacityReservation) (domain.CapacityReservation, error)
	ReleaseReservation(ctx context.Context, reservationID uuid.UUID) error
	ConsumeReservation(ctx context.Context, reservationID uuid.UUID) error

	CreateQCResult(ctx context.Context, result domain.QCResult) error

	RecordEvent(ctx context.Context, id, orderID uuid.UUID, lotID *uuid.UUID, eventType string, payload any) error

	// --- compensation paths (batch 13) --------------------------------------

	CreateAmendment(ctx context.Context, id uuid.UUID, in domain.BulkOrderAmendment) (domain.BulkOrderAmendment, error)
	DecideAmendment(ctx context.Context, amendmentID uuid.UUID, status domain.AmendmentStatus) (domain.BulkOrderAmendment, error)
	ReduceBulkOrderQuantity(ctx context.Context, orderID uuid.UUID, quantity int32, totalValuePaise int64, state domain.BulkOrderState) (domain.BulkOrder, error)
	ExtendBulkOrderDeadline(ctx context.Context, orderID uuid.UUID, requiredBy time.Time, state domain.BulkOrderState) (domain.BulkOrder, error)

	// --- payment split, escrow, SHG (batch 13) ------------------------------

	CreatePaymentSplit(ctx context.Context, id, orderID uuid.UUID, grossTotal, commissionTotal, netTotal int64) (domain.PaymentSplit, bool, error)
	InsertPaymentSplitLine(ctx context.Context, id, splitID uuid.UUID, payeeID string, payeeType domain.PayeeType, lotID uuid.UUID, gross, commission, net int64, payoutRef string) (domain.PaymentSplitLine, bool, error)
	MarkSplitLineSettled(ctx context.Context, lineID uuid.UUID, settlementRef string) (domain.PaymentSplitLine, error)
	MarkPaymentSplitSettled(ctx context.Context, splitID uuid.UUID) (domain.PaymentSplit, bool, error)

	CreateEscrowMilestone(ctx context.Context, id, orderID uuid.UUID, lotID *uuid.UUID, trigger domain.MilestoneTrigger, amountPaise int64) (domain.EscrowMilestone, bool, error)
	ReleaseEscrowMilestone(ctx context.Context, id uuid.UUID, releaseRef string) (domain.EscrowMilestone, error)
}

// Store is collab-svc's read surface plus its transaction entry point.
type Store interface {
	InTx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error

	GetBulkOrder(ctx context.Context, id uuid.UUID) (domain.BulkOrder, error)
	GetLot(ctx context.Context, id uuid.UUID) (domain.OrderLot, error)
	ListLotsForOrder(ctx context.Context, orderID uuid.UUID) ([]domain.OrderLot, error)
	GetReservationForLot(ctx context.Context, lotID uuid.UUID) (domain.CapacityReservation, bool, error)

	// ListingFeasibility returns the listing's own lead time (the feasibility
	// check's TypicalLeadTimeDays) and the craft/product ids the allocator
	// needs, keyed by listing id.
	ListingFeasibility(ctx context.Context, listingID uuid.UUID) (domain.ListingInfo, error)

	// CandidatesForOrder returns every artisan eligible to make the order's
	// craft, with capacity, on-time rate and cluster proximity already
	// computed server-side (a SQL join, not something this package should
	// pull rows apart to compute). preferredArtisanIDs, when non-empty,
	// restricts the result to that set.
	CandidatesForOrder(ctx context.Context, in domain.CandidateQuery) ([]domain.Candidate, error)

	// ReapExpiredReservations releases every HELD reservation whose
	// expires_at has passed, and expires the OFFERED lot each one was
	// backing, in one call per row so a crash mid-sweep leaves no
	// reservation half-freed. This is the reaper's only entry point — see
	// RunReservationReaper. orderIDs is the distinct set of bulk orders
	// affected, batch 13's hook for the "re-offer to the next candidate"
	// half of the lot-offer-expiry compensation path.
	ReapExpiredReservations(ctx context.Context, batchSize int32) (released int, orderIDs []uuid.UUID, err error)

	// --- compensation paths (batch 13) --------------------------------------

	GetAmendment(ctx context.Context, id uuid.UUID) (domain.BulkOrderAmendment, error)
	CountFailedQC(ctx context.Context, lotID uuid.UUID) (int, error)

	// --- payment split, escrow, SHG (batch 13) ------------------------------

	ListSHGMemberShares(ctx context.Context, shgID uuid.UUID) ([]domain.SHGMemberShare, error)
	GetPaymentSplit(ctx context.Context, orderID uuid.UUID) (domain.PaymentSplit, bool, error)
	ListPendingMilestonesForLot(ctx context.Context, lotID uuid.UUID) ([]domain.EscrowMilestone, error)
}

// Fulfilment is the collective-fulfilment saga coordinator.
type Fulfilment struct {
	store          Store
	reservationTTL time.Duration
	log            *slog.Logger
	now            func() time.Time
}

// NewFulfilment builds the saga coordinator. reservationTTL of zero uses
// defaultReservationTTL.
func NewFulfilment(store Store, reservationTTL time.Duration, log *slog.Logger) *Fulfilment {
	if reservationTTL <= 0 {
		reservationTTL = defaultReservationTTL
	}
	return &Fulfilment{store: store, reservationTTL: reservationTTL, log: log, now: func() time.Time { return time.Now().UTC() }}
}

// --- CreateBulkOrder ---------------------------------------------------------

// CreateBulkOrderInput is what a buyer supplies to register a requirement.
type CreateBulkOrderInput struct {
	BuyerID        string
	ListingID      uuid.UUID
	Quantity       int32
	RequiredBy     time.Time
	Customisations map[string]string
	Notes          *string
	// EscrowEnabled opts this order into milestone-based escrow. Never valid
	// for anything but a bulk order, which is all this service handles.
	EscrowEnabled  bool
	IdempotencyKey string
}

// CreateBulkOrder validates the requirement, checks deadline feasibility
// against the listing's own lead time, and persists the order in ALLOCATING
// with its order.bulk.requested event, atomically. The idempotency key is
// enforced at the DB level — bulk_order has a UNIQUE(buyer_id, idempotency_key)
// constraint (migration 019) the repo's insert is guarded by, not through
// pkg/idempotency.Do, which cannot safely wrap a multi-step operation that
// must survive a crash mid-flight. A replayed call with the same key returns
// the order that was already created rather than erroring or duplicating it.
func (f *Fulfilment) CreateBulkOrder(ctx context.Context, in CreateBulkOrderInput) (domain.BulkOrder, error) {
	if in.Quantity <= 0 {
		return domain.BulkOrder{}, pkgdomain.InvalidInput("quantity must be positive")
	}
	if in.BuyerID == "" {
		return domain.BulkOrder{}, pkgdomain.InvalidInput("buyer_id is required")
	}
	if in.IdempotencyKey == "" {
		return domain.BulkOrder{}, pkgdomain.InvalidInput("idempotency_key is required")
	}

	listing, err := f.store.ListingFeasibility(ctx, in.ListingID)
	if err != nil {
		return domain.BulkOrder{}, fmt.Errorf("loading listing %s: %w", in.ListingID, err)
	}

	feasible, reason := domain.CheckDeadlineFeasible(domain.FeasibilityInput{
		TypicalLeadTimeDays:  listing.TypicalLeadTimeDays,
		AllocationBufferDays: allocationBufferDays,
		Now:                  f.now(),
		RequiredBy:           in.RequiredBy,
	})
	if !feasible {
		return domain.BulkOrder{}, pkgdomain.InvalidInput(reason)
	}

	orderID := ids.New()
	totalPaise := listing.UnitPricePaise * int64(in.Quantity)

	order := domain.BulkOrder{
		ID:              orderID,
		BuyerID:         in.BuyerID,
		ListingID:       in.ListingID,
		ProductID:       listing.ProductID,
		Quantity:        in.Quantity,
		UnitPricePaise:  listing.UnitPricePaise,
		TotalValuePaise: totalPaise,
		RequiredBy:      in.RequiredBy,
		State:           domain.BulkOrderAllocating,
		Customisations:  in.Customisations,
		Notes:           in.Notes,
		EscrowEnabled:   in.EscrowEnabled,
	}

	var created domain.BulkOrder
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		var isNew bool
		created, isNew, err = tx.CreateBulkOrder(ctx, orderID, order, in.IdempotencyKey)
		if err != nil {
			return err
		}
		if !isNew {
			// A replay: the row already existed under this buyer's key, and
			// its event/outbox rows were written the first time. Nothing
			// further to do.
			return nil
		}
		if err := tx.RecordEvent(ctx, ids.New(), created.ID, nil, "ORDER_CREATED", created); err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), created.ID.String(),
			topics.OrderBulkRequested, in.IdempotencyKey, bulkOrderRequestedPayload(created))
	})
	if err != nil {
		return domain.BulkOrder{}, fmt.Errorf("creating bulk order: %w", err)
	}
	return created, nil
}

// --- GetOrder -----------------------------------------------------------------

// GetOrder loads one bulk order with its lots.
func (f *Fulfilment) GetOrder(ctx context.Context, orderID uuid.UUID) (domain.BulkOrder, []domain.OrderLot, error) {
	order, err := f.store.GetBulkOrder(ctx, orderID)
	if err != nil {
		return domain.BulkOrder{}, nil, fmt.Errorf("loading bulk order %s: %w", orderID, err)
	}
	lots, err := f.store.ListLotsForOrder(ctx, orderID)
	if err != nil {
		return domain.BulkOrder{}, nil, fmt.Errorf("listing lots for order %s: %w", orderID, err)
	}
	return order, lots, nil
}

// --- ProposeAllocation -------------------------------------------------------

// ProposeAllocationInput controls one allocation pass.
type ProposeAllocationInput struct {
	BulkOrderID         uuid.UUID
	PreferredArtisanIDs []uuid.UUID
	MaxUnitsPerArtisan  int32
	ResponseWindow      time.Duration
	DryRun              bool
	IdempotencyKey      string
}

// ProposeAllocationResult is what one allocation pass produced.
type ProposeAllocationResult struct {
	Lots                []domain.OrderLot
	UnallocatedQuantity int32
	ShortfallReason     string
}

// ProposeAllocation ranks eligible artisans, decomposes the order's
// unallocated quantity into lots (never below domain.MinLotSize), reserves
// capacity per lot with a TTL, persists the lots as OFFERED, and emits
// order.lot.offered per lot — all in one transaction, so a lot never exists
// without its reservation or its offer event. DryRun computes the same
// decomposition without writing or offering anything.
func (f *Fulfilment) ProposeAllocation(ctx context.Context, in ProposeAllocationInput) (ProposeAllocationResult, error) {
	order, err := f.store.GetBulkOrder(ctx, in.BulkOrderID)
	if err != nil {
		return ProposeAllocationResult{}, fmt.Errorf("loading bulk order %s: %w", in.BulkOrderID, err)
	}
	if order.State != domain.BulkOrderAllocating && order.State != domain.BulkOrderPartiallyAllocated {
		return ProposeAllocationResult{}, pkgdomain.Conflict(
			"bulk order is not accepting allocation in state " + string(order.State))
	}

	existingLots, err := f.store.ListLotsForOrder(ctx, order.ID)
	if err != nil {
		return ProposeAllocationResult{}, fmt.Errorf("listing lots for order %s: %w", order.ID, err)
	}
	var pendingOffered int32
	for _, l := range existingLots {
		if l.State == domain.LotOffered {
			pendingOffered += l.Quantity
		}
	}

	// remaining excludes both accepted quantity and quantity already offered
	// but not yet answered — an OFFERED lot already reserves that capacity
	// against this order, so a redundant re-run (e.g. Kafka's at-least-once
	// redelivery of order.lot.declined calling HandleLotDeclined twice) must
	// not offer the same freed capacity a second time.
	remaining := order.Quantity - order.AllocatedQuantity - pendingOffered
	if remaining <= 0 {
		return ProposeAllocationResult{}, pkgdomain.Conflict("bulk order already fully allocated")
	}

	listing, err := f.store.ListingFeasibility(ctx, order.ListingID)
	if err != nil {
		return ProposeAllocationResult{}, fmt.Errorf("loading listing %s: %w", order.ListingID, err)
	}

	periodStart, periodEnd := capacityMonth(order.RequiredBy)
	candidates, err := f.store.CandidatesForOrder(ctx, domain.CandidateQuery{
		CraftID:             listing.CraftID,
		PreferredArtisanIDs: in.PreferredArtisanIDs,
		PeriodStart:         periodStart,
		PeriodEnd:           periodEnd,
	})
	if err != nil {
		return ProposeAllocationResult{}, fmt.Errorf("loading allocation candidates: %w", err)
	}

	ranked := domain.RankCandidates(candidates, remaining)
	proposed, unallocated := domain.Decompose(ranked, remaining, in.MaxUnitsPerArtisan)

	if in.DryRun {
		return ProposeAllocationResult{Lots: proposedToDraftLots(proposed, listing.UnitPricePaise), UnallocatedQuantity: unallocated}, nil
	}

	window := in.ResponseWindow
	if window <= 0 {
		window = 48 * time.Hour
	}
	now := f.now()
	respondsBy := now.Add(window)

	var lots []domain.OrderLot
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		for _, p := range proposed {
			lotID := ids.New()
			lot := domain.OrderLot{
				ID:             lotID,
				BulkOrderID:    order.ID,
				ArtisanID:      p.ArtisanID,
				ClusterID:      p.ClusterID,
				Quantity:       p.Quantity,
				UnitPricePaise: listing.UnitPricePaise,
				LotValuePaise:  listing.UnitPricePaise * int64(p.Quantity),
				State:          domain.LotOffered,
				OfferedAt:      now,
				RespondsBy:     respondsBy,
			}
			created, err := tx.CreateLot(ctx, lotID, lot)
			if err != nil {
				return fmt.Errorf("creating lot for artisan %s: %w", p.ArtisanID, err)
			}

			reservationID := ids.New()
			_, err = tx.CreateReservation(ctx, reservationID, domain.CapacityReservation{
				ID:          reservationID,
				ArtisanID:   p.ArtisanID,
				ListingID:   order.ListingID,
				LotID:       lotID,
				Units:       p.Quantity,
				PeriodStart: periodStart,
				PeriodEnd:   periodEnd,
				ExpiresAt:   now.Add(f.reservationTTL),
				State:       domain.ReservationHeld,
			})
			if err != nil {
				return fmt.Errorf("reserving capacity for artisan %s: %w", p.ArtisanID, err)
			}

			if err := tx.RecordEvent(ctx, ids.New(), order.ID, &lotID, "LOT_OFFERED", created); err != nil {
				return err
			}
			if err := outbox.Enqueue(ctx, tx, ids.New().String(), lotID.String(),
				topics.OrderLotOffered, fmt.Sprintf("%s:%s", in.IdempotencyKey, lotID), lotOfferedPayload(created)); err != nil {
				return err
			}
			if order.EscrowEnabled {
				if err := f.createLotEscrowMilestones(ctx, tx, order.ID, created); err != nil {
					return fmt.Errorf("creating escrow milestones for lot %s: %w", lotID, err)
				}
			}
			lots = append(lots, created)
		}

		// unallocated > 0 covers both a partial shortfall (some lots placed,
		// some quantity left over) and a fully exhausted candidate pool (no
		// lots placed at all) — the latter is exactly the "if the candidate
		// pool is exhausted, transition to PARTIALLY_ALLOCATED" half of the
		// lot-offer-expiry compensation path (service.RunReservationReaper
		// re-runs this method after an expiry, and this is what lets that
		// re-run register the shortfall instead of silently doing nothing).
		if unallocated > 0 {
			if _, err := tx.TransitionBulkOrder(ctx, order.ID, order.State, domain.BulkOrderPartiallyAllocated); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ProposeAllocationResult{}, fmt.Errorf("proposing allocation for order %s: %w", order.ID, err)
	}

	result := ProposeAllocationResult{Lots: lots, UnallocatedQuantity: unallocated}
	if unallocated > 0 {
		result.ShortfallReason = fmt.Sprintf("no eligible artisan capacity for %d of %d units", unallocated, order.Quantity)
	}
	return result, nil
}

// --- RespondToLot -------------------------------------------------------------

// RespondToLotInput carries an artisan's accept or decline.
type RespondToLotInput struct {
	LotID            uuid.UUID
	ArtisanID        uuid.UUID
	Accept           bool
	PromisedShipDate *time.Time
	DeclineReason    *string
	IdempotencyKey   string
}

// RespondToLot records an artisan's accept or decline. It is idempotent on
// (lot id, artisan id): a lot that is no longer OFFERED because this exact
// artisan already answered it returns the lot's current state rather than
// erroring, so a retried request is safe; a lot that moved on for any other
// reason (expired, or answered under a different artisan id, which should
// never happen but is checked anyway) is a conflict.
//
// Accepting releases nothing; the reservation converts from HELD to CONSUMED
// in the same transaction. Declining releases the reservation and re-offers
// the freed capacity to the next-ranked candidate who was not already
// offered a lot on this order.
func (f *Fulfilment) RespondToLot(ctx context.Context, in RespondToLotInput) (domain.OrderLot, error) {
	lot, err := f.store.GetLot(ctx, in.LotID)
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("loading lot %s: %w", in.LotID, err)
	}
	if lot.ArtisanID != in.ArtisanID {
		return domain.OrderLot{}, pkgdomain.Forbidden("lot does not belong to this artisan")
	}

	wantState := domain.LotDeclined
	if in.Accept {
		wantState = domain.LotAccepted
	}
	if lot.State == wantState {
		// Replay of an already-applied response: idempotent no-op.
		return lot, nil
	}
	if lot.State != domain.LotOffered {
		return domain.OrderLot{}, pkgdomain.Conflict("lot is no longer awaiting a response, state is " + string(lot.State))
	}
	if in.Accept && in.PromisedShipDate == nil {
		return domain.OrderLot{}, pkgdomain.InvalidInput("promised_ship_date is required to accept a lot")
	}
	if !in.Accept && (in.DeclineReason == nil || *in.DeclineReason == "") {
		return domain.OrderLot{}, pkgdomain.InvalidInput("decline_reason is required to decline a lot")
	}

	var updated domain.OrderLot
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		fields := domain.LotTransitionFields{}
		if in.Accept {
			now := f.now()
			fields.AcceptedAt = &now
			fields.PromisedShipDate = in.PromisedShipDate
		} else {
			fields.DeclineReason = in.DeclineReason
		}

		var err error
		updated, err = tx.TransitionLot(ctx, in.LotID, domain.LotOffered, wantState, fields)
		if err != nil {
			return err
		}

		reservation, hasReservation, err := f.store.GetReservationForLot(ctx, in.LotID)
		if err != nil {
			return fmt.Errorf("loading reservation for lot %s: %w", in.LotID, err)
		}

		if in.Accept {
			if hasReservation {
				if err := tx.ConsumeReservation(ctx, reservation.ID); err != nil {
					return err
				}
			}
			order, err := tx.IncrementAllocatedQuantity(ctx, lot.BulkOrderID, lot.Quantity)
			if err != nil {
				return err
			}
			// Confirm once accepted lots cover the full quantity. order here
			// is IncrementAllocatedQuantity's own return value — the freshly
			// updated row from inside this same transaction — never a
			// separate read that could observe a stale AllocatedQuantity.
			if order.AllocatedQuantity >= order.Quantity {
				if _, err := tx.TransitionBulkOrder(ctx, order.ID, order.State, domain.BulkOrderConfirmed); err != nil {
					return err
				}
				if err := tx.RecordEvent(ctx, ids.New(), order.ID, nil, "ORDER_CONFIRMED", order); err != nil {
					return err
				}
				if err := outbox.Enqueue(ctx, tx, ids.New().String(), order.ID.String(),
					topics.OrderFulfilmentCompleted, in.IdempotencyKey+":confirmed", orderConfirmedPayload(order)); err != nil {
					return err
				}
			}
			if err := tx.RecordEvent(ctx, ids.New(), lot.BulkOrderID, &lot.ID, "LOT_ACCEPTED", updated); err != nil {
				return err
			}
			if err := f.releaseLotMilestone(ctx, tx, lot.BulkOrderID, updated, domain.MilestoneAdvance, in.IdempotencyKey); err != nil {
				return err
			}
			return outbox.Enqueue(ctx, tx, ids.New().String(), lot.ID.String(),
				topics.OrderLotAccepted, in.IdempotencyKey, lotAcceptedPayload(updated))
		}

		if hasReservation {
			if err := tx.ReleaseReservation(ctx, reservation.ID); err != nil {
				return err
			}
		}
		if err := tx.RecordEvent(ctx, ids.New(), lot.BulkOrderID, &lot.ID, "LOT_DECLINED", updated); err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), lot.ID.String(),
			topics.OrderLotDeclined, in.IdempotencyKey, lotDeclinedPayload(updated))
	})
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("responding to lot %s: %w", in.LotID, err)
	}

	// Re-offering the freed capacity is NOT triggered inline here. The
	// order.lot.declined event just enqueued in the same transaction as the
	// decline is what drives it: a dedicated consumer in cmd/collab-svc,
	// subscribed to topics.OrderLotDeclined, calls ProposeAllocation once
	// that event is actually published. This is durable across a crash
	// between commit and an inline call the way the earlier design was not
	// — the outbox row survives the process, so the reoffer is
	// at-least-once even if collab-svc dies the instant after this
	// function returns.
	return updated, nil
}

// --- ReportProgress and SubmitQC ---------------------------------------------

// ReportProgressInput records production progress against an accepted lot.
type ReportProgressInput struct {
	LotID          uuid.UUID
	ArtisanID      uuid.UUID
	ProgressPct    int32
	IdempotencyKey string
}

// ReportProgress moves a lot ACCEPTED -> IN_PRODUCTION on its first report,
// and on to QC_PENDING once ProgressPct reaches 100. Idempotent: reporting
// the same or a lower percentage again on an already-IN_PRODUCTION lot just
// updates the stored value without erroring.
func (f *Fulfilment) ReportProgress(ctx context.Context, in ReportProgressInput) (domain.OrderLot, error) {
	if in.ProgressPct < 0 || in.ProgressPct > 100 {
		return domain.OrderLot{}, pkgdomain.InvalidInput("progress_pct must be between 0 and 100")
	}
	lot, err := f.store.GetLot(ctx, in.LotID)
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("loading lot %s: %w", in.LotID, err)
	}
	if lot.ArtisanID != in.ArtisanID {
		return domain.OrderLot{}, pkgdomain.Forbidden("lot does not belong to this artisan")
	}
	// QC_FAILED is a legal starting point too: a rework resubmission is just
	// another progress report, moving the lot back toward QC_PENDING once the
	// artisan says it's ready again -- CanTransitionLot(QC_FAILED, QC_PENDING)
	// already allows this edge, ReportProgress just never exercised it.
	if lot.State != domain.LotAccepted && lot.State != domain.LotInProduction && lot.State != domain.LotQCFailed {
		return domain.OrderLot{}, pkgdomain.Conflict("lot is not in production, state is " + string(lot.State))
	}

	target := domain.LotInProduction
	if in.ProgressPct >= 100 {
		target = domain.LotQCPending
	}
	from := lot.State
	// Same-state progress updates (e.g. 40% then 70%, both IN_PRODUCTION) are
	// a plain field update on the row's current state, not a transition
	// domain.CanTransitionLot needs to know about — only cross-state moves
	// are checked against the transition table.
	if from != target && !domain.CanTransitionLot(from, target) {
		return domain.OrderLot{}, pkgdomain.Conflict("cannot move lot from " + string(from) + " to " + string(target))
	}

	pct := in.ProgressPct
	var updated domain.OrderLot
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		updated, err = tx.TransitionLot(ctx, in.LotID, from, target, domain.LotTransitionFields{ProgressPct: &pct})
		if err != nil {
			return err
		}
		if err := tx.RecordEvent(ctx, ids.New(), lot.BulkOrderID, &lot.ID, "LOT_PROGRESSED", updated); err != nil {
			return err
		}
		if from != domain.LotInProduction && target == domain.LotInProduction {
			if err := f.releaseLotMilestone(ctx, tx, lot.BulkOrderID, updated, domain.MilestoneProductionStart, in.IdempotencyKey); err != nil {
				return err
			}
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), lot.ID.String(),
			topics.OrderLotProgressed, in.IdempotencyKey, lotProgressedPayload(updated))
	})
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("reporting progress on lot %s: %w", in.LotID, err)
	}
	return updated, nil
}

// RequestReallocationInput gives up an artisan's own accepted lot.
type RequestReallocationInput struct {
	LotID          uuid.UUID
	ArtisanID      uuid.UUID
	Reason         string
	IdempotencyKey string
}

// RequestReallocation is the artisan-initiated counterpart to SubmitQC's
// give-up branch: the same ACCEPTED/IN_PRODUCTION/QC_FAILED -> REALLOCATED
// edge the domain transition table already allows for a QC-driven give-up,
// now reachable by the artisan themself when they know upfront they cannot
// finish -- a non-punitive path instead of silently missing the ship date
// and forcing a QC failure to discover it. Mirrors SubmitQC's own give-up
// handling: no payment split line is ever generated for a lot that never
// reaches COMPLETED, and the freed units are reoffered the same way.
func (f *Fulfilment) RequestReallocation(ctx context.Context, in RequestReallocationInput) (domain.OrderLot, error) {
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
		// Replay of an already-applied give-up: idempotent no-op.
		return lot, nil
	}
	from := lot.State
	if !domain.CanTransitionLot(from, domain.LotReallocated) {
		return domain.OrderLot{}, pkgdomain.Conflict("cannot give up a lot in state " + string(from))
	}

	var updated domain.OrderLot
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		updated, err = tx.TransitionLot(ctx, in.LotID, from, domain.LotReallocated, domain.LotTransitionFields{DeclineReason: &in.Reason})
		if err != nil {
			return err
		}
		reservation, hasReservation, err := f.store.GetReservationForLot(ctx, in.LotID)
		if err != nil {
			return fmt.Errorf("loading reservation for lot %s: %w", in.LotID, err)
		}
		if hasReservation {
			if err := tx.ReleaseReservation(ctx, reservation.ID); err != nil {
				return err
			}
		}
		return tx.RecordEvent(ctx, ids.New(), lot.BulkOrderID, &lot.ID, "LOT_GAVE_UP", updated)
	})
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("requesting reallocation for lot %s: %w", in.LotID, err)
	}
	// Same as SubmitQC's give-up branch: reoffer freed capacity after commit,
	// not from inside the transaction -- see reofferAfterGiveUp's own doc
	// comment for the durability tradeoff this accepts.
	f.reofferAfterGiveUp(ctx, lot.BulkOrderID, in.IdempotencyKey)
	return updated, nil
}

// SubmitQCInput records a quality inspection outcome.
type SubmitQCInput struct {
	LotID          uuid.UUID
	InspectorID    string
	Passed         bool
	Notes          *string
	MediaIDs       []uuid.UUID
	Defects        []domain.QCDefect
	IdempotencyKey string
}

// CancelBulkOrderInput cancels a bulk order before production has started.
type CancelBulkOrderInput struct {
	BulkOrderID    uuid.UUID
	Reason         string
	IdempotencyKey string
}

// CancelBulkOrderResult reports what CancelBulkOrder released.
type CancelBulkOrderResult struct {
	Order          domain.BulkOrder
	ReleasedLotIDs []uuid.UUID
}

// CancelBulkOrder moves a bulk order to CANCELLED and releases every open
// commitment against it: an OFFERED lot's reservation is released and the
// lot itself moves to DECLINED (the closest existing lot state to "this
// offer is now void" — the schema has no separate CANCELLED lot state), and
// an ACCEPTED lot's CONSUMED reservation is also released, since the order
// backing it no longer exists. Only legal from ALLOCATING,
// PARTIALLY_ALLOCATED or CONFIRMED — once any lot has reached
// IN_PRODUCTION the domain transition table has no CANCELLED edge out of
// that state, and CanTransitionBulkOrder rejects it up front: an artisan
// already mid-production is a bigger commitment than a plain cancel should
// undo.
func (f *Fulfilment) CancelBulkOrder(ctx context.Context, in CancelBulkOrderInput) (CancelBulkOrderResult, error) {
	if in.Reason == "" {
		return CancelBulkOrderResult{}, pkgdomain.InvalidInput("reason is required")
	}

	order, err := f.store.GetBulkOrder(ctx, in.BulkOrderID)
	if err != nil {
		return CancelBulkOrderResult{}, fmt.Errorf("loading bulk order %s: %w", in.BulkOrderID, err)
	}
	if order.State == domain.BulkOrderCancelled {
		// Replay of an already-applied cancellation: idempotent no-op.
		return CancelBulkOrderResult{Order: order}, nil
	}
	if !domain.CanTransitionBulkOrder(order.State, domain.BulkOrderCancelled) {
		return CancelBulkOrderResult{}, pkgdomain.Conflict(
			"bulk order cannot be cancelled from state " + string(order.State))
	}

	lots, err := f.store.ListLotsForOrder(ctx, in.BulkOrderID)
	if err != nil {
		return CancelBulkOrderResult{}, fmt.Errorf("listing lots for order %s: %w", in.BulkOrderID, err)
	}

	var released []uuid.UUID
	var cancelled domain.BulkOrder
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		cancelled, err = tx.TransitionBulkOrder(ctx, in.BulkOrderID, order.State, domain.BulkOrderCancelled)
		if err != nil {
			return err
		}

		for _, lot := range lots {
			if lot.State != domain.LotOffered && lot.State != domain.LotAccepted {
				continue
			}
			reservation, hasReservation, err := f.store.GetReservationForLot(ctx, lot.ID)
			if err != nil {
				return fmt.Errorf("loading reservation for lot %s: %w", lot.ID, err)
			}
			if hasReservation && reservation.State != domain.ReservationReleased {
				if err := tx.ReleaseReservation(ctx, reservation.ID); err != nil {
					return err
				}
			}
			if lot.State == domain.LotOffered {
				reason := "bulk order cancelled: " + in.Reason
				updatedLot, err := tx.TransitionLot(ctx, lot.ID, domain.LotOffered, domain.LotDeclined,
					domain.LotTransitionFields{DeclineReason: &reason})
				if err != nil {
					return err
				}
				if err := tx.RecordEvent(ctx, ids.New(), in.BulkOrderID, &lot.ID, "LOT_DECLINED", toLotEventPayload(updatedLot)); err != nil {
					return err
				}
			}
			released = append(released, lot.ID)
		}

		if err := tx.RecordEvent(ctx, ids.New(), in.BulkOrderID, nil, "ORDER_CANCELLED", bulkOrderEventPayload{
			BulkOrderID: cancelled.ID.String(), BuyerID: cancelled.BuyerID, ListingID: cancelled.ListingID.String(),
			Quantity: cancelled.Quantity, State: string(cancelled.State),
		}); err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), cancelled.ID.String(),
			topics.OrderBulkCancelled, in.IdempotencyKey, orderConfirmedPayload(cancelled))
	})
	if err != nil {
		return CancelBulkOrderResult{}, fmt.Errorf("cancelling bulk order %s: %w", in.BulkOrderID, err)
	}
	return CancelBulkOrderResult{Order: cancelled, ReleasedLotIDs: released}, nil
}

// SubmitQC moves a lot from QC_PENDING to COMPLETED on a pass, to
// REALLOCATED when a fail includes a CRITICAL defect, or to QC_FAILED
// otherwise, and once every lot on the order is COMPLETED, moves the order
// itself to COMPLETED. The inspection and its defects are persisted as a
// QCResult (repo.CreateQCResult) alongside the lot transition and its own
// bulk_order_event row, all in the same transaction.
func (f *Fulfilment) SubmitQC(ctx context.Context, in SubmitQCInput) (domain.OrderLot, error) {
	if in.Passed && len(in.Defects) > 0 {
		return domain.OrderLot{}, pkgdomain.InvalidInput("defects must be empty when passed is true")
	}
	for i, d := range in.Defects {
		if d.Code == "" {
			return domain.OrderLot{}, pkgdomain.InvalidInput(fmt.Sprintf("defects[%d].code is required", i))
		}
		if d.AffectedUnits < 0 {
			return domain.OrderLot{}, pkgdomain.InvalidInput(fmt.Sprintf("defects[%d].affected_units must not be negative", i))
		}
	}

	lot, err := f.store.GetLot(ctx, in.LotID)
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("loading lot %s: %w", in.LotID, err)
	}
	if lot.State != domain.LotQCPending {
		return domain.OrderLot{}, pkgdomain.Conflict("lot is not awaiting QC, state is " + string(lot.State))
	}

	// A lot only reaches QC_PENDING a second time via ResubmitForQC (rework),
	// so any prior failed qc_result row here means this submission is the
	// rework's own re-inspection — a second failure, whatever its severity,
	// gives up on the lot rather than opening a second rework window.
	priorFailures, err := f.store.CountFailedQC(ctx, in.LotID)
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("counting prior QC failures for lot %s: %w", in.LotID, err)
	}

	target := domain.LotCompleted
	defects := make([]domain.QCDefect, len(in.Defects))
	for i, d := range in.Defects {
		d.ID = ids.New()
		defects[i] = d
	}
	result := domain.QCResult{
		ID: ids.New(), LotID: in.LotID, InspectorID: in.InspectorID, Passed: in.Passed,
		Notes: in.Notes, MediaIDs: in.MediaIDs, Defects: defects, InspectedAt: f.now(),
	}
	fields := domain.LotTransitionFields{}
	if !in.Passed {
		switch {
		case priorFailures > 0:
			target = domain.LotReallocated
		case result.HasCriticalDefect():
			target = domain.LotReallocated
		default:
			target = domain.LotQCFailed
			deadline := f.now().Add(defaultReworkWindow)
			fields.ReworkDeadline = &deadline
		}
	}

	var updated domain.OrderLot
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		updated, err = tx.TransitionLot(ctx, in.LotID, domain.LotQCPending, target, fields)
		if err != nil {
			return err
		}
		if err := tx.CreateQCResult(ctx, result); err != nil {
			return err
		}
		if err := tx.RecordEvent(ctx, ids.New(), lot.BulkOrderID, &lot.ID, "QC_RECORDED", qcRecordedPayload{
			lotEventPayload: toLotEventPayload(updated),
			InspectorID:     in.InspectorID,
			Passed:          in.Passed,
		}); err != nil {
			return err
		}

		if target == domain.LotReallocated {
			// This lot's own split is withheld — RequestPaymentSplit only
			// ever pays DELIVERED (COMPLETED) lots, so a reallocated lot
			// simply never generates a payment_split_line, and every other
			// lot on the order is untouched. The reoffer that sends its
			// units to a fresh lot happens after this transaction commits —
			// see the call below.
			return nil
		}
		if target == domain.LotQCFailed {
			return nil
		}

		if err := outbox.Enqueue(ctx, tx, ids.New().String(), updated.ID.String(),
			topics.OrderLotCompleted, in.IdempotencyKey+":lot-completed", lotCompletedPayload(updated)); err != nil {
			return err
		}

		// target == LotCompleted: a passed lot releases its QC_PASSED
		// tranche and, once every lot that will ever complete has, closes
		// out the order.
		if err := f.releaseLotMilestone(ctx, tx, lot.BulkOrderID, updated, domain.MilestoneQCPassed, in.IdempotencyKey); err != nil {
			return err
		}

		lots, err := f.store.ListLotsForOrder(ctx, lot.BulkOrderID)
		if err != nil {
			return fmt.Errorf("listing lots for order %s: %w", lot.BulkOrderID, err)
		}
		if !allLotsSettled(lots, lot.ID, target) {
			return nil
		}
		order, err := tx.TransitionBulkOrder(ctx, lot.BulkOrderID, domain.BulkOrderInProduction, domain.BulkOrderCompleted)
		if err != nil {
			return err
		}
		if err := tx.RecordEvent(ctx, ids.New(), order.ID, nil, "ORDER_COMPLETED", order); err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), order.ID.String(),
			topics.OrderFulfilmentCompleted, in.IdempotencyKey+":order-completed", orderConfirmedPayload(order))
	})
	if err != nil {
		return domain.OrderLot{}, fmt.Errorf("submitting QC for lot %s: %w", in.LotID, err)
	}
	if target == domain.LotReallocated {
		f.reofferAfterGiveUp(ctx, lot.BulkOrderID, in.IdempotencyKey)
	}
	return updated, nil
}

// allLotsSettled reports whether every lot on the order has reached a
// terminal outcome — COMPLETED (delivered) or REALLOCATED (given up on,
// replaced by whatever lot its units moved to) — treating override as l's
// own not-yet-persisted target state. A REALLOCATED lot counts as settled
// here for the same reason SettlePartial exists: its units live on in a
// replacement lot, which is itself checked in its own right, so counting the
// original a second time would never let a chain of reallocations converge.
func allLotsSettled(lots []domain.OrderLot, overrideID uuid.UUID, override domain.LotState) bool {
	for _, l := range lots {
		state := l.State
		if l.ID == overrideID {
			state = override
		}
		if state != domain.LotCompleted && state != domain.LotReallocated {
			return false
		}
	}
	return true
}

// reofferAfterGiveUp best-effort re-runs ProposeAllocation for orderID right
// after a lot is reallocated (dropout or a second QC failure), so its units
// go back out to the next candidate without a human having to notice the
// shortfall. Called after the reallocating transaction has already
// committed — never from inside it, since ProposeAllocation reads the
// order's own committed state (unallocated quantity, current lots) to
// decide what to do next.
//
// ponytail: called inline, not via an outbox-published event like
// HandleLotDeclined's reoffer — REALLOCATED has no Kafka topic of its own
// yet, so a crash between the reallocating call returning and this running
// leaves the freed capacity unpicked-up until the next manual
// ProposeAllocation call. Upgrade by adding an order.lot.reallocated topic
// and a durable consumer once that scope is needed. A failure here is
// logged, not returned — the reallocation itself already committed
// successfully and must not be reported as failed because its follow-up
// reoffer didn't run.
func (f *Fulfilment) reofferAfterGiveUp(ctx context.Context, orderID uuid.UUID, idempotencyKey string) {
	if _, err := f.ProposeAllocation(ctx, ProposeAllocationInput{
		BulkOrderID: orderID, IdempotencyKey: "reoffer:" + idempotencyKey,
	}); err != nil && !errors.Is(err, pkgdomain.ErrConflict) {
		f.log.ErrorContext(ctx, "reoffering capacity after lot give-up", "bulk_order_id", orderID, "error", err)
	}
}

// --- reoffer consumer ----------------------------------------------------------

// declinedLotPayload mirrors lotEventPayload's JSON shape (see
// lotDeclinedPayload) — this package writes that payload and this method
// reads it back, so the two stay in the same package rather than needing a
// shared type across a package boundary.
type declinedLotPayload struct {
	BulkOrderID string `json:"bulk_order_id"`
}

// HandleLotDeclined is a Kafka consumer entry point for topics.OrderLotDeclined:
// it re-runs ProposeAllocation for the order whose lot was just declined,
// which is what actually offers the freed capacity to the next candidate.
// This makes the reoffer durable — it fires from the outbox-published event,
// not from an inline call inside RespondToLot's own request path, so it
// still happens even if collab-svc crashes the instant RespondToLot's
// transaction commits. Kafka's own at-least-once delivery means this may run
// more than once for the same decline; that is safe because ProposeAllocation
// only ever offers lots for an order's currently-unallocated remainder — a
// redundant call against an order that has since become fully allocated (or
// cancelled) is a correctly-rejected no-op, not a duplicate offer.
func (f *Fulfilment) HandleLotDeclined(ctx context.Context, payload []byte) error {
	var p declinedLotPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("decoding declined-lot payload: %w", err)
	}
	orderID, err := uuid.Parse(p.BulkOrderID)
	if err != nil {
		return fmt.Errorf("declined-lot payload bulk_order_id %q is not a uuid: %w", p.BulkOrderID, err)
	}

	_, err = f.ProposeAllocation(ctx, ProposeAllocationInput{
		BulkOrderID:    orderID,
		IdempotencyKey: "reoffer:" + p.BulkOrderID,
	})
	if err != nil {
		if errors.Is(err, pkgdomain.ErrConflict) {
			// The order is no longer in an allocatable state (fully
			// allocated already, cancelled, or further along) — expected on
			// a redundant redelivery or a decline that arrived after the
			// order moved on for another reason. Not an error worth retrying.
			f.log.InfoContext(ctx, "skipping reoffer, order is no longer allocatable",
				"bulk_order_id", orderID, "error", err)
			return nil
		}
		return fmt.Errorf("reoffering capacity for order %s: %w", orderID, err)
	}
	return nil
}

// --- reaper -------------------------------------------------------------------

// RunReservationReaper sweeps expired capacity reservations on a ticker until
// ctx is cancelled. This is the enforcement mechanism the spec requires: a
// reservation's TTL is honoured by this DB-driven sweep, not by an
// in-process timer that dies with the process holding it.
//
// Batch 13: this is also compensation path (a) in full — the sweep itself
// (repo.ReapExpiredReservations) releases each expired reservation, expires
// its OFFERED lot and writes a LOT_EXPIRED audit event; this loop then
// re-runs ProposeAllocation for every affected order, which is what actually
// re-offers the freed capacity to the next-ranked candidate and (via the
// unallocated>0 fix in ProposeAllocation) moves the order to
// PARTIALLY_ALLOCATED if the candidate pool is exhausted.
func (f *Fulfilment) RunReservationReaper(ctx context.Context, interval time.Duration, batchSize int32) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			released, orderIDs, err := f.store.ReapExpiredReservations(ctx, batchSize)
			if err != nil {
				f.log.ErrorContext(ctx, "capacity reservation reaper sweep failed", "error", err)
				continue
			}
			if released > 0 {
				f.log.InfoContext(ctx, "reaped expired capacity reservations", "count", released)
			}
			for _, orderID := range orderIDs {
				if _, err := f.ProposeAllocation(ctx, ProposeAllocationInput{
					BulkOrderID: orderID, IdempotencyKey: "reoffer:expiry:" + orderID.String(),
				}); err != nil && !errors.Is(err, pkgdomain.ErrConflict) {
					f.log.ErrorContext(ctx, "reoffering capacity after lot expiry", "bulk_order_id", orderID, "error", err)
				}
			}
		}
	}
}

// --- helpers -------------------------------------------------------------------

// capacityMonth returns the half-open [start, end) UTC month interval that
// contains requiredBy, matching capacity_reservation's period columns.
func capacityMonth(requiredBy time.Time) (start, end time.Time) {
	requiredBy = requiredBy.UTC()
	start = time.Date(requiredBy.Year(), requiredBy.Month(), 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0)
	return start, end
}

// createLotEscrowMilestones funds ADVANCE, PRODUCTION_START and QC_PASSED
// tranches for one newly-offered lot, split off its own lot value so the
// three tranches always sum back to exactly the lot's worth (money.SplitPct).
// Called only when the order is EscrowEnabled — this is the one and only
// place escrow milestones get created, so a lot never exists without them.
func (f *Fulfilment) createLotEscrowMilestones(ctx context.Context, tx Tx, orderID uuid.UUID, lot domain.OrderLot) error {
	shares, err := money.New(lot.LotValuePaise).SplitPct([]int{escrowAdvancePct, escrowProductionStartPct, escrowQCPassedPct})
	if err != nil {
		return fmt.Errorf("splitting lot value into escrow tranches: %w", err)
	}
	triggers := []domain.MilestoneTrigger{domain.MilestoneAdvance, domain.MilestoneProductionStart, domain.MilestoneQCPassed}
	for i, trigger := range triggers {
		if _, _, err := tx.CreateEscrowMilestone(ctx, ids.New(), orderID, &lot.ID, trigger, shares[i].Paise()); err != nil {
			return err
		}
	}
	return nil
}

// releaseLotMilestone releases lot's tranche for trigger, if the order is
// escrowed and that tranche is still pending. A non-escrowed order or a lot
// with no such tranche (this trigger, or escrow at all, was never funded)
// is a silent no-op — every call site below fires unconditionally regardless
// of whether the order opted into escrow.
func (f *Fulfilment) releaseLotMilestone(ctx context.Context, tx Tx, orderID uuid.UUID, lot domain.OrderLot, trigger domain.MilestoneTrigger, idempotencyKey string) error {
	pending, err := f.store.ListPendingMilestonesForLot(ctx, lot.ID)
	if err != nil {
		return fmt.Errorf("loading pending escrow milestones for lot %s: %w", lot.ID, err)
	}
	for _, m := range pending {
		if m.Trigger != trigger {
			continue
		}
		released, err := tx.ReleaseEscrowMilestone(ctx, m.ID, "escrow-release:"+m.ID.String())
		if err != nil {
			return fmt.Errorf("releasing escrow milestone %s: %w", m.ID, err)
		}
		if err := tx.RecordEvent(ctx, ids.New(), orderID, &lot.ID, "ESCROW_MILESTONE_RELEASED", released); err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), lot.ID.String(),
			topics.EscrowMilestoneReleased, idempotencyKey+":escrow:"+string(trigger), released)
	}
	return nil
}

func proposedToDraftLots(proposed []domain.ProposedLot, unitPricePaise int64) []domain.OrderLot {
	lots := make([]domain.OrderLot, 0, len(proposed))
	for _, p := range proposed {
		lots = append(lots, domain.OrderLot{
			ArtisanID:      p.ArtisanID,
			ClusterID:      p.ClusterID,
			Quantity:       p.Quantity,
			UnitPricePaise: unitPricePaise,
			LotValuePaise:  unitPricePaise * int64(p.Quantity),
			State:          domain.LotOffered,
		})
	}
	return lots
}

// --- outbox payloads -----------------------------------------------------------
//
// These mirror events.v1's envelope payload shapes closely enough for the
// handler layer to unmarshal, but are defined locally rather than importing
// pkg/pb (this package must stay protobuf-free per the service-layer rule).

type bulkOrderEventPayload struct {
	BulkOrderID string `json:"bulk_order_id"`
	BuyerID     string `json:"buyer_id"`
	ListingID   string `json:"listing_id"`
	Quantity    int32  `json:"quantity"`
	State       string `json:"state"`
}

func bulkOrderRequestedPayload(o domain.BulkOrder) bulkOrderEventPayload {
	return bulkOrderEventPayload{
		BulkOrderID: o.ID.String(), BuyerID: o.BuyerID, ListingID: o.ListingID.String(),
		Quantity: o.Quantity, State: string(o.State),
	}
}

func orderConfirmedPayload(o domain.BulkOrder) bulkOrderEventPayload {
	return bulkOrderEventPayload{
		BulkOrderID: o.ID.String(), BuyerID: o.BuyerID, ListingID: o.ListingID.String(),
		Quantity: o.Quantity, State: string(o.State),
	}
}

type lotEventPayload struct {
	LotID       string `json:"lot_id"`
	BulkOrderID string `json:"bulk_order_id"`
	ArtisanID   string `json:"artisan_id"`
	Quantity    int32  `json:"quantity"`
	State       string `json:"state"`
	ProgressPct int32  `json:"progress_pct"`
}

func lotOfferedPayload(l domain.OrderLot) lotEventPayload    { return toLotEventPayload(l) }
func lotAcceptedPayload(l domain.OrderLot) lotEventPayload   { return toLotEventPayload(l) }
func lotDeclinedPayload(l domain.OrderLot) lotEventPayload   { return toLotEventPayload(l) }
func lotProgressedPayload(l domain.OrderLot) lotEventPayload { return toLotEventPayload(l) }
func lotCompletedPayload(l domain.OrderLot) lotEventPayload  { return toLotEventPayload(l) }

func toLotEventPayload(l domain.OrderLot) lotEventPayload {
	return lotEventPayload{
		LotID: l.ID.String(), BulkOrderID: l.BulkOrderID.String(), ArtisanID: l.ArtisanID.String(),
		Quantity: l.Quantity, State: string(l.State), ProgressPct: l.ProgressPct,
	}
}

type qcRecordedPayload struct {
	lotEventPayload
	InspectorID string `json:"inspector_id"`
	Passed      bool   `json:"passed"`
}
