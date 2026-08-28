package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/segfaultsyndicate/kalakriti/pkg/auth"
	"github.com/segfaultsyndicate/kalakriti/pkg/money"
	commonv1 "github.com/segfaultsyndicate/kalakriti/proto/common/v1"
	insightv1 "github.com/segfaultsyndicate/kalakriti/proto/insight/v1"
	"github.com/segfaultsyndicate/kalakriti/services/insight-svc/internal/insight/service"
)

type Handler struct {
	insightv1.UnimplementedInsightServiceServer
	svc           *service.Service
	verifyBaseURL string
}

func New(svc *service.Service, verifyBaseURL string) *Handler {
	return &Handler{svc: svc, verifyBaseURL: verifyBaseURL}
}

func (h *Handler) GetArtisansByCategory(ctx context.Context, req *insightv1.GetArtisansByCategoryRequest) (*insightv1.GetArtisansByCategoryResponse, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return nil, status.Error(codes.PermissionDenied, "MINISTRY role required")
	}

	var stateCode, district *string
	if req.StateCode != nil {
		stateCode = req.StateCode
	}
	if req.District != nil {
		district = req.District
	}

	rows, err := h.svc.GetArtisansByCategory(ctx, stateCode, district)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching aggregates: %v", err)
	}

	resp := &insightv1.GetArtisansByCategoryResponse{Rows: make([]*insightv1.ArtisanCategoryRow, len(rows))}
	for i, row := range rows {
		resp.Rows[i] = &insightv1.ArtisanCategoryRow{
			StateCode:      row.StateCode,
			District:       row.District,
			SocialCategory: row.SocialCategory,
			ArtisanCount:   row.ArtisanCount,
			VerifiedCount:  row.VerifiedCount,
		}
	}
	return resp, nil
}

func (h *Handler) GetListingsByCraftMonth(ctx context.Context, req *insightv1.GetListingsByCraftMonthRequest) (*insightv1.GetListingsByCraftMonthResponse, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return nil, status.Error(codes.PermissionDenied, "MINISTRY role required")
	}

	var from, to *time.Time
	var craftID *uuid.UUID
	if req.FromDate != nil {
		t := req.FromDate.AsTime()
		from = &t
	}
	if req.ToDate != nil {
		t := req.ToDate.AsTime()
		to = &t
	}
	if req.CraftId != nil {
		id, err := uuid.Parse(*req.CraftId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid craft_id: %v", err)
		}
		craftID = &id
	}

	rows, err := h.svc.GetListingsByCraftMonth(ctx, from, to, craftID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching aggregates: %v", err)
	}

	resp := &insightv1.GetListingsByCraftMonthResponse{Rows: make([]*insightv1.ListingCraftMonthRow, len(rows))}
	for i, row := range rows {
		resp.Rows[i] = &insightv1.ListingCraftMonthRow{
			CraftId:      row.CraftID,
			CraftName:    row.CraftName,
			Month:        timestamppb.New(row.Month),
			ListingCount: row.ListingCount,
			ArtisanCount: row.ArtisanCount,
		}
	}
	return resp, nil
}

func (h *Handler) GetEarningsByDistrict(ctx context.Context, req *insightv1.GetEarningsByDistrictRequest) (*insightv1.GetEarningsByDistrictResponse, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return nil, status.Error(codes.PermissionDenied, "MINISTRY role required")
	}

	var stateCode, district *string
	if req.StateCode != nil {
		stateCode = req.StateCode
	}
	if req.District != nil {
		district = req.District
	}

	rows, err := h.svc.GetEarningsByDistrict(ctx, stateCode, district)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching aggregates: %v", err)
	}

	resp := &insightv1.GetEarningsByDistrictResponse{Rows: make([]*insightv1.EarningsDistrictRow, len(rows))}
	for i, row := range rows {
		resp.Rows[i] = &insightv1.EarningsDistrictRow{
			StateCode:    row.StateCode,
			District:     row.District,
			TotalGmv:     money.New(row.TotalGMVPaise).ToProto(),
			TotalNet:     money.New(row.TotalNetPaise).ToProto(),
			ArtisanCount: row.ArtisanCount,
			AvgEarnings:  money.New(row.AvgEarnings).ToProto(),
		}
	}
	return resp, nil
}

func (h *Handler) GetIncomeComparison(ctx context.Context, req *insightv1.GetIncomeComparisonRequest) (*insightv1.GetIncomeComparisonResponse, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return nil, status.Error(codes.PermissionDenied, "MINISTRY role required")
	}

	var stateCode, district *string
	if req.StateCode != nil {
		stateCode = req.StateCode
	}
	if req.District != nil {
		district = req.District
	}

	rows, err := h.svc.GetIncomeComparison(ctx, stateCode, district)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching aggregates: %v", err)
	}

	resp := &insightv1.GetIncomeComparisonResponse{Rows: make([]*insightv1.IncomeComparisonRow, len(rows))}
	for i, row := range rows {
		resp.Rows[i] = &insightv1.IncomeComparisonRow{
			StateCode:     row.StateCode,
			District:      row.District,
			MedianBefore:  money.New(row.MedianBeforePaise).ToProto(),
			MedianAfter:   money.New(row.MedianAfterPaise).ToProto(),
			ArtisanCount:  row.ArtisanCount,
		}
	}
	return resp, nil
}

