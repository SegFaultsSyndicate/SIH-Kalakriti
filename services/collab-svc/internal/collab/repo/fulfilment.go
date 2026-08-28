// services/collab-svc/internal/collab/repo/fulfilment.go
package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/segfaultsyndicate/kalakriti/pkg/ids"

	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/domain"
	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/repo/db"
)

// --- reads (Repo, autocommit) -------------------------------------------------

// GetBulkOrder loads one bulk order.
func (r *Repo) GetBulkOrder(ctx context.Context, id uuid.UUID) (domain.BulkOrder, error) {
	row, err := r.q.GetBulkOrder(ctx, id)
	if err != nil {
		return domain.BulkOrder{}, translate(err, "bulk order")
	}
	return toDomainBulkOrder(row), nil
}

// GetLot loads one order lot.
func (r *Repo) GetLot(ctx context.Context, id uuid.UUID) (domain.OrderLot, error) {
	row, err := r.q.GetOrderLot(ctx, id)
	if err != nil {
		return domain.OrderLot{}, translate(err, "order lot")
	}
	return toDomainLot(row), nil
}

// ListLotsForOrder loads every lot on one bulk order.
func (r *Repo) ListLotsForOrder(ctx context.Context, orderID uuid.UUID) ([]domain.OrderLot, error) {
	rows, err := r.q.ListLotsByOrder(ctx, orderID)
	if err != nil {
		return nil, translate(err, "order lots")
	}
	out := make([]domain.OrderLot, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainLot(row))
	}
	return out, nil
}

// GetReservationForLot loads the capacity reservation backing one lot, if any
// — a lot created before this batch's reservation coupling, or one whose
// reservation has already been deleted by a cascade, legitimately has none.
func (r *Repo) GetReservationForLot(ctx context.Context, lotID uuid.UUID) (domain.CapacityReservation, bool, error) {
	row, err := r.q.GetReservationForLot(ctx, lotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CapacityReservation{}, false, nil
		}
		return domain.CapacityReservation{}, false, translate(err, "capacity reservation")
	}
	return toDomainReservation(row), true, nil
}

// ListingFeasibility loads the saga's own view of a listing.
func (r *Repo) ListingFeasibility(ctx context.Context, listingID uuid.UUID) (domain.ListingInfo, error) {
	row, err := r.q.GetListingForFulfilment(ctx, listingID)
	if err != nil {
		return domain.ListingInfo{}, translate(err, "listing")
	}
	return domain.ListingInfo{
		ProductID:           row.ProductID,
		CraftID:             row.CraftID,
		ArtisanID:           row.ArtisanID,
		UnitPricePaise:      row.UnitPricePaise,
		TypicalLeadTimeDays: row.TypicalLeadTimeDays,
		Owner: domain.ListingOwner{
			OwnerType:  domain.ListingOwnerType(row.OwnerType),
			OwnerSHGID: row.OwnerShgID,
		},
	}, nil
}

// CandidatesForOrder loads every eligible artisan for one craft with
// capacity, on-time rate and cluster proximity already computed.
func (r *Repo) CandidatesForOrder(ctx context.Context, in domain.CandidateQuery) ([]domain.Candidate, error) {
	var hub uuid.UUID
	if in.HubClusterID != nil {
		hub = *in.HubClusterID
	}
	rows, err := r.q.ListAllocationCandidates(ctx, db.ListAllocationCandidatesParams{
		CraftID:             in.CraftID,
		PeriodStart:         in.PeriodStart,
		HubClusterID:        hub,
		RestrictToPreferred: len(in.PreferredArtisanIDs) > 0,
		PreferredArtisanIDs: in.PreferredArtisanIDs,
	})
	if err != nil {
		return nil, translate(err, "allocation candidates")
	}
	out := make([]domain.Candidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Candidate{
			ArtisanID:      row.ArtisanID,
			ClusterID:      row.PrimaryClusterID,
			AvailableUnits: row.AvailableUnits,
			OnTimeRate:     float64(row.OnTimeRate),
			SameCluster:    row.SameCluster,
		})
	}
	return out, nil
}

