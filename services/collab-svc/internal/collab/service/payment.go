// services/collab-svc/internal/collab/service/payment.go

// Payment split, escrow milestone release and the SHG revenue split (batch
// 13). Escrow's own release calls live in fulfilment.go, next to the
// production-event transitions that trigger them (accept, first progress
// report, QC pass) — this file is the payout side: turning DELIVERED lots
// into an outbound payout instruction, never a balance the platform holds.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/money"
	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/collab-svc/internal/collab/domain"
)

// RequestPaymentSplitInput asks for one bulk order's payout instruction.
type RequestPaymentSplitInput struct {
	BulkOrderID uuid.UUID
	// CommissionPct is the platform's whole-percentage cut of each payee's
	// gross share.
	CommissionPct  int
	IdempotencyKey string
}

// payeeShare is one payee's percentage of a lot's proceeds, computed before
// any money changes hands.
type payeeShare struct {
	payeeID   string
	payeeType domain.PayeeType
	sharePct  int
}

// RequestPaymentSplit computes and persists one bulk order's payout
// instruction, proportional to DELIVERED (COMPLETED) lots only — a lot that
// was declined, expired, dropped out of or reallocated away from simply
// contributes nothing, withholding its share without anyone having to
// remember to exclude it. Each lot's own lot_value_paise is already an exact
// figure (unit price times quantity, fixed at offer time), so no rounding
// step is needed to know a lot's gross share of the order; the only place
// paisa can be lost is dividing that gross further — commission
// (Money.MulPct) and, for an SHG-owned listing, the member roster
// (Money.SplitPct) — and both of those are exact by construction.
//
// Idempotent the same way CreateBulkOrder is: a retried call with the same
// order id returns the split that already exists (created is false) rather
// than computing and inserting a second one, and the outbox event fires only
// on the call that actually creates it — an already-settled artisan is
// therefore never re-credited no matter how many times this is called.
func (f *Fulfilment) RequestPaymentSplit(ctx context.Context, in RequestPaymentSplitInput) (domain.PaymentSplit, error) {
	if in.CommissionPct < 0 || in.CommissionPct > 100 {
		return domain.PaymentSplit{}, pkgdomain.InvalidInput("commission_pct must be between 0 and 100")
	}

	order, err := f.store.GetBulkOrder(ctx, in.BulkOrderID)
	if err != nil {
		return domain.PaymentSplit{}, fmt.Errorf("loading bulk order %s: %w", in.BulkOrderID, err)
	}
	lots, err := f.store.ListLotsForOrder(ctx, in.BulkOrderID)
	if err != nil {
		return domain.PaymentSplit{}, fmt.Errorf("listing lots for order %s: %w", in.BulkOrderID, err)
	}
	listing, err := f.store.ListingFeasibility(ctx, order.ListingID)
	if err != nil {
		return domain.PaymentSplit{}, fmt.Errorf("loading listing %s: %w", order.ListingID, err)
	}

	payees, err := f.buildLotPayees(ctx, listing)
	if err != nil {
		return domain.PaymentSplit{}, err
	}

	splitID := ids.New()
	var grossTotal, commissionTotal, netTotal int64
	type pendingLine struct {
		lotID                  uuid.UUID
		payeeID                string
		payeeType              domain.PayeeType
		gross, commission, net int64
	}
	var pending []pendingLine

	for _, lot := range lots {
		if lot.State != domain.LotCompleted {
			continue
		}
		shares, err := money.New(lot.LotValuePaise).SplitPct(sharePcts(payees))
		if err != nil {
			return domain.PaymentSplit{}, fmt.Errorf("splitting lot %s across payees: %w", lot.ID, err)
		}
		for i, payee := range payees {
			gross := shares[i]
			commission := gross.MulPct(in.CommissionPct)
			net, err := gross.Sub(commission)
			if err != nil {
				return domain.PaymentSplit{}, err
			}
			pending = append(pending, pendingLine{
				lotID: lot.ID, payeeID: payee.payeeID, payeeType: payee.payeeType,
				gross: gross.Paise(), commission: commission.Paise(), net: net.Paise(),
			})
			grossTotal += gross.Paise()
			commissionTotal += commission.Paise()
			netTotal += net.Paise()
		}
	}

	var split domain.PaymentSplit
	var isNewSplit bool
	err = f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		split, isNewSplit, err = tx.CreatePaymentSplit(ctx, splitID, order.ID, grossTotal, commissionTotal, netTotal)
		if err != nil {
			return err
		}
		for _, p := range pending {
			line, _, err := tx.InsertPaymentSplitLine(ctx, ids.New(), split.ID, p.payeeID, p.payeeType, p.lotID,
				p.gross, p.commission, p.net, "payee:"+p.payeeID)
			if err != nil {
				return err
			}
			split.Lines = append(split.Lines, line)
		}
		if !isNewSplit {
			// Replay: the header (and, by the same per-line idempotency, its
			// lines) already existed. Nothing further to announce.
			return nil
		}
		if err := tx.RecordEvent(ctx, ids.New(), order.ID, nil, "PAYMENT_SPLIT_REQUESTED", split); err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), order.ID.String(),
			topics.PaymentSplitRequested, in.IdempotencyKey, split)
	})
	if err != nil {
		return domain.PaymentSplit{}, fmt.Errorf("requesting payment split for order %s: %w", in.BulkOrderID, err)
	}
	return split, nil
}

