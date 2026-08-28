// services/core-svc/internal/core/service/collective.go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// --- cluster -----------------------------------------------------------------

// CreateCluster registers a cluster. Clusters are administrative units, so only
// field staff and the ministry may create them.
func (s *Identity) CreateCluster(ctx context.Context, in domain.CreateClusterInput) (domain.Cluster, error) {
	if err := in.Validate(); err != nil {
		return domain.Cluster{}, err
	}
	if _, err := auth.RequireRole(ctx, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return domain.Cluster{}, err
	}

	clusterID := ids.New()
	var created domain.Cluster
	err := s.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		created, err = tx.CreateCluster(ctx, clusterID, in)
		return err
	})
	if err != nil {
		return domain.Cluster{}, err
	}
	s.log.InfoContext(ctx, "cluster created", "cluster_id", clusterID, "name", in.Name)
	return created, nil
}

// GetCluster reads one cluster, including its member count.
func (s *Identity) GetCluster(ctx context.Context, id uuid.UUID) (domain.Cluster, error) {
	if id == uuid.Nil {
		return domain.Cluster{}, fmt.Errorf("cluster_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	return s.store.GetCluster(ctx, id)
}

// UpdateCluster patches a cluster.
func (s *Identity) UpdateCluster(ctx context.Context, in domain.UpdateClusterInput) (domain.Cluster, error) {
	if err := in.Validate(); err != nil {
		return domain.Cluster{}, err
	}
	if _, err := auth.RequireRole(ctx, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return domain.Cluster{}, err
	}
	return s.store.UpdateCluster(ctx, in)
}

// AddClusterMember adds an artisan to a cluster or updates their standing.
func (s *Identity) AddClusterMember(
	ctx context.Context,
	clusterID, artisanID uuid.UUID,
	role domain.ClusterMemberRole,
) (domain.ClusterMember, error) {
	if clusterID == uuid.Nil || artisanID == uuid.Nil {
		return domain.ClusterMember{}, fmt.Errorf("cluster_id and artisan_id are required: %w", pkgdomain.ErrInvalidInput)
	}
	if role == "" {
		role = domain.ClusterRoleMember
	}
	if !role.Valid() {
		return domain.ClusterMember{}, fmt.Errorf("unknown cluster member role %q: %w", role, pkgdomain.ErrInvalidInput)
	}
	if _, err := auth.RequireRole(ctx, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return domain.ClusterMember{}, err
	}

	err := s.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		return tx.UpsertClusterMember(ctx, clusterID, artisanID, role)
	})
	if err != nil {
		return domain.ClusterMember{}, err
	}
	return s.store.GetClusterMember(ctx, clusterID, artisanID)
}

// RemoveClusterMember removes an artisan from a cluster, reporting whether a row
// was actually removed so a repeated call is a visible no-op rather than an error.
func (s *Identity) RemoveClusterMember(ctx context.Context, clusterID, artisanID uuid.UUID) (bool, error) {
	if clusterID == uuid.Nil || artisanID == uuid.Nil {
		return false, fmt.Errorf("cluster_id and artisan_id are required: %w", pkgdomain.ErrInvalidInput)
	}
	if _, err := auth.RequireRole(ctx, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return false, err
	}

	var removed bool
	err := s.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		removed, err = tx.RemoveClusterMember(ctx, clusterID, artisanID)
		return err
	})
	return removed, err
}

// ListClusterMembers pages a cluster's roster.
func (s *Identity) ListClusterMembers(ctx context.Context, clusterID uuid.UUID, page domain.Page) ([]domain.ClusterMember, error) {
	if clusterID == uuid.Nil {
		return nil, fmt.Errorf("cluster_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if _, err := s.store.GetCluster(ctx, clusterID); err != nil {
		return nil, err
	}
	return s.store.ListClusterMembers(ctx, clusterID, page)
}

// --- self-help group ---------------------------------------------------------

// CreateSelfHelpGroup registers an SHG together with its founding share split.
// The group row and every member row commit in one transaction, so a group can
// never exist with a roster that does not sum to 100.
func (s *Identity) CreateSelfHelpGroup(ctx context.Context, in domain.CreateSHGInput) (domain.SelfHelpGroup, []domain.SHGMember, error) {
	if err := in.Validate(); err != nil {
		return domain.SelfHelpGroup{}, nil, err
	}
	if _, err := auth.RequireRole(ctx, auth.RoleClusterOfficer, auth.RoleMinistry, auth.RoleArtisan); err != nil {
		return domain.SelfHelpGroup{}, nil, err
	}

	shgID := ids.New()
	var (
		created domain.SelfHelpGroup
		roster  []domain.SHGMember
	)
	err := s.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		created, err = tx.CreateSHG(ctx, shgID, in)
		if err != nil {
			return err
		}
		if err := tx.ReplaceSHGMembers(ctx, shgID, in.Members); err != nil {
			return err
		}
		roster, err = tx.ListSHGMembers(ctx, shgID)
		return err
	})
	if err != nil {
		return domain.SelfHelpGroup{}, nil, err
	}

	s.log.InfoContext(ctx, "self-help group created",
		"shg_id", shgID, "name", in.Name, "members", len(in.Members))
	return created, roster, nil
}

// GetSelfHelpGroup reads one SHG.
func (s *Identity) GetSelfHelpGroup(ctx context.Context, id uuid.UUID) (domain.SelfHelpGroup, error) {
	if id == uuid.Nil {
		return domain.SelfHelpGroup{}, fmt.Errorf("self_help_group_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	return s.store.GetSHG(ctx, id)
}

// SetSelfHelpGroupMembers replaces an SHG's roster wholesale. Shares are
// validated here before any write, and the deferred database trigger enforces
// the same invariant at COMMIT whatever path the write took.
//
// Only the group's authorised signatory, a cluster officer or the ministry may
// change a share split, because these numbers divide real money.
func (s *Identity) SetSelfHelpGroupMembers(
	ctx context.Context,
	shgID uuid.UUID,
	members []domain.SHGMemberShare,
) ([]domain.SHGMember, error) {
	if shgID == uuid.Nil {
		return nil, fmt.Errorf("self_help_group_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if err := domain.ValidateSHGShares(members); err != nil {
		return nil, err
	}

	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	group, err := s.store.GetSHG(ctx, shgID)
	if err != nil {
		return nil, err
	}
	if !principal.HasRole(auth.RoleClusterOfficer, auth.RoleMinistry) {
		if group.SignatoryArtisanID == nil || group.SignatoryArtisanID.String() != principal.Subject {
			return nil, fmt.Errorf(
				"only the group's signatory or a cluster officer may change its share split: %w",
				pkgdomain.ErrForbidden)
		}
	}

	var roster []domain.SHGMember
	err = s.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		if err := tx.ReplaceSHGMembers(ctx, shgID, members); err != nil {
			return err
		}
		var err error
		roster, err = tx.ListSHGMembers(ctx, shgID)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "self-help group roster replaced", "shg_id", shgID, "members", len(members))
	return roster, nil
}

// ListSelfHelpGroupMembers reads an SHG's roster.
func (s *Identity) ListSelfHelpGroupMembers(ctx context.Context, shgID uuid.UUID) ([]domain.SHGMember, error) {
	if shgID == uuid.Nil {
		return nil, fmt.Errorf("self_help_group_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if _, err := s.store.GetSHG(ctx, shgID); err != nil {
		return nil, err
	}
	return s.store.ListSHGMembers(ctx, shgID)
}
