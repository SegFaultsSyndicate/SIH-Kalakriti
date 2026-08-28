// services/collab-svc/internal/collab/domain/fulfilment.go

// Package domain holds collab-svc's business types and pure logic: the saga
// state machine's legal transitions, the allocation algorithm that splits a
// bulk order into lots, and the deadline feasibility check. Nothing here does
// SQL or knows about protobuf or gRPC.
package domain

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

// BulkOrderState is where a whole buyer order sits in the saga. Mirrors
// fulfilment.v1.BulkOrderState without the UNSPECIFIED member.
type BulkOrderState string

// The bulk order lifecycle. AMENDMENT_PENDING exists in the proto and schema
// for a later batch (mid-production scope changes); this batch's saga does
// not drive any transition into or out of it. CANCELLED is driven by this
// batch — see CancelBulkOrder in service/fulfilment.go — for the case of a
// buyer or system cancellation before production has started.
const (
	BulkOrderAllocating         BulkOrderState = "ALLOCATING"
	BulkOrderPartiallyAllocated BulkOrderState = "PARTIALLY_ALLOCATED"
	BulkOrderConfirmed          BulkOrderState = "CONFIRMED"
	BulkOrderInProduction       BulkOrderState = "IN_PRODUCTION"
	BulkOrderAmendmentPending   BulkOrderState = "AMENDMENT_PENDING"
	BulkOrderCompleted          BulkOrderState = "COMPLETED"
	BulkOrderCancelled          BulkOrderState = "CANCELLED"
)

// LotState is where one artisan's slice of an order sits. Mirrors
// fulfilment.v1.LotState without the UNSPECIFIED member.
type LotState string

// The lot lifecycle.
const (
	LotOffered      LotState = "OFFERED"
	LotAccepted     LotState = "ACCEPTED"
	LotDeclined     LotState = "DECLINED"
	LotExpired      LotState = "EXPIRED"
	LotInProduction LotState = "IN_PRODUCTION"
	LotQCPending    LotState = "QC_PENDING"
	LotQCFailed     LotState = "QC_FAILED"
	LotCompleted    LotState = "COMPLETED"
	LotReallocated  LotState = "REALLOCATED"
)

// ReservationState is whether a capacity hold is live, spent or returned.
type ReservationState string

// The reservation lifecycle.
const (
	ReservationHeld     ReservationState = "HELD"
	ReservationConsumed ReservationState = "CONSUMED"
	ReservationReleased ReservationState = "RELEASED"
)

// MinLotSize is the smallest quantity the allocator will ever offer as one
// lot. A remainder smaller than this is folded into the previous candidate's
// lot rather than offered on its own.
//
// ponytail: a fixed platform-wide floor, not a per-craft configured value —
// promote to config if a craft ever needs a different minimum.
const MinLotSize = 5