// ReapExpiredReservations releases every HELD reservation past its
// expires_at and expires the OFFERED lot each one backs, one row per call so
// a crash mid-sweep never leaves a reservation half-freed: each release and
// its lot's expiry happen in the same statement pair inside one short
// transaction per batch member, not the whole batch in one transaction —
// a batch member that fails does not block the rest from being reaped on
// the next tick.
//
// Batch 13: also writes a LOT_EXPIRED audit event per expired lot (every
// compensation path writes one, this is no exception) and returns the
// distinct set of bulk orders affected, so the caller
// (service.RunReservationReaper) can re-run ProposeAllocation against each —
// the "release the reservation and re-offer to the next candidate" half of
// the lot-offer-expiry compensation path. Recording the event and reoffering
// live in two different transactions on purpose: a crash between them just
// leaves the reoffer for the next reaper tick to retry, since
// ProposeAllocation against an order with no unallocated remainder is a safe
// no-op (same reasoning as HandleLotDeclined's redelivery tolerance).
func (r *Repo) ReapExpiredReservations(ctx context.Context, batchSize int32) (released int, orderIDs []uuid.UUID, err error) {
	err = r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		reservations, err := tx.q.ReleaseExpiredReservationsReturningOrders(ctx, batchSize)
		if err != nil {
			return translate(err, "expired capacity reservations")
		}
		if len(reservations) == 0 {
			return nil
		}
		lotIDs := make([]uuid.UUID, 0, len(reservations))
		for _, res := range reservations {
			lotIDs = append(lotIDs, res.LotID)
		}
		expired, err := tx.q.ExpireLotsByIDReturningOrders(ctx, lotIDs)
		if err != nil {
			return translate(err, "expiring lots for released reservations")
		}
		seen := map[uuid.UUID]bool{}
		for _, lot := range expired {
			if err := tx.RecordEvent(ctx, ids.New(), lot.BulkOrderID, &lot.ID, "LOT_EXPIRED", toDomainLot(lot)); err != nil {
				return err
			}
			if !seen[lot.BulkOrderID] {
				seen[lot.BulkOrderID] = true
				orderIDs = append(orderIDs, lot.BulkOrderID)
			}
		}
		released = len(expired)
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	return released, orderIDs, nil
}

// --- writes (Tx) -----------------------------------------------------------

// CreateBulkOrder inserts a bulk order, or returns the row that already
// exists under (buyer_id, idempotency_key) on a replay. created is false
// only on that replay path.
func (t *Tx) CreateBulkOrder(ctx context.Context, id uuid.UUID, in domain.BulkOrder, idempotencyKey string) (domain.BulkOrder, bool, error) {
	customisations, err := json.Marshal(in.Customisations)
	if err != nil {
		return domain.BulkOrder{}, false, fmt.Errorf("encoding customisations: %w", err)
	}
	row, err := t.q.CreateBulkOrder(ctx, db.CreateBulkOrderParams{
		ID:              id,
		BuyerID:         in.BuyerID,
		ListingID:       in.ListingID,
		ProductID:       in.ProductID,
		Quantity:        in.Quantity,
		UnitPricePaise:  in.UnitPricePaise,
		TotalValuePaise: in.TotalValuePaise,
		RequiredBy:      in.RequiredBy,
		Customisations:  customisations,
		Notes:           in.Notes,
		IdempotencyKey:  idempotencyKey,
		CreatedBy:       "buyer:" + in.BuyerID,
		EscrowEnabled:   in.EscrowEnabled,
	})
	if err != nil {
		return domain.BulkOrder{}, false, translate(err, "bulk order")
	}
	return toDomainBulkOrder(row.BulkOrder), row.IsNew, nil
}

// TransitionBulkOrder moves a bulk order from one state to another, guarded
// by the current state matching from — a zero-row update means another
// writer got there first, reported as ErrConflict.
func (t *Tx) TransitionBulkOrder(ctx context.Context, orderID uuid.UUID, from, to domain.BulkOrderState) (domain.BulkOrder, error) {
	row, err := t.q.TransitionBulkOrderState(ctx, db.TransitionBulkOrderStateParams{
		ID:            orderID,
		ExpectedState: db.BulkOrderState(from),
		NextState:     db.BulkOrderState(to),
	})
	if err != nil {
		return domain.BulkOrder{}, translate(err, "bulk order transition")
	}
	return toDomainBulkOrder(row), nil
}

