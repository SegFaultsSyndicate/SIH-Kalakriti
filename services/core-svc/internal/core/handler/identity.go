// services/core-svc/internal/core/handler/identity.go

// Package handler exposes core-svc's gRPC surface. Handlers convert protobuf to
// domain types, call the service, and map domain errors onto gRPC status codes
// through pkg/domain. They hold no business logic and never touch the database.
package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Identity implements identity.v1.IdentityService.
type Identity struct {
	identityv1.UnimplementedIdentityServiceServer
	svc *service.Identity
}

// NewIdentity builds the identity handler.
func NewIdentity(svc *service.Identity) *Identity {
	return &Identity{svc: svc}
}

// PublicMethods are the RPCs callable without an access token: the two steps of
// the login handshake and the refresh exchange (which authenticates itself
// with the refresh token in its body rather than a bearer header), plus the
// read-only catalog/identity RPCs backing bff's public browsing routes
// (GET /crafts, /listings, /artisans/:id/storefront, /feed/process, and the
// unauthenticated /v/:code provenance page) -- bff calls these on behalf of
// anonymous buyers, with no bearer token to forward.
func PublicMethods() auth.PublicMethods {
	return auth.NewPublicMethods(
		"/identity.v1.IdentityService/RequestOtp",
		"/identity.v1.IdentityService/VerifyOtp",
		"/identity.v1.IdentityService/RefreshToken",
		"/identity.v1.IdentityService/GetArtisan",
		"/catalog.v1.OntologyService/ListCrafts",
		"/catalog.v1.OntologyService/GetCraft",
		"/catalog.v1.CatalogService/ListListings",
		"/catalog.v1.CatalogService/GetListing",
		"/catalog.v1.CatalogService/GetProvenanceByShortCode",
		"/trends.v1.TrendService/ListTrendLinks",
		"/b2b.v1.B2BService/ListNearbyBoutiques",
		"/b2b.v1.B2BService/ListCompanies",
		// bff mounts POST /companies and GET /companies/:id on its
		// unauthenticated route group (services/bff/internal/bff/server.go),
		// and neither service.B2B.RegisterCompany nor .GetCompany calls
		// auth.RequirePrincipal -- self-service company/boutique
		// registration is intentionally public, same as ListCompanies
		// above. This allow-list entry was simply missing, so an anonymous
		// caller got 401 from the interceptor before ever reaching that
		// intentionally-public handler. See WIRING_AUDIT_PLAN.md F-9.
		"/b2b.v1.B2BService/RegisterCompany",
		"/b2b.v1.B2BService/GetCompany",
		"/badges.v1.BadgeService/ListBadgeCatalog",
		"/badges.v1.BadgeService/ListArtisanBadges",
		"/schemes.v1.SchemeService/ListSchemes",
	)
}

// RequestOtp sends a one-time code to a phone number.
func (h *Identity) RequestOtp(ctx context.Context, req *identityv1.RequestOtpRequest) (*identityv1.RequestOtpResponse, error) {
	language := ""
	if langs := languagesFromProto([]commonv1.Language{req.GetLanguage()}); len(langs) > 0 {
		language = langs[0]
	}

	challenge, err := h.svc.RequestOtp(ctx, req.GetPhoneE164(), language)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.RequestOtpResponse{
		ChallengeId: challenge.ID,
		ExpiresAt:   timestamppb.New(challenge.ExpiresAt),
		DevMode:     challenge.DevMode,
	}, nil
}

// VerifyOtp exchanges a challenge and code for a token pair.
func (h *Identity) VerifyOtp(ctx context.Context, req *identityv1.VerifyOtpRequest) (*identityv1.VerifyOtpResponse, error) {
	result, err := h.svc.VerifyOtp(ctx, req.GetChallengeId(), req.GetPhoneE164(), req.GetCode(), req.GetDevRole())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	resp := &identityv1.VerifyOtpResponse{
		Tokens:     tokensToProto(result.Tokens),
		Registered: result.Registered,
	}
	if result.ArtisanID != "" {
		id := result.ArtisanID
		resp.ArtisanId = &id
	}
	return resp, nil
}

// RefreshToken exchanges a refresh token for a fresh pair.
func (h *Identity) RefreshToken(ctx context.Context, req *identityv1.RefreshTokenRequest) (*identityv1.RefreshTokenResponse, error) {
	tokens, err := h.svc.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.RefreshTokenResponse{Tokens: tokensToProto(tokens)}, nil
}

