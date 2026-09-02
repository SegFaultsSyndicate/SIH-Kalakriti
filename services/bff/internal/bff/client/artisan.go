// services/bff/internal/bff/client/artisan.go
package client

import (
	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"
)

// Artisan is bff's view of core-svc's artisan profile RPCs.
type Artisan struct {
	identity identityv1.IdentityServiceClient
}

// NewArtisan builds the artisan client, sharing conn with core-svc's other services.
func NewArtisan(conn grpc.ClientConnInterface) *Artisan {
	return &Artisan{identity: identityv1.NewIdentityServiceClient(conn)}
}

// Register is not wired: RegisterArtisanRequest requires craft_ids (at least
// one) and a region, neither of which handler.ArtisanService.Register's
// signature (or the documented POST /artisans body: display_name + language
// only) has anywhere to carry. Submitting the request without them would
// just fail core-svc's own validation on every call. Needs a bff API
// contract decision, not a guessed default, before this can wire up.
func (a *Artisan) Register(principalID, displayName, phone, language string) (string, error) {
	return "", domain.Unavailable("artisan registration is not wired: craft and region are required by core-svc but are not yet collected by this endpoint")
}

// GetProfile fetches one artisan's profile.
func (a *Artisan) GetProfile(artisanID string) (map[string]any, error) {
	ctx, cancel := withTimeout()
	defer cancel()

	resp, err := a.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: artisanID})
	if err != nil {
		return nil, grpcErr(err)
	}

	art := resp.GetArtisan()
	profile := map[string]any{
		"id":           art.GetId(),
		"display_name": art.GetDisplayName(),
		"phone_e164":   art.GetPhoneE164(),
		"craft_ids":    art.GetCraftIds(),
		"verified":     art.GetVerified(),
	}
	if art.ClusterId != nil {
		profile["cluster_id"] = *art.ClusterId
	}
	if art.Bio != nil {
		profile["bio"] = *art.Bio
	}
	if art.YearsOfExperience != nil {
		profile["years_of_experience"] = *art.YearsOfExperience
	}
	if region := art.GetRegion(); region != nil {
		profile["region"] = map[string]any{
			"state_code": region.GetStateCode(),
			"district":   region.GetDistrict(),
		}
	}
	return profile, nil
}

// UpdateProfile patches the mutable fields of an artisan profile. Only the
// scalar fields present in updates are sent; anything not in this map is
// left alone, matching UpdateArtisanProfileRequest's own patch semantics.
func (a *Artisan) UpdateProfile(artisanID string, updates map[string]any) error {
	ctx, cancel := withTimeout()
	defer cancel()

	req := &identityv1.UpdateArtisanProfileRequest{ArtisanId: artisanID}
	if v, ok := updates["display_name"].(string); ok {
		req.DisplayName = &v
	}
	if v, ok := updates["bio"].(string); ok {
		req.Bio = &v
	}
	if v, ok := updates["cluster_id"].(string); ok {
		req.ClusterId = &v
	}
	if v, ok := updates["photo_media_id"].(string); ok {
		req.PhotoMediaId = &v
	}
	if v, ok := updates["years_of_experience"].(float64); ok { // JSON numbers decode as float64
		years := int32(v)
		req.YearsOfExperience = &years
	}

	if _, err := a.identity.UpdateArtisanProfile(ctx, req); err != nil {
		return grpcErr(err)
	}
	return nil
}
