// services/bff/internal/bff/client/artisan.go
package client

import (
	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"
)

// maxCraftIDs caps how many craft ids a single registration can list, so a
// malicious or buggy client can't force an unbounded request downstream.
const maxCraftIDs = 20

// Artisan is bff's view of core-svc's artisan profile RPCs.
type Artisan struct {
	identity identityv1.IdentityServiceClient
}

// NewArtisan builds the artisan client, sharing conn with core-svc's other services.
func NewArtisan(conn grpc.ClientConnInterface) *Artisan {
	return &Artisan{identity: identityv1.NewIdentityServiceClient(conn)}
}

// Register creates an artisan profile for an already-verified phone number.
// fields carries the POST /artisans JSON body verbatim: display_name,
// craft_ids ([]string, at least one), languages ([]string, BCP-47-ish names
// as languageToProto expects), region ({state_code, district?, block?,
// village?, pincode?}), and the optional cluster_id/pehchan_id/
// pm_vishwakarma_id/years_of_experience/bio.
func (a *Artisan) Register(phone, idempotencyKey string, fields map[string]any) (string, error) {
	craftIDs, err := stringSlice(fields, "craft_ids")
	if err != nil {
		return "", err
	}
	if len(craftIDs) == 0 {
		return "", domain.InvalidInput("craft_ids: at least one is required")
	}
	if len(craftIDs) > maxCraftIDs {
		return "", domain.InvalidInput("craft_ids: too many")
	}

	langNames, err := stringSlice(fields, "languages")
	if err != nil {
		return "", err
	}
	languages := make([]commonv1.Language, len(langNames))
	for i, name := range langNames {
		languages[i] = languageToProto(name)
	}

	region, err := geoRegion(fields["region"])
	if err != nil {
		return "", err
	}

	displayName, _ := fields["display_name"].(string)

	req := &identityv1.RegisterArtisanRequest{
		DisplayName:    displayName,
		PhoneE164:      phone,
		CraftIds:       craftIDs,
		Languages:      languages,
		Region:         region,
		IdempotencyKey: idempotencyKey,
	}
	if v, ok := fields["cluster_id"].(string); ok {
		req.ClusterId = &v
	}
	if v, ok := fields["pehchan_id"].(string); ok {
		req.PehchanId = &v
	}
	if v, ok := fields["pm_vishwakarma_id"].(string); ok {
		req.PmVishwakarmaId = &v
	}
	if v, ok := fields["years_of_experience"].(float64); ok { // JSON numbers decode as float64
		years := int32(v)
		req.YearsOfExperience = &years
	}
	if v, ok := fields["bio"].(string); ok {
		req.Bio = &v
	}

	ctx, cancel := withTimeout()
	defer cancel()

	resp, err := a.identity.RegisterArtisan(ctx, req)
	if err != nil {
		return "", grpcErr(err)
	}
	return resp.GetArtisan().GetId(), nil
}

// stringSlice reads a JSON-decoded []any field as []string, rejecting any
// element that isn't a string rather than silently dropping it.
func stringSlice(fields map[string]any, key string) ([]string, error) {
	raw, ok := fields[key].([]any)
	if !ok {
		if fields[key] != nil {
			return nil, domain.InvalidInput(key + ": must be an array of strings")
		}
		return nil, nil
	}
	out := make([]string, len(raw))
	for i, v := range raw {
		s, ok := v.(string)
		if !ok {
			return nil, domain.InvalidInput(key + ": must be an array of strings")
		}
		out[i] = s
	}
	return out, nil
}

// geoRegion converts the JSON "region" object into a GeoRegion. state_code is
// required; the rest mirror GetProfile's own region shape.
func geoRegion(raw any) (*commonv1.GeoRegion, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, domain.InvalidInput("region: is required")
	}
	stateCode, _ := m["state_code"].(string)
	if stateCode == "" {
		return nil, domain.InvalidInput("region.state_code: is required")
	}
	region := &commonv1.GeoRegion{StateCode: stateCode}
	if v, ok := m["district"].(string); ok {
		region.District = &v
	}
	if v, ok := m["block"].(string); ok {
		region.Block = &v
	}
	if v, ok := m["village"].(string); ok {
		region.Village = &v
	}
	if v, ok := m["pincode"].(string); ok {
		region.Pincode = &v
	}
	return region, nil
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
