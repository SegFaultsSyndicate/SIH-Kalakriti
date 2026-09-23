package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	insightv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/insight/v1"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/domain"
)

// impactFilter reads the request filter and clamps it for the caller:
// MINISTRY sees everything; a CLUSTER_OFFICER only their own state (and
// district, when their scope has one), whatever the request asked for.
func impactFilter(ctx context.Context, in *insightv1.ImpactFilter) (domain.ImpactFilter, error) {
	p, err := auth.RequireRole(ctx, auth.RoleMinistry, auth.RoleClusterOfficer)
	if err != nil {
		return domain.ImpactFilter{}, err
	}
	f := domain.ImpactFilter{
		StateCode: nonEmpty(in.GetStateCode()), District: nonEmpty(in.GetDistrict()),
		SocialCategory: nonEmpty(in.GetSocialCategory()), Corporation: nonEmpty(in.GetCorporation()),
	}
	if f.FromMonth, err = parseMonth("from_month", in.GetFromMonth()); err != nil {
		return f, err
	}
	if f.ToMonth, err = parseMonth("to_month", in.GetToMonth()); err != nil {
		return f, err
	}
	if p.Role == auth.RoleClusterOfficer {
		if p.ScopeState == "" {
			return f, pkgdomain.Forbidden("your account has no region scope; ask an administrator to set one")
		}
		f.StateCode = &p.ScopeState
		if p.ScopeDistrict != "" {
			f.District = &p.ScopeDistrict
		}
	}
	return f, nil
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseMonth(field, v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01", v)
	if err != nil {
		return nil, pkgdomain.InvalidInput(field + " must be YYYY-MM")
	}
	return &t, nil
}

func (h *Handler) GetImpactSummary(ctx context.Context, req *insightv1.GetImpactSummaryRequest) (*insightv1.GetImpactSummaryResponse, error) {
	f, err := impactFilter(ctx, req.GetFilter())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	s, err := h.svc.GetImpactSummary(ctx, f)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	resp := &insightv1.GetImpactSummaryResponse{
		Suppressed: s.Suppressed, Beneficiaries: s.Beneficiaries, ActiveSellers_90D: s.ActiveSellers90d,
		UpliftSample: s.UpliftSample, MedianUpliftPct: s.MedianUpliftPct, DigitalSharePct: s.DigitalSharePct,
		CertificatesIssued: s.CertificatesIssued, FinanceLinked: s.FinanceLinked, FinanceVerified: s.FinanceVerified,
	}
	if s.RefreshedAt != nil {
		resp.RefreshedAt = timestamppb.New(*s.RefreshedAt)
	}
	return resp, nil
}

func (h *Handler) GetImpactByGroup(ctx context.Context, req *insightv1.GetImpactByGroupRequest) (*insightv1.GetImpactByGroupResponse, error) {
	f, err := impactFilter(ctx, req.GetFilter())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	rows, err := h.svc.GetImpactByGroup(ctx, req.GetGroupBy(), f)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*insightv1.ImpactGroupRow, len(rows))
	for i, r := range rows {
		out[i] = &insightv1.ImpactGroupRow{
			Group: r.Group, StateCode: r.StateCode, Suppressed: r.Suppressed, ArtisanCount: r.ArtisanCount,
			ActiveSellers_90D: r.ActiveSellers90d, WithBaselineCount: r.WithBaselineCount,
			MedianBaselineMonthlyPaise: r.MedianBaselinePaise, MedianCurrentMonthlyPaise: r.MedianCurrentPaise,
			MedianUpliftPct: r.MedianUpliftPct, PlatformIncomePaise_90D: r.PlatformIncomePaise,
			OfflineIncomePaise_90D: r.OfflineIncomePaise, FairIncomePaise_90D: r.FairIncomePaise,
		}
	}
	return &insightv1.GetImpactByGroupResponse{Rows: out}, nil
}

func (h *Handler) GetSalesMix(ctx context.Context, req *insightv1.GetSalesMixRequest) (*insightv1.GetSalesMixResponse, error) {
	f, err := impactFilter(ctx, req.GetFilter())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	months, err := h.svc.GetSalesMix(ctx, f)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*insightv1.SalesMixMonth, len(months))
	for i, m := range months {
		out[i] = &insightv1.SalesMixMonth{
			Month: m.Month.Format("2006-01"), Suppressed: m.Suppressed, PlatformPaise: m.PlatformPaise,
			FairPaise: m.FairPaise, OtherOfflinePaise: m.OtherOfflinePaise,
		}
	}
	return &insightv1.GetSalesMixResponse{Months: out}, nil
}

