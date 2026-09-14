// services/core-svc/internal/core/service/b2b.go

package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// B2BStore is the persistence surface the B2B service reads through.
type B2BStore interface {
	InTx(ctx context.Context, fn func(ctx context.Context, tx B2BTx) error) error

	GetCompany(ctx context.Context, id uuid.UUID) (domain.Company, error)
	GetCompanyByUserID(ctx context.Context, userID string) (domain.Company, error)
	ListCompanies(ctx context.Context, filter domain.CompanyFilter) ([]domain.Company, error)
	GetPlatformCommissionStats(ctx context.Context) (domain.PlatformCommissionStats, error)
	ListCompanySales(ctx context.Context, companyID uuid.UUID, page domain.Page) ([]domain.CompanySaleSettlement, error)

	GetCompanyInterest(ctx context.Context, id uuid.UUID) (domain.CompanyInterest, error)
	ListArtisanLeads(ctx context.Context, artisanID uuid.UUID, status *domain.InterestStatus, page domain.Page) ([]domain.CompanyInterest, error)

	ListPartnershipsByArtisan(ctx context.Context, artisanID uuid.UUID, activeOnly bool, page domain.Page) ([]domain.SupplyPartnership, error)
	ListPartnershipsByCompany(ctx context.Context, companyID uuid.UUID, activeOnly bool, page domain.Page) ([]domain.SupplyPartnership, error)

	ListBoutiqueMatchesForArtisan(ctx context.Context, artisanID uuid.UUID, limit int32) ([]domain.BoutiqueMatch, error)
	ListNearbyBoutiques(ctx context.Context, lat, lng, radiusKm float64, craftID *uuid.UUID, limit int32) ([]domain.Company, error)
}

// B2BTx is the transactional surface for B2B writes.
type B2BTx interface {
	outbox.Enqueuer
	CreateCompany(ctx context.Context, id uuid.UUID, in domain.RegisterCompanyInput, bps int32) (domain.Company, error)
	VerifyCompany(ctx context.Context, in domain.VerifyCompanyInput, bps int32) (domain.Company, error)
	RecordCompanySaleSettlement(ctx context.Context, id uuid.UUID, in domain.RecordCompanySaleInput, bps int32, feePaise, netPaise int64) (domain.CompanySaleSettlement, error)
	IncrementCompanySales(ctx context.Context, companyID uuid.UUID, grossPaise, feePaise int64) error

	CreateCompanyInterest(ctx context.Context, id uuid.UUID, in domain.ExpressInterestInput) (domain.CompanyInterest, error)
	RespondToInterest(ctx context.Context, interestID uuid.UUID, artisanID uuid.UUID, decision domain.InterestStatus) (domain.CompanyInterest, error)
	CreatePartnership(ctx context.Context, id uuid.UUID, in domain.CreatePartnershipInput) (domain.SupplyPartnership, error)
}

// B2B is the service for company connections, verification, interest management, and partnerships.
type B2B struct {
	store B2BStore
	log   *slog.Logger
	now   func() time.Time
}

// NewB2B builds the B2B service.
func NewB2B(store B2BStore, log *slog.Logger) *B2B {
	return &B2B{store: store, log: log, now: time.Now}
}

// RegisterCompany creates a new company profile in PENDING verification state.
func (s *B2B) RegisterCompany(ctx context.Context, in domain.RegisterCompanyInput) (domain.Company, error) {
	if err := in.Validate(); err != nil {
		return domain.Company{}, err
	}

	bps := domain.CommissionRateBpsForType(in.Type)

	var company domain.Company
	if err := s.store.InTx(ctx, func(ctx context.Context, tx B2BTx) error {
		id := ids.New()
		var err error
		company, err = tx.CreateCompany(ctx, id, in, bps)
		if err != nil {
			return err
		}

		payload := map[string]string{
			"company_id":    id.String(),
			"company_name":  in.Name,
			"company_type":  string(in.Type),
			"contact_name":  in.ContactName,
			"contact_phone": in.ContactPhone,
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), id.String(), topics.CompanyRegistered, id.String(), payload)
	}); err != nil {
		return domain.Company{}, err
	}

	s.log.Info("company_registered", "company_id", company.ID, "type", company.Type, "commission_rate_bps", bps)
	return company, nil
}

