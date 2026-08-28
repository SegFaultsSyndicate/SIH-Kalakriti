// services/core-svc/internal/core/domain/collective.go
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

// ClusterMemberRole is an artisan's standing inside a cluster. Values match the
// cluster_member_role Postgres enum from migration 013.
type ClusterMemberRole string

const (
	// ClusterRoleMember is an ordinary member.
	ClusterRoleMember ClusterMemberRole = "MEMBER"
	// ClusterRoleCoordinator coordinates the cluster and vouches for new members.
	ClusterRoleCoordinator ClusterMemberRole = "COORDINATOR"
	// ClusterRoleMaster is a master craftsperson recognised for technique verification.
	ClusterRoleMaster ClusterMemberRole = "MASTER"
)

// Valid reports whether r is one of the known cluster roles.
func (r ClusterMemberRole) Valid() bool {
	switch r {
	case ClusterRoleMember, ClusterRoleCoordinator, ClusterRoleMaster:
		return true
	default:
		return false
	}
}

// String returns the role's wire and database representation.
func (r ClusterMemberRole) String() string { return string(r) }

// Cluster is a geographic concentration of artisans sharing a craft and a market.
type Cluster struct {
	ID                   uuid.UUID
	Name                 string
	Region               Region
	CoordinatorPhoneE164 *string
	ArtisanCount         int32
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// CreateClusterInput is what the service needs to register a cluster.
type CreateClusterInput struct {
	Name                 string
	Region               Region
	CoordinatorPhoneE164 *string
}

// Validate checks the cluster's required and pattern-constrained fields.
func (in CreateClusterInput) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required: %w", pkgdomain.ErrInvalidInput)
	}
	if err := in.Region.Validate(); err != nil {
		return err
	}
	if in.CoordinatorPhoneE164 != nil && !phoneE164Pattern.MatchString(*in.CoordinatorPhoneE164) {
		return fmt.Errorf("coordinator_phone_e164 %q must be E.164: %w", *in.CoordinatorPhoneE164, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// UpdateClusterInput patches a cluster; a nil field is left untouched.
type UpdateClusterInput struct {
	ClusterID            uuid.UUID
	Name                 *string
	Region               *Region
	CoordinatorPhoneE164 *string
}

// Validate checks only the fields actually being patched.
func (in UpdateClusterInput) Validate() error {
	if in.ClusterID == uuid.Nil {
		return fmt.Errorf("cluster_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return fmt.Errorf("name must not be blank: %w", pkgdomain.ErrInvalidInput)
	}
	if in.Region != nil {
		if err := in.Region.Validate(); err != nil {
			return err
		}
	}
	if in.CoordinatorPhoneE164 != nil && !phoneE164Pattern.MatchString(*in.CoordinatorPhoneE164) {
		return fmt.Errorf("coordinator_phone_e164 %q must be E.164: %w", *in.CoordinatorPhoneE164, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// ClusterMember is one artisan's membership row in a cluster.
type ClusterMember struct {
	ClusterID   uuid.UUID
	ArtisanID   uuid.UUID
	DisplayName string
	Role        ClusterMemberRole
	JoinedAt    time.Time
}

// SelfHelpGroup is the registered collective that fronts finance and bulk acceptance.
type SelfHelpGroup struct {
	ID                 uuid.UUID
	Name               string
	RegistrationNo     string
	ClusterID          *uuid.UUID
	SignatoryArtisanID *uuid.UUID
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SHGMember is one artisan's membership and payout share in an SHG.
type SHGMember struct {
	SHGID       uuid.UUID
	ArtisanID   uuid.UUID
	DisplayName string
	SharePct    int32
	JoinedAt    time.Time
}

// SHGMemberShare is one member's share on write.
type SHGMemberShare struct {
	ArtisanID uuid.UUID
	SharePct  int32
}

// TotalSharePct is the percentage an SHG roster must sum to. Settlement divides
// real money by these numbers, so the invariant is exact, not approximate.
const TotalSharePct int32 = 100

// maxSHGMembers bounds a roster. A share is a whole percent, so more than 100
// members cannot each hold a non-zero share.
const maxSHGMembers = 100

// ValidateSHGShares checks a proposed roster: non-empty, no duplicate or nil
// artisan, every share in range, and the whole thing summing to exactly 100.
func ValidateSHGShares(members []SHGMemberShare) error {
	if len(members) == 0 {
		return fmt.Errorf("an SHG roster must have at least one member: %w", pkgdomain.ErrInvalidInput)
	}
	if len(members) > maxSHGMembers {
		return fmt.Errorf("an SHG roster may have at most %d members: %w", maxSHGMembers, pkgdomain.ErrInvalidInput)
	}

	seen := make(map[uuid.UUID]struct{}, len(members))
	var total int32
	for _, m := range members {
		if m.ArtisanID == uuid.Nil {
			return fmt.Errorf("member artisan_id must not be nil: %w", pkgdomain.ErrInvalidInput)
		}
		if _, dup := seen[m.ArtisanID]; dup {
			return fmt.Errorf("artisan %s appears twice in the roster: %w", m.ArtisanID, pkgdomain.ErrInvalidInput)
		}
		seen[m.ArtisanID] = struct{}{}

		if m.SharePct < 0 || m.SharePct > TotalSharePct {
			return fmt.Errorf("share_pct for artisan %s is %d, must be between 0 and %d: %w",
				m.ArtisanID, m.SharePct, TotalSharePct, pkgdomain.ErrInvalidInput)
		}
		total += m.SharePct
	}

	if total != TotalSharePct {
		return fmt.Errorf("member shares sum to %d, must sum to exactly %d: %w",
			total, TotalSharePct, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// CreateSHGInput is what the service needs to register an SHG and its roster.
type CreateSHGInput struct {
	Name               string
	RegistrationNo     string
	ClusterID          *uuid.UUID
	SignatoryArtisanID *uuid.UUID
	Members            []SHGMemberShare
}

// Validate checks the group's own fields and its founding share split.
func (in CreateSHGInput) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.RegistrationNo) == "" {
		return fmt.Errorf("registration_no is required: %w", pkgdomain.ErrInvalidInput)
	}
	if err := ValidateSHGShares(in.Members); err != nil {
		return err
	}
	// A signatory who is not a member cannot sign for the group's earnings.
	if in.SignatoryArtisanID != nil {
		found := false
		for _, m := range in.Members {
			if m.ArtisanID == *in.SignatoryArtisanID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("signatory %s is not one of the group's members: %w",
				*in.SignatoryArtisanID, pkgdomain.ErrInvalidInput)
		}
	}
	return nil
}
