// services/core-svc/internal/core/repo/assisted.go

package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// GetActiveStaffByPhone returns the active staff account for a phone, and
// false (no error) when the phone belongs to no active staff member -- the
// ordinary case for every artisan login.
func (r *Repo) GetActiveStaffByPhone(ctx context.Context, phone string) (domain.StaffAccount, bool, error) {
	row, err := r.q.GetActiveStaffByPhone(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StaffAccount{}, false, nil
	}
	if err != nil {
		return domain.StaffAccount{}, false, translate(err, "staff account")
	}
	return staffFromRow(row), true, nil
}

func (r *Repo) GetStaffAccount(ctx context.Context, id uuid.UUID) (domain.StaffAccount, error) {
	row, err := r.q.GetStaffAccount(ctx, id)
	if err != nil {
		return domain.StaffAccount{}, translate(err, "staff account")
	}
	return staffFromRow(row), nil
}

func (r *Repo) CreateStaffAccount(ctx context.Context, s domain.StaffAccount) (domain.StaffAccount, error) {
	row, err := r.q.CreateStaffAccount(ctx, db.CreateStaffAccountParams{
		ID: s.ID, PhoneE164: s.PhoneE164, DisplayName: s.DisplayName, Role: db.StaffRole(s.Role),
		StateCode: s.StateCode, District: s.District, ClusterID: s.ClusterID, CscID: s.CSCID,
		CreatedBy: s.CreatedBy,
	})
	if err != nil {
		return domain.StaffAccount{}, translate(err, "staff account")
	}
	return staffFromRow(row), nil
}

func (r *Repo) ListStaffAccounts(ctx context.Context, stateCode *string) ([]domain.StaffAccount, error) {
	rows, err := r.q.ListStaffAccounts(ctx, stateCode)
	if err != nil {
		return nil, translate(err, "staff accounts")
	}
	out := make([]domain.StaffAccount, len(rows))
	for i, row := range rows {
		out[i] = staffFromRow(row)
	}
	return out, nil
}

func (r *Repo) SetStaffActive(ctx context.Context, id uuid.UUID, active bool) (domain.StaffAccount, error) {
	row, err := r.q.SetStaffActive(ctx, db.SetStaffActiveParams{ID: id, Active: active})
	if err != nil {
		return domain.StaffAccount{}, translate(err, "staff account")
	}
	return staffFromRow(row), nil
}

func (r *Repo) UpsertAssistedLink(ctx context.Context, l domain.AssistedLink) (domain.AssistedLink, error) {
	row, err := r.q.UpsertAssistedLink(ctx, db.UpsertAssistedLinkParams{
		ID: l.ID, AgentID: l.AgentID, ArtisanID: l.ArtisanID, ConsentMethod: l.ConsentMethod,
		ConsentRef: l.ConsentRef, NeedsReview: l.NeedsReview,
	})
	if err != nil {
		return domain.AssistedLink{}, translate(err, "assisted link")
	}
	return domain.AssistedLink{
		ID: row.ID, AgentID: row.AgentID, ArtisanID: row.ArtisanID, ConsentMethod: row.ConsentMethod,
		ConsentRef: row.ConsentRef, ConsentAt: row.ConsentAt, NeedsReview: row.NeedsReview, RevokedAt: row.RevokedAt,
	}, nil
}

func (r *Repo) HasActiveAssistedLink(ctx context.Context, agentID, artisanID uuid.UUID) (bool, error) {
	ok, err := r.q.HasActiveAssistedLink(ctx, db.HasActiveAssistedLinkParams{AgentID: agentID, ArtisanID: artisanID})
	if err != nil {
		return false, translate(err, "assisted link")
	}
	return ok, nil
}

func (r *Repo) RevokeAssistedLink(ctx context.Context, linkID, artisanID uuid.UUID) (bool, error) {
	n, err := r.q.RevokeAssistedLink(ctx, db.RevokeAssistedLinkParams{ID: linkID, ArtisanID: artisanID})
	if err != nil {
		return false, translate(err, "assisted link")
	}
	return n > 0, nil
}

func (r *Repo) ListAgentArtisans(ctx context.Context, agentID uuid.UUID) ([]domain.AgentArtisan, error) {
	rows, err := r.q.ListAgentArtisans(ctx, agentID)
	if err != nil {
		return nil, translate(err, "agent artisans")
	}
	out := make([]domain.AgentArtisan, len(rows))
	for i, row := range rows {
		out[i] = domain.AgentArtisan{
			ArtisanID: row.ID, DisplayName: row.DisplayName, Village: row.Village, District: row.District,
			StateCode: row.StateCode, PhotoMediaID: row.PhotoMediaID, ConsentMethod: row.ConsentMethod,
			NeedsReview: row.NeedsReview, LinkedAt: row.LinkedAt, LastActivityAt: epochToNil(row.LastActivityAt),
			DraftCount: row.DraftCount,
		}
	}
	return out, nil
}