// BulkOrder is one buyer requirement for many units of a single listing.
type BulkOrder struct {
	ID                uuid.UUID
	BuyerID           string
	ListingID         uuid.UUID
	ProductID         uuid.UUID
	Quantity          int32
	UnitPricePaise    int64
	TotalValuePaise   int64
	RequiredBy        time.Time
	State             BulkOrderState
	AllocatedQuantity int32
	Customisations    map[string]string
	Notes             *string
	IdempotencyKey    string
	// EscrowEnabled opts this order into milestone-based escrow (batch 13):
	// buyer funds held and released per delivered-and-QC-passed lot rather
	// than paid out in one instruction at completion. Never set for anything
	// other than a bulk order — this schema has no single-artisan direct
	// sale to gate it against.
	EscrowEnabled bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OrderLot is the slice of a bulk order offered to one artisan.
type OrderLot struct {
	ID                    uuid.UUID
	BulkOrderID           uuid.UUID
	ArtisanID             uuid.UUID
	ClusterID             *uuid.UUID
	Quantity              int32
	UnitPricePaise        int64
	LotValuePaise         int64
	State                 LotState
	OfferedAt             time.Time
	RespondsBy            time.Time
	AcceptedAt            *time.Time
	PromisedShipDate      *time.Time
	ProgressPct           int32
	CapacityReservationID *uuid.UUID
	DeclineReason         *string
	ReallocatedFromLotID  *uuid.UUID
	DropoutReason         *string
	ReworkDeadline        *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// CapacityReservation holds part of an artisan's monthly capacity against a lot.
type CapacityReservation struct {
	ID          uuid.UUID
	ArtisanID   uuid.UUID
	ListingID   uuid.UUID
	LotID       uuid.UUID
	Units       int32
	PeriodStart time.Time
	PeriodEnd   time.Time
	ExpiresAt   time.Time
	State       ReservationState
}

// ListingInfo is the subset of a listing the saga needs to check feasibility
// and run allocation.
type ListingInfo struct {
	ProductID           uuid.UUID
	CraftID             uuid.UUID
	ArtisanID           uuid.UUID
	UnitPricePaise      int64
	TypicalLeadTimeDays int32
	// Owner is who the order's proceeds are paid to — the maker (ArtisanID
	// above) unless the listing is SHG-owned, in which case the payment
	// split fans out across the group's roster instead. See
	// service.buildLotPayees.
	Owner ListingOwner
}

// CandidateQuery narrows the candidate scan to one order's craft and
// (optionally) a preferred artisan set and/or hub cluster.
type CandidateQuery struct {
	CraftID                uuid.UUID
	PreferredArtisanIDs    []uuid.UUID
	HubClusterID           *uuid.UUID
	PeriodStart, PeriodEnd time.Time
}

// LotTransitionFields carries the extra columns a lot transition sets beyond
// state itself — which ones apply depends on the target state, and the repo
// only writes the ones relevant to it.
type LotTransitionFields struct {
	AcceptedAt       *time.Time
	PromisedShipDate *time.Time
	DeclineReason    *string
	ProgressPct      *int32
	// DropoutReason is set only on a dropout-driven REALLOCATED transition —
	// separate from DeclineReason because the DB ties that column to state =
	// DECLINED specifically (order_lot_decline_reason_check).
	DropoutReason *string
	// ReworkDeadline is set only on the QC_PENDING -> QC_FAILED transition,
	// starting the rework window a first-time non-critical failure gets.
	ReworkDeadline *time.Time
}

// DefectSeverity grades a QC fault. Mirrors fulfilment.v1.DefectSeverity
// without the UNSPECIFIED member.
type DefectSeverity string

// The severities a QC defect can carry. CRITICAL is what drives a lot to
// REALLOCATED rather than leaving it QC_FAILED — see lotTransitions.
const (
	DefectMinor    DefectSeverity = "MINOR"
	DefectMajor    DefectSeverity = "MAJOR"
	DefectCritical DefectSeverity = "CRITICAL"
)

// QCDefect is one fault an inspector recorded against a lot.
type QCDefect struct {
	ID            uuid.UUID
	Code          string
	Description   string
	Severity      DefectSeverity
	MediaIDs      []uuid.UUID
	AffectedUnits int32
}

// QCResult is one inspection outcome for one lot.
type QCResult struct {
	ID          uuid.UUID
	LotID       uuid.UUID
	InspectorID string
	Passed      bool
	Notes       *string
	MediaIDs    []uuid.UUID
	Defects     []QCDefect
	InspectedAt time.Time
}

// HasCriticalDefect reports whether any recorded defect is severe enough to
// send the lot to REALLOCATED instead of leaving it QC_FAILED.
func (r QCResult) HasCriticalDefect() bool {
	for _, d := range r.Defects {
		if d.Severity == DefectCritical {
			return true
		}
	}
	return false
}

// --- saga transition tables -----------------------------------------------

// bulkOrderTransitions enumerates every legal BulkOrder state change this
// batch's saga performs. A transition not listed here is rejected with
// ErrConflict. CANCELLED is reachable from any pre-production state; once
// IN_PRODUCTION, cancellation is an amendment-flow concern out of this
// batch's scope, not a plain state flip (artisans are already committed).
//
// AMENDMENT_PENDING is driven by this batch (service.ProposeAmendment) when
// accepted lots cannot cover the order's quantity and no further candidates
// exist: the order pauses for a buyer decision rather than staying stuck in
// PARTIALLY_ALLOCATED indefinitely. Deciding the amendment either re-plans
// (back to ALLOCATING/PARTIALLY_ALLOCATED, see service.DecideAmendment) or
// cancels outright on a decline.
var bulkOrderTransitions = map[BulkOrderState]map[BulkOrderState]bool{
	BulkOrderAllocating: {
		BulkOrderPartiallyAllocated: true, BulkOrderConfirmed: true,
		BulkOrderAmendmentPending: true, BulkOrderCancelled: true,
	},
	BulkOrderPartiallyAllocated: {
		BulkOrderPartiallyAllocated: true, BulkOrderConfirmed: true,
		BulkOrderAmendmentPending: true, BulkOrderCancelled: true,
	},
	BulkOrderConfirmed: {BulkOrderInProduction: true, BulkOrderCancelled: true},
	// COMPLETED from IN_PRODUCTION covers both the normal path (every lot
	// COMPLETED, see service.SubmitQC) and service.SettlePartial: a dropout
	// that cannot be refilled before the deadline settles the order on
	// whatever lots actually delivered rather than waiting on a slot that
	// will never fill. Both call sites already only fire when no lot is
	// still open, so one edge covers both.
	BulkOrderInProduction: {BulkOrderCompleted: true},
	BulkOrderAmendmentPending: {
		// Confirmed is reachable directly when an accepted REDUCE_QUANTITY
		// amendment turns out to already be fully covered by lots accepted
		// before the shortfall was even raised.
		BulkOrderAllocating: true, BulkOrderPartiallyAllocated: true,
		BulkOrderConfirmed: true, BulkOrderCancelled: true,
	},
}

// CanTransitionBulkOrder reports whether the saga may move a bulk order from
// one state to another.
func CanTransitionBulkOrder(from, to BulkOrderState) bool {
	return bulkOrderTransitions[from][to]
}

// lotTransitions enumerates every legal OrderLot state change. QC_PENDING can
// go to REALLOCATED directly on a CRITICAL defect (per DefectSeverity's own
// proto doc: "Unsaleable; the lot is reallocated.").
//
// QC_FAILED now has two ways out (batch 13): QC_PENDING on a rework
// resubmission within the rework window (service.ResubmitForQC), or straight
// to REALLOCATED when that window lapses without one (service.ExpireRework).
// A second QC failure is not a distinct edge — it re-enters QC_PENDING via
// the same resubmission and is caught by service.SubmitQC counting prior
// failed qc_result rows for the lot, sending it to REALLOCATED instead of
// QC_FAILED again.
//
// ACCEPTED and IN_PRODUCTION can both go to REALLOCATED directly — an
// artisan dropping out before or during production (service.Dropout) — with
// no state in between; DropoutReason on the transition explains why.
var lotTransitions = map[LotState]map[LotState]bool{
	LotOffered:      {LotAccepted: true, LotDeclined: true, LotExpired: true},
	LotAccepted:     {LotInProduction: true, LotReallocated: true},
	LotInProduction: {LotQCPending: true, LotReallocated: true},
	LotQCPending:    {LotCompleted: true, LotQCFailed: true, LotReallocated: true},
	LotQCFailed:     {LotQCPending: true, LotReallocated: true},
}

// CanTransitionLot reports whether the saga may move a lot from one state to
// another.
func CanTransitionLot(from, to LotState) bool {
	return lotTransitions[from][to]
}

// --- feasibility -------------------------------------------------------------

// FeasibilityInput is what CheckDeadlineFeasible needs about the craft being
// ordered and when the order is being placed.
type FeasibilityInput struct {
	// TypicalLeadTimeDays is read off the listing (listing.lead_time_days),
	// not a separate per-craft table — every made-to-order listing already
	// states its own lead time, and that is a truer number than a
	// craft-wide average would be.
	TypicalLeadTimeDays int32
	// AllocationBufferDays covers the time ProposeAllocation/OfferLots/
	// RespondToLot need before production can even start; production lead
	// time alone understates how long the whole saga takes.
	AllocationBufferDays int32
	Now                  time.Time
	RequiredBy           time.Time
}

// CheckDeadlineFeasible reports whether RequiredBy leaves enough runway for
// allocation plus the listing's own typical lead time. reason is set only
// when infeasible, phrased for direct display to the buyer.
func CheckDeadlineFeasible(in FeasibilityInput) (feasible bool, reason string) {
	needed := time.Duration(in.AllocationBufferDays+in.TypicalLeadTimeDays) * 24 * time.Hour
	earliest := in.Now.Add(needed)
	if in.RequiredBy.Before(earliest) {
		return false, "deadline too soon: this craft typically needs " +
			formatDays(in.TypicalLeadTimeDays+in.AllocationBufferDays) +
			" from order placement to delivery"
	}
	return true, ""
}

func formatDays(d int32) string {
	if d == 1 {
		return "1 day"
	}
	return strconv.Itoa(int(d)) + " days"
}
