// services/core-svc/internal/core/domain/catalog.go
package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// ListingType is the commercial posture of a listing. Values match the
// listing_type Postgres enum.
type ListingType string

const (
	// ListingMadeToOrder is produced after the order is accepted.
	ListingMadeToOrder ListingType = "MADE_TO_ORDER"
	// ListingReadyStock is already made and on hand.
	ListingReadyStock ListingType = "READY_STOCK"
)

// Valid reports whether t is a known listing type.
func (t ListingType) Valid() bool {
	return t == ListingMadeToOrder || t == ListingReadyStock
}

// String returns the type's wire and database representation.
func (t ListingType) String() string { return string(t) }

// ListingState is where a listing sits in its approval lifecycle. Values match
// the listing_state Postgres enum.
type ListingState string

const (
	// StateDraft is being assembled and is not visible to buyers.
	StateDraft ListingState = "DRAFT"
	// StatePendingArtisanApproval is waiting for the artisan to sign off on generated copy.
	StatePendingArtisanApproval ListingState = "PENDING_ARTISAN_APPROVAL"
	// StatePublished is live and searchable.
	StatePublished ListingState = "PUBLISHED"
	// StateSuspended is withdrawn by an operator and recoverable.
	StateSuspended ListingState = "SUSPENDED"
)

// String returns the state's wire and database representation.
func (s ListingState) String() string { return string(s) }

// Valid reports whether s is a known listing state.
func (s ListingState) Valid() bool {
	_, ok := listingTransitions[s]
	return ok
}

// listingTransitions is the whole lifecycle, in one place:
//
//	DRAFT -> PENDING_ARTISAN_APPROVAL -> PUBLISHED -> SUSPENDED -> PUBLISHED
//
// Anything absent from this table is illegal, self-transitions included: a
// second approval of an already published listing is a client bug, not a no-op.
var listingTransitions = map[ListingState]map[ListingState]struct{}{
	StateDraft:                  {StatePendingArtisanApproval: {}},
	StatePendingArtisanApproval: {StatePublished: {}},
	StatePublished:              {StateSuspended: {}},
	StateSuspended:              {StatePublished: {}},
}

// CanTransitionTo reports whether next is a legal successor of s.
func (s ListingState) CanTransitionTo(next ListingState) bool {
	_, ok := listingTransitions[s][next]
	return ok
}

