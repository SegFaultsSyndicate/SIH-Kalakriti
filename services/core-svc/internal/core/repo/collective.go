// services/core-svc/internal/core/repo/collective.go
package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// isNotFound reports whether err came back as a not-found from translate.
func isNotFound(err error) bool { return errors.Is(err, pkgdomain.ErrNotFound) }

// --- cluster -----------------------------------------------------------------

// CreateCluster inserts a cluster row.
func (t *Tx) CreateCluster(ctx context.Context, id uuid.UUID, in domain.CreateClusterInput) (domain.Cluster, error) {
	row, err := t.q.CreateCluster(ctx, db.CreateClusterParams{
		ID:                   id,
		Name:                 in.Name,
		StateCode:            in.Region.StateCode,
		District:             in.Region.District,
		Block:                in.Region.Block,
		Village:              in.Region.Village,
		Pincode:              in.Region.Pincode,
		CoordinatorPhoneE164: in.CoordinatorPhoneE164,
	})
	if err != nil {
		return domain.Cluster{}, translate(err, "cluster")
	}
	return clusterFromRow(row), nil
}

// GetCluster reads one cluster by id, including its member count.
func (r *Repo) GetCluster(ctx context.Context, id uuid.UUID) (domain.Cluster, error) {
	row, err := r.q.GetCluster(ctx, id)
	if err != nil {
		return domain.Cluster{}, translate(err, "cluster")
	}
	c := clusterFromRow(row)

	count, err := r.q.CountClusterMembers(ctx, id)
	if err != nil {
		return domain.Cluster{}, translate(err, "cluster member count")
	}
	c.ArtisanCount = int32(count)
	return c, nil
}

// UpdateCluster patches a cluster; nil fields are left alone by COALESCE.
func (r *Repo) UpdateCluster(ctx context.Context, in domain.UpdateClusterInput) (domain.Cluster, error) {
	params := db.UpdateClusterParams{
		ID:                   in.ClusterID,
		Name:                 in.Name,
		CoordinatorPhoneE164: in.CoordinatorPhoneE164,
	}
	if in.Region != nil {
		params.StateCode = &in.Region.StateCode
		params.District = in.Region.District
		params.Block = in.Region.Block
		params.Village = in.Region.Village
		params.Pincode = in.Region.Pincode
	}

	row, err := r.q.UpdateCluster(ctx, params)
	if err != nil {
		return domain.Cluster{}, translate(err, "cluster")
	}
	return clusterFromRow(row), nil
}

// UpsertClusterMember adds an artisan to a cluster or updates their role.
func (t *Tx) UpsertClusterMember(ctx context.Context, clusterID, artisanID uuid.UUID, role domain.ClusterMemberRole) error {
	err := t.q.UpsertClusterMember(ctx, db.UpsertClusterMemberParams{
		ClusterID: clusterID,
		ArtisanID: artisanID,
		Role:      db.ClusterMemberRole(role),
	})
	return translate(err, "cluster member")
}

// GetClusterMember reads one membership row, joined to the artisan's name.
func (r *Repo) GetClusterMember(ctx context.Context, clusterID, artisanID uuid.UUID) (domain.ClusterMember, error) {
	row, err := r.q.GetClusterMember(ctx, db.GetClusterMemberParams{
		ClusterID: clusterID,
		ArtisanID: artisanID,
	})
	if err != nil {
		return domain.ClusterMember{}, translate(err, "cluster member")
	}
	return domain.ClusterMember{
		ClusterID:   row.ClusterID,
		ArtisanID:   row.ArtisanID,
		DisplayName: row.DisplayName,
		Role:        domain.ClusterMemberRole(row.Role),
		JoinedAt:    row.JoinedAt,
	}, nil
}

// RemoveClusterMember deletes a membership, reporting whether a row was removed.
func (t *Tx) RemoveClusterMember(ctx context.Context, clusterID, artisanID uuid.UUID) (bool, error) {
	n, err := t.q.RemoveClusterMember(ctx, db.RemoveClusterMemberParams{
		ClusterID: clusterID,
		ArtisanID: artisanID,
	})
	if err != nil {
		return false, translate(err, "cluster member")
	}
	return n > 0, nil
}

