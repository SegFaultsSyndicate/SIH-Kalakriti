// services/collab-svc/internal/collab/repo/payment.go
package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ZoroNewbie00/kalakriti/services/collab-svc/internal/collab/domain"
	"github.com/ZoroNewbie00/kalakriti/services/collab-svc/internal/collab/repo/db"
)

// --- amendments (Tx) ---------------------------------------------------------

// CreateAmendment raises a buyer amendment proposal. The one-pending-per-order
// unique index (migrations/021) makes a second proposal while one is still
// pending a conflict rather than a silent double-propose.
func (t *Tx) CreateAmendment(ctx context.Context, id uuid.UUID, in domain.BulkOrderAmendment) (domain.BulkOrderAmendment, error) {
	row, err := t.q.CreateAmendment(ctx, db.CreateAmendmentParams{
		ID:                 id,
		BulkOrderID:        in.BulkOrderID,
		AmendmentType:      db.AmendmentType(in.Type),
		ProposedQuantity:   in.ProposedQuantity,
		ProposedRequiredBy: in.ProposedRequiredBy,
		Reason:             in.Reason,
	})
	if err != nil {
		return domain.BulkOrderAmendment{}, translate(err, "bulk order amendment")
	}
	return toDomainAmendment(row), nil
}

// DecideAmendment moves a PENDING amendment to ACCEPTED or DECLINED. Zero rows
// (already decided) is reported as ErrConflict, matching every other guarded
// transition in this package.
func (t *Tx) DecideAmendment(ctx context.Context, amendmentID uuid.UUID, status domain.AmendmentStatus) (domain.BulkOrderAmendment, error) {
	row, err := t.q.DecideAmendment(ctx, db.DecideAmendmentParams{ID: amendmentID, Status: db.AmendmentStatus(status)})
	if err != nil {
		return domain.BulkOrderAmendment{}, translate(err, "bulk order amendment decision")
	}
	return toDomainAmendment(row), nil
}

// ReduceBulkOrderQuantity applies an accepted REDUCE_QUANTITY amendment:
// quantity, total_value_paise (recomputed by the caller from the pinned unit
// price so no rounding drifts in) and the order's next state all move
// together.
func (t *Tx) ReduceBulkOrderQuantity(ctx context.Context, orderID uuid.UUID, quantity int32, totalValuePaise int64, state domain.BulkOrderState) (domain.BulkOrder, error) {
	row, err := t.q.ReduceBulkOrderQuantity(ctx, db.ReduceBulkOrderQuantityParams{
		ID: orderID, Quantity: quantity, TotalValuePaise: totalValuePaise, State: db.BulkOrderState(state),
	})
	if err != nil {
		return domain.BulkOrder{}, translate(err, "bulk order quantity reduction")
	}
	return toDomainBulkOrder(row), nil
}

// ExtendBulkOrderDeadline applies an accepted EXTEND_DEADLINE amendment.
func (t *Tx) ExtendBulkOrderDeadline(ctx context.Context, orderID uuid.UUID, requiredBy time.Time, state domain.BulkOrderState) (domain.BulkOrder, error) {
	row, err := t.q.ExtendBulkOrderDeadline(ctx, db.ExtendBulkOrderDeadlineParams{
		ID: orderID, RequiredBy: requiredBy, State: db.BulkOrderState(state),
	})
	if err != nil {
		return domain.BulkOrder{}, translate(err, "bulk order deadline extension")
	}
	return toDomainBulkOrder(row), nil
}

// GetAmendment loads one amendment by id (Store, autocommit read).
func (r *Repo) GetAmendment(ctx context.Context, id uuid.UUID) (domain.BulkOrderAmendment, error) {
	row, err := r.q.GetAmendment(ctx, id)
	if err != nil {
		return domain.BulkOrderAmendment{}, translate(err, "bulk order amendment")
	}
	return toDomainAmendment(row), nil
}