func (h *Handler) GetFinanceCoverage(ctx context.Context, req *insightv1.GetFinanceCoverageRequest) (*insightv1.GetFinanceCoverageResponse, error) {
	f, err := impactFilter(ctx, req.GetFilter())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	rows, err := h.svc.GetFinanceCoverage(ctx, f)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*insightv1.FinanceCoverageRow, len(rows))
	for i, r := range rows {
		out[i] = &insightv1.FinanceCoverageRow{
			Corporation: r.Corporation, Suppressed: r.Suppressed, Beneficiaries: r.Beneficiaries,
			Verified: r.Verified, SelfReported: r.SelfReported, ActiveSellers_90D: r.ActiveSellers90d,
			MedianCoverageRatio: r.MedianCoverageRatio,
		}
	}
	return &insightv1.GetFinanceCoverageResponse{Rows: out}, nil
}

func (h *Handler) GetLiteracyFunnel(ctx context.Context, req *insightv1.GetLiteracyFunnelRequest) (*insightv1.GetLiteracyFunnelResponse, error) {
	f, err := impactFilter(ctx, req.GetFilter())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	rows, err := h.svc.GetLiteracyFunnel(ctx, f)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*insightv1.LiteracyFunnelRow, len(rows))
	for i, r := range rows {
		out[i] = &insightv1.LiteracyFunnelRow{
			StateCode: r.StateCode, District: r.District, Suppressed: r.Suppressed, Artisans: r.Artisans,
			Started: r.Started, HalfWay: r.HalfWay, Certified: r.Certified,
		}
	}
	return &insightv1.GetLiteracyFunnelResponse{Rows: out}, nil
}

// --- literacy certificate ---

// selfArtisan allows only the artisan themselves (or an agent acting for
// them, who arrives as that artisan).
func selfArtisan(ctx context.Context, artisanID string) (uuid.UUID, error) {
	p, err := auth.RequireRole(ctx, auth.RoleArtisan)
	if err != nil {
		return uuid.Nil, err
	}
	if p.Subject != artisanID {
		return uuid.Nil, pkgdomain.Forbidden("you can only act on your own certificate")
	}
	id, err := uuid.Parse(artisanID)
	if err != nil {
		return uuid.Nil, pkgdomain.InvalidInput("artisan_id must be a UUID")
	}
	return id, nil
}

func (h *Handler) certificateToProto(ctx context.Context, c *domain.LiteracyCertificate) (*insightv1.LiteracyCertificate, error) {
	url, err := h.svc.CertificateDownloadURL(ctx, c)
	if err != nil {
		return nil, err
	}
	return &insightv1.LiteracyCertificate{
		Id: c.ID.String(), ShortCode: c.ShortCode, IssuedAt: timestamppb.New(c.IssuedAt),
		DownloadUrl: url, VerificationUrl: h.svc.CertificateVerifyURL(c.ShortCode),
	}, nil
}

func (h *Handler) IssueLiteracyCertificate(ctx context.Context, req *insightv1.IssueLiteracyCertificateRequest) (*insightv1.IssueLiteracyCertificateResponse, error) {
	id, err := selfArtisan(ctx, req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	c, err := h.svc.IssueLiteracyCertificate(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	pc, err := h.certificateToProto(ctx, c)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &insightv1.IssueLiteracyCertificateResponse{Certificate: pc}, nil
}

func (h *Handler) GetLiteracyCertificate(ctx context.Context, req *insightv1.GetLiteracyCertificateRequest) (*insightv1.GetLiteracyCertificateResponse, error) {
	id, err := selfArtisan(ctx, req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	c, err := h.svc.GetLiteracyCertificate(ctx, id)
	if err != nil || c == nil {
		return &insightv1.GetLiteracyCertificateResponse{}, pkgdomain.GRPCError(err)
	}
	pc, err := h.certificateToProto(ctx, c)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &insightv1.GetLiteracyCertificateResponse{Certificate: pc}, nil
}

// VerifyLiteracyCertificate is public: anyone holding the printed code can
// check it. It reveals only name, craft and place.
func (h *Handler) VerifyLiteracyCertificate(ctx context.Context, req *insightv1.VerifyLiteracyCertificateRequest) (*insightv1.VerifyLiteracyCertificateResponse, error) {
	c, valid, err := h.svc.VerifyLiteracyCertificate(ctx, req.GetShortCode())
	if err != nil {
		// Could not check (storage down): say so rather than calling a
		// genuine certificate invalid.
		return nil, pkgdomain.GRPCError(pkgdomain.Unavailable("certificate check is unavailable, try again shortly"))
	}
	if c == nil {
		return nil, pkgdomain.GRPCError(pkgdomain.NotFound("no certificate with that code"))
	}
	return &insightv1.VerifyLiteracyCertificateResponse{
		Valid: valid, ArtisanName: c.ArtisanName, CraftName: c.CraftName,
		District: c.District, StateCode: c.StateCode, IssuedAt: timestamppb.New(c.IssuedAt),
	}, nil
}
