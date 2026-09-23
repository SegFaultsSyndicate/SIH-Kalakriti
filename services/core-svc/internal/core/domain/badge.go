// services/core-svc/internal/core/domain/badge.go

package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// BadgeKind distinguishes an admin-conferred badge from one the badge
// consumer grants automatically when an activity threshold is crossed.
type BadgeKind string

const (
	BadgeKindEarned    BadgeKind = "EARNED"
	BadgeKindConferred BadgeKind = "CONFERRED"
)

// BadgeTier is the tier of a tiered earned badge; untiered badges carry nil.
type BadgeTier string

const (
	BadgeTierBronze BadgeTier = "BRONZE"
	BadgeTierSilver BadgeTier = "SILVER"
	BadgeTierGold   BadgeTier = "GOLD"
)

// BadgeMetric is a countable artisan activity an earned badge's threshold is
// measured against.
type BadgeMetric string

const (
	MetricListingsPublished BadgeMetric = "LISTINGS_PUBLISHED"
	MetricProvenanceSealed  BadgeMetric = "PROVENANCE_SEALED"
	MetricLotsAccepted      BadgeMetric = "LOTS_ACCEPTED"
	MetricLotsCompleted     BadgeMetric = "LOTS_COMPLETED"
	MetricLessonsCompleted  BadgeMetric = "LESSONS_COMPLETED"
)

// Badge is one catalog entry: either a conferred recognition or one tier of
// an earned badge family.
type Badge struct {
	ID        uuid.UUID
	Code      string
	Kind      BadgeKind
	Tier      *BadgeTier
	IconName  string
	Metric    *BadgeMetric
	Threshold *int64
	SortOrder int32
}

// ArtisanBadge is one active grant, joined with its catalog entry.
type ArtisanBadge struct {
	Badge     Badge
	GrantedAt time.Time
	GrantedBy string
	Evidence  *string // raw JSON, if present
}

// BadgeProgressEntry is the artisan's current count toward one metric.
type BadgeProgressEntry struct {
	Metric    BadgeMetric
	Value     int64
	UpdatedAt time.Time
}

// GrantBadgeInput is what an admin supplies to confer a badge.
type GrantBadgeInput struct {
	ArtisanID uuid.UUID
	BadgeID   uuid.UUID
	GrantedBy string
	Evidence  *string
}

// Validate checks a grant request has everything it needs. It does not check
// that BadgeID refers to a CONFERRED badge -- that check needs a DB read and
// lives in the service layer (see service.Badges.GrantBadge).
func (in GrantBadgeInput) Validate() error {
	if in.ArtisanID == uuid.Nil {
		return fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.BadgeID == uuid.Nil {
		return fmt.Errorf("badge_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.GrantedBy == "" {
		return fmt.Errorf("granted_by is required: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// RevokeBadgeInput is what an admin supplies to revoke a badge.
type RevokeBadgeInput struct {
	ArtisanID uuid.UUID
	BadgeID   uuid.UUID
	RevokedBy string
	Reason    string
}

// Validate checks a revoke request has everything it needs.
func (in RevokeBadgeInput) Validate() error {
	if in.ArtisanID == uuid.Nil {
		return fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.BadgeID == uuid.Nil {
		return fmt.Errorf("badge_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.RevokedBy == "" {
		return fmt.Errorf("revoked_by is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.Reason == "" {
		return fmt.Errorf("reason is required: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}
