// services/collab-svc/internal/collab/service/payment_test.go
package service

import (
	"context"
	"math/rand"
	"testing"

	"github.com/google/uuid"

	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/domain"
)

// seedOrderWithLots wires up a bulk order plus listing directly (bypassing
// the allocation saga, which RequestPaymentSplit does not touch) and returns
// the order alongside the lots as stored, so a test can assert against the
// exact ids/values it seeded.
func seedOrderWithLots(store *fakeStore, owner domain.ListingOwner, lots []domain.OrderLot) (domain.BulkOrder, []domain.OrderLot) {
	listingID := uuid.New()
	store.listings[listingID] = domain.ListingInfo{ArtisanID: uuid.New(), Owner: owner}
	order := domain.BulkOrder{ID: uuid.New(), BuyerID: "buyer-1", ListingID: listingID, State: domain.BulkOrderInProduction}
	store.orders[order.ID] = order
	for i := range lots {
		lots[i].ID = uuid.New()
		lots[i].BulkOrderID = order.ID
		store.lots[lots[i].ID] = lots[i]
	}
	return order, lots
}

func TestRequestPaymentSplitPaysOnlyDeliveredLotsExactly(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	delivered1 := domain.OrderLot{ArtisanID: uuid.New(), Quantity: 20, UnitPricePaise: 10007, LotValuePaise: 20 * 10007, State: domain.LotCompleted}
	delivered2 := domain.OrderLot{ArtisanID: uuid.New(), Quantity: 13, UnitPricePaise: 9999, LotValuePaise: 13 * 9999, State: domain.LotCompleted}
	gaveUp := domain.OrderLot{ArtisanID: uuid.New(), Quantity: 7, UnitPricePaise: 10000, LotValuePaise: 70000, State: domain.LotReallocated}
	order, seeded := seedOrderWithLots(store, domain.ListingOwner{OwnerType: domain.ListingOwnerArtisan}, []domain.OrderLot{delivered1, delivered2, gaveUp})
	gaveUpID := seeded[2].ID

	split, err := f.RequestPaymentSplit(ctx, RequestPaymentSplitInput{BulkOrderID: order.ID, CommissionPct: 15, IdempotencyKey: "pay-1"})
	if err != nil {
		t.Fatalf("RequestPaymentSplit: %v", err)
	}
	if len(split.Lines) != 2 {
		t.Fatalf("expected 2 lines (delivered lots only), got %d: %+v", len(split.Lines), split.Lines)
	}
	wantGross := delivered1.LotValuePaise + delivered2.LotValuePaise
	if split.GrossTotalPaise != wantGross {
		t.Errorf("gross total = %d, want %d", split.GrossTotalPaise, wantGross)
	}
	var sumGross, sumCommission, sumNet int64
	for _, l := range split.Lines {
		if l.LotID == gaveUpID {
			t.Fatalf("a reallocated lot must never appear in the split: %+v", l)
		}
		if l.GrossAmountPaise != l.CommissionPaise+l.NetAmountPaise {
			t.Errorf("line %s: gross %d != commission %d + net %d", l.ID, l.GrossAmountPaise, l.CommissionPaise, l.NetAmountPaise)
		}
		sumGross += l.GrossAmountPaise
		sumCommission += l.CommissionPaise
		sumNet += l.NetAmountPaise
	}
	if sumGross != split.GrossTotalPaise || sumCommission != split.CommissionTotalPaise || sumNet != split.NetTotalPaise {
		t.Errorf("line sums (%d/%d/%d) do not match split totals (%d/%d/%d)",
			sumGross, sumCommission, sumNet, split.GrossTotalPaise, split.CommissionTotalPaise, split.NetTotalPaise)
	}
	assertHasOutbox(t, store, "payment.split.requested")
}

