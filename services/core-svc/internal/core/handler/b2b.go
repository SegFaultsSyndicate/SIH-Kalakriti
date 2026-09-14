// services/core-svc/internal/core/handler/b2b.go

package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	b2bv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/b2b/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// B2B implements b2b.v1.B2BService.
type B2B struct {
	b2bv1.UnimplementedB2BServiceServer
	svc *service.B2B
}

// NewB2B builds the B2B handler.
func NewB2B(svc *service.B2B) *B2B {
	return &B2B{svc: svc}
}

// RegisterCompany onboard a business entity.
func (h *B2B) RegisterCompany(ctx context.Context, req *b2bv1.RegisterCompanyRequest) (*b2bv1.RegisterCompanyResponse, error) {
	in := domain.RegisterCompanyInput{
		Name:               req.GetName(),
		Type:               domain.CompanyType(companyTypeFromProto(req.GetType())),
		GSTIN:              req.Gstin,
		ContactName:        req.GetContactName(),
		ContactPhone:       req.GetContactPhone(),
		ContactEmail:       req.ContactEmail,
		Website:            req.Website,
		IncomeStatementURL: req.GetIncomeStatementUrl(),
		AcceptsConsignment: req.GetAcceptsConsignment(),
		MinOrderValuePaise: req.MinOrderValuePaise,
	}

	if r := req.GetRegion(); r != nil {
		in.Region = domain.Region{
			StateCode: r.GetStateCode(),
			District:  r.District,
		}
	}

	for _, idStr := range req.GetPreferredCraftIds() {
		if id, err := uuid.Parse(idStr); err == nil {
			in.PreferredCraftIDs = append(in.PreferredCraftIDs, id)
		}
	}

	if loc := req.GetStoreLocation(); loc != nil {
		lat := loc.GetLatitude()
		lng := loc.GetLongitude()
		addr := loc.GetAddress()
		in.StoreLocation = domain.StoreLocation{
			Latitude:  &lat,
			Longitude: &lng,
			Address:   &addr,
			City:      loc.City,
			Pincode:   loc.Pincode,
		}
	}

	company, err := h.svc.RegisterCompany(ctx, in)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.RegisterCompanyResponse{
		Company: companyToProto(company),
	}, nil
}

