// services/core-svc/internal/core/service/artisan.go
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// artisanRegistered mirrors events.v1.ArtisanRegistered's payload fields.
type artisanRegistered struct {
	ArtisanID       string   `json:"artisan_id"`
	DisplayName     string   `json:"display_name"`
	CraftIDs        []string `json:"craft_ids"`
	ClusterID       *string  `json:"cluster_id,omitempty"`
	SelfHelpGroupID *string  `json:"self_help_group_id,omitempty"`
	Languages       []string `json:"languages"`
	StateCode       string   `json:"state_code"`
	District        *string  `json:"district,omitempty"`
}

// RegisterArtisan creates an artisan profile and emits artisan.registered. The
// profile row, its craft edges, any cluster membership and the outbox event all
// commit in a single transaction: either the artisan exists and the event is
// queued, or neither happened.
//
// idempotencyKey is the caller's retry key; it is carried into the outbox row so
// a replayed registration cannot enqueue the same event twice.
func (s *Identity) RegisterArtisan(ctx context.Context, in domain.RegisterArtisanInput, idempotencyKey string) (domain.Artisan, error) {
	if err := in.Validate(); err != nil {
		return domain.Artisan{}, err
	}
	if idempotencyKey == "" {
		return domain.Artisan{}, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}

	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return domain.Artisan{}, err
	}
	// A caller may register only the phone number they proved control of during
	// login — otherwise anyone with a token could claim someone else's number.
	// Field staff registering an artisan by proxy are the documented exception.
	if !principal.HasRole(auth.RoleClusterOfficer, auth.RoleMinistry) {
		if principal.PhoneE164 == "" || principal.PhoneE164 != in.PhoneE164 {
			return domain.Artisan{}, fmt.Errorf(
				"a registration must use the phone number verified at login: %w", pkgdomain.ErrForbidden)
		}
	}
	if in.CreatedBy == "" {
		in.CreatedBy = principal.Subject
		if in.CreatedBy == "" {
			in.CreatedBy = "self-registration"
		}
	}

	// The unique constraint on artisan.phone_e164 is the real guard and is
	// translated to ErrConflict by the repo. This pre-check exists so the common
	// case produces a clear message naming the phone number, rather than a
	// constraint name, and so a doomed registration never opens a transaction.
	if _, exists, err := s.store.ArtisanExistsByPhone(ctx, in.PhoneE164); err != nil {
		return domain.Artisan{}, fmt.Errorf("checking whether %s is registered: %w", in.PhoneE164, err)
	} else if exists {
		return domain.Artisan{}, fmt.Errorf("an artisan is already registered with phone %s: %w",
			in.PhoneE164, pkgdomain.ErrConflict)
	}

	artisanID := ids.New()

	var created domain.Artisan
	err = s.store.InTx(ctx, func(ctx context.Context, tx Tx) error {
		var err error
		created, err = tx.CreateArtisan(ctx, artisanID, in)
		if err != nil {
			return err
		}

		if in.ClusterID != nil {
			if err := tx.UpsertClusterMember(ctx, *in.ClusterID, artisanID, domain.ClusterRoleMember); err != nil {
				return fmt.Errorf("adding the new artisan to cluster %s: %w", *in.ClusterID, err)
			}
		}

		payload := artisanRegistered{
			ArtisanID:   artisanID.String(),
			DisplayName: created.DisplayName,
			CraftIDs:    uuidsToStrings(in.CraftIDs),
			Languages:   in.Languages,
			StateCode:   in.Region.StateCode,
			District:    in.Region.District,
		}
		if in.ClusterID != nil {
			clusterID := in.ClusterID.String()
			payload.ClusterID = &clusterID
		}

		return outbox.Enqueue(ctx, tx,
			ids.New().String(),
			artisanID.String(),
			topics.ArtisanRegistered,
			idempotencyKey,
			s.newEvent(artisanID, idempotencyKey, payload),
		)
	})
	if err != nil {
		return domain.Artisan{}, err
	}

	s.log.InfoContext(ctx, "artisan registered",
		"artisan_id", artisanID, "cluster_id", in.ClusterID, "crafts", len(in.CraftIDs))
	return created, nil
}

// GetArtisan reads one artisan. Any authenticated caller may read a profile:
// profiles are the public face of the marketplace.
func (s *Identity) GetArtisan(ctx context.Context, id uuid.UUID) (domain.Artisan, error) {
	if id == uuid.Nil {
		return domain.Artisan{}, fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	return s.store.GetArtisan(ctx, id)
}

// GetArtisanByPhone reads one artisan by phone. A phone number is a login
// identifier rather than public information, so this is restricted to the
// artisan themselves and to field staff.
func (s *Identity) GetArtisanByPhone(ctx context.Context, phone string) (domain.Artisan, error) {
	if phone == "" {
		return domain.Artisan{}, fmt.Errorf("phone_e164 is required: %w", pkgdomain.ErrInvalidInput)
	}

	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return domain.Artisan{}, err
	}

	found, err := s.store.GetArtisanByPhone(ctx, phone)
	if err != nil {
		return domain.Artisan{}, err
	}
	// Resolved after the read so that "not yours" and "does not exist" are told
	// apart only for callers already entitled to know.
	if principal.HasRole(auth.RoleClusterOfficer, auth.RoleMinistry) || principal.Subject == found.ID.String() {
		return found, nil
	}
	return domain.Artisan{}, fmt.Errorf("artisan not found: %w", pkgdomain.ErrNotFound)
}

// UpdateArtisanProfile patches a profile. An artisan may edit their own; a
// cluster officer may edit any.
func (s *Identity) UpdateArtisanProfile(ctx context.Context, in domain.UpdateArtisanInput) (domain.Artisan, error) {
	if err := in.Validate(); err != nil {
		return domain.Artisan{}, err
	}
	if _, err := auth.RequireSelfOrRole(ctx, in.ArtisanID.String(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return domain.Artisan{}, err
	}
	return s.store.UpdateArtisan(ctx, in)
}

// ListArtisansByCluster pages a cluster's artisans.
func (s *Identity) ListArtisansByCluster(ctx context.Context, clusterID uuid.UUID, page domain.Page) ([]domain.Artisan, error) {
	if clusterID == uuid.Nil {
		return nil, fmt.Errorf("cluster_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	// Fail with a clear not-found rather than an empty page when the cluster
	// itself does not exist.
	if _, err := s.store.GetCluster(ctx, clusterID); err != nil {
		return nil, err
	}
	return s.store.ListArtisansByCluster(ctx, clusterID, page)
}

// uuidsToStrings renders a list of ids for an event payload.
func uuidsToStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

// isConflict reports whether err is a uniqueness conflict, used where the
// service wants to add context to a constraint violation the repo translated.
func isConflict(err error) bool { return errors.Is(err, pkgdomain.ErrConflict) }