// TestRequestPaymentSplitIsIdempotentNeverDoubleCredits calls
// RequestPaymentSplit twice for the same order and asserts the second call
// is a pure replay: same split id, same lines, exactly one outbox row.
func TestRequestPaymentSplitIsIdempotentNeverDoubleCredits(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	lot := domain.OrderLot{ArtisanID: uuid.New(), Quantity: 10, UnitPricePaise: 10000, LotValuePaise: 100000, State: domain.LotCompleted}
	order, _ := seedOrderWithLots(store, domain.ListingOwner{OwnerType: domain.ListingOwnerArtisan}, []domain.OrderLot{lot})

	in := RequestPaymentSplitInput{BulkOrderID: order.ID, CommissionPct: 10, IdempotencyKey: "pay-1"}
	first, err := f.RequestPaymentSplit(ctx, in)
	if err != nil {
		t.Fatalf("first RequestPaymentSplit: %v", err)
	}
	second, err := f.RequestPaymentSplit(ctx, in)
	if err != nil {
		t.Fatalf("replayed RequestPaymentSplit: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("replay created a second split: first %s, second %s", first.ID, second.ID)
	}
	if len(store.splits) != 1 {
		t.Errorf("expected exactly one stored split, got %d", len(store.splits))
	}
	if len(second.Lines) != 1 {
		t.Errorf("expected exactly one line after the replay, got %d", len(second.Lines))
	}
	outboxHits := 0
	for _, row := range store.outbox {
		if row.topic == "payment.split.requested" {
			outboxHits++
		}
	}
	if outboxHits != 1 {
		t.Errorf("expected exactly one payment.split.requested outbox row across the replay, got %d", outboxHits)
	}
}

// TestSettleSplitLineNeverDoubleCreditsOnRetry settles the same line twice
// (a retried settlement webhook) and asserts the second call is a no-op:
// settled_at does not move and payment.settled fires exactly once.
func TestSettleSplitLineNeverDoubleCreditsOnRetry(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	lot := domain.OrderLot{ArtisanID: uuid.New(), Quantity: 10, UnitPricePaise: 10000, LotValuePaise: 100000, State: domain.LotCompleted}
	order, _ := seedOrderWithLots(store, domain.ListingOwner{OwnerType: domain.ListingOwnerArtisan}, []domain.OrderLot{lot})
	split, err := f.RequestPaymentSplit(ctx, RequestPaymentSplitInput{BulkOrderID: order.ID, CommissionPct: 10, IdempotencyKey: "pay-1"})
	if err != nil {
		t.Fatalf("RequestPaymentSplit: %v", err)
	}
	lineID := split.Lines[0].ID

	in := SettleSplitLineInput{LineID: lineID, SettlementRef: "psp-ref-1", IdempotencyKey: "settle-1"}
	if _, err := f.SettleSplitLine(ctx, in); err != nil {
		t.Fatalf("first SettleSplitLine: %v", err)
	}
	firstSettledAt := store.lines[lineID].SettledAt
	if firstSettledAt == nil {
		t.Fatalf("expected the line to be settled")
	}

	if _, err := f.SettleSplitLine(ctx, in); err != nil {
		t.Fatalf("retried SettleSplitLine: %v", err)
	}
	if got := store.lines[lineID].SettledAt; got == nil || !got.Equal(*firstSettledAt) {
		t.Errorf("settled_at moved on retry: first %v, second %v", firstSettledAt, got)
	}
	settledEvents := 0
	for _, row := range store.outbox {
		if row.topic == "payment.settled" {
			settledEvents++
		}
	}
	if settledEvents != 1 {
		t.Errorf("expected exactly one payment.settled outbox row across the retry, got %d", settledEvents)
	}
}

// TestSHGOwnedLotSplitsByMemberPercentagesExactly is the SHG nested-split
// requirement: an SHG-owned lot's share fans out into one line per member, by
// their own share_pct, summing back to exactly the lot's value.
func TestSHGOwnedLotSplitsByMemberPercentagesExactly(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	f := newTestFulfilment(store)
	ctx := context.Background()

	shgID := uuid.New()
	member1, member2, member3 := uuid.New(), uuid.New(), uuid.New()
	store.shgMembers[shgID] = []domain.SHGMemberShare{
		{ArtisanID: member1, SharePct: 50},
		{ArtisanID: member2, SharePct: 30},
		{ArtisanID: member3, SharePct: 20},
	}

	lot := domain.OrderLot{Quantity: 7, UnitPricePaise: 10007, LotValuePaise: 7 * 10007, State: domain.LotCompleted} // 70049, does not divide evenly
	order, _ := seedOrderWithLots(store, domain.ListingOwner{OwnerType: domain.ListingOwnerSHG, OwnerSHGID: &shgID}, []domain.OrderLot{lot})

	split, err := f.RequestPaymentSplit(ctx, RequestPaymentSplitInput{BulkOrderID: order.ID, CommissionPct: 5, IdempotencyKey: "pay-1"})
	if err != nil {
		t.Fatalf("RequestPaymentSplit: %v", err)
	}
	if len(split.Lines) != 3 {
		t.Fatalf("expected one line per SHG member, got %d", len(split.Lines))
	}
	var sumGross int64
	seen := map[uuid.UUID]bool{}
	for _, l := range split.Lines {
		seen[uuid.MustParse(l.PayeeID)] = true
		sumGross += l.GrossAmountPaise
	}
	for _, m := range []uuid.UUID{member1, member2, member3} {
		if !seen[m] {
			t.Errorf("member %s has no payment line", m)
		}
	}
	if sumGross != lot.LotValuePaise {
		t.Errorf("sum of member gross shares = %d, want exactly the lot value %d", sumGross, lot.LotValuePaise)
	}
}

// TestPaymentSplitConservesMoneyAcrossRandomFixtures is the acceptance
// criterion: over 100 randomised quantity/price fixtures (some SHG-owned,
// exercising the nested split too), the sum of every payment split line adds
// back to exactly the delivered lots' total value — no paisa created or lost.
func TestPaymentSplitConservesMoneyAcrossRandomFixtures(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(20260827))

	for i := 0; i < 100; i++ {
		store := newFakeStore()
		f := newTestFulfilment(store)
		ctx := context.Background()

		var owner domain.ListingOwner
		var wantLines int
		numLots := 1 + rng.Intn(5)
		if rng.Intn(2) == 0 {
			owner = domain.ListingOwner{OwnerType: domain.ListingOwnerArtisan}
			wantLines = numLots
		} else {
			shgID := uuid.New()
			numMembers := 1 + rng.Intn(4)
			pcts := randomPercentagesSummingTo100(rng, numMembers)
			members := make([]domain.SHGMemberShare, numMembers)
			for m := range members {
				members[m] = domain.SHGMemberShare{ArtisanID: uuid.New(), SharePct: int32(pcts[m])}
			}
			store.shgMembers[shgID] = members
			owner = domain.ListingOwner{OwnerType: domain.ListingOwnerSHG, OwnerSHGID: &shgID}
			wantLines = numLots * numMembers
		}

		var lots []domain.OrderLot
		var wantTotal int64
		for l := 0; l < numLots; l++ {
			quantity := int32(1 + rng.Intn(500))
			unitPrice := int64(1 + rng.Intn(1_000_00)) // up to ₹1,000.00
			value := unitPrice * int64(quantity)
			lots = append(lots, domain.OrderLot{Quantity: quantity, UnitPricePaise: unitPrice, LotValuePaise: value, State: domain.LotCompleted})
			wantTotal += value
		}
		// A withheld lot (dropped out or reallocated) never contributes.
		lots = append(lots, domain.OrderLot{Quantity: 999, UnitPricePaise: 999, LotValuePaise: 999 * 999, State: domain.LotReallocated})

		order, _ := seedOrderWithLots(store, owner, lots)
		commissionPct := rng.Intn(31) // 0..30

		split, err := f.RequestPaymentSplit(ctx, RequestPaymentSplitInput{
			BulkOrderID: order.ID, CommissionPct: commissionPct, IdempotencyKey: "pay-1",
		})
		if err != nil {
			t.Fatalf("case %d: RequestPaymentSplit: %v", i, err)
		}
		if len(split.Lines) != wantLines {
			t.Fatalf("case %d: got %d lines, want %d", i, len(split.Lines), wantLines)
		}
		var sumGross, sumCommission, sumNet int64
		for _, ln := range split.Lines {
			if ln.GrossAmountPaise != ln.CommissionPaise+ln.NetAmountPaise {
				t.Fatalf("case %d: line gross %d != commission %d + net %d", i, ln.GrossAmountPaise, ln.CommissionPaise, ln.NetAmountPaise)
			}
			sumGross += ln.GrossAmountPaise
			sumCommission += ln.CommissionPaise
			sumNet += ln.NetAmountPaise
		}
		if sumGross != wantTotal {
			t.Fatalf("case %d: sum of line gross = %d, want exactly the delivered total %d", i, sumGross, wantTotal)
		}
		if split.GrossTotalPaise != wantTotal {
			t.Fatalf("case %d: split gross total = %d, want %d", i, split.GrossTotalPaise, wantTotal)
		}
		if sumCommission != split.CommissionTotalPaise || sumNet != split.NetTotalPaise {
			t.Fatalf("case %d: line sums (%d/%d) do not match split totals (%d/%d)", i, sumCommission, sumNet, split.CommissionTotalPaise, split.NetTotalPaise)
		}
		if sumNet+sumCommission != sumGross {
			t.Fatalf("case %d: net + commission = %d, want gross %d", i, sumNet+sumCommission, sumGross)
		}
	}
}

// randomPercentagesSummingTo100 generates n whole percentages that sum to
// exactly 100, matching the DB-constraint-trigger guarantee shg_member's
// share_pct roster actually carries.
func randomPercentagesSummingTo100(rng *rand.Rand, n int) []int {
	if n == 1 {
		return []int{100}
	}
	pcts := make([]int, n)
	remaining := 100
	for i := 0; i < n-1; i++ {
		maxShare := remaining - (n - 1 - i) // leave at least 1 for every remaining member
		if maxShare < 1 {
			maxShare = 1
		}
		share := 1 + rng.Intn(maxShare)
		pcts[i] = share
		remaining -= share
	}
	pcts[n-1] = remaining
	return pcts
}