func (h *Handler) GetDyingCrafts(ctx context.Context, req *insightv1.GetDyingCraftsRequest) (*insightv1.GetDyingCraftsResponse, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return nil, status.Error(codes.PermissionDenied, "MINISTRY role required")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	rows, err := h.svc.GetDyingCrafts(ctx, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching dying crafts: %v", err)
	}

	resp := &insightv1.GetDyingCraftsResponse{Rows: make([]*insightv1.DyingCraftRow, len(rows))}
	for i, row := range rows {
		resp.Rows[i] = &insightv1.DyingCraftRow{
			CraftId:         row.CraftID,
			CraftName:       row.CraftName,
			DeclineRate:     row.DeclineRate,
			PeakArtisans:    row.PeakArtisans,
			CurrentArtisans: row.CurrentArtisans,
		}
	}
	return resp, nil
}

func (h *Handler) RefreshMaterializedViews(ctx context.Context, req *insightv1.RefreshMaterializedViewsRequest) (*insightv1.RefreshMaterializedViewsResponse, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return nil, status.Error(codes.PermissionDenied, "MINISTRY role required")
	}

	if err := h.svc.RefreshMaterializedViews(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "refreshing views: %v", err)
	}

	return &insightv1.RefreshMaterializedViewsResponse{
		RefreshedAt: timestamppb.New(time.Now().UTC()),
	}, nil
}

func (h *Handler) GenerateIncomeStatement(ctx context.Context, req *insightv1.GenerateIncomeStatementRequest) (*insightv1.GenerateIncomeStatementResponse, error) {
	artisanID, err := uuid.Parse(req.ArtisanId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid artisan_id: %v", err)
	}

	// Artisan can only request their own statement.
	if _, err := auth.RequireSelfOrRole(ctx, req.ArtisanId); err != nil {
		return nil, status.Error(codes.PermissionDenied, "can only generate own income statement")
	}

	start := req.PeriodStart.AsTime()
	end := req.PeriodEnd.AsTime()
	if end.Before(start) || end.Equal(start) {
		return nil, status.Error(codes.InvalidArgument, "period_end must be after period_start")
	}

	stmt, err := h.svc.GenerateIncomeStatement(ctx, artisanID, start, end)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generating statement: %v", err)
	}

	downloadURL, err := h.svc.GetStatementDownloadURL(ctx, stmt.S3Key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generating download URL: %v", err)
	}

	return &insightv1.GenerateIncomeStatementResponse{
		StatementId:     stmt.ID.String(),
		ShortCode:       stmt.ShortCode,
		DownloadUrl:     downloadURL,
		VerificationUrl: h.verifyBaseURL + "/v/statement/" + stmt.ShortCode,
	}, nil
}

func (h *Handler) GetIncomeStatements(ctx context.Context, req *insightv1.GetIncomeStatementsRequest) (*insightv1.GetIncomeStatementsResponse, error) {
	artisanID, err := uuid.Parse(req.ArtisanId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid artisan_id: %v", err)
	}

	// Artisan can only fetch their own statements.
	if _, err := auth.RequireSelfOrRole(ctx, req.ArtisanId); err != nil {
		return nil, status.Error(codes.PermissionDenied, "can only fetch own statements")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	statements, err := h.svc.GetIncomeStatements(ctx, artisanID, limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching statements: %v", err)
	}

	resp := &insightv1.GetIncomeStatementsResponse{Statements: make([]*insightv1.IncomeStatementSummary, len(statements))}
	for i, s := range statements {
		downloadURL, err := h.svc.GetStatementDownloadURL(ctx, s.S3Key)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generating download URL: %v", err)
		}
		resp.Statements[i] = &insightv1.IncomeStatementSummary{
			StatementId:  s.ID.String(),
			PeriodStart:  timestamppb.New(s.PeriodStart),
			PeriodEnd:    timestamppb.New(s.PeriodEnd),
			OrderCount:   s.OrderCount,
			GrossAmount:  money.New(s.GrossPaise).ToProto(),
			NetAmount:    money.New(s.NetPaise).ToProto(),
			FeeAmount:    money.New(s.FeePaise).ToProto(),
			ShortCode:    s.ShortCode,
			DownloadUrl:  downloadURL,
			CreatedAt:    timestamppb.New(s.CreatedAt),
		}
	}
	return resp, nil
}

func (h *Handler) VerifyIncomeStatement(ctx context.Context, req *insightv1.VerifyIncomeStatementRequest) (*insightv1.VerifyIncomeStatementResponse, error) {
	// Public endpoint, no auth required.
	stmt, valid, err := h.svc.VerifyIncomeStatement(ctx, req.ShortCode)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "verifying statement: %v", err)
	}

	return &insightv1.VerifyIncomeStatementResponse{
		Valid:        valid,
		ArtisanId:    stmt.ArtisanID.String(),
		ArtisanName:  stmt.ArtisanName,
		PeriodStart:  timestamppb.New(stmt.PeriodStart),
		PeriodEnd:    timestamppb.New(stmt.PeriodEnd),
		OrderCount:   stmt.OrderCount,
		GrossAmount:  money.New(stmt.GrossPaise).ToProto(),
		NetAmount:    money.New(stmt.NetPaise).ToProto(),
		FeeAmount:    money.New(stmt.FeePaise).ToProto(),
		IssuedAt:     timestamppb.New(stmt.CreatedAt),
	}, nil
}
