// services/core-svc/internal/core/handler/finance.go

package handler

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	financev1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/finance/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Finance implements finance.v1.FinanceService.
type Finance struct {
	financev1.UnimplementedFinanceServiceServer
	svc *service.Finance
}

// NewFinance builds the finance handler.
func NewFinance(svc *service.Finance) *Finance { return &Finance{svc: svc} }

func (h *Finance) LinkFinance(ctx context.Context, req *financev1.LinkFinanceRequest) (*financev1.LinkFinanceResponse, error) {
	start, err := parseOptionalDate("repayment_start", req.RepaymentStart)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	link, err := h.svc.LinkFinance(ctx, service.LinkFinanceInput{
		Corporation: req.GetCorporation(), Reference: req.GetReference(),
		Terms: domain.FinanceTerms{
			ChannelizingAgency: req.ChannelizingAgency, SanctionedPaise: req.SanctionedPaise,
			EmiPaise: req.EmiPaise, EmiDayOfMonth: req.EmiDayOfMonth, RepaymentStart: start,
		},
		ConsentGiven: req.GetConsentGiven(), ConsentVersion: req.GetConsentVersion(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &financev1.LinkFinanceResponse{Link: financeLinkToProto(link)}, nil
}

func (h *Finance) ListMyFinanceLinks(ctx context.Context, _ *financev1.ListMyFinanceLinksRequest) (*financev1.ListMyFinanceLinksResponse, error) {
	links, err := h.svc.ListMyFinanceLinks(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*financev1.FinanceLink, len(links))
	for i, l := range links {
		out[i] = financeLinkToProto(l)
	}
	return &financev1.ListMyFinanceLinksResponse{Links: out}, nil
}

func (h *Finance) UpdateFinanceLink(ctx context.Context, req *financev1.UpdateFinanceLinkRequest) (*financev1.UpdateFinanceLinkResponse, error) {
	id, err := parseUUID("id", req.GetId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	start, err := parseOptionalDate("repayment_start", req.RepaymentStart)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	link, err := h.svc.UpdateFinanceLink(ctx, id, domain.FinanceTerms{
		ChannelizingAgency: req.ChannelizingAgency, SanctionedPaise: req.SanctionedPaise,
		EmiPaise: req.EmiPaise, EmiDayOfMonth: req.EmiDayOfMonth, RepaymentStart: start,
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &financev1.UpdateFinanceLinkResponse{Link: financeLinkToProto(link)}, nil
}

func (h *Finance) DeleteFinanceLink(ctx context.Context, req *financev1.DeleteFinanceLinkRequest) (*financev1.DeleteFinanceLinkResponse, error) {
	id, err := parseUUID("id", req.GetId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	deleted, err := h.svc.DeleteFinanceLink(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &financev1.DeleteFinanceLinkResponse{Deleted: deleted}, nil
}

func (h *Finance) ReviewFinanceLink(ctx context.Context, req *financev1.ReviewFinanceLinkRequest) (*financev1.ReviewFinanceLinkResponse, error) {
	id, err := parseUUID("id", req.GetId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	link, err := h.svc.ReviewFinanceLink(ctx, id, req.GetVerified(), req.Reason)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &financev1.ReviewFinanceLinkResponse{Link: financeLinkToProto(link)}, nil
}

func (h *Finance) ListFinanceLinksForReview(ctx context.Context, req *financev1.ListFinanceLinksForReviewRequest) (*financev1.ListFinanceLinksForReviewResponse, error) {
	links, err := h.svc.ListFinanceLinksForReview(ctx, req.Status, req.StateCode, req.District)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*financev1.FinanceLinkForReview, len(links))
	for i, l := range links {
		out[i] = &financev1.FinanceLinkForReview{
			Id: l.ID.String(), ArtisanId: l.ArtisanID.String(), ArtisanName: l.ArtisanName,
			StateCode: l.StateCode, District: l.District, Corporation: l.Corporation,
			ChannelizingAgency: l.ChannelizingAgency, ReferenceLast4: l.ReferenceLast4,
			SanctionedPaise: l.SanctionedPaise, EmiPaise: l.EmiPaise, EmiDayOfMonth: l.EmiDayOfMonth,
			Status: l.Status, CreatedAt: timestamppb.New(l.CreatedAt),
		}
	}
	return &financev1.ListFinanceLinksForReviewResponse{Links: out}, nil
}

func (h *Finance) GetRepaymentCoverage(ctx context.Context, req *financev1.GetRepaymentCoverageRequest) (*financev1.GetRepaymentCoverageResponse, error) {
	c, err := h.svc.GetRepaymentCoverage(ctx, req.GetMonth())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &financev1.GetRepaymentCoverageResponse{Coverage: &financev1.RepaymentCoverage{
		Month: c.Month.Format("2006-01"), EmiPaise: c.EmiPaise,
		EarnedPlatformPaise: c.EarnedPlatformPaise, EarnedOfflinePaise: c.EarnedOfflinePaise,
		PendingPaise: c.PendingPaise, CoverageRatio: c.CoverageRatio, DaysToEmi: c.DaysToEmi,
		Status: c.Status, AnyVerified: c.AnyVerified,
	}}, nil
}

func financeLinkToProto(l domain.FinanceLink) *financev1.FinanceLink {
	return &financev1.FinanceLink{
		Id: l.ID.String(), Corporation: l.Corporation, ChannelizingAgency: l.ChannelizingAgency,
		ReferenceLast4: l.ReferenceLast4, SanctionedPaise: l.SanctionedPaise, EmiPaise: l.EmiPaise,
		EmiDayOfMonth: l.EmiDayOfMonth, RepaymentStart: formatOptionalDate(l.RepaymentStart),
		Status: l.Status, VerifiedAt: optionalTimestamp(l.VerifiedAt), RejectReason: l.RejectReason,
		ConsentAt: timestamppb.New(l.ConsentAt), ConsentVersion: l.ConsentVersion,
		CreatedAt: timestamppb.New(l.CreatedAt),
	}
}

// --- date helpers shared by the finance and income handlers ---

const dateLayout = "2006-01-02"

func parseDate(field, v string) (time.Time, error) {
	t, err := time.Parse(dateLayout, v)
	if err != nil {
		return time.Time{}, pkgdomain.InvalidInput(field + " must be YYYY-MM-DD")
	}
	return t, nil
}

func parseOptionalDate(field string, v *string) (*time.Time, error) {
	if v == nil || *v == "" {
		return nil, nil
	}
	t, err := parseDate(field, *v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func formatOptionalDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}

func optionalTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