func toDomainAmendment(row db.BulkOrderAmendment) domain.BulkOrderAmendment {
	return domain.BulkOrderAmendment{
		ID:                 row.ID,
		BulkOrderID:        row.BulkOrderID,
		Type:               domain.AmendmentType(row.AmendmentType),
		ProposedQuantity:   row.ProposedQuantity,
		ProposedRequiredBy: row.ProposedRequiredBy,
		Reason:             row.Reason,
		Status:             domain.AmendmentStatus(row.Status),
		DecidedAt:          row.DecidedAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

// --- QC rework support (Store, autocommit read) ------------------------------

// CountFailedQC reports how many failed inspections a lot already has,
// letting SubmitQC tell a first failure (goes to QC_FAILED with a rework
// window) from a second (goes straight to REALLOCATED) apart.
func (r *Repo) CountFailedQC(ctx context.Context, lotID uuid.UUID) (int, error) {
	n, err := r.q.CountFailedQC(ctx, lotID)
	if err != nil {
		return 0, translate(err, "qc failure count")
	}
	return int(n), nil
}

// --- listing ownership and SHG roster (Store, autocommit read) --------------

// ListSHGMemberShares loads one self-help group's payout roster. Every row's
// share_pct sums to 100 across the whole roster by DB constraint trigger
// (migrations/013), which is exactly the invariant service.buildLotPayees
// relies on to split a lot's gross paisa-exact with money.SplitPct.
func (r *Repo) ListSHGMemberShares(ctx context.Context, shgID uuid.UUID) ([]domain.SHGMemberShare, error) {
	rows, err := r.q.ListSHGMemberShares(ctx, shgID)
	if err != nil {
		return nil, translate(err, "shg member shares")
	}
	out := make([]domain.SHGMemberShare, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.SHGMemberShare{ArtisanID: row.ArtisanID, SharePct: row.SharePct})
	}
	return out, nil
}

// --- payment split (Tx) ------------------------------------------------------

// CreatePaymentSplit inserts a bulk order's payout header, or returns the one
// that already exists (created is false) — the get-or-insert idiom every
// idempotent create in this package uses, here guarding against a retried
// RequestPaymentSplit call producing a second payout instruction for the same
// order.
func (t *Tx) CreatePaymentSplit(ctx context.Context, id, orderID uuid.UUID, grossTotal, commissionTotal, netTotal int64) (domain.PaymentSplit, bool, error) {
	row, err := t.q.CreatePaymentSplitIdempotent(ctx, db.CreatePaymentSplitIdempotentParams{
		ID: id, BulkOrderID: orderID,
		GrossTotalPaise: grossTotal, CommissionTotalPaise: commissionTotal, NetTotalPaise: netTotal,
	})
	if err != nil {
		return domain.PaymentSplit{}, false, translate(err, "payment split")
	}
	return toDomainPaymentSplit(row.PaymentSplit), row.IsNew, nil
}

// InsertPaymentSplitLine inserts one payee's share of one lot, or returns the
// line that already exists (created is false) on a retry — the same
// unique-on-(split, lot, payee) guard that stops an already-settled artisan
// from ever being re-credited, since the caller only enqueues
// payment.split.requested for lines where created is true.
func (t *Tx) InsertPaymentSplitLine(ctx context.Context, id, splitID uuid.UUID, payeeID string, payeeType domain.PayeeType, lotID uuid.UUID, gross, commission, net int64, payoutRef string) (domain.PaymentSplitLine, bool, error) {
	row, err := t.q.InsertPaymentSplitLineIdempotent(ctx, db.InsertPaymentSplitLineIdempotentParams{
		ID: id, PaymentSplitID: splitID, PayeeID: payeeID, PayeeType: db.PayeeType(payeeType), LotID: lotID,
		GrossAmountPaise: gross, CommissionPaise: commission, NetAmountPaise: net, PayoutRef: payoutRef,
	})
	if err != nil {
		return domain.PaymentSplitLine{}, false, translate(err, "payment split line")
	}
	return toDomainPaymentSplitLine(row.PaymentSplitLine), row.IsNew, nil
}

// MarkSplitLineSettled records one payee's settlement. Already-settled is a
// zero-row update (ErrNotFound), which the service layer treats as an
// idempotent no-op — the exact "never re-credited under retry" guard.
func (t *Tx) MarkSplitLineSettled(ctx context.Context, lineID uuid.UUID, settlementRef string) (domain.PaymentSplitLine, error) {
	row, err := t.q.MarkSplitLineSettled(ctx, db.MarkSplitLineSettledParams{ID: lineID, SettlementRef: &settlementRef})
	if err != nil {
		return domain.PaymentSplitLine{}, translate(err, "payment split line settlement")
	}
	return toDomainPaymentSplitLine(row), nil
}

// MarkPaymentSplitSettled flips a split's settled_at once every one of its
// lines has settled — the WHERE NOT EXISTS guard in the query itself is what
// enforces that, so this either fully applies or is a no-op, never partial.
func (t *Tx) MarkPaymentSplitSettled(ctx context.Context, splitID uuid.UUID) (domain.PaymentSplit, bool, error) {
	row, err := t.q.MarkPaymentSplitSettled(ctx, splitID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PaymentSplit{}, false, nil
		}
		return domain.PaymentSplit{}, false, translate(err, "payment split settlement")
	}
	return toDomainPaymentSplit(row), true, nil
}

