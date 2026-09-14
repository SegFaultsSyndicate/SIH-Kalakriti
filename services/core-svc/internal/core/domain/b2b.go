// services/core-svc/internal/core/domain/b2b.go

package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// CompanyType classifies a business entity.
type CompanyType string

const (
	CompanyTypeRetailer    CompanyType = "RETAILER"
	CompanyTypeBoutique    CompanyType = "BOUTIQUE"
	CompanyTypeExporter    CompanyType = "EXPORTER"
	CompanyTypeInstitution CompanyType = "INSTITUTION"
)

// VerificationStatus tracks the administrative legitimacy verification lifecycle.
type VerificationStatus string

const (
	VerificationStatusPending  VerificationStatus = "PENDING"
	VerificationStatusVerified VerificationStatus = "VERIFIED"
	VerificationStatusRejected VerificationStatus = "REJECTED"
)

// InterestStatus tracks the lifecycle of a company's interest in an artisan.
type InterestStatus string

const (
	InterestStatusPending  InterestStatus = "PENDING"
	InterestStatusAccepted InterestStatus = "ACCEPTED"
	InterestStatusDeclined InterestStatus = "DECLINED"
)

// MatchStatus tracks boutique-artisan match lifecycle.
type MatchStatus string

const (
	MatchStatusSuggested MatchStatus = "SUGGESTED"
	MatchStatusContacted MatchStatus = "CONTACTED"
	MatchStatusActive    MatchStatus = "ACTIVE"
	MatchStatusDeclined  MatchStatus = "DECLINED"
)