// GetCompany fetches one company by id.
func (h *B2B) GetCompany(ctx context.Context, req *b2bv1.GetCompanyRequest) (*b2bv1.GetCompanyResponse, error) {
	id, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	company, err := h.svc.GetCompany(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.GetCompanyResponse{
		Company: companyToProto(company),
	}, nil
}

// GetMyCompany fetches the caller's company profile.
func (h *B2B) GetMyCompany(ctx context.Context, _ *b2bv1.GetMyCompanyRequest) (*b2bv1.GetMyCompanyResponse, error) {
	company, err := h.svc.GetMyCompany(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.GetMyCompanyResponse{
		Company: companyToProto(company),
	}, nil
}

// ListCompanies pages through companies.
func (h *B2B) ListCompanies(ctx context.Context, req *b2bv1.ListCompaniesRequest) (*b2bv1.ListCompaniesResponse, error) {
	page, err := pageFromProto(req.GetPage())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	filter := domain.CompanyFilter{
		StateCode:    req.StateCode,
		VerifiedOnly: req.GetVerifiedOnly(),
		Page:         page,
	}

	if req.Type != nil {
		ct := domain.CompanyType(companyTypeFromProto(req.GetType()))
		filter.Type = &ct
	}

	if req.VerificationStatus != nil {
		vs := verificationStatusFromProto(req.GetVerificationStatus())
		filter.VerificationStatus = &vs
	}

	companies, err := h.svc.ListCompanies(ctx, filter)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	pbCompanies := make([]*b2bv1.Company, len(companies))
	for i, c := range companies {
		pbCompanies[i] = companyToProto(c)
	}

	var lastID string
	if len(companies) > 0 {
		lastID = companies[len(companies)-1].ID.String()
	}

	return &b2bv1.ListCompaniesResponse{
		Companies: pbCompanies,
		Page:      nextPage(lastID, len(companies), req.GetPage().GetPageSize()),
	}, nil
}

// VerifyCompany processes an admin's verification decision.
func (h *B2B) VerifyCompany(ctx context.Context, req *b2bv1.VerifyCompanyRequest) (*b2bv1.VerifyCompanyResponse, error) {
	cid, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	in := domain.VerifyCompanyInput{
		CompanyID:       cid,
		Decision:        verificationStatusFromProto(req.GetDecision()),
		RejectionReason: req.RejectionReason,
		AdminUserID:     req.GetAdminUserId(),
	}

	company, err := h.svc.VerifyCompany(ctx, in)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.VerifyCompanyResponse{
		Company: companyToProto(company),
	}, nil
}

// GetPlatformCommissionStats returns platform financial stats.
func (h *B2B) GetPlatformCommissionStats(ctx context.Context, _ *b2bv1.GetPlatformCommissionStatsRequest) (*b2bv1.GetPlatformCommissionStatsResponse, error) {
	stats, err := h.svc.GetPlatformCommissionStats(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.GetPlatformCommissionStatsResponse{
		TotalCompanies:       stats.TotalCompanies,
		PendingVerifications: stats.PendingVerifications,
		VerifiedCompanies:    stats.VerifiedCompanies,
		TotalSalesPaise:      stats.TotalSalesPaise,
		TotalCommissionPaise: stats.TotalCommissionPaise,
	}, nil
}

// RecordCompanySale settles a sale with platform fee deduction.
func (h *B2B) RecordCompanySale(ctx context.Context, req *b2bv1.RecordCompanySaleRequest) (*b2bv1.RecordCompanySaleResponse, error) {
	cid, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	in := domain.RecordCompanySaleInput{
		CompanyID:        cid,
		OrderID:          req.GetOrderId(),
		ProductName:      req.GetProductName(),
		BuyerID:          req.GetBuyerId(),
		GrossAmountPaise: req.GetGrossAmountPaise(),
	}

	settlement, err := h.svc.RecordCompanySale(ctx, in)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.RecordCompanySaleResponse{
		Settlement: &b2bv1.CompanySaleSettlement{
			Id:                settlement.ID.String(),
			CompanyId:         settlement.CompanyID.String(),
			OrderId:           settlement.OrderID,
			ProductName:       settlement.ProductName,
			BuyerId:           settlement.BuyerID,
			GrossAmountPaise:  settlement.GrossAmountPaise,
			CommissionRateBps: settlement.CommissionRateBps,
			PlatformFeePaise:  settlement.PlatformFeePaise,
			NetPayoutPaise:    settlement.NetPayoutPaise,
			SettledAt:         timestamppb.New(settlement.SettledAt),
		},
	}, nil
}

// ListCompanySales lists sales for a company.
func (h *B2B) ListCompanySales(ctx context.Context, req *b2bv1.ListCompanySalesRequest) (*b2bv1.ListCompanySalesResponse, error) {
	cid, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	page, err := pageFromProto(req.GetPage())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	sales, err := h.svc.ListCompanySales(ctx, cid, page)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	pbSales := make([]*b2bv1.CompanySaleSettlement, len(sales))
	for i, s := range sales {
		pbSales[i] = &b2bv1.CompanySaleSettlement{
			Id:                s.ID.String(),
			CompanyId:         s.CompanyID.String(),
			OrderId:           s.OrderID,
			ProductName:       s.ProductName,
			BuyerId:           s.BuyerID,
			GrossAmountPaise:  s.GrossAmountPaise,
			CommissionRateBps: s.CommissionRateBps,
			PlatformFeePaise:  s.PlatformFeePaise,
			NetPayoutPaise:    s.NetPayoutPaise,
			SettledAt:         timestamppb.New(s.SettledAt),
		}
	}

	var lastID string
	if len(sales) > 0 {
		lastID = sales[len(sales)-1].ID.String()
	}

	return &b2bv1.ListCompanySalesResponse{
		Sales: pbSales,
		Page:  nextPage(lastID, len(sales), req.GetPage().GetPageSize()),
	}, nil
}

// ExpressInterest records a company's outreach to an artisan.
func (h *B2B) ExpressInterest(ctx context.Context, req *b2bv1.ExpressInterestRequest) (*b2bv1.ExpressInterestResponse, error) {
	cid, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}
	aid, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	interest, err := h.svc.ExpressInterest(ctx, domain.ExpressInterestInput{
		CompanyID: cid,
		ArtisanID: aid,
		Message:   req.GetMessage(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.ExpressInterestResponse{
		Interest: companyInterestToProto(interest),
	}, nil
}

// RespondToInterest updates an interest expression.
func (h *B2B) RespondToInterest(ctx context.Context, req *b2bv1.RespondToInterestRequest) (*b2bv1.RespondToInterestResponse, error) {
	iid, err := uuid.Parse(req.GetInterestId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}
	aid, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	decision := domain.InterestStatusAccepted
	if req.GetDecision() == b2bv1.InterestStatus_INTEREST_STATUS_DECLINED {
		decision = domain.InterestStatusDeclined
	}

	interest, err := h.svc.RespondToInterest(ctx, iid, aid, decision)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.RespondToInterestResponse{
		Interest: companyInterestToProto(interest),
	}, nil
}

// ListArtisanLeads lists inbound leads for an artisan.
func (h *B2B) ListArtisanLeads(ctx context.Context, req *b2bv1.ListArtisanLeadsRequest) (*b2bv1.ListArtisanLeadsResponse, error) {
	aid, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	var status *domain.InterestStatus
	if req.Status != nil {
		s := domain.InterestStatus(req.GetStatus().String())
		status = &s
	}

	page, err := pageFromProto(req.GetPage())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	leads, err := h.svc.ListArtisanLeads(ctx, aid, status, page)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	pbLeads := make([]*b2bv1.CompanyInterest, len(leads))
	for i, l := range leads {
		pbLeads[i] = companyInterestToProto(l)
	}

	var lastID string
	if len(leads) > 0 {
		lastID = leads[len(leads)-1].ID.String()
	}

	return &b2bv1.ListArtisanLeadsResponse{
		Leads: pbLeads,
		Page:  nextPage(lastID, len(leads), req.GetPage().GetPageSize()),
	}, nil
}

// CreatePartnership creates a partnership.
func (h *B2B) CreatePartnership(ctx context.Context, req *b2bv1.CreatePartnershipRequest) (*b2bv1.CreatePartnershipResponse, error) {
	cid, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}
	aid, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}
	crid, err := uuid.Parse(req.GetCraftId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	var renewalDate *timestamppb.Timestamp
	if req.RenewalDate != nil {
		renewalDate = req.RenewalDate
	}

	p, err := h.svc.CreatePartnership(ctx, domain.CreatePartnershipInput{
		CompanyID:   cid,
		ArtisanID:   aid,
		CraftID:     crid,
		Terms:       req.GetTerms(),
		RenewalDate: timestampToTime(renewalDate),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.CreatePartnershipResponse{
		Partnership: supplyPartnershipToProto(p),
	}, nil
}

// ListPartnerships lists partnerships for artisan or company.
func (h *B2B) ListPartnerships(ctx context.Context, req *b2bv1.ListPartnershipsRequest) (*b2bv1.ListPartnershipsResponse, error) {
	page, err := pageFromProto(req.GetPage())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	activeOnly := req.GetActiveOnly()

	var partnerships []domain.SupplyPartnership

	if req.ArtisanId != nil {
		aid, pErr := uuid.Parse(*req.ArtisanId)
		if pErr != nil {
			return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
		}
		partnerships, err = h.svc.ListPartnershipsByArtisan(ctx, aid, activeOnly, page)
	} else if req.CompanyId != nil {
		cid, pErr := uuid.Parse(*req.CompanyId)
		if pErr != nil {
			return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
		}
		partnerships, err = h.svc.ListPartnershipsByCompany(ctx, cid, activeOnly, page)
	} else {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	pbPartnerships := make([]*b2bv1.SupplyPartnership, len(partnerships))
	for i, p := range partnerships {
		pbPartnerships[i] = supplyPartnershipToProto(p)
	}

	var lastID string
	if len(partnerships) > 0 {
		lastID = partnerships[len(partnerships)-1].ID.String()
	}

	return &b2bv1.ListPartnershipsResponse{
		Partnerships: pbPartnerships,
		Page:         nextPage(lastID, len(partnerships), req.GetPage().GetPageSize()),
	}, nil
}

// ListBoutiqueMatches returns recommendations for an artisan.
func (h *B2B) ListBoutiqueMatches(ctx context.Context, req *b2bv1.ListBoutiqueMatchesRequest) (*b2bv1.ListBoutiqueMatchesResponse, error) {
	aid, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	matches, err := h.svc.ListBoutiqueMatches(ctx, aid, req.GetLimit())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	pbMatches := make([]*b2bv1.BoutiqueMatch, len(matches))
	for i, m := range matches {
		pbMatches[i] = &b2bv1.BoutiqueMatch{
			Id:           m.ID.String(),
			CompanyId:    m.CompanyID.String(),
			ArtisanId:    m.ArtisanID.String(),
			MatchScore:   m.MatchScore,
			Status:       b2bv1.MatchStatus(b2bv1.MatchStatus_value["MATCH_STATUS_"+string(m.Status)]),
			BoutiqueName: m.BoutiqueName,
			BoutiqueLocation: &b2bv1.StoreLocation{
				Latitude:  derefFloat(m.BoutiqueLocation.Latitude),
				Longitude: derefFloat(m.BoutiqueLocation.Longitude),
				Address:   derefString(m.BoutiqueLocation.Address),
				City:      m.BoutiqueLocation.City,
			},
			OverlappingCrafts: m.OverlappingCrafts,
			CreatedAt:         timestamppb.New(m.CreatedAt),
		}
	}

	return &b2bv1.ListBoutiqueMatchesResponse{
		Matches: pbMatches,
	}, nil
}

// ContactBoutique initiates artisan outreach to a boutique.
func (h *B2B) ContactBoutique(ctx context.Context, req *b2bv1.ContactBoutiqueRequest) (*b2bv1.ContactBoutiqueResponse, error) {
	cid, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}
	aid, err := uuid.Parse(req.GetArtisanId())
	if err != nil {
		return nil, pkgdomain.GRPCError(pkgdomain.ErrInvalidInput)
	}

	interest, err := h.svc.ExpressInterest(ctx, domain.ExpressInterestInput{
		CompanyID: cid,
		ArtisanID: aid,
		Message:   req.GetMessage(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	return &b2bv1.ContactBoutiqueResponse{
		Interest: companyInterestToProto(interest),
	}, nil
}

// ListNearbyBoutiques returns boutiques filtered by location radius.
func (h *B2B) ListNearbyBoutiques(ctx context.Context, req *b2bv1.ListNearbyBoutiquesRequest) (*b2bv1.ListNearbyBoutiquesResponse, error) {
	var craftID *uuid.UUID
	if req.CraftId != nil {
		if id, err := uuid.Parse(*req.CraftId); err == nil {
			craftID = &id
		}
	}

	boutiques, err := h.svc.ListNearbyBoutiques(ctx, req.GetLatitude(), req.GetLongitude(), req.GetRadiusKm(), craftID, req.GetLimit())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	pbBoutiques := make([]*b2bv1.Company, len(boutiques))
	for i, b := range boutiques {
		pbBoutiques[i] = companyToProto(b)
	}

	return &b2bv1.ListNearbyBoutiquesResponse{
		Boutiques: pbBoutiques,
	}, nil
}

// Helpers

func companyTypeFromProto(t b2bv1.CompanyType) string {
	switch t {
	case b2bv1.CompanyType_COMPANY_TYPE_RETAILER:
		return "RETAILER"
	case b2bv1.CompanyType_COMPANY_TYPE_BOUTIQUE:
		return "BOUTIQUE"
	case b2bv1.CompanyType_COMPANY_TYPE_EXPORTER:
		return "EXPORTER"
	case b2bv1.CompanyType_COMPANY_TYPE_INSTITUTION:
		return "INSTITUTION"
	default:
		return "RETAILER"
	}
}

func companyTypeToProto(t domain.CompanyType) b2bv1.CompanyType {
	switch t {
	case domain.CompanyTypeRetailer:
		return b2bv1.CompanyType_COMPANY_TYPE_RETAILER
	case domain.CompanyTypeBoutique:
		return b2bv1.CompanyType_COMPANY_TYPE_BOUTIQUE
	case domain.CompanyTypeExporter:
		return b2bv1.CompanyType_COMPANY_TYPE_EXPORTER
	case domain.CompanyTypeInstitution:
		return b2bv1.CompanyType_COMPANY_TYPE_INSTITUTION
	default:
		return b2bv1.CompanyType_COMPANY_TYPE_UNSPECIFIED
	}
}

func verificationStatusFromProto(s b2bv1.VerificationStatus) domain.VerificationStatus {
	switch s {
	case b2bv1.VerificationStatus_VERIFICATION_STATUS_VERIFIED:
		return domain.VerificationStatusVerified
	case b2bv1.VerificationStatus_VERIFICATION_STATUS_REJECTED:
		return domain.VerificationStatusRejected
	default:
		return domain.VerificationStatusPending
	}
}

func verificationStatusToProto(s domain.VerificationStatus) b2bv1.VerificationStatus {
	switch s {
	case domain.VerificationStatusVerified:
		return b2bv1.VerificationStatus_VERIFICATION_STATUS_VERIFIED
	case domain.VerificationStatusRejected:
		return b2bv1.VerificationStatus_VERIFICATION_STATUS_REJECTED
	default:
		return b2bv1.VerificationStatus_VERIFICATION_STATUS_PENDING
	}
}

func companyToProto(c domain.Company) *b2bv1.Company {
	craftIDs := make([]string, len(c.PreferredCraftIDs))
	for i, id := range c.PreferredCraftIDs {
		craftIDs[i] = id.String()
	}

	var verifiedAt *timestamppb.Timestamp
	if c.VerifiedAt != nil {
		verifiedAt = timestamppb.New(*c.VerifiedAt)
	}

	return &b2bv1.Company{
		Id:                 c.ID.String(),
		UserId:             c.UserID,
		Name:               c.Name,
		Type:               companyTypeToProto(c.Type),
		Gstin:              c.GSTIN,
		ContactName:        c.ContactName,
		ContactPhone:       c.ContactPhone,
		ContactEmail:       c.ContactEmail,
		Website:            c.Website,
		Region: &commonv1.GeoRegion{
			StateCode: c.StateCode,
			District:  c.District,
		},
		Verified:              c.Verified,
		VerifiedBy:            c.VerifiedBy,
		VerifiedAt:            verifiedAt,
		RejectionReason:       c.RejectionReason,
		IncomeStatementUrl:    c.IncomeStatementURL,
		CommissionRateBps:     c.CommissionRateBps,
		TotalSalesPaise:       c.TotalSalesPaise,
		CommissionEarnedPaise: c.CommissionEarnedPaise,
		AcceptsConsignment:    c.AcceptsConsignment,
		MinOrderValuePaise:    c.MinOrderValuePaise,
		PreferredCraftIds:     craftIDs,
		StoreLocation: &b2bv1.StoreLocation{
			Latitude:  derefFloat(c.StoreLocation.Latitude),
			Longitude: derefFloat(c.StoreLocation.Longitude),
			Address:   derefString(c.StoreLocation.Address),
			City:      c.StoreLocation.City,
			Pincode:   c.StoreLocation.Pincode,
		},
		VerificationStatus: verificationStatusToProto(c.VerificationStatus),
	}
}

func companyInterestToProto(i domain.CompanyInterest) *b2bv1.CompanyInterest {
	var respAt *timestamppb.Timestamp
	if i.RespondedAt != nil {
		respAt = timestamppb.New(*i.RespondedAt)
	}

	status := b2bv1.InterestStatus_INTEREST_STATUS_PENDING
	switch i.Status {
	case domain.InterestStatusAccepted:
		status = b2bv1.InterestStatus_INTEREST_STATUS_ACCEPTED
	case domain.InterestStatusDeclined:
		status = b2bv1.InterestStatus_INTEREST_STATUS_DECLINED
	}

	return &b2bv1.CompanyInterest{
		Id:          i.ID.String(),
		CompanyId:   i.CompanyID.String(),
		ArtisanId:   i.ArtisanID.String(),
		Message:     i.Message,
		Status:      status,
		RespondedAt: respAt,
		CompanyName: i.CompanyName,
		CompanyType: companyTypeToProto(i.CompanyType),
		CreatedAt:   timestamppb.New(i.CreatedAt),
	}
}

func supplyPartnershipToProto(p domain.SupplyPartnership) *b2bv1.SupplyPartnership {
	var renewalDate *timestamppb.Timestamp
	if p.RenewalDate != nil {
		renewalDate = timestamppb.New(*p.RenewalDate)
	}

	return &b2bv1.SupplyPartnership{
		Id:          p.ID.String(),
		CompanyId:   p.CompanyID.String(),
		ArtisanId:   p.ArtisanID.String(),
		CraftId:     p.CraftID.String(),
		Terms:       p.Terms,
		RenewalDate: renewalDate,
		Active:      p.Active,
		CompanyName: p.CompanyName,
		ArtisanName: p.ArtisanName,
	}
}

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func timestampToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
