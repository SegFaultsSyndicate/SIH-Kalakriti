// services/collab-svc/internal/collab/domain/allocation_test.go
package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewRandom()
	if err != nil {
		t.Fatalf("generating test uuid: %v", err)
	}
	return id
}

// TestDecomposeCoversQuantityExactlyNoLotBelowMinimum is the acceptance
// criterion verbatim: a 500-unit order across artisans with capacity 30-80
// allocates fully with no lot below MinLotSize.
func TestDecomposeCoversQuantityExactlyNoLotBelowMinimum(t *testing.T) {
	t.Parallel()

	capacities := []int32{80, 75, 70, 65, 60, 55, 50, 45, 40, 35, 30}
	var candidates []Candidate
	for _, units := range capacities {
		candidates = append(candidates, Candidate{
			ArtisanID:      mustUUID(t),
			AvailableUnits: units,
			OnTimeRate:     DefaultOnTimeRate,
		})
	}

	const orderQty = 500
	ranked := RankCandidates(candidates, orderQty)
	lots, unallocated := Decompose(ranked, orderQty, 0)

	var total int32
	for _, l := range lots {
		if l.Quantity < MinLotSize {
			t.Errorf("lot for %s has quantity %d, below MinLotSize %d", l.ArtisanID, l.Quantity, MinLotSize)
		}
		total += l.Quantity
	}
	if total != orderQty {
		t.Errorf("allocated total = %d, want %d", total, orderQty)
	}
	if unallocated != 0 {
		t.Errorf("unallocated = %d, want 0", unallocated)
	}
}

func TestDecomposeSkipsCandidateBelowMinLotSize(t *testing.T) {
	t.Parallel()

	a, b := mustUUID(t), mustUUID(t)
	candidates := []Candidate{
		{ArtisanID: a, AvailableUnits: 100, OnTimeRate: 1.0},
		{ArtisanID: b, AvailableUnits: 3, OnTimeRate: 1.0}, // below MinLotSize
	}
	// After a takes 100 of a 102-unit order, only 2 units remain — b's slice
	// would be 2 < MinLotSize, and must be skipped rather than offered.
	lots, unallocated := Decompose(candidates, 102, 0)

	if len(lots) != 1 || lots[0].ArtisanID != a {
		t.Fatalf("expected exactly one lot for %s, got %+v", a, lots)
	}
	if unallocated != 2 {
		t.Errorf("unallocated = %d, want 2", unallocated)
	}
}

func TestDecomposeRespectsMaxPerArtisan(t *testing.T) {
	t.Parallel()

	a := mustUUID(t)
	candidates := []Candidate{{ArtisanID: a, AvailableUnits: 200, OnTimeRate: 1.0}}

	lots, unallocated := Decompose(candidates, 150, 50)

	if len(lots) != 1 || lots[0].Quantity != 50 {
		t.Fatalf("expected one 50-unit lot capped by maxPerArtisan, got %+v", lots)
	}
	if unallocated != 100 {
		t.Errorf("unallocated = %d, want 100 (150 - 50 capped lot)", unallocated)
	}
}

func TestDecomposeReportsShortfallWhenCapacityInsufficient(t *testing.T) {
	t.Parallel()

	candidates := []Candidate{
		{ArtisanID: mustUUID(t), AvailableUnits: 40, OnTimeRate: 1.0},
		{ArtisanID: mustUUID(t), AvailableUnits: 30, OnTimeRate: 1.0},
	}
	lots, unallocated := Decompose(candidates, 100, 0)

	var total int32
	for _, l := range lots {
		total += l.Quantity
	}
	if total != 70 {
		t.Errorf("allocated total = %d, want 70", total)
	}
	if unallocated != 30 {
		t.Errorf("unallocated = %d, want 30", unallocated)
	}
}

func TestRankCandidatesPrefersCapacityFitThenOnTimeRateThenCluster(t *testing.T) {
	t.Parallel()

	hub := mustUUID(t)
	exactFit := Candidate{ArtisanID: mustUUID(t), AvailableUnits: 100, OnTimeRate: 1.0, ClusterID: &hub, SameCluster: true}
	overshoot := Candidate{ArtisanID: mustUUID(t), AvailableUnits: 500, OnTimeRate: 1.0}
	unreliable := Candidate{ArtisanID: mustUUID(t), AvailableUnits: 100, OnTimeRate: 0.2}

	ranked := RankCandidates([]Candidate{overshoot, unreliable, exactFit}, 100)

	if ranked[0].ArtisanID != exactFit.ArtisanID {
		t.Errorf("expected the exact-fit, same-cluster, reliable candidate ranked first, got %+v", ranked[0])
	}
	if ranked[len(ranked)-1].ArtisanID != unreliable.ArtisanID {
		t.Errorf("expected the unreliable candidate ranked last, got %+v", ranked[len(ranked)-1])
	}
}

