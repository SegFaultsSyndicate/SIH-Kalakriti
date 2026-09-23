// services/core-svc/internal/core/handler/income.go

package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	impactv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/impact/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Income implements impact.v1.IncomeService.
type Income struct {
	impactv1.UnimplementedIncomeServiceServer
	svc *service.Income
}

// NewIncome builds the income handler.
func NewIncome(svc *service.Income) *Income { return &Income{svc: svc} }

func (h *Income) SetIncomeBaseline(ctx context.Context, req *impactv1.SetIncomeBaselineRequest) (*impactv1.SetIncomeBaselineResponse, error) {
	b, err := h.svc.SetIncomeBaseline(ctx, domain.IncomeBaseline{
		MonthlyBracket: req.GetMonthlyBracket(), MonthlyPaise: req.MonthlyPaise,
		FairsPerYear: req.FairsPerYear, FairIncomeBracket: req.FairIncomeBracket,
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &impactv1.SetIncomeBaselineResponse{Baseline: baselineToProto(&b)}, nil
}

func (h *Income) GetIncomeBaseline(ctx context.Context, _ *impactv1.GetIncomeBaselineRequest) (*impactv1.GetIncomeBaselineResponse, error) {
	b, err := h.svc.GetIncomeBaseline(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &impactv1.GetIncomeBaselineResponse{Baseline: baselineToProto(b)}, nil
}

func (h *Income) LogOfflineSale(ctx context.Context, req *impactv1.LogOfflineSaleRequest) (*impactv1.LogOfflineSaleResponse, error) {
	id, err := parseUUID("id", req.GetId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	soldOn, err := parseDate("sold_on", req.GetSoldOn())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	s, err := h.svc.LogOfflineSale(ctx, domain.OfflineSale{
		ID: id, Channel: req.GetChannel(), EventName: req.EventName, AmountPaise: req.GetAmountPaise(), SoldOn: soldOn,
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &impactv1.LogOfflineSaleResponse{Sale: offlineSaleToProto(s)}, nil
}

func (h *Income) ListOfflineSales(ctx context.Context, req *impactv1.ListOfflineSalesRequest) (*impactv1.ListOfflineSalesResponse, error) {
	sales, err := h.svc.ListOfflineSales(ctx, req.GetLimit())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*impactv1.OfflineSale, len(sales))
	for i, s := range sales {
		out[i] = offlineSaleToProto(s)
	}
	return &impactv1.ListOfflineSalesResponse{Sales: out}, nil
}

func (h *Income) DeleteOfflineSale(ctx context.Context, req *impactv1.DeleteOfflineSaleRequest) (*impactv1.DeleteOfflineSaleResponse, error) {
	id, err := parseUUID("id", req.GetId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	deleted, err := h.svc.DeleteOfflineSale(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &impactv1.DeleteOfflineSaleResponse{Deleted: deleted}, nil
}

func (h *Income) GetMyIncomeSummary(ctx context.Context, _ *impactv1.GetMyIncomeSummaryRequest) (*impactv1.GetMyIncomeSummaryResponse, error) {
	s, err := h.svc.GetMyIncomeSummary(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	months := make([]*impactv1.MonthIncome, len(s.Months))
	for i, m := range s.Months {
		months[i] = &impactv1.MonthIncome{Month: m.Month.Format("2006-01"), PlatformPaise: m.PlatformPaise, OfflinePaise: m.OfflinePaise}
	}
	return &impactv1.GetMyIncomeSummaryResponse{Summary: &impactv1.IncomeSummary{
		Baseline: baselineToProto(s.Baseline), BaselineMonthlyPaise: s.BaselineMonthlyPaise, Months: months,
		PlatformPaise_90D: s.Facts.PlatformPaise90d, OfflinePaise_90D: s.Facts.OfflinePaise90d,
		FairPaise_90D: s.Facts.FairPaise90d, PlatformPendingPaise: s.Facts.PlatformPendingPaise,
		CurrentMonthlyPaise: s.CurrentMonthlyPaise, UpliftPct: s.UpliftPct,
		InsufficientData: s.InsufficientData, DigitalSharePct: s.DigitalSharePct,
	}}, nil
}

func baselineToProto(b *domain.IncomeBaseline) *impactv1.IncomeBaseline {
	if b == nil {
		return nil
	}
	return &impactv1.IncomeBaseline{
		MonthlyBracket: b.MonthlyBracket, MonthlyPaise: b.MonthlyPaise, FairsPerYear: b.FairsPerYear,
		FairIncomeBracket: b.FairIncomeBracket, CapturedAt: timestamppb.New(b.CapturedAt), Source: b.Source,
	}
}

func offlineSaleToProto(s domain.OfflineSale) *impactv1.OfflineSale {
	return &impactv1.OfflineSale{
		Id: s.ID.String(), Channel: s.Channel, EventName: s.EventName, AmountPaise: s.AmountPaise,
		SoldOn: s.SoldOn.Format(dateLayout), CreatedAt: timestamppb.New(s.CreatedAt),
	}
}