// IncrementAllocatedQuantity adds delta to allocated_quantity. This reads
// then writes inside the caller's own transaction, so the increment is
// consistent with whatever else that transaction does — a raw UPDATE ... SET
// allocated_quantity = allocated_quantity + delta would be simpler but sqlc's
// generated UpdateBulkOrderAllocation query takes the target value directly
// (it also carries the order's own state), so the read happens here rather
// than duplicating a second hand-written query.
func (t *Tx) IncrementAllocatedQuantity(ctx context.Context, orderID uuid.UUID, delta int32) (domain.BulkOrder, error) {
	current, err := t.q.GetBulkOrderForUpdate(ctx, orderID)
	if err != nil {
		return domain.BulkOrder{}, translate(err, "bulk order")
	}
	next := current.AllocatedQuantity + delta
	state := current.State
	row, err := t.q.UpdateBulkOrderAllocation(ctx, db.UpdateBulkOrderAllocationParams{
		ID:                orderID,
		AllocatedQuantity: next,
		State:             state,
	})
	if err != nil {
		return domain.BulkOrder{}, translate(err, "bulk order allocation")
	}
	return toDomainBulkOrder(row), nil
}

// CreateLot inserts a new order lot.
func (t *Tx) CreateLot(ctx context.Context, id uuid.UUID, in domain.OrderLot) (domain.OrderLot, error) {
	row, err := t.q.InsertOrderLot(ctx, db.InsertOrderLotParams{
		ID:                   id,
		BulkOrderID:          in.BulkOrderID,
		ArtisanID:            in.ArtisanID,
		ClusterID:            in.ClusterID,
		Quantity:             in.Quantity,
		UnitPricePaise:       in.UnitPricePaise,
		LotValuePaise:        in.LotValuePaise,
		RespondsBy:           in.RespondsBy,
		ReallocatedFromLotID: in.ReallocatedFromLotID,
	})
	if err != nil {
		return domain.OrderLot{}, translate(err, "order lot")
	}
	return toDomainLot(row), nil
}

// TransitionLot moves a lot from one state to another, applying whichever of
// fields' values are relevant to the target state.
//
// Batch 13 additions: QC_PENDING reached from QC_FAILED is a rework
// resubmission (StartRework, guarded by the rework window), never a progress
// update, so it is checked ahead of the generic progress-update case;
// QC_FAILED itself now requires a rework deadline; REALLOCATED is reached
// either by a mid-production dropout (from ACCEPTED/IN_PRODUCTION) or by a
// QC-side give-up (from QC_PENDING/QC_FAILED, second failure or a
// rework-window timeout), backed by two different queries because only the
// dropout path records a reason.
func (t *Tx) TransitionLot(ctx context.Context, lotID uuid.UUID, from, to domain.LotState, fields domain.LotTransitionFields) (domain.OrderLot, error) {
	switch to {
	case domain.LotAccepted:
		row, err := t.q.AcceptOrderLot(ctx, db.AcceptOrderLotParams{
			ID:               lotID,
			PromisedShipDate: fields.PromisedShipDate,
		})
		if err != nil {
			return domain.OrderLot{}, translate(err, "order lot acceptance")
		}
		return toDomainLot(row), nil
	case domain.LotQCPending:
		if from == domain.LotQCFailed {
			row, err := t.q.StartRework(ctx, lotID)
			if err != nil {
				return domain.OrderLot{}, translate(err, "order lot rework resubmission")
			}
			return toDomainLot(row), nil
		}
		fallthrough
	case domain.LotInProduction:
		pct := int32(0)
		if fields.ProgressPct != nil {
			pct = *fields.ProgressPct
		}
		row, err := t.q.SetLotProgress(ctx, db.SetLotProgressParams{
			ID:          lotID,
			ProgressPct: pct,
			State:       db.LotState(to),
		})
		if err != nil {
			return domain.OrderLot{}, translate(err, "order lot progress")
		}
		return toDomainLot(row), nil
	case domain.LotQCFailed:
		if fields.ReworkDeadline == nil {
			return domain.OrderLot{}, fmt.Errorf("transitioning lot %s to QC_FAILED: rework deadline is required", lotID)
		}
		row, err := t.q.SetLotReworkDeadline(ctx, db.SetLotReworkDeadlineParams{
			ID:             lotID,
			ReworkDeadline: fields.ReworkDeadline,
		})
		if err != nil {
			return domain.OrderLot{}, translate(err, "order lot rework deadline")
		}
		return toDomainLot(row), nil
	case domain.LotReallocated:
		if from == domain.LotAccepted || from == domain.LotInProduction {
			row, err := t.q.DropoutLot(ctx, db.DropoutLotParams{ID: lotID, DropoutReason: fields.DropoutReason})
			if err != nil {
				return domain.OrderLot{}, translate(err, "order lot dropout")
			}
			return toDomainLot(row), nil
		}
		row, err := t.q.ReallocateLot(ctx, lotID)
		if err != nil {
			return domain.OrderLot{}, translate(err, "order lot reallocation")
		}
		return toDomainLot(row), nil
	default:
		row, err := t.q.TransitionLotState(ctx, db.TransitionLotStateParams{
			ID:            lotID,
			ExpectedState: db.LotState(from),
			NextState:     db.LotState(to),
			DeclineReason: fields.DeclineReason,
		})
		if err != nil {
			return domain.OrderLot{}, translate(err, "order lot transition")
		}
		return toDomainLot(row), nil
	}
}

