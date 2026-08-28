// services/core-svc/internal/core/domain/artisan.go

// Package domain holds core-svc's business types. These structs are what the
// service layer takes and returns; protobuf and sqlc types are converted at the
// handler and repo edges respectively, never leaked inward.
package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

// phoneE164Pattern mirrors the artisan_phone_e164_check constraint in migration 002,
// so a bad number is rejected with a useful message before it reaches Postgres.
var phoneE164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// pincodePattern mirrors the cluster_pincode_check constraint in migration 002.
var pincodePattern = regexp.MustCompile(`^[1-9][0-9]{5}$`)

// maxLanguages bounds the languages list so a single profile cannot carry the
// entire Eighth Schedule and blow out every response that embeds it.
const maxLanguages = 8

// Region is where an artisan or cluster is located.
type Region struct {
	StateCode string
	District  *string
	Block     *string
	Village   *string
	Pincode   *string
}

// Validate checks the region's required and pattern-constrained fields.
func (r Region) Validate() error {
	if strings.TrimSpace(r.StateCode) == "" {
		return fmt.Errorf("region.state_code is required: %w", pkgdomain.ErrInvalidInput)
	}
	if r.Pincode != nil && !pincodePattern.MatchString(*r.Pincode) {
		return fmt.Errorf("region.pincode %q is not a six-digit Indian PIN: %w", *r.Pincode, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// Artisan is one maker: the unit that owns products and accepts order lots.
type Artisan struct {
	ID                uuid.UUID
	UserID            *string
	DisplayName       string
	PhoneE164         string
	PehchanID         *string
	PMVishwakarmaID   *string
	PrimaryClusterID  *uuid.UUID
	Region            Region
	Languages         []string
	CraftIDs          []uuid.UUID
	YearsOfExperience *int32
	Bio               *string
	PhotoMediaID      *uuid.UUID
	Verified          bool
	CreatedBy         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// RegisterArtisanInput is what the service needs to create an artisan.
type RegisterArtisanInput struct {
	DisplayName       string
	PhoneE164         string
	CraftIDs          []uuid.UUID
	Languages         []string
	Region            Region
	ClusterID         *uuid.UUID
	PehchanID         *string
	PMVishwakarmaID   *string
	YearsOfExperience *int32
	Bio               *string
	// CreatedBy is the subject id of the actor performing the registration,
	// which is the artisan themselves for self-service and an officer's id
	// when field staff register someone by proxy.
	CreatedBy string
}

// Validate checks every field the database constrains, plus the business rules
// the schema cannot express, so a bad request fails before a transaction opens.
func (in RegisterArtisanInput) Validate() error {
	if strings.TrimSpace(in.DisplayName) == "" {
		return fmt.Errorf("display_name is required: %w", pkgdomain.ErrInvalidInput)
	}
	if len(in.DisplayName) > 200 {
		return fmt.Errorf("display_name must be at most 200 characters: %w", pkgdomain.ErrInvalidInput)
	}
	if !phoneE164Pattern.MatchString(in.PhoneE164) {
		return fmt.Errorf("phone_e164 %q must be E.164, e.g. +919876543210: %w", in.PhoneE164, pkgdomain.ErrInvalidInput)
	}
	if len(in.CraftIDs) == 0 {
		return fmt.Errorf("at least one craft_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if err := uniqueUUIDs(in.CraftIDs, "craft_ids"); err != nil {
		return err
	}
	if len(in.Languages) == 0 {
		return fmt.Errorf("at least one language is required: %w", pkgdomain.ErrInvalidInput)
	}
	if len(in.Languages) > maxLanguages {
		return fmt.Errorf("at most %d languages may be listed: %w", maxLanguages, pkgdomain.ErrInvalidInput)
	}
	if err := in.Region.Validate(); err != nil {
		return err
	}
	if in.YearsOfExperience != nil && *in.YearsOfExperience < 0 {
		return fmt.Errorf("years_of_experience must not be negative: %w", pkgdomain.ErrInvalidInput)
	}
	if in.PehchanID != nil && strings.TrimSpace(*in.PehchanID) == "" {
		return fmt.Errorf("pehchan_id must not be blank when present: %w", pkgdomain.ErrInvalidInput)
	}
	if in.PMVishwakarmaID != nil && strings.TrimSpace(*in.PMVishwakarmaID) == "" {
		return fmt.Errorf("pm_vishwakarma_id must not be blank when present: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// UpdateArtisanInput patches an artisan profile. A nil field is left untouched;
// Languages and CraftIDs replace the stored lists wholesale when non-nil.
type UpdateArtisanInput struct {
	ArtisanID         uuid.UUID
	DisplayName       *string
	Languages         []string
	Region            *Region
	ClusterID         *uuid.UUID
	YearsOfExperience *int32
	Bio               *string
	PhotoMediaID      *uuid.UUID
}

// Validate checks only the fields actually being patched.
func (in UpdateArtisanInput) Validate() error {
	if in.ArtisanID == uuid.Nil {
		return fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.DisplayName != nil && strings.TrimSpace(*in.DisplayName) == "" {
		return fmt.Errorf("display_name must not be blank: %w", pkgdomain.ErrInvalidInput)
	}
	if len(in.Languages) > maxLanguages {
		return fmt.Errorf("at most %d languages may be listed: %w", maxLanguages, pkgdomain.ErrInvalidInput)
	}
	if in.Region != nil {
		if err := in.Region.Validate(); err != nil {
			return err
		}
	}
	if in.YearsOfExperience != nil && *in.YearsOfExperience < 0 {
		return fmt.Errorf("years_of_experience must not be negative: %w", pkgdomain.ErrInvalidInput)
	}
	return nil
}

// Page is a keyset page request. Cursor is the last id from the previous page.
type Page struct {
	Cursor *uuid.UUID
	Size   int32
}

// DefaultPageSize and MaxPageSize bound what a caller may ask for.
const (
	DefaultPageSize int32 = 50
	MaxPageSize     int32 = 200
)

// Normalise clamps the page size into range, so a caller asking for a million
// rows gets a sane page rather than an error or an unbounded scan.
func (p Page) Normalise() Page {
	if p.Size <= 0 {
		p.Size = DefaultPageSize
	}
	if p.Size > MaxPageSize {
		p.Size = MaxPageSize
	}
	return p
}

// uniqueUUIDs rejects a list containing a nil or repeated id.
func uniqueUUIDs(ids []uuid.UUID, field string) error {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			return fmt.Errorf("%s must not contain a nil uuid: %w", field, pkgdomain.ErrInvalidInput)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("%s contains %s twice: %w", field, id, pkgdomain.ErrInvalidInput)
		}
		seen[id] = struct{}{}
	}
	return nil
}