func TestRankCandidatesIsDeterministicOnTies(t *testing.T) {
	t.Parallel()

	a := Candidate{ArtisanID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), AvailableUnits: 50, OnTimeRate: 1.0}
	b := Candidate{ArtisanID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), AvailableUnits: 50, OnTimeRate: 1.0}

	r1 := RankCandidates([]Candidate{b, a}, 50)
	r2 := RankCandidates([]Candidate{a, b}, 50)

	if r1[0].ArtisanID != r2[0].ArtisanID {
		t.Errorf("ranking is not deterministic across input order: %v vs %v", r1[0].ArtisanID, r2[0].ArtisanID)
	}
	if r1[0].ArtisanID != a.ArtisanID {
		t.Errorf("expected lexicographically-lower uuid %s to win the tie, got %s", a.ArtisanID, r1[0].ArtisanID)
	}
}

func TestCheckDeadlineFeasible(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		leadTime     int32
		buffer       int32
		requiredBy   time.Time
		wantFeasible bool
	}{
		{
			name: "far enough out", leadTime: 14, buffer: 3,
			requiredBy: now.AddDate(0, 0, 20), wantFeasible: true,
		},
		{
			name: "exactly on the boundary is feasible", leadTime: 14, buffer: 3,
			requiredBy: now.AddDate(0, 0, 17), wantFeasible: true,
		},
		{
			name: "one day short", leadTime: 14, buffer: 3,
			requiredBy: now.AddDate(0, 0, 16), wantFeasible: false,
		},
		{
			name: "far too soon", leadTime: 30, buffer: 5,
			requiredBy: now.AddDate(0, 0, 2), wantFeasible: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			feasible, reason := CheckDeadlineFeasible(FeasibilityInput{
				TypicalLeadTimeDays: tt.leadTime, AllocationBufferDays: tt.buffer,
				Now: now, RequiredBy: tt.requiredBy,
			})
			if feasible != tt.wantFeasible {
				t.Errorf("feasible = %v, want %v", feasible, tt.wantFeasible)
			}
			if !feasible && reason == "" {
				t.Error("expected a non-empty reason when infeasible")
			}
			if feasible && reason != "" {
				t.Errorf("expected an empty reason when feasible, got %q", reason)
			}
		})
	}
}

func TestCanTransitionBulkOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from, to BulkOrderState
		want     bool
	}{
		{BulkOrderAllocating, BulkOrderPartiallyAllocated, true},
		{BulkOrderAllocating, BulkOrderConfirmed, true},
		{BulkOrderPartiallyAllocated, BulkOrderConfirmed, true},
		{BulkOrderConfirmed, BulkOrderInProduction, true},
		{BulkOrderInProduction, BulkOrderCompleted, true},
		{BulkOrderCompleted, BulkOrderInProduction, false},  // no going backwards
		{BulkOrderAllocating, BulkOrderInProduction, false}, // no skipping confirmation
		{BulkOrderAllocating, BulkOrderAllocating, false},
	}
	for _, tt := range tests {
		if got := CanTransitionBulkOrder(tt.from, tt.to); got != tt.want {
			t.Errorf("CanTransitionBulkOrder(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestCanTransitionLot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from, to LotState
		want     bool
	}{
		{LotOffered, LotAccepted, true},
		{LotOffered, LotDeclined, true},
		{LotOffered, LotExpired, true},
		{LotAccepted, LotInProduction, true},
		{LotInProduction, LotQCPending, true},
		{LotQCPending, LotCompleted, true},
		{LotQCPending, LotQCFailed, true},
		{LotDeclined, LotAccepted, false},
		{LotOffered, LotCompleted, false},
	}
	for _, tt := range tests {
		if got := CanTransitionLot(tt.from, tt.to); got != tt.want {
			t.Errorf("CanTransitionLot(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
