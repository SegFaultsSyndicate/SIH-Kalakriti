// services/core-svc/internal/core/repo/b2b.go

package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// CreateCompany inserts a new company in PENDING state.
func (t *Tx) CreateCompany(ctx context.Context, id uuid.UUID, in domain.RegisterCompanyInput, bps int32) (domain.Company, error) {
	row, err := t.q.CreateCompany(ctx, db.CreateCompanyParams{
		ID:                 id,
		UserID:             in.UserID,
		Name:               in.Name,
		CompanyType:        db.CompanyType(in.Type),
		Gstin:              in.GSTIN,
		ContactName:        in.ContactName,
		ContactPhone:       in.ContactPhone,
		ContactEmail:       in.ContactEmail,
		Website:            in.Website,
		StateCode:          in.Region.StateCode,
		District:           in.Region.District,
		VerificationStatus: db.VerificationStatusPENDING,
		Verified:           false,
		IncomeStatementUrl: in.IncomeStatementURL,
		CommissionRateBps:  bps,
		AcceptsConsignment: in.AcceptsConsignment,
		MinOrderValuePaise: in.MinOrderValuePaise,
		PreferredCraftIds:  in.PreferredCraftIDs,
		StoreLatitude:      in.StoreLocation.Latitude,
		StoreLongitude:     in.StoreLocation.Longitude,
		StoreAddress:       in.StoreLocation.Address,
		StoreCity:          in.StoreLocation.City,
		StorePincode:       in.StoreLocation.Pincode,
	})
	if err != nil {
		return domain.Company{}, translate(err, "company")
	}
	return companyFromRow(row), nil
}

// VerifyCompany updates verification status and commission parameters.
func (t *Tx) VerifyCompany(ctx context.Context, in domain.VerifyCompanyInput, bps int32) (domain.Company, error) {
	isVerified := in.Decision == domain.VerificationStatusVerified
	adminID := in.AdminUserID
	row, err := t.q.VerifyCompany(ctx, db.VerifyCompanyParams{
		VerificationStatus: db.VerificationStatus(in.Decision),
		Verified:           isVerified,
		VerifiedBy:         &adminID,
		RejectionReason:    in.RejectionReason,
		CommissionRateBps:  bps,
		ID:                 in.CompanyID,
	})
	if err != nil {
		return domain.Company{}, translate(err, "company verification")
	}
	return companyFromRow(row), nil
}

// RecordCompanySaleSettlement writes a settlement record into the ledger.
func (t *Tx) RecordCompanySaleSettlement(ctx context.Context, id uuid.UUID, in domain.RecordCompanySaleInput, bps int32, feePaise, netPaise int64) (domain.CompanySaleSettlement, error) {
	row, err := t.q.RecordCompanySaleSettlement(ctx, db.RecordCompanySaleSettlementParams{
		ID:                id,
		CompanyID:         in.CompanyID,
		OrderID:           in.OrderID,
		ProductName:       in.ProductName,
		BuyerID:           in.BuyerID,
		GrossAmountPaise:  in.GrossAmountPaise,
		CommissionRateBps: bps,
		PlatformFeePaise:  feePaise,
		NetPayoutPaise:    netPaise,
	})
	if err != nil {
		return domain.CompanySaleSettlement{}, translate(err, "company sale settlement")
	}
	return saleSettlementFromRow(row), nil
}

// IncrementCompanySales atomically adds to the company's gross and platform fee counters.
func (t *Tx) IncrementCompanySales(ctx context.Context, companyID uuid.UUID, grossPaise, feePaise int64) error {
	_, err := t.q.IncrementCompanySales(ctx, db.IncrementCompanySalesParams{
		ID:               companyID,
		GrossAmountPaise: grossPaise,
		PlatformFeePaise: feePaise,
	})
	if err != nil {
		return translate(err, "company sales totals")
	}
	return nil
}

// CreateCompanyInterest writes an interest expression.
func (t *Tx) CreateCompanyInterest(ctx context.Context, id uuid.UUID, in domain.ExpressInterestInput) (domain.CompanyInterest, error) {
	row, err := t.q.CreateCompanyInterest(ctx, db.CreateCompanyInterestParams{
		ID:        id,
		CompanyID: in.CompanyID,
		ArtisanID: in.ArtisanID,
		Message:   in.Message,
	})
	if err != nil {
		return domain.CompanyInterest{}, translate(err, "company interest")
	}
	return companyInterestFromRow(row), nil
}