func (r *Repo) ListArtisanHelpers(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanHelper, error) {
	rows, err := r.q.ListArtisanHelpers(ctx, artisanID)
	if err != nil {
		return nil, translate(err, "artisan helpers")
	}
	out := make([]domain.ArtisanHelper, len(rows))
	for i, row := range rows {
		out[i] = domain.ArtisanHelper{
			LinkID: row.LinkID, AgentID: row.AgentID, DisplayName: row.DisplayName, Role: string(row.Role),
			CSCID: row.CscID, ConsentMethod: row.ConsentMethod, LinkedAt: row.LinkedAt,
		}
	}
	return out, nil
}

func (r *Repo) ListLinksNeedingReview(ctx context.Context, stateCode, district *string) ([]domain.LinkForReview, error) {
	rows, err := r.q.ListLinksNeedingReview(ctx, db.ListLinksNeedingReviewParams{StateCode: stateCode, District: district})
	if err != nil {
		return nil, translate(err, "links needing review")
	}
	out := make([]domain.LinkForReview, len(rows))
	for i, row := range rows {
		out[i] = domain.LinkForReview{
			LinkID: row.ID, AgentID: row.AgentID, AgentName: row.AgentName, ArtisanID: row.ArtisanID,
			ArtisanName: row.ArtisanName, District: row.District, StateCode: row.StateCode,
			ConsentRef: row.ConsentRef, CreatedAt: row.CreatedAt,
		}
	}
	return out, nil
}

func (r *Repo) MarkLinkReviewed(ctx context.Context, linkID uuid.UUID, reviewer string) (bool, error) {
	n, err := r.q.MarkLinkReviewed(ctx, db.MarkLinkReviewedParams{ID: linkID, ReviewedBy: &reviewer})
	if err != nil {
		return false, translate(err, "assisted link review")
	}
	return n > 0, nil
}

// InsertAssistedAudit appends one audit_log row for a write made on an
// artisan's behalf. metadata is a JSON object (may be nil).
func (r *Repo) InsertAssistedAudit(ctx context.Context, a domain.AssistedAudit, metadata []byte) error {
	actor, subject := a.ActorID, a.SubjectID
	err := r.q.InsertAssistedAudit(ctx, db.InsertAssistedAuditParams{
		ActorID: &actor, Action: a.Action, ResourceType: a.ResourceType, ResourceID: a.ResourceID,
		SubjectID: &subject, Metadata: metadata,
	})
	return translate(err, "assisted audit")
}

func (r *Repo) ListAgentProductivity(ctx context.Context, stateCode, district *string) ([]domain.AgentProductivity, error) {
	rows, err := r.q.ListAgentProductivity(ctx, db.ListAgentProductivityParams{StateCode: stateCode, District: district})
	if err != nil {
		return nil, translate(err, "agent productivity")
	}
	out := make([]domain.AgentProductivity, len(rows))
	for i, row := range rows {
		out[i] = domain.AgentProductivity{
			AgentID: row.ID, DisplayName: row.DisplayName, Role: string(row.Role), StateCode: row.StateCode,
			District: row.District, CSCID: row.CscID, Active: row.Active, ArtisansOnboarded: row.ArtisansOnboarded,
			ListingsCreated: row.ListingsCreated, LastActiveAt: epochToNil(row.LastActiveAt),
		}
	}
	return out, nil
}

// GetListingHelper returns the agent who created a listing on the artisan's
// behalf, or ok=false when the artisan listed it themself.
func (r *Repo) GetListingHelper(ctx context.Context, listingID uuid.UUID) (name string, cscID *string, ok bool, err error) {
	row, err := r.q.GetListingHelper(ctx, listingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, translate(err, "listing helper")
	}
	return row.DisplayName, row.CscID, true, nil
}

func staffFromRow(row db.StaffAccount) domain.StaffAccount {
	return domain.StaffAccount{
		ID: row.ID, PhoneE164: row.PhoneE164, DisplayName: row.DisplayName, Role: string(row.Role),
		StateCode: row.StateCode, District: row.District, ClusterID: row.ClusterID, CSCID: row.CscID,
		Active: row.Active, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt,
	}
}

// epochToNil undoes the queries' COALESCE(..., 'epoch') -- sqlc cannot infer
// that a LEFT JOIN LATERAL column is nullable, so "never" is sent as epoch.
func epochToNil(t time.Time) *time.Time {
	if t.Unix() == 0 {
		return nil
	}
	return &t
}