// CreateReservation inserts a capacity reservation.
func (t *Tx) CreateReservation(ctx context.Context, id uuid.UUID, in domain.CapacityReservation) (domain.CapacityReservation, error) {
	row, err := t.q.ReserveCapacity(ctx, db.ReserveCapacityParams{
		ID:          id,
		ArtisanID:   in.ArtisanID,
		ListingID:   in.ListingID,
		LotID:       in.LotID,
		Units:       in.Units,
		PeriodStart: in.PeriodStart,
		PeriodEnd:   in.PeriodEnd,
		ExpiresAt:   in.ExpiresAt,
	})
	if err != nil {
		return domain.CapacityReservation{}, translate(err, "capacity reservation")
	}
	return toDomainReservation(row), nil
}

// ReleaseReservation gives back a HELD reservation's capacity.
func (t *Tx) ReleaseReservation(ctx context.Context, reservationID uuid.UUID) error {
	_, err := t.q.SetReservationState(ctx, db.SetReservationStateParams{
		ID:    reservationID,
		State: db.ReservationState(domain.ReservationReleased),
	})
	return translate(err, "capacity reservation release")
}

// ConsumeReservation marks a HELD reservation spent by an accepted lot.
func (t *Tx) ConsumeReservation(ctx context.Context, reservationID uuid.UUID) error {
	_, err := t.q.SetReservationState(ctx, db.SetReservationStateParams{
		ID:    reservationID,
		State: db.ReservationState(domain.ReservationConsumed),
	})
	return translate(err, "capacity reservation consumption")
}

// CreateQCResult persists one inspection outcome and its defects. Two insert
// statements (result, then one row per defect) rather than a single JSON blob
// column, so a per-defect query ("every CRITICAL defect this month") stays a
// plain SQL scan instead of a jsonb traversal.
func (t *Tx) CreateQCResult(ctx context.Context, result domain.QCResult) error {
	if err := t.q.InsertQCResult(ctx, db.InsertQCResultParams{
		ID:          result.ID,
		LotID:       result.LotID,
		InspectorID: result.InspectorID,
		Passed:      result.Passed,
		Notes:       result.Notes,
		MediaIds:    result.MediaIDs,
		InspectedAt: result.InspectedAt,
	}); err != nil {
		return translate(err, "qc result")
	}
	for _, d := range result.Defects {
		if err := t.q.InsertQCDefect(ctx, db.InsertQCDefectParams{
			ID:            d.ID,
			QcResultID:    result.ID,
			Code:          d.Code,
			Description:   d.Description,
			Severity:      db.DefectSeverity(d.Severity),
			MediaIds:      d.MediaIDs,
			AffectedUnits: d.AffectedUnits,
		}); err != nil {
			return translate(err, "qc defect")
		}
	}
	return nil
}

