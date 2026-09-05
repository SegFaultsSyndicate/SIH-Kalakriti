// services/bff/internal/bff/client/insight.go
package client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	insightv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/insight/v1"
)

// Insight is bff's view of insight-svc's ministry dashboard aggregates and
// artisan income statements — it satisfies both handler.InsightService and
// handler.StatementService.
type Insight struct {
	insight insightv1.InsightServiceClient
}

// NewInsight builds the insight client.
func NewInsight(conn grpc.ClientConnInterface) *Insight {
	return &Insight{insight: insightv1.NewInsightServiceClient(conn)}
}

// stringFilter reads an optional string filter out of a loose filters map,
// treating "" the same as absent so callers need not omit the key.
func stringFilter(filters map[string]any, key string) *string {
	v, ok := filters[key].(string)
	if !ok || v == "" {
		return nil
	}
	return &v
}

func moneyMap(m *commonv1.Money) map[string]any {
	if m == nil {
		return nil
	}
	return map[string]any{"amount_paise": m.GetAmountPaise(), "currency_code": m.GetCurrencyCode()}
}

// GetEarningsByDistrict returns total and average earnings per district.
func (in *Insight) GetEarningsByDistrict(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	req := &insightv1.GetEarningsByDistrictRequest{
		StateCode: stringFilter(filters, "state_code"),
		District:  stringFilter(filters, "district"),
	}
	resp, err := in.insight.GetEarningsByDistrict(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	rows := make([]map[string]any, 0, len(resp.GetRows()))
	for _, r := range resp.GetRows() {
		rows = append(rows, map[string]any{
			"state_code":    r.GetStateCode(),
			"district":      r.GetDistrict(),
			"total_gmv":     moneyMap(r.GetTotalGmv()),
			"total_net":     moneyMap(r.GetTotalNet()),
			"artisan_count": r.GetArtisanCount(),
			"avg_earnings":  moneyMap(r.GetAvgEarnings()),
		})
	}
	return rows, nil
}

// GetIncomeComparison returns before/after median income per district.
func (in *Insight) GetIncomeComparison(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	req := &insightv1.GetIncomeComparisonRequest{
		StateCode: stringFilter(filters, "state_code"),
		District:  stringFilter(filters, "district"),
	}
	resp, err := in.insight.GetIncomeComparison(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	rows := make([]map[string]any, 0, len(resp.GetRows()))
	for _, r := range resp.GetRows() {
		rows = append(rows, map[string]any{
			"state_code":    r.GetStateCode(),
			"district":      r.GetDistrict(),
			"median_before": moneyMap(r.GetMedianBefore()),
			"median_after":  moneyMap(r.GetMedianAfter()),
			"artisan_count": r.GetArtisanCount(),
		})
	}
	return rows, nil
}

// GetArtisansByCategory returns artisan counts by state/district/social
// category. Unlike GetEarningsByDistrict and GetIncomeComparison, insight-svc
// applies no small-bucket suppression to this query -- every row it returns
// is raw, including buckets under 5 artisans. It doubles as the district
// roster the ministry dashboard diffs the suppressed endpoints against: any
// district present here but absent from an earnings/income response was
// suppressed, not empty.
func (in *Insight) GetArtisansByCategory(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	req := &insightv1.GetArtisansByCategoryRequest{
		StateCode: stringFilter(filters, "state_code"),
		District:  stringFilter(filters, "district"),
	}
	resp, err := in.insight.GetArtisansByCategory(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	rows := make([]map[string]any, 0, len(resp.GetRows()))
	for _, r := range resp.GetRows() {
		rows = append(rows, map[string]any{
			"state_code":      r.GetStateCode(),
			"district":        r.GetDistrict(),
			"social_category": r.GetSocialCategory(),
			"artisan_count":   r.GetArtisanCount(),
			"verified_count":  r.GetVerifiedCount(),
		})
	}
	return rows, nil
}

// GetListingsByCraftMonth returns listing/artisan counts by craft and month,
// unsuppressed (see GetArtisansByCategory's doc comment on why that matters).
func (in *Insight) GetListingsByCraftMonth(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	req := &insightv1.GetListingsByCraftMonthRequest{}
	if craftID := stringFilter(filters, "craft_id"); craftID != nil {
		req.CraftId = craftID
	}
	if from, ok := filters["from_date"].(time.Time); ok {
		req.FromDate = timestamppb.New(from)
	}
	if to, ok := filters["to_date"].(time.Time); ok {
		req.ToDate = timestamppb.New(to)
	}

	resp, err := in.insight.GetListingsByCraftMonth(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	rows := make([]map[string]any, 0, len(resp.GetRows()))
	for _, r := range resp.GetRows() {
		rows = append(rows, map[string]any{
			"craft_id":      r.GetCraftId(),
			"craft_name":    r.GetCraftName(),
			"month":         r.GetMonth().AsTime().Format("2006-01-02"),
			"listing_count": r.GetListingCount(),
			"artisan_count": r.GetArtisanCount(),
		})
	}
	return rows, nil
}

// RefreshMaterializedViews triggers a manual refresh of the dashboard's
// materialized views and returns when it completed.
func (in *Insight) RefreshMaterializedViews(ctx context.Context) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := in.insight.RefreshMaterializedViews(ctx, &insightv1.RefreshMaterializedViewsRequest{})
	if err != nil {
		return nil, grpcErr(err)
	}
	return map[string]any{"refreshed_at": resp.GetRefreshedAt().AsTime().Format(time.RFC3339)}, nil
}

// GetDyingCrafts returns the crafts with the steepest artisan decline.
func (in *Insight) GetDyingCrafts(ctx context.Context, limit int32) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := in.insight.GetDyingCrafts(ctx, &insightv1.GetDyingCraftsRequest{Limit: limit})
	if err != nil {
		return nil, grpcErr(err)
	}

	rows := make([]map[string]any, 0, len(resp.GetRows()))
	for _, r := range resp.GetRows() {
		rows = append(rows, map[string]any{
			"craft_id":         r.GetCraftId(),
			"craft_name":       r.GetCraftName(),
			"decline_rate":     r.GetDeclineRate(),
			"peak_artisans":    r.GetPeakArtisans(),
			"current_artisans": r.GetCurrentArtisans(),
		})
	}
	return rows, nil
}

// parseStatementBound accepts either a bare date or a full RFC3339 timestamp,
// since handler.StatementService's signature carries the period bounds as
// plain strings with no documented format (POST /statements has no OpenAPI
// entry to pin one).
//
// exclusiveEnd matters only for a bare date: insight-svc's earnings query
// bounds settled_at with `< period_end` (exclusive), so a caller writing
// "2024-01-31" as the end of the period means to include all of the 31st —
// that requires rolling to the start of Feb 1, not stopping at the 31st's
// first instant. An RFC3339 timestamp is taken as the exact instant the
// caller gave, with no adjustment either way.
func parseStatementBound(s string, exclusiveEnd bool) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if exclusiveEnd {
			t = t.AddDate(0, 0, 1)
		}
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, domain.InvalidInput("dates must be RFC3339 or YYYY-MM-DD: " + s)
}

// GenerateStatement kicks off income statement generation for one artisan's
// period and returns the full response -- short_code, download_url and
// verification_url included, not just the id -- since GetStatement (below)
// cannot fetch any of that back afterwards.
func (in *Insight) GenerateStatement(ctx context.Context, artisanID string, start, end string) (map[string]any, error) {
	periodStart, err := parseStatementBound(start, false)
	if err != nil {
		return nil, err
	}
	periodEnd, err := parseStatementBound(end, true)
	if err != nil {
		return nil, err
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := in.insight.GenerateIncomeStatement(ctx, &insightv1.GenerateIncomeStatementRequest{
		ArtisanId:   artisanID,
		PeriodStart: timestamppb.New(periodStart),
		PeriodEnd:   timestamppb.New(periodEnd),
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	return map[string]any{
		"statement_id":     resp.GetStatementId(),
		"short_code":       resp.GetShortCode(),
		"download_url":     resp.GetDownloadUrl(),
		"verification_url": resp.GetVerificationUrl(),
	}, nil
}

// ListIncomeStatements lists an artisan's own past statements, newest first
// -- the earnings-over-time read model /earnings needs, and the only way to
// see a statement generated earlier: see GetStatement's own doc comment on
// why there is no fetch-by-id.
func (in *Insight) ListIncomeStatements(ctx context.Context, artisanID string, limit, offset int32) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := in.insight.GetIncomeStatements(ctx, &insightv1.GetIncomeStatementsRequest{
		ArtisanId: artisanID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]map[string]any, 0, len(resp.GetStatements()))
	for _, s := range resp.GetStatements() {
		out = append(out, map[string]any{
			"statement_id": s.GetStatementId(),
			"period_start": s.GetPeriodStart().AsTime().Format(time.RFC3339),
			"period_end":   s.GetPeriodEnd().AsTime().Format(time.RFC3339),
			"order_count":  s.GetOrderCount(),
			"gross_amount": moneyMap(s.GetGrossAmount()),
			"net_amount":   moneyMap(s.GetNetAmount()),
			"fee_amount":   moneyMap(s.GetFeeAmount()),
			"short_code":   s.GetShortCode(),
			"download_url": s.GetDownloadUrl(),
			"created_at":   s.GetCreatedAt().AsTime().Format(time.RFC3339),
		})
	}
	return out, nil
}

// GetStatement is not wired: insight-svc has no RPC that fetches one
// statement by its bare id — only a list scoped to an artisan
// (GetIncomeStatements, see ListIncomeStatements above) or a lookup by
// public short code (VerifyIncomeStatement). handler.StatementService's
// GetStatement signature has neither an artisan id nor a short code to hand
// it, so there is no RPC this can call.
func (in *Insight) GetStatement(ctx context.Context, statementID string) (map[string]any, error) {
	return nil, domain.Unavailable("fetching a statement by id is not supported: insight-svc only lists by artisan or verifies by short code")
}
