package domain

import (
	"time"

	"github.com/google/uuid"
)

// ImpactFilter narrows every impact query. nil = no filter.
type ImpactFilter struct {
	StateCode      *string
	District       *string
	SocialCategory *string
	Corporation    *string
	FromMonth      *time.Time
	ToMonth        *time.Time
}

// ImpactArtisan is one artisan's row of mv_impact_artisan.
type ImpactArtisan struct {
	ArtisanID            uuid.UUID
	StateCode            string
	District             string
	SocialCategory       string
	Corporations         []string
	FinanceVerified      bool
	RegisteredAt         time.Time
	BaselineBracket      string // "" = none captured
	BaselineMonthlyPaise *int64
	Platform90d          int64
	Offline90d           int64
	Fair90d              int64
	LastSaleAt           *time.Time
}

// ImpactSummary is the dashboard's headline KPIs for one cohort.
type ImpactSummary struct {
	Suppressed         bool
	Beneficiaries      int64
	ActiveSellers90d   int64
	UpliftSample       int64
	MedianUpliftPct    *float64
	DigitalSharePct    *float64
	CertificatesIssued int64
	FinanceLinked      int64
	FinanceVerified    int64
	RefreshedAt        *time.Time
}

// ImpactGroupRow is one district / social category / corporation.
type ImpactGroupRow struct {
	Group                string
	StateCode            string
	Suppressed           bool
	ArtisanCount         int64
	ActiveSellers90d     int64
	WithBaselineCount    int64
	MedianBaselinePaise  int64
	MedianCurrentPaise   int64 // over the same artisans as MedianBaselinePaise
	MedianUpliftPct      *float64
	PlatformIncomePaise  int64
	OfflineIncomePaise   int64
	FairIncomePaise      int64
}

// SalesMixMonth is one month of platform vs offline sales for a cohort.
type SalesMixMonth struct {
	Month             time.Time
	Suppressed        bool
	ArtisanCount      int64
	PlatformPaise     int64
	FairPaise         int64
	OtherOfflinePaise int64
}

// FinanceCoverageFact is one non-rejected finance link with its artisan's income.
type FinanceCoverageFact struct {
	ArtisanID   uuid.UUID
	Corporation string
	Status      string
	EmiPaise    int64
	Platform90d int64
	Offline90d  int64
}

// FinanceCoverageRow is one corporation's beneficiaries on the platform.
type FinanceCoverageRow struct {
	Corporation         string
	Suppressed          bool
	Beneficiaries       int64
	Verified            int64
	SelfReported        int64
	ActiveSellers90d    int64
	MedianCoverageRatio *float64
}

// LiteracyFunnelRow is one district's progress through the literacy track.
type LiteracyFunnelRow struct {
	StateCode  string
	District   string
	Suppressed bool
	Artisans   int64
	Started    int64
	HalfWay    int64
	Certified  int64
}

// LiteracyCertificate is an issued digital literacy certificate.
type LiteracyCertificate struct {
	ID          uuid.UUID
	ArtisanID   uuid.UUID
	IssuedAt    time.Time
	ShortCode   string
	Signature   []byte
	PublicKeyID string
	S3Key       string
	// Verification view only.
	ArtisanName string
	StateCode   string
	District    string
	CraftName   string
}