// ValidateTransition returns ErrInvalidInput naming the attempted pair when the
// move is not one the lifecycle allows.
func ValidateTransition(from, to ListingState) error {
	if !from.Valid() {
		return fmt.Errorf("unknown listing state %q: %w", from, pkgdomain.ErrInvalidInput)
	}
	if !to.Valid() {
		return fmt.Errorf("unknown listing state %q: %w", to, pkgdomain.ErrInvalidInput)
	}
	if !from.CanTransitionTo(to) {
		return fmt.Errorf("illegal listing state transition %s -> %s: %w", from, to, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// ValidateTransitionFrom checks a move that is only meaningful out of one
// state. Approval publishes a listing only from PENDING_ARTISAN_APPROVAL, even
// though SUSPENDED -> PUBLISHED is a legal edge reached by reinstatement; this
// is what keeps the two paths from standing in for each other.
func ValidateTransitionFrom(want, from, to ListingState) error {
	if from != want {
		return fmt.Errorf("illegal listing state transition %s -> %s: only a listing in %s may make this move: %w",
			from, to, want, pkgdomain.ErrInvalidInput)
	}
	return ValidateTransition(from, to)
}

// AttributeSource says who supplied an attribute value. Values match the
// attribute_source Postgres enum.
type AttributeSource string

const (
	// SourceModel is the vision or language stack's proposal.
	SourceModel AttributeSource = "MODEL"
	// SourceArtisan is the maker's own answer, which outranks the model's.
	SourceArtisan AttributeSource = "ARTISAN"
	// SourceCurator is a human curator's correction, which also outranks the model's.
	SourceCurator AttributeSource = "CURATOR"
)

// Valid reports whether s is a known attribute source.
func (s AttributeSource) Valid() bool {
	return s == SourceModel || s == SourceArtisan || s == SourceCurator
}

// Authoritative reports whether a value from this source may be overwritten by
// the model. Artisan and curator values never may.
func (s AttributeSource) Authoritative() bool { return s == SourceArtisan || s == SourceCurator }

// String returns the source's wire and database representation.
func (s AttributeSource) String() string { return string(s) }

// ListingMediaRole is the part a media asset plays on a listing. Values match
// the listing_media_role Postgres enum.
type ListingMediaRole string

const (
	// MediaRoleGallery is an ordinary gallery asset.
	MediaRoleGallery ListingMediaRole = "GALLERY"
	// MediaRolePrimaryImage is the one image used as the listing's thumbnail.
	MediaRolePrimaryImage ListingMediaRole = "PRIMARY_IMAGE"
	// MediaRoleProcessVideo is the one clip showing the craft being practised.
	MediaRoleProcessVideo ListingMediaRole = "PROCESS_VIDEO"
)

// Valid reports whether r is a known media role.
func (r ListingMediaRole) Valid() bool {
	return r == MediaRoleGallery || r == MediaRolePrimaryImage || r == MediaRoleProcessVideo
}

// String returns the role's wire and database representation.
func (r ListingMediaRole) String() string { return string(r) }

// MediaKind mirrors the media_kind Postgres enum, which the media rows carry.
type MediaKind string

const (
	// MediaImage is a still photograph.
	MediaImage MediaKind = "IMAGE"
	// MediaVideo is moving footage.
	MediaVideo MediaKind = "VIDEO"
	// MediaAudio is audio, typically a voice note.
	MediaAudio MediaKind = "AUDIO"
	// MediaDocument is a PDF or scanned certificate.
	MediaDocument MediaKind = "DOCUMENT"
)

// Dimensions is a measured extent in millimetres and grams.
type Dimensions struct {
	LengthMM *int32
	WidthMM  *int32
	HeightMM *int32
	WeightG  *int32
}

// Packaging drives freight quoting and the pre-checkout warning.
type Packaging struct {
	Fragile               bool
	Oversized             bool
	RequiresCustomCrating bool
	Packed                Dimensions
}

// Product is the physical item. One product may be offered by more than one
// listing over its life.
type Product struct {
	ID               uuid.UUID
	ArtisanID        uuid.UUID
	CraftID          uuid.UUID
	WorkingTitle     string
	Dimensions       Dimensions
	Materials        []string
	Techniques       []string
	Colours          []string
	Motifs           []string
	VoiceNoteMediaID *uuid.UUID
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CreateProductInput is what the service needs to register a product.
type CreateProductInput struct {
	ArtisanID        uuid.UUID
	CraftID          uuid.UUID
	WorkingTitle     string
	Dimensions       Dimensions
	Materials        []string
	Techniques       []string
	Colours          []string
	Motifs           []string
	VoiceNoteMediaID *uuid.UUID
	MediaIDs         []uuid.UUID
	CreatedBy        string
}

// maxWorkingTitle bounds the artisan's own title before AI copy replaces it.
const maxWorkingTitle = 200

// Validate checks the fields the schema constrains plus the ones it cannot.
func (in CreateProductInput) Validate() error {
	if in.ArtisanID == uuid.Nil {
		return fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.CraftID == uuid.Nil {
		return fmt.Errorf("craft_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.WorkingTitle) == "" {
		return fmt.Errorf("working_title is required: %w", pkgdomain.ErrInvalidInput)
	}
	if len(in.WorkingTitle) > maxWorkingTitle {
		return fmt.Errorf("working_title must be at most %d characters: %w", maxWorkingTitle, pkgdomain.ErrInvalidInput)
	}
	if err := in.Dimensions.validate("dimensions"); err != nil {
		return err
	}
	return uniqueUUIDs(in.MediaIDs, "media_ids")
}

// validate rejects a non-positive measurement, mirroring the table's check
// constraints so the caller gets a named field rather than a constraint name.
func (d Dimensions) validate(field string) error {
	for name, v := range map[string]*int32{
		"length_mm": d.LengthMM, "width_mm": d.WidthMM,
		"height_mm": d.HeightMM, "weight_g": d.WeightG,
	} {
		if v != nil && *v <= 0 {
			return fmt.Errorf("%s.%s must be positive: %w", field, name, pkgdomain.ErrInvalidInput)
		}
	}
	return nil
}

// Listing is the sellable offer: one product, one price, one commercial posture.
type Listing struct {
	ID               uuid.UUID
	ProductID        uuid.UUID
	ArtisanID        uuid.UUID
	Type             ListingType
	State            ListingState
	PricePaise       int64
	CurrencyCode     string
	StockQuantity    *int32
	MinOrderQuantity int32
	LeadTimeDays     *int32
	CapacityPerMonth *int32
	AcceptingOrders  bool
	AdvancePct       *int32
	Packaging        Packaging
	ProvenanceID     *uuid.UUID
	GICertified      bool
	// NeedsDescription is set when copy generation failed but everything above
	// it succeeded: the listing is a usable draft waiting on the artisan's own
	// words rather than a failure they can do nothing about.
	NeedsDescription bool
	PublishedAt      *time.Time
	SuspensionReason *string
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time

	Translations []ListingTranslation
	Attributes   []ListingAttribute
	Media        []ListingMedia
}

// ListingTranslation is buyer-facing copy for one listing in one language.
type ListingTranslation struct {
	ListingID        uuid.UUID
	Language         string
	Title            string
	Description      string
	Highlights       []string
	MachineGenerated bool
	EditedBy         *string
	UpdatedAt        time.Time
}

// Validate checks one translation before it is upserted.
func (t ListingTranslation) Validate() error {
	if t.Language == "" {
		return fmt.Errorf("translation language is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(t.Title) == "" {
		return fmt.Errorf("translation title for %s is required: %w", t.Language, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// ListingAttribute is one typed fact about a listing, with the confidence and
// the provenance of whoever asserted it.
type ListingAttribute struct {
	ID         uuid.UUID
	ListingID  uuid.UUID
	Name       string
	Value      string
	Confidence float32
	Source     AttributeSource
	CreatedAt  time.Time
}

// Validate checks one attribute before it is written.
func (a ListingAttribute) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("attribute name is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(a.Value) == "" {
		return fmt.Errorf("attribute %q has no value: %w", a.Name, pkgdomain.ErrInvalidInput)
	}
	if a.Confidence < 0 || a.Confidence > 1 {
		return fmt.Errorf("attribute %q has confidence %v, must be in [0,1]: %w",
			a.Name, a.Confidence, pkgdomain.ErrInvalidInput)
	}
	if !a.Source.Valid() {
		return fmt.Errorf("attribute %q has unknown source %q: %w", a.Name, a.Source, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// ListingMedia is one media asset attached to a listing, in display order.
type ListingMedia struct {
	ListingID uuid.UUID
	MediaID   uuid.UUID
	Ordinal   int32
	Role      ListingMediaRole
	Kind      MediaKind
}

// MediaOwnership is the slice of a media row the catalog needs in order to
// decide whether a listing may attach it: who uploaded it and what it is.
type MediaOwnership struct {
	MediaID   uuid.UUID
	ArtisanID uuid.UUID
	Kind      MediaKind
}

// UpsertListingInput creates a listing or replaces the mutable parts of one.
// A nil ListingID creates; a non-nil one updates that listing in place.
type UpsertListingInput struct {
	ListingID        *uuid.UUID
	ProductID        uuid.UUID
	Type             ListingType
	PricePaise       int64
	CurrencyCode     string
	StockQuantity    *int32
	MinOrderQuantity int32
	LeadTimeDays     *int32
	CapacityPerMonth *int32
	AcceptingOrders  bool
	AdvancePct       *int32
	Packaging        Packaging
	GICertified      bool
	Translations     []ListingTranslation
	CreatedBy        string
}

// Validate enforces the commercial rules the schema only half expresses: a
// made-to-order listing must be quotable and allocatable, ready stock must be
// decrementable, and an advance is a whole percentage of order value.
func (in UpsertListingInput) Validate() error {
	if in.ProductID == uuid.Nil {
		return fmt.Errorf("product_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if !in.Type.Valid() {
		return fmt.Errorf("listing type %q is not one of MADE_TO_ORDER, READY_STOCK: %w",
			in.Type, pkgdomain.ErrInvalidInput)
	}
	if in.PricePaise < 0 {
		return fmt.Errorf("price must not be negative: %w", pkgdomain.ErrInvalidInput)
	}
	if in.CurrencyCode != "" && in.CurrencyCode != "INR" {
		return fmt.Errorf("currency_code %q is not supported, only INR: %w", in.CurrencyCode, pkgdomain.ErrInvalidInput)
	}
	if in.MinOrderQuantity <= 0 {
		return fmt.Errorf("min_order_quantity must be positive: %w", pkgdomain.ErrInvalidInput)
	}
	if in.AdvancePct != nil && (*in.AdvancePct < 0 || *in.AdvancePct > 100) {
		return fmt.Errorf("advance_pct is %d, must be between 0 and 100: %w", *in.AdvancePct, pkgdomain.ErrInvalidInput)
	}
	if err := in.Packaging.Packed.validate("packaging.packed_dimensions"); err != nil {
		return err
	}

	switch in.Type {
	case ListingMadeToOrder:
		if in.LeadTimeDays == nil || *in.LeadTimeDays <= 0 {
			return fmt.Errorf("a MADE_TO_ORDER listing requires a positive lead_time_days: %w", pkgdomain.ErrInvalidInput)
		}
		if in.CapacityPerMonth == nil || *in.CapacityPerMonth <= 0 {
			return fmt.Errorf("a MADE_TO_ORDER listing requires a positive capacity_per_month: %w", pkgdomain.ErrInvalidInput)
		}
	case ListingReadyStock:
		if in.StockQuantity == nil {
			return fmt.Errorf("a READY_STOCK listing requires stock_quantity: %w", pkgdomain.ErrInvalidInput)
		}
		if *in.StockQuantity < 0 {
			return fmt.Errorf("stock_quantity must not be negative: %w", pkgdomain.ErrInvalidInput)
		}
	}

	for _, t := range in.Translations {
		if err := t.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ListingFilter is the set of narrowings ListListings accepts.
type ListingFilter struct {
	ArtisanID *uuid.UUID
	ClusterID *uuid.UUID
	CraftID   *uuid.UUID
	State     *ListingState
	Type      *ListingType
}

// AttachMediaInput replaces a listing's media set wholesale.
type AttachMediaInput struct {
	ListingID uuid.UUID
	Items     []ListingMedia
}

// Validate checks ordering and the two singleton roles. Media existence,
// ownership and kind are checked by the service, which can read the media rows.
func (in AttachMediaInput) Validate() error {
	if in.ListingID == uuid.Nil {
		return fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if len(in.Items) == 0 {
		return fmt.Errorf("at least one media item is required: %w", pkgdomain.ErrInvalidInput)
	}

	seenMedia := make(map[uuid.UUID]struct{}, len(in.Items))
	seenOrdinal := make(map[int32]struct{}, len(in.Items))
	var primaries, videos int
	for _, item := range in.Items {
		if item.MediaID == uuid.Nil {
			return fmt.Errorf("media_id must not be nil: %w", pkgdomain.ErrInvalidInput)
		}
		if _, dup := seenMedia[item.MediaID]; dup {
			return fmt.Errorf("media %s is attached twice: %w", item.MediaID, pkgdomain.ErrInvalidInput)
		}
		seenMedia[item.MediaID] = struct{}{}

		if item.Ordinal < 0 {
			return fmt.Errorf("ordinal for media %s must not be negative: %w", item.MediaID, pkgdomain.ErrInvalidInput)
		}
		if _, dup := seenOrdinal[item.Ordinal]; dup {
			return fmt.Errorf("ordinal %d is used twice: %w", item.Ordinal, pkgdomain.ErrInvalidInput)
		}
		seenOrdinal[item.Ordinal] = struct{}{}

		if !item.Role.Valid() {
			return fmt.Errorf("media %s has unknown role %q: %w", item.MediaID, item.Role, pkgdomain.ErrInvalidInput)
		}
		switch item.Role {
		case MediaRolePrimaryImage:
			primaries++
		case MediaRoleProcessVideo:
			videos++
		}
	}
	if primaries > 1 {
		return fmt.Errorf("a listing may designate only one primary image, got %d: %w", primaries, pkgdomain.ErrInvalidInput)
	}
	if videos > 1 {
		return fmt.Errorf("a listing may designate only one process video, got %d: %w", videos, pkgdomain.ErrInvalidInput)
	}
	return nil
}