// VerifyCompany lets an admin approve or reject a company's registration.
func (s *B2B) VerifyCompany(ctx context.Context, in domain.VerifyCompanyInput) (domain.Company, error) {
	if err := in.Validate(); err != nil {
		return domain.Company{}, err
	}

	target, err := s.store.GetCompany(ctx, in.CompanyID)
	if err != nil {
		return domain.Company{}, err
	}

	bps := domain.CommissionRateBpsForType(target.Type)

	var verifiedCompany domain.Company
	if err := s.store.InTx(ctx, func(ctx context.Context, tx B2BTx) error {
		var err error
		verifiedCompany, err = tx.VerifyCompany(ctx, in, bps)
		if err != nil {
			return err
		}

		topic := topics.CompanyVerified
		if in.Decision == domain.VerificationStatusRejected {
			topic = topics.CompanyRejected
		}

		rateStr := "0.5"
		if bps == 100 {
			rateStr = "1.0"
		}

		reason := ""
		if in.RejectionReason != nil {
			reason = *in.RejectionReason
		}

		payload := map[string]string{
			"company_id":    target.ID.String(),
			"company_name":  target.Name,
			"company_type":  string(target.Type),
			"contact_name":  target.ContactName,
			"contact_phone": target.ContactPhone,
			"rate":          rateStr,
			"reason":        reason,
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), target.ID.String(), topic, target.ID.String(), payload)
	}); err != nil {
		return domain.Company{}, err
	}

	s.log.Info("company_verified", "company_id", in.CompanyID, "decision", in.Decision, "verified_by", in.AdminUserID)
	return verifiedCompany, nil
}

// RecordCompanySale records a product purchase from a company with platform fee deduction.
func (s *B2B) RecordCompanySale(ctx context.Context, in domain.RecordCompanySaleInput) (domain.CompanySaleSettlement, error) {
	if err := in.Validate(); err != nil {
		return domain.CompanySaleSettlement{}, err
	}

	company, err := s.store.GetCompany(ctx, in.CompanyID)
	if err != nil {
		return domain.CompanySaleSettlement{}, err
	}

	feePaise, netPaise := domain.CalculatePlatformCommission(in.GrossAmountPaise, company.CommissionRateBps)

	var settlement domain.CompanySaleSettlement
	if err := s.store.InTx(ctx, func(ctx context.Context, tx B2BTx) error {
		id := ids.New()
		var err error
		settlement, err = tx.RecordCompanySaleSettlement(ctx, id, in, company.CommissionRateBps, feePaise, netPaise)
		if err != nil {
			return err
		}

		if err := tx.IncrementCompanySales(ctx, in.CompanyID, in.GrossAmountPaise, feePaise); err != nil {
			return err
		}

		rateStr := "0.5"
		if company.CommissionRateBps == 100 {
			rateStr = "1.0"
		}

		payload := map[string]any{
			"company_id":    company.ID.String(),
			"company_name":  company.Name,
			"contact_phone": company.ContactPhone,
			"order_id":      in.OrderID,
			"product_name":  in.ProductName,
			"gross_paise":   in.GrossAmountPaise,
			"fee_paise":     feePaise,
			"net_paise":     netPaise,
			"rate":          rateStr,
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), id.String(), topics.CompanySaleSettled, id.String(), payload)
	}); err != nil {
		return domain.CompanySaleSettlement{}, err
	}

	s.log.Info("company_sale_recorded", "company_id", in.CompanyID, "gross", in.GrossAmountPaise, "fee", feePaise)
	return settlement, nil
}

// GetPlatformCommissionStats retrieves platform treasury metrics.
func (s *B2B) GetPlatformCommissionStats(ctx context.Context) (domain.PlatformCommissionStats, error) {
	return s.store.GetPlatformCommissionStats(ctx)
}

// ListCompanySales lists settlement entries for a given company.
func (s *B2B) ListCompanySales(ctx context.Context, companyID uuid.UUID, page domain.Page) ([]domain.CompanySaleSettlement, error) {
	page = page.Normalise()
	return s.store.ListCompanySales(ctx, companyID, page)
}

// GetCompany fetches one company by id.
func (s *B2B) GetCompany(ctx context.Context, id uuid.UUID) (domain.Company, error) {
	return s.store.GetCompany(ctx, id)
}

// GetMyCompany fetches the calling user's company profile.
func (s *B2B) GetMyCompany(ctx context.Context) (domain.Company, error) {
	p, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return domain.Company{}, fmt.Errorf("not authenticated: %w", pkgdomain.ErrForbidden)
	}
	return s.store.GetCompanyByUserID(ctx, p.Subject)
}

// ListCompanies pages through companies with optional filters.
func (s *B2B) ListCompanies(ctx context.Context, filter domain.CompanyFilter) ([]domain.Company, error) {
	filter.Page = filter.Page.Normalise()
	return s.store.ListCompanies(ctx, filter)
}