// GetPaymentSplit loads one order's payout instruction with every line
// (Store, autocommit read).
func (r *Repo) GetPaymentSplit(ctx context.Context, orderID uuid.UUID) (domain.PaymentSplit, bool, error) {
	row, err := r.q.GetPaymentSplitByOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PaymentSplit{}, false, nil
		}
		return domain.PaymentSplit{}, false, translate(err, "payment split")
	}
	lines, err := r.q.ListSplitLinesByOrder(ctx, orderID)
	if err != nil {
		return domain.PaymentSplit{}, false, translate(err, "payment split lines")
	}
	split := toDomainPaymentSplit(row)
	for _, l := range lines {
		split.Lines = append(split.Lines, toDomainPaymentSplitLine(l))
	}
	return split, true, nil
}

func toDomainPaymentSplit(row db.PaymentSplit) domain.PaymentSplit {
	return domain.PaymentSplit{
		ID:                   row.ID,
		BulkOrderID:          row.BulkOrderID,
		GrossTotalPaise:      row.GrossTotalPaise,
		CommissionTotalPaise: row.CommissionTotalPaise,
		NetTotalPaise:        row.NetTotalPaise,
		SettledAt:            row.SettledAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func toDomainPaymentSplitLine(row db.PaymentSplitLine) domain.PaymentSplitLine {
	return domain.PaymentSplitLine{
		ID:               row.ID,
		PaymentSplitID:   row.PaymentSplitID,
		PayeeID:          row.PayeeID,
		PayeeType:        domain.PayeeType(row.PayeeType),
		LotID:            row.LotID,
		GrossAmountPaise: row.GrossAmountPaise,
		CommissionPaise:  row.CommissionPaise,
		NetAmountPaise:   row.NetAmountPaise,
		PayoutRef:        row.PayoutRef,
		SettlementRef:    row.SettlementRef,
		SettledAt:        row.SettledAt,
	}
}

// --- escrow (Tx + Store) ------------------------------------------------------

// CreateEscrowMilestone inserts one tranche, or reports it already exists
// (created is false) — ProposeAllocation calls this once per (lot, trigger)
// when an order is escrow-enabled, and a retried allocation pass must not
// double-fund a tranche.
func (t *Tx) CreateEscrowMilestone(ctx context.Context, id, orderID uuid.UUID, lotID *uuid.UUID, trigger domain.MilestoneTrigger, amountPaise int64) (domain.EscrowMilestone, bool, error) {
	row, err := t.q.CreateEscrowMilestone(ctx, db.CreateEscrowMilestoneParams{
		ID: id, BulkOrderID: orderID, LotID: lotID, Trigger: db.MilestoneTrigger(trigger), AmountPaise: amountPaise,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.EscrowMilestone{}, false, nil
		}
		return domain.EscrowMilestone{}, false, translate(err, "escrow milestone")
	}
	return toDomainEscrowMilestone(row), true, nil
}

// ReleaseEscrowMilestone releases one tranche. Already-released is a
// zero-row update surfaced as ErrNotFound, which the service layer treats as
// an idempotent no-op.
func (t *Tx) ReleaseEscrowMilestone(ctx context.Context, id uuid.UUID, releaseRef string) (domain.EscrowMilestone, error) {
	row, err := t.q.ReleaseEscrowMilestone(ctx, db.ReleaseEscrowMilestoneParams{ID: id, ReleaseRef: &releaseRef})
	if err != nil {
		return domain.EscrowMilestone{}, translate(err, "escrow milestone release")
	}
	return toDomainEscrowMilestone(row), nil
}

// ListPendingMilestonesForLot loads one lot's unreleased tranches (Store,
// autocommit read) — what RespondToLot/ReportProgress/SubmitQC each check to
// decide whether their trigger has a tranche waiting.
func (r *Repo) ListPendingMilestonesForLot(ctx context.Context, lotID uuid.UUID) ([]domain.EscrowMilestone, error) {
	rows, err := r.q.ListMilestonesForLot(ctx, lotID)
	if err != nil {
		return nil, translate(err, "escrow milestones")
	}
	out := make([]domain.EscrowMilestone, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainEscrowMilestone(row))
	}
	return out, nil
}

func toDomainEscrowMilestone(row db.EscrowMilestone) domain.EscrowMilestone {
	return domain.EscrowMilestone{
		ID: row.ID, BulkOrderID: row.BulkOrderID, LotID: row.LotID,
		Trigger: domain.MilestoneTrigger(row.Trigger), AmountPaise: row.AmountPaise,
		Released: row.Released, ReleasedAt: row.ReleasedAt, ReleaseRef: row.ReleaseRef,
	}
}