// RegisterArtisan creates an artisan profile for a verified phone number.
func (h *Identity) RegisterArtisan(ctx context.Context, req *identityv1.RegisterArtisanRequest) (*identityv1.RegisterArtisanResponse, error) {
	craftIDs, err := parseUUIDs("craft_ids", req.GetCraftIds())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	clusterID, err := parseOptionalUUID("cluster_id", req.ClusterId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	in := domain.RegisterArtisanInput{
		DisplayName:       req.GetDisplayName(),
		PhoneE164:         req.GetPhoneE164(),
		CraftIDs:          craftIDs,
		Languages:         languagesFromProto(req.GetLanguages()),
		Region:            regionFromProto(req.GetRegion()),
		ClusterID:         clusterID,
		PehchanID:         req.PehchanId,
		PMVishwakarmaID:   req.PmVishwakarmaId,
		YearsOfExperience: req.YearsOfExperience,
		Bio:               req.Bio,
		SocialCategory:    req.SocialCategory,
	}

	artisan, err := h.svc.RegisterArtisan(ctx, in, req.GetIdempotencyKey())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.RegisterArtisanResponse{Artisan: artisanToProto(artisan)}, nil
}

// GetArtisan fetches one artisan by id.
func (h *Identity) GetArtisan(ctx context.Context, req *identityv1.GetArtisanRequest) (*identityv1.GetArtisanResponse, error) {
	id, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	artisan, err := h.svc.GetArtisan(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.GetArtisanResponse{Artisan: artisanToProto(artisan)}, nil
}

// GetArtisanByPhone fetches one artisan by phone number.
func (h *Identity) GetArtisanByPhone(ctx context.Context, req *identityv1.GetArtisanByPhoneRequest) (*identityv1.GetArtisanByPhoneResponse, error) {
	artisan, err := h.svc.GetArtisanByPhone(ctx, req.GetPhoneE164())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.GetArtisanByPhoneResponse{Artisan: artisanToProto(artisan)}, nil
}

// UpdateArtisanProfile patches the mutable fields of a profile.
func (h *Identity) UpdateArtisanProfile(ctx context.Context, req *identityv1.UpdateArtisanProfileRequest) (*identityv1.UpdateArtisanProfileResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	clusterID, err := parseOptionalUUID("cluster_id", req.ClusterId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	photoID, err := parseOptionalUUID("photo_media_id", req.PhotoMediaId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	in := domain.UpdateArtisanInput{
		ArtisanID:         artisanID,
		DisplayName:       req.DisplayName,
		Languages:         languagesFromProto(req.GetLanguages()),
		ClusterID:         clusterID,
		YearsOfExperience: req.YearsOfExperience,
		Bio:               req.Bio,
		PhotoMediaID:      photoID,
	}
	if req.Region != nil {
		region := regionFromProto(req.Region)
		in.Region = &region
	}

	artisan, err := h.svc.UpdateArtisanProfile(ctx, in)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.UpdateArtisanProfileResponse{Artisan: artisanToProto(artisan)}, nil
}

// ListArtisansByCluster pages a cluster's artisans.
func (h *Identity) ListArtisansByCluster(ctx context.Context, req *identityv1.ListArtisansByClusterRequest) (*identityv1.ListArtisansByClusterResponse, error) {
	clusterID, err := parseUUID("cluster_id", req.GetClusterId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	page, err := pageFromProto(req.GetPage())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	artisans, err := h.svc.ListArtisansByCluster(ctx, clusterID, page)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	out := make([]*catalogv1.Artisan, 0, len(artisans))
	lastID := ""
	for _, a := range artisans {
		out = append(out, artisanToProto(a))
		lastID = a.ID.String()
	}
	return &identityv1.ListArtisansByClusterResponse{
		Artisans: out,
		Page:     nextPage(lastID, len(artisans), page.Normalise().Size),
	}, nil
}

// CreateCluster registers a cluster.
func (h *Identity) CreateCluster(ctx context.Context, req *identityv1.CreateClusterRequest) (*identityv1.CreateClusterResponse, error) {
	in := domain.CreateClusterInput{
		Name:                 req.GetName(),
		Region:               regionFromProto(req.GetRegion()),
		CoordinatorPhoneE164: req.CoordinatorPhoneE164,
	}
	cluster, err := h.svc.CreateCluster(ctx, in)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.CreateClusterResponse{Cluster: clusterToProto(cluster)}, nil
}

// GetCluster fetches one cluster by id.
func (h *Identity) GetCluster(ctx context.Context, req *identityv1.GetClusterRequest) (*identityv1.GetClusterResponse, error) {
	id, err := parseUUID("cluster_id", req.GetClusterId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	cluster, err := h.svc.GetCluster(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.GetClusterResponse{Cluster: clusterToProto(cluster)}, nil
}

// UpdateCluster patches a cluster.
func (h *Identity) UpdateCluster(ctx context.Context, req *identityv1.UpdateClusterRequest) (*identityv1.UpdateClusterResponse, error) {
	id, err := parseUUID("cluster_id", req.GetClusterId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	in := domain.UpdateClusterInput{
		ClusterID:            id,
		Name:                 req.Name,
		CoordinatorPhoneE164: req.CoordinatorPhoneE164,
	}
	if req.Region != nil {
		region := regionFromProto(req.Region)
		in.Region = &region
	}

	cluster, err := h.svc.UpdateCluster(ctx, in)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.UpdateClusterResponse{Cluster: clusterToProto(cluster)}, nil
}

// AddClusterMember adds an artisan to a cluster or updates their role.
func (h *Identity) AddClusterMember(ctx context.Context, req *identityv1.AddClusterMemberRequest) (*identityv1.AddClusterMemberResponse, error) {
	clusterID, err := parseUUID("cluster_id", req.GetClusterId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	member, err := h.svc.AddClusterMember(ctx, clusterID, artisanID, clusterRoleFromProto(req.GetRole()))
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.AddClusterMemberResponse{Member: clusterMemberToProto(member)}, nil
}

// RemoveClusterMember removes an artisan from a cluster.
func (h *Identity) RemoveClusterMember(ctx context.Context, req *identityv1.RemoveClusterMemberRequest) (*identityv1.RemoveClusterMemberResponse, error) {
	clusterID, err := parseUUID("cluster_id", req.GetClusterId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	removed, err := h.svc.RemoveClusterMember(ctx, clusterID, artisanID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.RemoveClusterMemberResponse{Removed: removed}, nil
}

// ListClusterMembers pages a cluster's roster.
func (h *Identity) ListClusterMembers(ctx context.Context, req *identityv1.ListClusterMembersRequest) (*identityv1.ListClusterMembersResponse, error) {
	clusterID, err := parseUUID("cluster_id", req.GetClusterId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	page, err := pageFromProto(req.GetPage())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	members, err := h.svc.ListClusterMembers(ctx, clusterID, page)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	out := make([]*identityv1.ClusterMember, 0, len(members))
	lastID := ""
	for _, m := range members {
		out = append(out, clusterMemberToProto(m))
		lastID = m.ArtisanID.String()
	}
	return &identityv1.ListClusterMembersResponse{
		Members: out,
		Page:    nextPage(lastID, len(members), page.Normalise().Size),
	}, nil
}

// CreateSelfHelpGroup registers an SHG and its founding share split.
func (h *Identity) CreateSelfHelpGroup(ctx context.Context, req *identityv1.CreateSelfHelpGroupRequest) (*identityv1.CreateSelfHelpGroupResponse, error) {
	clusterID, err := parseOptionalUUID("cluster_id", req.ClusterId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	signatoryID, err := parseOptionalUUID("signatory_artisan_id", req.SignatoryArtisanId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	members, err := shgSharesFromProto(req.GetMembers())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	group, roster, err := h.svc.CreateSelfHelpGroup(ctx, domain.CreateSHGInput{
		Name:               req.GetName(),
		RegistrationNo:     req.GetRegistrationNo(),
		ClusterID:          clusterID,
		SignatoryArtisanID: signatoryID,
		Members:            members,
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.CreateSelfHelpGroupResponse{
		SelfHelpGroup: shgToProto(group, roster),
		Members:       shgMembersToProto(roster),
	}, nil
}

// GetSelfHelpGroup fetches one SHG by id.
func (h *Identity) GetSelfHelpGroup(ctx context.Context, req *identityv1.GetSelfHelpGroupRequest) (*identityv1.GetSelfHelpGroupResponse, error) {
	id, err := parseUUID("self_help_group_id", req.GetSelfHelpGroupId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	group, err := h.svc.GetSelfHelpGroup(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	members, err := h.svc.ListSelfHelpGroupMembers(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.GetSelfHelpGroupResponse{SelfHelpGroup: shgToProto(group, members)}, nil
}

// SetSelfHelpGroupMembers replaces an SHG roster and its share split.
func (h *Identity) SetSelfHelpGroupMembers(ctx context.Context, req *identityv1.SetSelfHelpGroupMembersRequest) (*identityv1.SetSelfHelpGroupMembersResponse, error) {
	id, err := parseUUID("self_help_group_id", req.GetSelfHelpGroupId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	members, err := shgSharesFromProto(req.GetMembers())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	roster, err := h.svc.SetSelfHelpGroupMembers(ctx, id, members)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.SetSelfHelpGroupMembersResponse{Members: shgMembersToProto(roster)}, nil
}

// ListSelfHelpGroupMembers lists an SHG's roster.
func (h *Identity) ListSelfHelpGroupMembers(ctx context.Context, req *identityv1.ListSelfHelpGroupMembersRequest) (*identityv1.ListSelfHelpGroupMembersResponse, error) {
	id, err := parseUUID("self_help_group_id", req.GetSelfHelpGroupId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	members, err := h.svc.ListSelfHelpGroupMembers(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &identityv1.ListSelfHelpGroupMembersResponse{Members: shgMembersToProto(members)}, nil
}