// ExpressInterest records a company's interest in working with an artisan.
func (s *B2B) ExpressInterest(ctx context.Context, in domain.ExpressInterestInput) (domain.CompanyInterest, error) {
	if err := in.Validate(); err != nil {
		return domain.CompanyInterest{}, err
	}

	var interest domain.CompanyInterest
	if err := s.store.InTx(ctx, func(ctx context.Context, tx B2BTx) error {
		id := ids.New()
		var err error
		interest, err = tx.CreateCompanyInterest(ctx, id, in)
		if err != nil {
			return err
		}

		payload := map[string]string{
			"interest_id": id.String(),
			"company_id":  in.CompanyID.String(),
			"artisan_id":  in.ArtisanID.String(),
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), id.String(), topics.CompanyInterestExpressed, id.String(), payload)
	}); err != nil {
		return domain.CompanyInterest{}, err
	}

	s.log.Info("interest_expressed", "company_id", in.CompanyID, "artisan_id", in.ArtisanID)
	return interest, nil
}

// RespondToInterest lets an artisan accept or decline a company's interest.
func (s *B2B) RespondToInterest(ctx context.Context, interestID uuid.UUID, artisanID uuid.UUID, decision domain.InterestStatus) (domain.CompanyInterest, error) {
	if decision != domain.InterestStatusAccepted && decision != domain.InterestStatusDeclined {
		return domain.CompanyInterest{}, fmt.Errorf("decision must be ACCEPTED or DECLINED: %w", pkgdomain.ErrInvalidInput)
	}

	var interest domain.CompanyInterest
	if err := s.store.InTx(ctx, func(ctx context.Context, tx B2BTx) error {
		var err error
		interest, err = tx.RespondToInterest(ctx, interestID, artisanID, decision)
		if err != nil {
			return err
		}

		topic := topics.CompanyInterestAccepted
		payload := map[string]string{
			"interest_id": interestID.String(),
			"company_id":  interest.CompanyID.String(),
			"artisan_id":  artisanID.String(),
			"decision":    string(decision),
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), interestID.String(), topic, interestID.String(), payload)
	}); err != nil {
		return domain.CompanyInterest{}, err
	}

	s.log.Info("interest_responded", "interest_id", interestID, "decision", decision)
	return interest, nil
}

// ListArtisanLeads returns inbound interest requests for an artisan.
func (s *B2B) ListArtisanLeads(ctx context.Context, artisanID uuid.UUID, status *domain.InterestStatus, page domain.Page) ([]domain.CompanyInterest, error) {
	page = page.Normalise()
	return s.store.ListArtisanLeads(ctx, artisanID, status, page)
}

// CreatePartnership establishes an ongoing supply relationship.
func (s *B2B) CreatePartnership(ctx context.Context, in domain.CreatePartnershipInput) (domain.SupplyPartnership, error) {
	if err := in.Validate(); err != nil {
		return domain.SupplyPartnership{}, err
	}

	var partnership domain.SupplyPartnership
	if err := s.store.InTx(ctx, func(ctx context.Context, tx B2BTx) error {
		id := ids.New()
		var err error
		partnership, err = tx.CreatePartnership(ctx, id, in)
		if err != nil {
			return err
		}

		payload := map[string]string{
			"partnership_id": id.String(),
			"company_id":     in.CompanyID.String(),
			"artisan_id":     in.ArtisanID.String(),
			"craft_id":       in.CraftID.String(),
		}
		return outbox.Enqueue(ctx, tx, ids.New().String(), id.String(), topics.SupplyPartnershipCreated, id.String(), payload)
	}); err != nil {
		return domain.SupplyPartnership{}, err
	}

	s.log.Info("partnership_created", "company_id", in.CompanyID, "artisan_id", in.ArtisanID)
	return partnership, nil
}

// ListPartnershipsByArtisan returns an artisan's partnerships.
func (s *B2B) ListPartnershipsByArtisan(ctx context.Context, artisanID uuid.UUID, activeOnly bool, page domain.Page) ([]domain.SupplyPartnership, error) {
	page = page.Normalise()
	return s.store.ListPartnershipsByArtisan(ctx, artisanID, activeOnly, page)
}

// ListPartnershipsByCompany returns a company's partnerships.
func (s *B2B) ListPartnershipsByCompany(ctx context.Context, companyID uuid.UUID, activeOnly bool, page domain.Page) ([]domain.SupplyPartnership, error) {
	page = page.Normalise()
	return s.store.ListPartnershipsByCompany(ctx, companyID, activeOnly, page)
}

// ListBoutiqueMatches returns scored boutique recommendations for an artisan.
func (s *B2B) ListBoutiqueMatches(ctx context.Context, artisanID uuid.UUID, limit int32) ([]domain.BoutiqueMatch, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.ListBoutiqueMatchesForArtisan(ctx, artisanID, limit)
}

// ListNearbyBoutiques returns boutiques near a location.
func (s *B2B) ListNearbyBoutiques(ctx context.Context, lat, lng, radiusKm float64, craftID *uuid.UUID, limit int32) ([]domain.Company, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.ListNearbyBoutiques(ctx, lat, lng, radiusKm, craftID, limit)
}