// RespondToInterest updates an interest expression.
func (t *Tx) RespondToInterest(ctx context.Context, interestID, artisanID uuid.UUID, decision domain.InterestStatus) (domain.CompanyInterest, error) {
	row, err := t.q.RespondToInterest(ctx, db.RespondToInterestParams{
		Status:    db.InterestStatus(decision),
		ID:        interestID,
		ArtisanID: artisanID,
	})
	if err != nil {
		return domain.CompanyInterest{}, translate(err, "company interest response")
	}
	return companyInterestFromRow(row), nil
}

// CreatePartnership establishes an ongoing partnership.
func (t *Tx) CreatePartnership(ctx context.Context, id uuid.UUID, in domain.CreatePartnershipInput) (domain.SupplyPartnership, error) {
	row, err := t.q.CreateSupplyPartnership(ctx, db.CreateSupplyPartnershipParams{
		ID:          id,
		CompanyID:   in.CompanyID,
		ArtisanID:   in.ArtisanID,
		CraftID:     in.CraftID,
		Terms:       in.Terms,
		RenewalDate: in.RenewalDate,
	})
	if err != nil {
		return domain.SupplyPartnership{}, translate(err, "supply partnership")
	}
	return supplyPartnershipFromRow(row), nil
}

// GetCompany fetches one company by id.
func (r *Repo) GetCompany(ctx context.Context, id uuid.UUID) (domain.Company, error) {
	row, err := r.q.GetCompany(ctx, id)
	if err != nil {
		return domain.Company{}, translate(err, "company")
	}
	return companyFromRow(row), nil
}

// GetCompanyByUserID fetches a company by owner user id.
func (r *Repo) GetCompanyByUserID(ctx context.Context, userID string) (domain.Company, error) {
	row, err := r.q.GetCompanyByUserID(ctx, userID)
	if err != nil {
		return domain.Company{}, translate(err, "company")
	}
	return companyFromRow(row), nil
}

// ListCompanies returns filtered companies.
func (r *Repo) ListCompanies(ctx context.Context, filter domain.CompanyFilter) ([]domain.Company, error) {
	var cType *db.CompanyType
	if filter.Type != nil {
		ct := db.CompanyType(*filter.Type)
		cType = &ct
	}
	var vStatus *db.VerificationStatus
	if filter.VerificationStatus != nil {
		vs := db.VerificationStatus(*filter.VerificationStatus)
		vStatus = &vs
	}

	rows, err := r.q.ListCompanies(ctx, db.ListCompaniesParams{
		CompanyType:        cType,
		StateCode:          filter.StateCode,
		VerificationStatus: vStatus,
		VerifiedOnly:       filter.VerifiedOnly,
		After:              filter.Page.Cursor,
		PageSize:           filter.Page.Size,
	})
	if err != nil {
		return nil, translate(err, "companies")
	}

	out := make([]domain.Company, len(rows))
	for i, row := range rows {
		out[i] = companyFromRow(row)
	}
	return out, nil
}

// GetPlatformCommissionStats aggregates platform revenue and company counts.
func (r *Repo) GetPlatformCommissionStats(ctx context.Context) (domain.PlatformCommissionStats, error) {
	row, err := r.q.GetPlatformCommissionStats(ctx)
	if err != nil {
		return domain.PlatformCommissionStats{}, translate(err, "platform commission stats")
	}
	return domain.PlatformCommissionStats{
		TotalCompanies:       row.TotalCompanies,
		PendingVerifications: row.PendingVerifications,
		VerifiedCompanies:    row.VerifiedCompanies,
		TotalSalesPaise:      row.TotalSalesPaise,
		TotalCommissionPaise: row.TotalCommissionPaise,
	}, nil
}

// ListCompanySales returns settlement rows for a company.
func (r *Repo) ListCompanySales(ctx context.Context, companyID uuid.UUID, page domain.Page) ([]domain.CompanySaleSettlement, error) {
	rows, err := r.q.ListCompanySales(ctx, db.ListCompanySalesParams{
		CompanyID: companyID,
		PageSize:  page.Size,
	})
	if err != nil {
		return nil, translate(err, "company sales")
	}

	out := make([]domain.CompanySaleSettlement, len(rows))
	for i, row := range rows {
		out[i] = saleSettlementFromRow(row)
	}
	return out, nil
}