// ListClusterMembers pages a cluster's roster by keyset on artisan id.
func (r *Repo) ListClusterMembers(ctx context.Context, clusterID uuid.UUID, page domain.Page) ([]domain.ClusterMember, error) {
	page = page.Normalise()
	rows, err := r.q.ListClusterMembers(ctx, db.ListClusterMembersParams{
		ClusterID: clusterID,
		After:     page.Cursor,
		PageSize:  page.Size,
	})
	if err != nil {
		return nil, translate(err, "cluster members")
	}
	out := make([]domain.ClusterMember, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ClusterMember{
			ClusterID:   row.ClusterID,
			ArtisanID:   row.ArtisanID,
			DisplayName: row.DisplayName,
			Role:        domain.ClusterMemberRole(row.Role),
			JoinedAt:    row.JoinedAt,
		})
	}
	return out, nil
}

func clusterFromRow(row db.Cluster) domain.Cluster {
	return domain.Cluster{
		ID:   row.ID,
		Name: row.Name,
		Region: domain.Region{
			StateCode: row.StateCode,
			District:  row.District,
			Block:     row.Block,
			Village:   row.Village,
			Pincode:   row.Pincode,
		},
		CoordinatorPhoneE164: row.CoordinatorPhoneE164,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

// --- self-help group ---------------------------------------------------------

// CreateSHG inserts an SHG row.
func (t *Tx) CreateSHG(ctx context.Context, id uuid.UUID, in domain.CreateSHGInput) (domain.SelfHelpGroup, error) {
	row, err := t.q.CreateShg(ctx, db.CreateShgParams{
		ID:                 id,
		Name:               in.Name,
		RegistrationNo:     in.RegistrationNo,
		ClusterID:          in.ClusterID,
		SignatoryArtisanID: in.SignatoryArtisanID,
	})
	if err != nil {
		return domain.SelfHelpGroup{}, translate(err, "self-help group")
	}
	return shgFromRow(row), nil
}

// GetSHG reads one SHG by id.
func (r *Repo) GetSHG(ctx context.Context, id uuid.UUID) (domain.SelfHelpGroup, error) {
	row, err := r.q.GetShg(ctx, id)
	if err != nil {
		return domain.SelfHelpGroup{}, translate(err, "self-help group")
	}
	return shgFromRow(row), nil
}

// ReplaceSHGMembers swaps an SHG's whole roster inside the caller's transaction.
// The delete-then-insert shape is safe because the shares-sum-to-100 constraint
// trigger is DEFERRABLE INITIALLY DEFERRED: it fires at COMMIT, so the
// intermediate empty roster never trips it.
func (t *Tx) ReplaceSHGMembers(ctx context.Context, shgID uuid.UUID, members []domain.SHGMemberShare) error {
	if _, err := t.q.DeleteShgMembers(ctx, shgID); err != nil {
		return translate(err, "self-help group members")
	}
	for _, m := range members {
		if err := t.q.InsertShgMember(ctx, db.InsertShgMemberParams{
			ShgID:     shgID,
			ArtisanID: m.ArtisanID,
			SharePct:  m.SharePct,
		}); err != nil {
			return translate(err, "self-help group member")
		}
	}
	return nil
}

// ListSHGMembers reads an SHG's roster, joined to each artisan's name.
func (r *Repo) ListSHGMembers(ctx context.Context, shgID uuid.UUID) ([]domain.SHGMember, error) {
	rows, err := r.q.ListShgMembers(ctx, shgID)
	if err != nil {
		return nil, translate(err, "self-help group members")
	}
	return shgMembersFromRows(rows), nil
}

// ListSHGMembers reads an SHG's roster inside the caller's transaction, so a
// write can return the roster it just stored without a second round trip.
func (t *Tx) ListSHGMembers(ctx context.Context, shgID uuid.UUID) ([]domain.SHGMember, error) {
	rows, err := t.q.ListShgMembers(ctx, shgID)
	if err != nil {
		return nil, translate(err, "self-help group members")
	}
	return shgMembersFromRows(rows), nil
}

func shgMembersFromRows(rows []db.ListShgMembersRow) []domain.SHGMember {
	out := make([]domain.SHGMember, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.SHGMember{
			SHGID:       row.ShgID,
			ArtisanID:   row.ArtisanID,
			DisplayName: row.DisplayName,
			SharePct:    row.SharePct,
			JoinedAt:    row.JoinedAt,
		})
	}
	return out
}

func shgFromRow(row db.Shg) domain.SelfHelpGroup {
	return domain.SelfHelpGroup{
		ID:                 row.ID,
		Name:               row.Name,
		RegistrationNo:     row.RegistrationNo,
		ClusterID:          row.ClusterID,
		SignatoryArtisanID: row.SignatoryArtisanID,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}
