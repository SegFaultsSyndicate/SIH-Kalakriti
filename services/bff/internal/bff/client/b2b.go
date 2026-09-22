// services/bff/internal/bff/client/b2b.go
package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	b2bv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/b2b/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
)

// B2B is bff's outbound client to core-svc's B2BService.
type B2B struct {
	b2b b2bv1.B2BServiceClient
}

// NewB2B builds the client.
func NewB2B(conn grpc.ClientConnInterface) *B2B {
	return &B2B{b2b: b2bv1.NewB2BServiceClient(conn)}
}

// RegisterCompany registers a new company/boutique.
func (b *B2B) RegisterCompany(ctx context.Context, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	name, _ := fields["name"].(string)
	if name == "" {
		return nil, domain.InvalidInput("name: is required")
	}
	typeStr, _ := fields["type"].(string)
	contactName, _ := fields["contact_name"].(string)
	contactPhone, _ := fields["contact_phone"].(string)
	incomeStatementURL, _ := fields["income_statement_url"].(string)

	req := &b2bv1.RegisterCompanyRequest{
		Name:               name,
		Type:               b2bv1.CompanyType(b2bv1.CompanyType_value["COMPANY_TYPE_"+typeStr]),
		ContactName:        contactName,
		ContactPhone:       contactPhone,
		IncomeStatementUrl: incomeStatementURL,
	}

	if v, ok := fields["gstin"].(string); ok && v != "" {
		req.Gstin = &v
	}
	if v, ok := fields["contact_email"].(string); ok && v != "" {
		req.ContactEmail = &v
	}
	if v, ok := fields["website"].(string); ok && v != "" {
		req.Website = &v
	}
	if v, ok := fields["accepts_consignment"].(bool); ok {
		req.AcceptsConsignment = v
	}
	if v, ok := fields["min_order_value_paise"].(float64); ok {
		val := int64(v)
		req.MinOrderValuePaise = &val
	}

	if rRaw, ok := fields["region"].(map[string]any); ok {
		stCode, _ := rRaw["state_code"].(string)
		dist, _ := rRaw["district"].(string)
		req.Region = &commonv1.GeoRegion{StateCode: stCode, District: &dist}
	}

	if craftIDs, err := stringSlice(fields, "preferred_craft_ids"); err == nil && len(craftIDs) > 0 {
		req.PreferredCraftIds = craftIDs
	}

	if locRaw, ok := fields["store_location"].(map[string]any); ok {
		lat, _ := locRaw["latitude"].(float64)
		lng, _ := locRaw["longitude"].(float64)
		addr, _ := locRaw["address"].(string)
		city, _ := locRaw["city"].(string)
		pincode, _ := locRaw["pincode"].(string)
		req.StoreLocation = &b2bv1.StoreLocation{
			Latitude:  lat,
			Longitude: lng,
			Address:   addr,
			City:      &city,
			Pincode:   &pincode,
		}
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.RegisterCompany(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}
	return companyToMap(resp.GetCompany()), nil
}

// GetCompany fetches a company profile.
func (b *B2B) GetCompany(ctx context.Context, companyID string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.GetCompany(ctx, &b2bv1.GetCompanyRequest{CompanyId: companyID})
	if err != nil {
		return nil, grpcErr(err)
	}
	return companyToMap(resp.GetCompany()), nil
}

// GetMyCompany fetches the caller's company profile.
func (b *B2B) GetMyCompany(ctx context.Context, userID string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.GetMyCompany(ctx, &b2bv1.GetMyCompanyRequest{UserId: userID})
	if err != nil {
		return nil, grpcErr(err)
	}
	return companyToMap(resp.GetCompany()), nil
}

// ListCompanies pages through companies.
func (b *B2B) ListCompanies(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	req := &b2bv1.ListCompaniesRequest{}
	if st, ok := filters["state_code"].(string); ok && st != "" {
		req.StateCode = &st
	}
	if vo, ok := filters["verified_only"].(bool); ok {
		req.VerifiedOnly = &vo
	}
	if tStr, ok := filters["type"].(string); ok && tStr != "" {
		tVal := b2bv1.CompanyType(b2bv1.CompanyType_value["COMPANY_TYPE_"+tStr])
		req.Type = &tVal
	}
	if vsStr, ok := filters["verification_status"].(string); ok && vsStr != "" {
		vsVal := b2bv1.VerificationStatus(b2bv1.VerificationStatus_value["VERIFICATION_STATUS_"+vsStr])
		req.VerificationStatus = &vsVal
	}
	if ps, ok := filters["page_size"].(float64); ok {
		req.Page = &commonv1.PageRequest{PageSize: int32(ps)}
	}
	if pt, ok := filters["page_token"].(string); ok && pt != "" {
		if req.Page == nil {
			req.Page = &commonv1.PageRequest{PageSize: 20}
		}
		req.Page.PageToken = pt
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.ListCompanies(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]map[string]any, len(resp.GetCompanies()))
	for i, c := range resp.GetCompanies() {
		out[i] = companyToMap(c)
	}
	return out, nil
}

// VerifyCompany sets admin approval or rejection.
func (b *B2B) VerifyCompany(ctx context.Context, companyID, decision, rejectionReason, adminUserID string) (map[string]any, error) {
	req := &b2bv1.VerifyCompanyRequest{
		CompanyId:   companyID,
		Decision:    b2bv1.VerificationStatus(b2bv1.VerificationStatus_value["VERIFICATION_STATUS_"+decision]),
		AdminUserId: adminUserID,
	}
	if rejectionReason != "" {
		req.RejectionReason = &rejectionReason
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.VerifyCompany(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}
	return companyToMap(resp.GetCompany()), nil
}

// GetPlatformCommissionStats gets platform totals.
func (b *B2B) GetPlatformCommissionStats(ctx context.Context) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.GetPlatformCommissionStats(ctx, &b2bv1.GetPlatformCommissionStatsRequest{})
	if err != nil {
		return nil, grpcErr(err)
	}
	return map[string]any{
		"total_companies":        resp.GetTotalCompanies(),
		"pending_verifications":  resp.GetPendingVerifications(),
		"verified_companies":     resp.GetVerifiedCompanies(),
		"total_sales_paise":      resp.GetTotalSalesPaise(),
		"total_commission_paise": resp.GetTotalCommissionPaise(),
	}, nil
}

// RecordCompanySale deducts commission and credits platform ledger.
func (b *B2B) RecordCompanySale(ctx context.Context, companyID, orderID, productName, buyerID string, grossAmountPaise int64) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.RecordCompanySale(ctx, &b2bv1.RecordCompanySaleRequest{
		CompanyId:        companyID,
		OrderId:          orderID,
		ProductName:      productName,
		BuyerId:          buyerID,
		GrossAmountPaise: grossAmountPaise,
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	s := resp.GetSettlement()
	return map[string]any{
		"id":                  s.GetId(),
		"company_id":          s.GetCompanyId(),
		"order_id":            s.GetOrderId(),
		"product_name":        s.GetProductName(),
		"buyer_id":            s.GetBuyerId(),
		"gross_amount_paise":  s.GetGrossAmountPaise(),
		"commission_rate_bps": s.GetCommissionRateBps(),
		"platform_fee_paise":  s.GetPlatformFeePaise(),
		"net_payout_paise":    s.GetNetPayoutPaise(),
		"settled_at":          s.GetSettledAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// ListCompanySales lists sales settlements for a company.
func (b *B2B) ListCompanySales(ctx context.Context, companyID string, pageSize int32, pageToken string) ([]map[string]any, string, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.ListCompanySales(ctx, &b2bv1.ListCompanySalesRequest{
		CompanyId: companyID,
		Page: &commonv1.PageRequest{
			PageSize:  pageSize,
			PageToken: pageToken,
		},
	})
	if err != nil {
		return nil, "", grpcErr(err)
	}

	out := make([]map[string]any, len(resp.GetSales()))
	for i, s := range resp.GetSales() {
		out[i] = map[string]any{
			"id":                  s.GetId(),
			"company_id":          s.GetCompanyId(),
			"order_id":            s.GetOrderId(),
			"product_name":        s.GetProductName(),
			"buyer_id":            s.GetBuyerId(),
			"gross_amount_paise":  s.GetGrossAmountPaise(),
			"commission_rate_bps": s.GetCommissionRateBps(),
			"platform_fee_paise":  s.GetPlatformFeePaise(),
			"net_payout_paise":    s.GetNetPayoutPaise(),
			"settled_at":          s.GetSettledAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	nextPage := ""
	if resp.GetPage() != nil {
		nextPage = resp.GetPage().GetNextPageToken()
	}
	return out, nextPage, nil
}

// ExpressInterest records interest in an artisan.
func (b *B2B) ExpressInterest(ctx context.Context, companyID, artisanID, idempotencyKey, message string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.ExpressInterest(ctx, &b2bv1.ExpressInterestRequest{
		CompanyId: companyID,
		ArtisanId: artisanID,
		Message:   message,
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	i := resp.GetInterest()
	return map[string]any{
		"id":           i.GetId(),
		"company_id":   i.GetCompanyId(),
		"artisan_id":   i.GetArtisanId(),
		"message":      i.GetMessage(),
		"status":       i.GetStatus().String(),
		"company_name": i.GetCompanyName(),
		"company_type": i.GetCompanyType().String(),
		"created_at":   i.GetCreatedAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// RespondToInterest accepts or declines interest.
func (b *B2B) RespondToInterest(ctx context.Context, interestID, artisanID, decision string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	dVal := b2bv1.InterestStatus_INTEREST_STATUS_ACCEPTED
	if decision == "DECLINED" {
		dVal = b2bv1.InterestStatus_INTEREST_STATUS_DECLINED
	}

	resp, err := b.b2b.RespondToInterest(ctx, &b2bv1.RespondToInterestRequest{
		InterestId: interestID,
		ArtisanId:  artisanID,
		Decision:   dVal,
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	i := resp.GetInterest()
	return map[string]any{
		"id":         i.GetId(),
		"status":     i.GetStatus().String(),
		"company_id": i.GetCompanyId(),
		"artisan_id": i.GetArtisanId(),
	}, nil
}

// ListArtisanLeads lists leads for an artisan.
func (b *B2B) ListArtisanLeads(ctx context.Context, artisanID string, filters map[string]any) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	req := &b2bv1.ListArtisanLeadsRequest{ArtisanId: artisanID}
	if st, ok := filters["status"].(string); ok && st != "" {
		stVal := b2bv1.InterestStatus(b2bv1.InterestStatus_value["INTEREST_STATUS_"+st])
		req.Status = &stVal
	}

	resp, err := b.b2b.ListArtisanLeads(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]map[string]any, len(resp.GetLeads()))
	for idx, i := range resp.GetLeads() {
		out[idx] = map[string]any{
			"id":           i.GetId(),
			"company_id":   i.GetCompanyId(),
			"artisan_id":   i.GetArtisanId(),
			"message":      i.GetMessage(),
			"status":       i.GetStatus().String(),
			"company_name": i.GetCompanyName(),
			"company_type": i.GetCompanyType().String(),
			"created_at":   i.GetCreatedAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return out, nil
}

// CreatePartnership forms a supply agreement.
func (b *B2B) CreatePartnership(ctx context.Context, idempotencyKey string, fields map[string]any) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	cid, _ := fields["company_id"].(string)
	aid, _ := fields["artisan_id"].(string)
	crid, _ := fields["craft_id"].(string)
	terms, _ := fields["terms"].(string)

	resp, err := b.b2b.CreatePartnership(ctx, &b2bv1.CreatePartnershipRequest{
		CompanyId: cid,
		ArtisanId: aid,
		CraftId:   crid,
		Terms:     terms,
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	p := resp.GetPartnership()
	return map[string]any{
		"id":           p.GetId(),
		"company_id":   p.GetCompanyId(),
		"artisan_id":   p.GetArtisanId(),
		"craft_id":     p.GetCraftId(),
		"terms":        p.GetTerms(),
		"active":       p.GetActive(),
		"company_name": p.GetCompanyName(),
		"artisan_name": p.GetArtisanName(),
	}, nil
}

// ListPartnerships lists partnerships.
func (b *B2B) ListPartnerships(ctx context.Context, filters map[string]any) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	req := &b2bv1.ListPartnershipsRequest{}
	if aid, ok := filters["artisan_id"].(string); ok && aid != "" {
		req.ArtisanId = &aid
	}
	if cid, ok := filters["company_id"].(string); ok && cid != "" {
		req.CompanyId = &cid
	}

	resp, err := b.b2b.ListPartnerships(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]map[string]any, len(resp.GetPartnerships()))
	for i, p := range resp.GetPartnerships() {
		out[i] = map[string]any{
			"id":           p.GetId(),
			"company_id":   p.GetCompanyId(),
			"artisan_id":   p.GetArtisanId(),
			"craft_id":     p.GetCraftId(),
			"terms":        p.GetTerms(),
			"active":       p.GetActive(),
			"company_name": p.GetCompanyName(),
			"artisan_name": p.GetArtisanName(),
		}
	}
	return out, nil
}

// ListBoutiqueMatches lists recommended boutiques.
func (b *B2B) ListBoutiqueMatches(ctx context.Context, artisanID string, limit int32) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := b.b2b.ListBoutiqueMatches(ctx, &b2bv1.ListBoutiqueMatchesRequest{
		ArtisanId: artisanID,
		Limit:     limit,
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]map[string]any, len(resp.GetMatches()))
	for i, m := range resp.GetMatches() {
		loc := m.GetBoutiqueLocation()
		out[i] = map[string]any{
			"id":                 m.GetId(),
			"company_id":         m.GetCompanyId(),
			"artisan_id":         m.GetArtisanId(),
			"match_score":        m.GetMatchScore(),
			"status":             m.GetStatus().String(),
			"boutique_name":      m.GetBoutiqueName(),
			"overlapping_crafts": m.GetOverlappingCrafts(),
			"boutique_location": map[string]any{
				"latitude":  loc.GetLatitude(),
				"longitude": loc.GetLongitude(),
				"address":   loc.GetAddress(),
				"city":      loc.GetCity(),
			},
		}
	}
	return out, nil
}

// ContactBoutique initiates outreach.
func (b *B2B) ContactBoutique(ctx context.Context, artisanID, companyID, idempotencyKey, message string) (map[string]any, error) {
	return b.ExpressInterest(ctx, companyID, artisanID, idempotencyKey, message)
}

// ListNearbyBoutiques finds boutiques within a radius.
func (b *B2B) ListNearbyBoutiques(ctx context.Context, lat, lng, radiusKm float64, craftID string, limit int32) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	req := &b2bv1.ListNearbyBoutiquesRequest{
		Latitude:  lat,
		Longitude: lng,
		RadiusKm:  radiusKm,
		Limit:     limit,
	}
	if craftID != "" {
		req.CraftId = &craftID
	}

	resp, err := b.b2b.ListNearbyBoutiques(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]map[string]any, len(resp.GetBoutiques()))
	for i, c := range resp.GetBoutiques() {
		out[i] = companyToMap(c)
	}
	return out, nil
}

func companyToMap(c *b2bv1.Company) map[string]any {
	if c == nil {
		return nil
	}
	m := map[string]any{
		"id":                      c.GetId(),
		"user_id":                 c.GetUserId(),
		"name":                    c.GetName(),
		"type":                    c.GetType().String(),
		"contact_name":            c.GetContactName(),
		"contact_phone":           c.GetContactPhone(),
		"verified":                c.GetVerified(),
		"verification_status":     c.GetVerificationStatus().String(),
		"income_statement_url":    c.GetIncomeStatementUrl(),
		"commission_rate_bps":     c.GetCommissionRateBps(),
		"total_sales_paise":       c.GetTotalSalesPaise(),
		"commission_earned_paise": c.GetCommissionEarnedPaise(),
		"accepts_consignment":     c.GetAcceptsConsignment(),
		"min_order_value_paise":   c.GetMinOrderValuePaise(),
		"preferred_craft_ids":     c.GetPreferredCraftIds(),
	}
	if c.Gstin != nil {
		m["gstin"] = *c.Gstin
	}
	if c.ContactEmail != nil {
		m["contact_email"] = *c.ContactEmail
	}
	if c.Website != nil {
		m["website"] = *c.Website
	}
	if c.RejectionReason != nil {
		m["rejection_reason"] = *c.RejectionReason
	}
	if c.VerifiedBy != nil {
		m["verified_by"] = *c.VerifiedBy
	}
	if c.VerifiedAt != nil {
		m["verified_at"] = c.VerifiedAt.AsTime().Format("2006-01-02T15:04:05Z07:00")
	}
	if r := c.GetRegion(); r != nil {
		m["region"] = map[string]any{
			"state_code": r.GetStateCode(),
			"district":   r.GetDistrict(),
		}
	}
	if loc := c.GetStoreLocation(); loc != nil {
		m["store_location"] = map[string]any{
			"latitude":  loc.GetLatitude(),
			"longitude": loc.GetLongitude(),
			"address":   loc.GetAddress(),
			"city":      loc.GetCity(),
			"pincode":   loc.GetPincode(),
		}
	}
	return m
}