// GetCompanyInterest reads one interest expression.
func (r *Repo) GetCompanyInterest(ctx context.Context, id uuid.UUID) (domain.CompanyInterest, error) {
	row, err := r.q.GetCompanyInterest(ctx, id)
	if err != nil {
		return domain.CompanyInterest{}, translate(err, "company interest")
	}
	return companyInterestFromRow(row), nil
}

// ListArtisanLeads lists inbound leads for an artisan.
func (r *Repo) ListArtisanLeads(ctx context.Context, artisanID uuid.UUID, status *domain.InterestStatus, page domain.Page) ([]domain.CompanyInterest, error) {
	var dbStatus *db.InterestStatus
	if status != nil {
		s := db.InterestStatus(*status)
		dbStatus = &s
	}

	rows, err := r.q.ListArtisanLeads(ctx, db.ListArtisanLeadsParams{
		ArtisanID: artisanID,
		Status:    dbStatus,
		After:     page.Cursor,
		PageSize:  page.Size,
	})
	if err != nil {
		return nil, translate(err, "artisan leads")
	}

	out := make([]domain.CompanyInterest, len(rows))
	for i, row := range rows {
		out[i] = domain.CompanyInterest{
			ID:          row.ID,
			CompanyID:   row.CompanyID,
			ArtisanID:   row.ArtisanID,
			Message:     row.Message,
			Status:      domain.InterestStatus(row.Status),
			RespondedAt: row.RespondedAt,
			CompanyName: row.CompanyName,
			CompanyType: domain.CompanyType(row.CompanyType),
			CreatedAt:   row.CreatedAt,
		}
	}
	return out, nil
}