// Indian standard 15-character GSTIN format: 2 digits state code, 10 alphanumeric PAN,
// 1 entity number, 'Z' default, 1 check digit.
var gstinPattern = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$`)

// CommissionRateBpsForType calculates the platform fee basis points according to business rules:
// Exporters pay 1.00% (100 bps); Retailers, Boutiques, and Institutions pay 0.50% (50 bps).
func CommissionRateBpsForType(t CompanyType) int32 {
	switch t {
	case CompanyTypeExporter:
		return 100 // 1.00%
	case CompanyTypeRetailer, CompanyTypeBoutique, CompanyTypeInstitution:
		return 50 // 0.50%
	default:
		return 50
	}
}

// CalculatePlatformCommission returns platform_fee_paise and net_payout_paise using integer arithmetic.
func CalculatePlatformCommission(grossPaise int64, bps int32) (feePaise int64, netPaise int64) {
	feePaise = (grossPaise * int64(bps)) / 10000
	netPaise = grossPaise - feePaise
	return feePaise, netPaise
}

// StoreLocation holds a boutique's physical coordinates and address.
type StoreLocation struct {
	Latitude  *float64
	Longitude *float64
	Address   *string
	City      *string
	Pincode   *string
}

// Company is a verified commercial entity that purchases from artisans or sells products.
type Company struct {
	ID                    uuid.UUID
	UserID                string
	Name                  string
	Type                  CompanyType
	GSTIN                 *string
	ContactName           string
	ContactPhone          string
	ContactEmail          *string
	Website               *string
	StateCode             string
	District              *string
	VerificationStatus    VerificationStatus
	Verified              bool
	VerifiedBy            *string
	VerifiedAt            *time.Time
	RejectionReason       *string
	IncomeStatementURL    string
	CommissionRateBps     int32
	TotalSalesPaise       int64
	CommissionEarnedPaise int64
	AcceptsConsignment    bool
	MinOrderValuePaise    *int64
	PreferredCraftIDs     []uuid.UUID
	StoreLocation         StoreLocation
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// RegisterCompanyInput is what the service needs to register a company.
type RegisterCompanyInput struct {
	UserID             string
	Name               string
	Type               CompanyType
	GSTIN              *string
	ContactName        string
	ContactPhone       string
	ContactEmail       *string
	Website            *string
	Region             Region
	IncomeStatementURL string
	AcceptsConsignment bool
	MinOrderValuePaise *int64
	PreferredCraftIDs  []uuid.UUID
	StoreLocation      StoreLocation
}

// Validate checks registration fields before a transaction opens.
func (in RegisterCompanyInput) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.ContactName) == "" {
		return fmt.Errorf("contact_name is required: %w", pkgdomain.ErrInvalidInput)
	}
	if !phoneE164Pattern.MatchString(in.ContactPhone) {
		return fmt.Errorf("contact_phone %q must be E.164: %w", in.ContactPhone, pkgdomain.ErrInvalidInput)
	}
	switch in.Type {
	case CompanyTypeRetailer, CompanyTypeBoutique, CompanyTypeExporter, CompanyTypeInstitution:
		// valid
	default:
		return fmt.Errorf("invalid company_type %q: %w", in.Type, pkgdomain.ErrInvalidInput)
	}
	if in.GSTIN != nil && *in.GSTIN != "" {
		cleaned := strings.TrimSpace(*in.GSTIN)
		if !gstinPattern.MatchString(cleaned) {
			return fmt.Errorf("invalid GSTIN %q (must be standard 15-character Indian format): %w", cleaned, pkgdomain.ErrInvalidInput)
		}
	}
	if strings.TrimSpace(in.IncomeStatementURL) == "" {
		return fmt.Errorf("income_statement_url is required to verify legitimacy: %w", pkgdomain.ErrInvalidInput)
	}
	if err := in.Region.Validate(); err != nil {
		return err
	}
	if in.MinOrderValuePaise != nil && *in.MinOrderValuePaise < 0 {
		return fmt.Errorf("min_order_value_paise must not be negative: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// VerifyCompanyInput contains admin decision parameters.
type VerifyCompanyInput struct {
	CompanyID       uuid.UUID
	Decision        VerificationStatus
	RejectionReason *string
	AdminUserID     string
}

// Validate checks admin review input.
func (in VerifyCompanyInput) Validate() error {
	if in.CompanyID == uuid.Nil {
		return fmt.Errorf("company_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.AdminUserID) == "" {
		return fmt.Errorf("admin_user_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	switch in.Decision {
	case VerificationStatusVerified:
		// valid
	case VerificationStatusRejected:
		if in.RejectionReason == nil || strings.TrimSpace(*in.RejectionReason) == "" {
			return fmt.Errorf("rejection_reason is required when rejecting an application: %w", pkgdomain.ErrInvalidInput)
		}
	default:
		return fmt.Errorf("decision must be VERIFIED or REJECTED: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// CompanySaleSettlement records platform fee deduction from marketplace sales.
type CompanySaleSettlement struct {
	ID                uuid.UUID
	CompanyID         uuid.UUID
	OrderID           string
	ProductName       string
	BuyerID           string
	GrossAmountPaise  int64
	CommissionRateBps int32
	PlatformFeePaise  int64
	NetPayoutPaise    int64
	SettledAt         time.Time
}

// RecordCompanySaleInput is what the settlement engine passes to record a marketplace purchase.
type RecordCompanySaleInput struct {
	CompanyID        uuid.UUID
	OrderID          string
	ProductName      string
	BuyerID          string
	GrossAmountPaise int64
}

// Validate checks sale recording fields.
func (in RecordCompanySaleInput) Validate() error {
	if in.CompanyID == uuid.Nil {
		return fmt.Errorf("company_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.OrderID) == "" {
		return fmt.Errorf("order_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.ProductName) == "" {
		return fmt.Errorf("product_name is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.BuyerID) == "" {
		return fmt.Errorf("buyer_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.GrossAmountPaise <= 0 {
		return fmt.Errorf("gross_amount_paise must be positive: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// PlatformCommissionStats summarizes platform treasury and partner growth metrics.
type PlatformCommissionStats struct {
	TotalCompanies       int64
	PendingVerifications int64
	VerifiedCompanies    int64
	TotalSalesPaise      int64
	TotalCommissionPaise int64
}

// CompanyInterest is a company's expression of interest in working with an artisan.
type CompanyInterest struct {
	ID          uuid.UUID
	CompanyID   uuid.UUID
	ArtisanID   uuid.UUID
	Message     string
	Status      InterestStatus
	RespondedAt *time.Time
	CompanyName string
	CompanyType CompanyType
	CreatedAt   time.Time
}

// ExpressInterestInput is what the service needs to create an interest.
type ExpressInterestInput struct {
	CompanyID uuid.UUID
	ArtisanID uuid.UUID
	Message   string
}

// Validate checks interest expression fields.
func (in ExpressInterestInput) Validate() error {
	if in.CompanyID == uuid.Nil {
		return fmt.Errorf("company_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.ArtisanID == uuid.Nil {
		return fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// SupplyPartnership is an ongoing relationship between a company and an artisan.
type SupplyPartnership struct {
	ID          uuid.UUID
	CompanyID   uuid.UUID
	ArtisanID   uuid.UUID
	CraftID     uuid.UUID
	Terms       string
	RenewalDate *time.Time
	Active      bool
	CompanyName string
	ArtisanName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreatePartnershipInput is what the service needs to create a partnership.
type CreatePartnershipInput struct {
	CompanyID   uuid.UUID
	ArtisanID   uuid.UUID
	CraftID     uuid.UUID
	Terms       string
	RenewalDate *time.Time
}

// Validate checks partnership creation fields.
func (in CreatePartnershipInput) Validate() error {
	if in.CompanyID == uuid.Nil {
		return fmt.Errorf("company_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.ArtisanID == uuid.Nil {
		return fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.CraftID == uuid.Nil {
		return fmt.Errorf("craft_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// BoutiqueMatch is a scored recommendation pairing a boutique with an artisan.
type BoutiqueMatch struct {
	ID                uuid.UUID
	CompanyID         uuid.UUID
	ArtisanID         uuid.UUID
	MatchScore        float32
	Status            MatchStatus
	BoutiqueName      string
	BoutiqueLocation  StoreLocation
	OverlappingCrafts []string
	CreatedAt         time.Time
}

// CompanyFilter holds query parameters for listing companies.
type CompanyFilter struct {
	Type               *CompanyType
	StateCode          *string
	VerificationStatus *VerificationStatus
	VerifiedOnly       bool
	Page               Page
}
