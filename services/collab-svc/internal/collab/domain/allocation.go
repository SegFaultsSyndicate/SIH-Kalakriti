// services/collab-svc/internal/collab/domain/allocation.go
package domain

import (
	"sort"

	"github.com/google/uuid"
)

// Candidate is one artisan the allocator is considering for a bulk order,
// with every signal ranking needs already attached. The repo layer joins
// craft match, verified-technique and proximity server-side (they are cheap
// filters over indexed columns); this package only ranks and decomposes.
type Candidate struct {
	ArtisanID uuid.UUID
	ClusterID *uuid.UUID
	// AvailableUnits is capacity_per_month minus already-committed units
	// (accepted lots plus live reservations) in the order's required-by
	// month. Zero or negative means the artisan has no room and is dropped
	// before ranking.
	AvailableUnits int32
	// OnTimeRate is the fraction (0..1) of the artisan's past lots that
	// shipped by their promised_ship_date. An artisan with no completed
	// lots yet gets DefaultOnTimeRate so newcomers are not penalised below
	// their first job.
	OnTimeRate float64
	// SameCluster is true when the artisan shares a cluster with the
	// requested hub cluster, if one was given.
	//
	// ponytail: cluster has no lat/lng in this schema (migrations/002), so
	// "proximity to the cluster hub" is same-cluster-as-hub, a boolean, not
	// a distance. Upgrade to real geo distance if cluster ever gains
	// coordinates.
	SameCluster bool
}

// DefaultOnTimeRate is what a candidate with no completed lot history is
// scored with, so a first-time artisan is neither favoured nor unfairly
// pushed to the back of the queue.
const DefaultOnTimeRate = 1.0

// score ranks candidates highest-first: capacity fit rewards an artisan who
// can cover more of what remains without overshooting it, on-time rate
// rewards reliability, and same-cluster is a tiebreaking bonus for pickup and
// QC convenience. Weights are fixed constants, not configurable — this is a
// ranking heuristic, not a tunable model.
func score(c Candidate, remaining int32) float64 {
	fit := capacityFit(c.AvailableUnits, remaining)
	s := 0.5*fit + 0.4*c.OnTimeRate
	if c.SameCluster {
		s += 0.1
	}
	return s
}

// capacityFit is 1.0 when the candidate can cover exactly what remains
// without leftover capacity going to waste, decaying as the candidate either
// falls short of or overshoots the remaining quantity.
func capacityFit(available, remaining int32) float64 {
	if available <= 0 || remaining <= 0 {
		return 0
	}
	if available >= remaining {
		// Covers it in one lot; prefer the candidate whose capacity is
		// closest to remaining (fewest larger lots, per the spec's "prefer
		// fewer, larger lots") over one with far more headroom than needed.
		return float64(remaining) / float64(available)
	}
	return float64(available) / float64(remaining)
}

// RankCandidates orders candidates by score, highest first, breaking ties by
// artisan id so the ordering is deterministic given the same input.
func RankCandidates(candidates []Candidate, orderQuantity int32) []Candidate {
	ranked := append([]Candidate(nil), candidates...)
	sort.SliceStable(ranked, func(i, j int) bool {
		si, sj := score(ranked[i], orderQuantity), score(ranked[j], orderQuantity)
		if si != sj {
			return si > sj
		}
		return ranked[i].ArtisanID.String() < ranked[j].ArtisanID.String()
	})
	return ranked
}

// ProposedLot is one slice of the decomposition, before it is persisted or
// offered.
type ProposedLot struct {
	ArtisanID uuid.UUID
	ClusterID *uuid.UUID
	Quantity  int32
}

// Decompose splits quantity across ranked candidates, taking each in order
// and offering it the largest lot it can carry: min(remaining quantity,
// AvailableUnits, maxPerArtisan when set). A lot below MinLotSize is never
// offered — if a candidate's remaining share would fall under the floor, that
// candidate is skipped rather than given a token-sized lot, and its capacity
// is left for the next pass or reported as unallocated.
//
// Preferring fewer, larger lots falls directly out of taking candidates in
// ranked (highest-fit-first) order and offering each the largest lot it can
// carry, rather than spreading remaining quantity evenly across every
// candidate up front.
func Decompose(ranked []Candidate, quantity int32, maxPerArtisan int32) (lots []ProposedLot, unallocated int32) {
	remaining := quantity
	for _, c := range ranked {
		if remaining <= 0 {
			break
		}
		want := c.AvailableUnits
		if want <= 0 {
			continue
		}
		if want > remaining {
			want = remaining
		}
		if maxPerArtisan > 0 && want > maxPerArtisan {
			want = maxPerArtisan
		}
		if want < MinLotSize {
			continue
		}
		lots = append(lots, ProposedLot{ArtisanID: c.ArtisanID, ClusterID: c.ClusterID, Quantity: want})
		remaining -= want
	}
	return lots, remaining
}