// ListPartnershipsByArtisan lists active or all partnerships for an artisan.
func (r *Repo) ListPartnershipsByArtisan(ctx context.Context, artisanID uuid.UUID, activeOnly bool, page domain.Page) ([]domain.SupplyPartnership, error) {
	rows, err := r.q.ListPartnershipsByArtisan(ctx, db.ListPartnershipsByArtisanParams{
		ArtisanID:  artisanID,
		ActiveOnly: activeOnly,
		After:      page.Cursor,
		PageSize:   page.Size,
	})
	if err != nil {
		return nil, translate(err, "partnerships")
	}

	out := make([]domain.SupplyPartnership, len(rows))
	for i, row := range rows {
		out[i] = domain.SupplyPartnership{
			ID:          row.ID,
			CompanyID:   row.CompanyID,
			ArtisanID:   row.ArtisanID,
			CraftID:     row.CraftID,
			Terms:       row.Terms,
			RenewalDate: row.RenewalDate,
			Active:      row.Active,
			CompanyName: row.CompanyName,
			ArtisanName: row.ArtisanName,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return out, nil
}

// ListPartnershipsByCompany lists partnerships for a company.
func (r *Repo) ListPartnershipsByCompany(ctx context.Context, companyID uuid.UUID, activeOnly bool, page domain.Page) ([]domain.SupplyPartnership, error) {
	rows, err := r.q.ListPartnershipsByCompany(ctx, db.ListPartnershipsByCompanyParams{
		CompanyID:  companyID,
		ActiveOnly: activeOnly,
		After:      page.Cursor,
		PageSize:   page.Size,
	})
	if err != nil {
		return nil, translate(err, "partnerships")
	}

	out := make([]domain.SupplyPartnership, len(rows))
	for i, row := range rows {
		out[i] = domain.SupplyPartnership{
			ID:          row.ID,
			CompanyID:   row.CompanyID,
			ArtisanID:   row.ArtisanID,
			CraftID:     row.CraftID,
			Terms:       row.Terms,
			RenewalDate: row.RenewalDate,
			Active:      row.Active,
			CompanyName: row.CompanyName,
			ArtisanName: row.ArtisanName,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return out, nil
}

// ListBoutiqueMatchesForArtisan returns recommended matches.
func (r *Repo) ListBoutiqueMatchesForArtisan(ctx context.Context, artisanID uuid.UUID, limit int32) ([]domain.BoutiqueMatch, error) {
	rows, err := r.q.ListBoutiqueMatchesForArtisan(ctx, db.ListBoutiqueMatchesForArtisanParams{
		ArtisanID: artisanID,
		LimitVal:  limit,
	})
	if err != nil {
		return nil, translate(err, "boutique matches")
	}

	out := make([]domain.BoutiqueMatch, len(rows))
	for i, row := range rows {
		out[i] = domain.BoutiqueMatch{
			ID:         row.ID,
			CompanyID:  row.CompanyID,
			ArtisanID:  row.ArtisanID,
			MatchScore: row.MatchScore,
			Status:     domain.MatchStatus(row.Status),
			BoutiqueName: row.BoutiqueName,
			BoutiqueLocation: domain.StoreLocation{
				Latitude:  row.StoreLatitude,
				Longitude: row.StoreLongitude,
				Address:   row.StoreAddress,
				City:      row.StoreCity,
			},
			CreatedAt: row.CreatedAt,
		}
	}
	return out, nil
}

// ListNearbyBoutiques returns boutiques filtered by location bounding box.
func (r *Repo) ListNearbyBoutiques(ctx context.Context, lat, lng, radiusKm float64, craftID *uuid.UUID, limit int32) ([]domain.Company, error) {
	latDelta := radiusKm / 111.0
	lngDelta := radiusKm / (111.0 * 0.8) // roughly at Indian latitudes

	latMin := lat - latDelta
	latMax := lat + latDelta
	lngMin := lng - lngDelta
	lngMax := lng + lngDelta

	rows, err := r.q.ListNearbyBoutiques(ctx, db.ListNearbyBoutiquesParams{
		LatMin:   &latMin,
		LatMax:   &latMax,
		LngMin:   &lngMin,
		LngMax:   &lngMax,
		CraftID:  craftID,
		LimitVal: limit,
	})
	if err != nil {
		return nil, translate(err, "nearby boutiques")
	}

	out := make([]domain.Company, len(rows))
	for i, row := range rows {
		out[i] = companyFromRow(row)
	}
	return out, nil
}

func companyFromRow(row db.Company) domain.Company {
	return domain.Company{
		ID:                    row.ID,
		UserID:                row.UserID,
		Name:                  row.Name,
		Type:                  domain.CompanyType(row.CompanyType),
		GSTIN:                 row.Gstin,
		ContactName:           row.ContactName,
		ContactPhone:          row.ContactPhone,
		ContactEmail:          row.ContactEmail,
		Website:               row.Website,
		StateCode:             row.StateCode,
		District:              row.District,
		VerificationStatus:    domain.VerificationStatus(row.VerificationStatus),
		Verified:              row.Verified,
		VerifiedBy:            row.VerifiedBy,
		VerifiedAt:            row.VerifiedAt,
		RejectionReason:       row.RejectionReason,
		IncomeStatementURL:    row.IncomeStatementUrl,
		CommissionRateBps:     row.CommissionRateBps,
		TotalSalesPaise:       row.TotalSalesPaise,
		CommissionEarnedPaise: row.CommissionEarnedPaise,
		AcceptsConsignment:    row.AcceptsConsignment,
		MinOrderValuePaise:    row.MinOrderValuePaise,
		PreferredCraftIDs:     row.PreferredCraftIds,
		StoreLocation: domain.StoreLocation{
			Latitude:  row.StoreLatitude,
			Longitude: row.StoreLongitude,
			Address:   row.StoreAddress,
			City:      row.StoreCity,
			Pincode:   row.StorePincode,
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func companyInterestFromRow(row db.CompanyInterest) domain.CompanyInterest {
	return domain.CompanyInterest{
		ID:          row.ID,
		CompanyID:   row.CompanyID,
		ArtisanID:   row.ArtisanID,
		Message:     row.Message,
		Status:      domain.InterestStatus(row.Status),
		RespondedAt: row.RespondedAt,
		CreatedAt:   row.CreatedAt,
	}
}

func supplyPartnershipFromRow(row db.SupplyPartnership) domain.SupplyPartnership {
	return domain.SupplyPartnership{
		ID:          row.ID,
		CompanyID:   row.CompanyID,
		ArtisanID:   row.ArtisanID,
		CraftID:     row.CraftID,
		Terms:       row.Terms,
		RenewalDate: row.RenewalDate,
		Active:      row.Active,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func saleSettlementFromRow(row db.CompanySaleSettlement) domain.CompanySaleSettlement {
	return domain.CompanySaleSettlement{
		ID:                row.ID,
		CompanyID:         row.CompanyID,
		OrderID:           row.OrderID,
		ProductName:       row.ProductName,
		BuyerID:           row.BuyerID,
		GrossAmountPaise:  row.GrossAmountPaise,
		CommissionRateBps: row.CommissionRateBps,
		PlatformFeePaise:  row.PlatformFeePaise,
		NetPayoutPaise:    row.NetPayoutPaise,
		SettledAt:         row.SettledAt,
	}
}
