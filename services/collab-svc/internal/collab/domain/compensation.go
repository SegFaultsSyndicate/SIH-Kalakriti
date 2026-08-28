// services/collab-svc/internal/collab/domain/compensation.go

// Compensation-path and payment types: buyer amendments, payment splits,
// escrow milestones and the SHG revenue split. Same rules as the rest of this
// package — no SQL, no protobuf, pure business types.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// AmendmentType is what a buyer amendment proposal changes. Exactly one of
// BulkOrderAmendment's ProposedQuantity/ProposedRequiredBy is set, matching
// which type this is.
type AmendmentType string

// The two shapes an amendment can take. A proposal is never both at once —
// the saga picks whichever the shortfall calls for (see
// service.ProposeAmendment).
const (
	AmendmentReduceQuantity AmendmentType = "REDUCE_QUANTITY"
	AmendmentExtendDeadline AmendmentType = "EXTEND_DEADLINE"
)

// AmendmentStatus is where a buyer amendment sits.
type AmendmentStatus string

// The amendment lifecycle. Terminal in both directions — a decided amendment
// is never revisited; a new shortfall raises a new proposal.
const (
	AmendmentPendingStatus AmendmentStatus = "PENDING"
	AmendmentAccepted      AmendmentStatus = "ACCEPTED"
	AmendmentDeclined      AmendmentStatus = "DECLINED"
)

// BulkOrderAmendment is a proposed change to an order's quantity or deadline,
// raised when accepted lots cannot cover what was ordered.
type BulkOrderAmendment struct {
	ID                 uuid.UUID
	BulkOrderID        uuid.UUID
	Type               AmendmentType
	ProposedQuantity   *int32
	ProposedRequiredBy *time.Time
	Reason             string
	Status             AmendmentStatus
	DecidedAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// PayeeType says what kind of party receives a payment split line. Mirrors
// fulfilment.v1.PayeeType without the UNSPECIFIED member.
type PayeeType string

// The parties a payment split line can pay. An SHG-owned lot never gets a
// PayeeSHG line itself — its share is always exploded into one PayeeArtisan
// line per member (see service.buildLotPayees) — so PayeeSHG exists for
// forward compatibility with a group that chooses not to fan payouts out
// itself, which this batch's SHG roster (share_pct per member) does not do.
const (
	PayeeArtisan PayeeType = "ARTISAN"
	PayeeSHG     PayeeType = "SELF_HELP_GROUP"
	PayeeCluster PayeeType = "CLUSTER"
)

// PaymentSplitLine is one payee's share of one lot's proceeds.
type PaymentSplitLine struct {
	ID               uuid.UUID
	PaymentSplitID   uuid.UUID
	PayeeID          string
	PayeeType        PayeeType
	LotID            uuid.UUID
	GrossAmountPaise int64
	CommissionPaise  int64
	NetAmountPaise   int64
	PayoutRef        string
	SettlementRef    *string
	SettledAt        *time.Time
}

// PaymentSplit is the full payout instruction for one bulk order: an outbound
// set of instructions telling the payment rail who to pay and how much, never
// a balance the platform holds on anyone's behalf.
type PaymentSplit struct {
	ID                   uuid.UUID
	BulkOrderID          uuid.UUID
	GrossTotalPaise      int64
	CommissionTotalPaise int64
	NetTotalPaise        int64
	Lines                []PaymentSplitLine
	SettledAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// AllSettled reports whether every line has been marked settled.
func (p PaymentSplit) AllSettled() bool {
	for _, l := range p.Lines {
		if l.SettledAt == nil {
			return false
		}
	}
	return true
}

// MilestoneTrigger is the production event that releases an escrow tranche.
// Mirrors fulfilment.v1.MilestoneTrigger without the UNSPECIFIED member.
type MilestoneTrigger string

// The triggers this batch's saga can release. DISPATCH and DELIVERY are
// schema-ready (migrations/008) but not driven by this service — shipment is
// channel-svc's domain (per the project README's service split) and has no
// event this package consumes yet.
const (
	MilestoneAdvance         MilestoneTrigger = "ADVANCE"
	MilestoneProductionStart MilestoneTrigger = "PRODUCTION_START"
	MilestoneQCPassed        MilestoneTrigger = "QC_PASSED"
	MilestoneDispatch        MilestoneTrigger = "DISPATCH"
	MilestoneDelivery        MilestoneTrigger = "DELIVERY"
)

// EscrowMilestone is one tranche of buyer funds held against one lot,
// released when its trigger fires.
type EscrowMilestone struct {
	ID          uuid.UUID
	BulkOrderID uuid.UUID
	LotID       *uuid.UUID
	Trigger     MilestoneTrigger
	AmountPaise int64
	Released    bool
	ReleasedAt  *time.Time
	ReleaseRef  *string
}

// ListingOwnerType is who receives a listing's proceeds. Mirrors
// identity.v1's listing_owner_type without needing the artisan/maker
// distinction — that stays on listing.artisan_id, untouched by this type.
type ListingOwnerType string

// The two ownership shapes a listing can have.
const (
	ListingOwnerArtisan ListingOwnerType = "ARTISAN"
	ListingOwnerSHG     ListingOwnerType = "SHG"
)

// ListingOwner is the subset of a listing's ownership the payment split
// needs: who actually gets paid, as opposed to who made the item.
type ListingOwner struct {
	OwnerType  ListingOwnerType
	OwnerSHGID *uuid.UUID
}

// SHGMemberShare is one member's payout percentage of their group's earnings.
type SHGMemberShare struct {
	ArtisanID uuid.UUID
	SharePct  int32
}
