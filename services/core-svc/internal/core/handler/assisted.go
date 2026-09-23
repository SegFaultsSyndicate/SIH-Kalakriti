// services/core-svc/internal/core/handler/assisted.go

package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	assistedv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/assisted/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Staff implements assisted.v1.StaffService.
type Staff struct {
	assistedv1.UnimplementedStaffServiceServer
	svc *service.Assisted
}

// NewStaff builds the staff-account handler.
func NewStaff(svc *service.Assisted) *Staff { return &Staff{svc: svc} }

func (h *Staff) CreateStaff(ctx context.Context, req *assistedv1.CreateStaffRequest) (*assistedv1.CreateStaffResponse, error) {
	clusterID, err := parseOptionalUUID("cluster_id", req.ClusterId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	s, err := h.svc.CreateStaff(ctx, domain.StaffAccount{
		PhoneE164: req.GetPhoneE164(), DisplayName: req.GetDisplayName(), Role: req.GetRole(),
		StateCode: req.StateCode, District: req.District, ClusterID: clusterID, CSCID: req.CscId, Active: true,
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.CreateStaffResponse{Staff: staffToProto(s)}, nil
}

func (h *Staff) ListStaff(ctx context.Context, req *assistedv1.ListStaffRequest) (*assistedv1.ListStaffResponse, error) {
	staff, err := h.svc.ListStaff(ctx, req.StateCode)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*assistedv1.StaffAccount, len(staff))
	for i, s := range staff {
		out[i] = staffToProto(s)
	}
	return &assistedv1.ListStaffResponse{Staff: out}, nil
}

func (h *Staff) SetStaffActive(ctx context.Context, req *assistedv1.SetStaffActiveRequest) (*assistedv1.SetStaffActiveResponse, error) {
	id, err := parseUUID("id", req.GetId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	s, err := h.svc.SetStaffActive(ctx, id, req.GetActive())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.SetStaffActiveResponse{Staff: staffToProto(s)}, nil
}

func (h *Staff) GetMyStaffAccount(ctx context.Context, _ *assistedv1.GetMyStaffAccountRequest) (*assistedv1.GetMyStaffAccountResponse, error) {
	s, err := h.svc.GetMyStaffAccount(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.GetMyStaffAccountResponse{Staff: staffToProto(s)}, nil
}

func (h *Staff) ListAgentProductivity(ctx context.Context, req *assistedv1.ListAgentProductivityRequest) (*assistedv1.ListAgentProductivityResponse, error) {
	rows, err := h.svc.ListAgentProductivity(ctx, req.StateCode, req.District)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*assistedv1.AgentProductivity, len(rows))
	for i, r := range rows {
		out[i] = &assistedv1.AgentProductivity{
			AgentId: r.AgentID.String(), DisplayName: r.DisplayName, Role: r.Role,
			StateCode: r.StateCode, District: r.District, CscId: r.CSCID, Active: r.Active,
			ArtisansOnboarded: r.ArtisansOnboarded, ListingsCreated: r.ListingsCreated,
			LastActiveAt: optionalTimestamp(r.LastActiveAt),
		}
	}
	return &assistedv1.ListAgentProductivityResponse{Agents: out}, nil
}

func staffToProto(s domain.StaffAccount) *assistedv1.StaffAccount {
	var clusterID *string
	if s.ClusterID != nil {
		v := s.ClusterID.String()
		clusterID = &v
	}
	return &assistedv1.StaffAccount{
		Id: s.ID.String(), PhoneE164: s.PhoneE164, DisplayName: s.DisplayName, Role: s.Role,
		StateCode: s.StateCode, District: s.District, ClusterId: clusterID, CscId: s.CSCID,
		Active: s.Active, CreatedAt: timestamppb.New(s.CreatedAt),
	}
}

// Assisted implements assisted.v1.AssistedService.
type Assisted struct {
	assistedv1.UnimplementedAssistedServiceServer
	svc *service.Assisted
}

// NewAssisted builds the assisted-mode handler.
func NewAssisted(svc *service.Assisted) *Assisted { return &Assisted{svc: svc} }

func (h *Assisted) StartArtisanConsent(ctx context.Context, req *assistedv1.StartArtisanConsentRequest) (*assistedv1.StartArtisanConsentResponse, error) {
	c, err := h.svc.StartArtisanConsent(ctx, req.GetPhoneE164(), req.GetLanguage())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.StartArtisanConsentResponse{
		ChallengeId: c.ID, ExpiresAt: timestamppb.New(c.ExpiresAt), DevMode: c.DevMode,
	}, nil
}

func (h *Assisted) LinkArtisan(ctx context.Context, req *assistedv1.LinkArtisanRequest) (*assistedv1.LinkArtisanResponse, error) {
	in := service.LinkArtisanInput{
		PhoneE164: req.GetPhoneE164(), ConsentMethod: req.GetConsentMethod(),
		ChallengeID: req.GetChallengeId(), OTP: req.GetOtp(),
	}
	if req.Registration != nil {
		reg, err := registerInputFromProto(req.Registration)
		if err != nil {
			return nil, pkgdomain.GRPCError(err)
		}
		in.Registration = &reg
	}
	res, err := h.svc.LinkArtisan(ctx, in)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.LinkArtisanResponse{
		ArtisanId: res.ArtisanID.String(), LinkId: res.LinkID.String(),
		RegisteredNow: res.RegisteredNow, NeedsReview: res.NeedsReview,
	}, nil
}

func (h *Assisted) AttachVoiceConsent(ctx context.Context, req *assistedv1.AttachVoiceConsentRequest) (*assistedv1.AttachVoiceConsentResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	mediaID, err := parseUUID("media_id", req.GetMediaId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	ok, err := h.svc.AttachVoiceConsent(ctx, artisanID, mediaID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.AttachVoiceConsentResponse{Attached: ok}, nil
}

func (h *Assisted) ListMyArtisans(ctx context.Context, _ *assistedv1.ListMyArtisansRequest) (*assistedv1.ListMyArtisansResponse, error) {
	rows, err := h.svc.ListMyArtisans(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*assistedv1.AgentArtisan, len(rows))
	for i, a := range rows {
		var photo *string
		if a.PhotoMediaID != nil {
			v := a.PhotoMediaID.String()
			photo = &v
		}
		out[i] = &assistedv1.AgentArtisan{
			ArtisanId: a.ArtisanID.String(), DisplayName: a.DisplayName, Village: a.Village,
			District: a.District, StateCode: a.StateCode, PhotoMediaId: photo,
			ConsentMethod: a.ConsentMethod, NeedsReview: a.NeedsReview, LinkedAt: timestamppb.New(a.LinkedAt),
			LastActivityAt: optionalTimestamp(a.LastActivityAt), DraftCount: a.DraftCount,
		}
	}
	return &assistedv1.ListMyArtisansResponse{Artisans: out}, nil
}

func (h *Assisted) ResolveOnBehalf(ctx context.Context, req *assistedv1.ResolveOnBehalfRequest) (*assistedv1.ResolveOnBehalfResponse, error) {
	artisanID, err := parseUUID("artisan_id", req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	ok, name, err := h.svc.ResolveOnBehalf(ctx, artisanID)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.ResolveOnBehalfResponse{Allowed: ok, DisplayName: name}, nil
}

func (h *Assisted) ListMyHelpers(ctx context.Context, _ *assistedv1.ListMyHelpersRequest) (*assistedv1.ListMyHelpersResponse, error) {
	rows, err := h.svc.ListMyHelpers(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*assistedv1.Helper, len(rows))
	for i, r := range rows {
		out[i] = &assistedv1.Helper{
			LinkId: r.LinkID.String(), AgentId: r.AgentID.String(), DisplayName: r.DisplayName,
			Role: r.Role, CscId: r.CSCID, ConsentMethod: r.ConsentMethod, LinkedAt: timestamppb.New(r.LinkedAt),
		}
	}
	return &assistedv1.ListMyHelpersResponse{Helpers: out}, nil
}

func (h *Assisted) RevokeHelper(ctx context.Context, req *assistedv1.RevokeHelperRequest) (*assistedv1.RevokeHelperResponse, error) {
	id, err := parseUUID("link_id", req.GetLinkId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	ok, err := h.svc.RevokeHelper(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.RevokeHelperResponse{Revoked: ok}, nil
}

func (h *Assisted) ListLinksForReview(ctx context.Context, req *assistedv1.ListLinksForReviewRequest) (*assistedv1.ListLinksForReviewResponse, error) {
	rows, err := h.svc.ListLinksForReview(ctx, req.StateCode, req.District)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*assistedv1.LinkForReview, len(rows))
	for i, r := range rows {
		// consent_ref holds the recording's media id once it is attached;
		// before that it is a placeholder the reviewer should not see as an id.
		voice := ""
		if _, err := uuid.Parse(r.ConsentRef); err == nil {
			voice = r.ConsentRef
		}
		out[i] = &assistedv1.LinkForReview{
			LinkId: r.LinkID.String(), AgentId: r.AgentID.String(), AgentName: r.AgentName,
			ArtisanId: r.ArtisanID.String(), ArtisanName: r.ArtisanName, District: r.District,
			StateCode: r.StateCode, VoiceMediaId: voice, CreatedAt: timestamppb.New(r.CreatedAt),
		}
	}
	return &assistedv1.ListLinksForReviewResponse{Links: out}, nil
}

func (h *Assisted) MarkLinkReviewed(ctx context.Context, req *assistedv1.MarkLinkReviewedRequest) (*assistedv1.MarkLinkReviewedResponse, error) {
	id, err := parseUUID("link_id", req.GetLinkId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	ok, err := h.svc.MarkLinkReviewed(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.MarkLinkReviewedResponse{Reviewed: ok}, nil
}

func (h *Assisted) GetListingHelper(ctx context.Context, req *assistedv1.GetListingHelperRequest) (*assistedv1.GetListingHelperResponse, error) {
	id, err := parseUUID("listing_id", req.GetListingId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	name, csc, ok, err := h.svc.GetListingHelper(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &assistedv1.GetListingHelperResponse{Assisted: ok, DisplayName: name, CscId: csc}, nil
}