// buildLotPayees resolves who a listing's proceeds are paid to: the maker
// directly, or — for an SHG-owned listing — every member of its roster by
// their own share_pct, which sums to 100 by DB constraint trigger
// (migrations/013) and is therefore always safe to feed straight into
// money.SplitPct.
func (f *Fulfilment) buildLotPayees(ctx context.Context, listing domain.ListingInfo) ([]payeeShare, error) {
	if listing.Owner.OwnerType != domain.ListingOwnerSHG {
		return []payeeShare{{payeeID: listing.ArtisanID.String(), payeeType: domain.PayeeArtisan, sharePct: 100}}, nil
	}
	if listing.Owner.OwnerSHGID == nil {
		return nil, fmt.Errorf("listing owner type is SHG but no owner_shg_id is set")
	}
	members, err := f.store.ListSHGMemberShares(ctx, *listing.Owner.OwnerSHGID)
	if err != nil {
		return nil, fmt.Errorf("loading shg %s roster: %w", *listing.Owner.OwnerSHGID, err)
	}
	if len(members) == 0 {
		return nil, pkgdomain.Conflict("shg " + listing.Owner.OwnerSHGID.String() + " has no members to pay")
	}
	payees := make([]payeeShare, len(members))
	for i, m := range members {
		payees[i] = payeeShare{payeeID: m.ArtisanID.String(), payeeType: domain.PayeeArtisan, sharePct: int(m.SharePct)}
	}
	return payees, nil
}

func sharePcts(payees []payeeShare) []int {
	pcts := make([]int, len(payees))
	for i, p := range payees {
		pcts[i] = p.sharePct
	}
	return pcts
}

// SettleSplitLineInput records one payee's settlement.
type SettleSplitLineInput struct {
	LineID         uuid.UUID
	SettlementRef  string
	IdempotencyKey string
}

// SettleSplitLine marks one payment split line settled — what a PSP
// settlement webhook or reconciliation consumer calls once a payout
// instruction actually clears. Already-settled is a no-op (ErrNotFound from
// the guarded update is swallowed here, not surfaced), which is what makes a
// retried settlement notification safe: the exact "never re-credited under
// retry" guarantee, enforced at the DB row level by
// payment_split_line's own settled_at IS NULL guard. Once every line on the
// split has settled, the split header itself is marked settled and
// payment.settled is emitted.
func (f *Fulfilment) SettleSplitLine(ctx context.Context, in SettleSplitLineInput) (domain.PaymentSplitLine, error) {
	if in.SettlementRef == "" {
		return domain.PaymentSplitLine{}, pkgdomain.InvalidInput("settlement_ref is required")
	}

	var line domain.PaymentSplitLine
	err := f.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		line, err = tx.MarkSplitLineSettled(ctx, in.LineID, in.SettlementRef)
		if err != nil {
			if errors.Is(err, pkgdomain.ErrNotFound) {
				// Already settled (or an unknown id) — idempotent no-op.
				line = domain.PaymentSplitLine{}
				return nil
			}
			return err
		}
		split, applied, err := tx.MarkPaymentSplitSettled(ctx, line.PaymentSplitID)
		if err != nil {
			return err
		}
		if !applied {
			return nil
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), split.BulkOrderID.String(),
			topics.PaymentSettled, in.IdempotencyKey, split)
	})
	if err != nil {
		return domain.PaymentSplitLine{}, fmt.Errorf("settling payment split line %s: %w", in.LineID, err)
	}
	return line, nil
}