// RecordEvent appends one row to the saga's audit trail.
func (t *Tx) RecordEvent(ctx context.Context, id, orderID uuid.UUID, lotID *uuid.UUID, eventType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding event payload for %s: %w", eventType, err)
	}
	if err := t.q.InsertBulkOrderEvent(ctx, db.InsertBulkOrderEventParams{
		ID:          id,
		BulkOrderID: orderID,
		LotID:       lotID,
		EventType:   eventType,
		Payload:     body,
	}); err != nil {
		return translate(err, "bulk order event")
	}
	return nil
}

// --- row mapping -------------------------------------------------------------

func toDomainBulkOrder(row db.BulkOrder) domain.BulkOrder {
	var customisations map[string]string
	_ = json.Unmarshal(row.Customisations, &customisations)
	return domain.BulkOrder{
		ID:                row.ID,
		BuyerID:           row.BuyerID,
		ListingID:         row.ListingID,
		ProductID:         row.ProductID,
		Quantity:          row.Quantity,
		UnitPricePaise:    row.UnitPricePaise,
		TotalValuePaise:   row.TotalValuePaise,
		RequiredBy:        row.RequiredBy,
		State:             domain.BulkOrderState(row.State),
		AllocatedQuantity: row.AllocatedQuantity,
		Customisations:    customisations,
		Notes:             row.Notes,
		IdempotencyKey:    orEmpty(row.IdempotencyKey),
		EscrowEnabled:     row.EscrowEnabled,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toDomainLot(row db.OrderLot) domain.OrderLot {
	return domain.OrderLot{
		ID:                    row.ID,
		BulkOrderID:           row.BulkOrderID,
		ArtisanID:             row.ArtisanID,
		ClusterID:             row.ClusterID,
		Quantity:              row.Quantity,
		UnitPricePaise:        row.UnitPricePaise,
		LotValuePaise:         row.LotValuePaise,
		State:                 domain.LotState(row.State),
		OfferedAt:             row.OfferedAt,
		RespondsBy:            row.RespondsBy,
		AcceptedAt:            row.AcceptedAt,
		PromisedShipDate:      row.PromisedShipDate,
		ProgressPct:           row.ProgressPct,
		CapacityReservationID: row.CapacityReservationID,
		DeclineReason:         row.DeclineReason,
		ReallocatedFromLotID:  row.ReallocatedFromLotID,
		DropoutReason:         row.DropoutReason,
		ReworkDeadline:        row.ReworkDeadline,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}

func toDomainReservation(row db.CapacityReservation) domain.CapacityReservation {
	return domain.CapacityReservation{
		ID:          row.ID,
		ArtisanID:   row.ArtisanID,
		ListingID:   row.ListingID,
		LotID:       row.LotID,
		Units:       row.Units,
		PeriodStart: row.PeriodStart,
		PeriodEnd:   row.PeriodEnd,
		ExpiresAt:   row.ExpiresAt,
		State:       domain.ReservationState(row.State),
	}
}

func orEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// BulkOrderEventRow is one row of the saga's audit trail, in the shape
// WatchOrder's replay needs: the raw event_type and JSON payload, decoded
// into a typed proto event one layer up in the handler package (which is
// where protobuf types are allowed to appear — this package stays
// protobuf-free like every other repo package in the project).
type BulkOrderEventRow struct {
	ID          uuid.UUID
	BulkOrderID uuid.UUID
	EventType   string
	Payload     []byte
	OccurredAt  time.Time
}

// ListBulkOrderEventsSince loads one order's audit trail, oldest first,
// optionally bounded to events at or after since.
func (r *Repo) ListBulkOrderEventsSince(ctx context.Context, orderID uuid.UUID, since *time.Time) ([]BulkOrderEventRow, error) {
	rows, err := r.q.ListBulkOrderEventsSince(ctx, db.ListBulkOrderEventsSinceParams{
		BulkOrderID: orderID,
		Since:       since,
	})
	if err != nil {
		return nil, translate(err, "bulk order events")
	}
	out := make([]BulkOrderEventRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, BulkOrderEventRow{
			ID: row.ID, BulkOrderID: row.BulkOrderID, EventType: row.EventType,
			Payload: row.Payload, OccurredAt: row.OccurredAt,
		})
	}
	return out, nil
}
