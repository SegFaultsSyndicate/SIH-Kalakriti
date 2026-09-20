// services/core-svc/internal/core/service/catalog.go
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// CatalogTx is the transactional surface the catalog writes through. It embeds
// outbox.Enqueuer so a publish and its event commit together.
type CatalogTx interface {
	outbox.Enqueuer

	CreateProduct(ctx context.Context, id uuid.UUID, in domain.CreateProductInput) (domain.Product, error)
	AttachProductMedia(ctx context.Context, productID, artisanID uuid.UUID, mediaIDs []uuid.UUID) error

	CreateListing(ctx context.Context, id, artisanID uuid.UUID, in domain.UpsertListingInput) (domain.Listing, error)
	UpdateListing(ctx context.Context, in domain.UpsertListingInput) (domain.Listing, error)
	// TransitionListingState moves a listing from the state the caller believes
	// it is in; a zero-row update means another writer got there first, which
	// the repo reports as ErrConflict.
	TransitionListingState(ctx context.Context, listingID uuid.UUID, from, to domain.ListingState, reason *string) (domain.Listing, error)

	UpsertListingTranslation(ctx context.Context, t domain.ListingTranslation) (domain.ListingTranslation, error)
	ListListingAttributes(ctx context.Context, listingID uuid.UUID) ([]domain.ListingAttribute, error)
	UpsertListingAttribute(ctx context.Context, a domain.ListingAttribute) error
	// DeleteListingAttributesByName removes every value stored for one attribute
	// name whose source is not authoritative, which is how an artisan's answer
	// displaces the model's guesses for that name.
	DeleteListingAttributesByName(ctx context.Context, listingID uuid.UUID, name string) (int64, error)
	ReplaceListingMedia(ctx context.Context, listingID uuid.UUID, items []domain.ListingMedia) error
}

// CatalogStore is the catalog's read surface plus its transaction entry point.
type CatalogStore interface {
	InTx(ctx context.Context, fn func(ctx context.Context, tx CatalogTx) error) error

	GetProduct(ctx context.Context, id uuid.UUID) (domain.Product, error)
	GetListing(ctx context.Context, id uuid.UUID) (domain.Listing, error)
	GetListingDetail(ctx context.Context, id uuid.UUID) (domain.Listing, error)
	ListListings(ctx context.Context, filter domain.ListingFilter, page domain.Page) ([]domain.Listing, error)
	ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]domain.ListingMedia, error)
	ListListingAttributes(ctx context.Context, listingID uuid.UUID) ([]domain.ListingAttribute, error)
	ListMediaOwnership(ctx context.Context, mediaIDs []uuid.UUID) ([]domain.MediaOwnership, error)
	GetSHGForArtisan(ctx context.Context, artisanID uuid.UUID) (domain.SelfHelpGroup, error)
}

// CraftIndex is the slice of the ontology registry the catalog needs: checking
// that a declared craft exists, without a database round trip on every write.
type CraftIndex interface {
	Craft(id uuid.UUID) (domain.Craft, bool)
}

// Catalog is core-svc's product and listing service. Every authorisation
// decision about a listing is made here, never in a handler.
type Catalog struct {
	store  CatalogStore
	crafts CraftIndex
	log    *slog.Logger
	now    func() time.Time
}

// NewCatalog builds the catalog service.
func NewCatalog(store CatalogStore, crafts CraftIndex, log *slog.Logger) *Catalog {
	return &Catalog{store: store, crafts: crafts, log: log, now: time.Now}
}

// listingMediaAttached mirrors events.v1.ListingMediaAttached's payload
// fields -- what triggers the cataloguing pipeline.
type listingMediaAttached struct {
	ListingID string `json:"listing_id"`
	ProductID string `json:"product_id"`
	ArtisanID string `json:"artisan_id"`
}

// listingPublished mirrors events.v1.CatalogListingPublished's payload fields.
type listingPublished struct {
	ListingID    string   `json:"listing_id"`
	ProductID    string   `json:"product_id"`
	ArtisanID    string   `json:"artisan_id"`
	CraftID      string   `json:"craft_id"`
	Type         string   `json:"type"`
	PricePaise   int64    `json:"price_paise"`
	CurrencyCode string   `json:"currency_code"`
	Languages    []string `json:"languages"`
	GICertified  bool     `json:"gi_certified"`
	Reinstated   bool     `json:"reinstated"`
}

// newCatalogEvent builds an outbox envelope for one aggregate.
func (s *Catalog) newCatalogEvent(aggregateID uuid.UUID, idempotencyKey string, payload any) event {
	return event{
		Header: eventHeader{
			EventID:        ids.New().String(),
			OccurredAt:     s.now().UTC(),
			AggregateID:    aggregateID.String(),
			IdempotencyKey: idempotencyKey,
			SchemaVersion:  schemaVersion,
			Producer:       producerName,
		},
		Payload: payload,
	}
}

// --- product -----------------------------------------------------------------

// CreateProduct registers a physical item against an artisan. The craft is
// checked against the in-memory ontology rather than the database, so an unknown
// craft fails before a transaction opens.
func (s *Catalog) CreateProduct(ctx context.Context, in domain.CreateProductInput, idempotencyKey string) (domain.Product, error) {
	if idempotencyKey == "" {
		return domain.Product{}, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}
	principal, err := s.authoriseFor(ctx, in.ArtisanID)
	if err != nil {
		return domain.Product{}, err
	}
	craft, ok := s.crafts.Craft(in.CraftID)
	if !ok {
		return domain.Product{}, fmt.Errorf("craft %s is not in the ontology: %w", in.CraftID, pkgdomain.ErrInvalidInput)
	}
	// The artisan app's story step (Batch 8's UI) deliberately marks this
	// field optional -- the whole product leans on AI-assisted copy rather
	// than asking a low-literacy artisan to type one -- but Validate below
	// still hard-required it, matching neither the frontend's own UI nor
	// maxWorkingTitle's doc comment ("bounds the artisan's own title before
	// AI copy replaces it", implying a missing one is expected and meant to
	// be replaced, not rejected). Confirmed live: leaving it blank left the
	// listing-creation step permanently stuck with no visible error.
	// Defaulting to the craft's own display name keeps every downstream
	// consumer of WorkingTitle (search indexing, the artisan's own listing
	// list) meaningful until GenerateDescription's copy replaces it.
	if strings.TrimSpace(in.WorkingTitle) == "" {
		in.WorkingTitle = craft.DisplayName
	}
	if err := in.Validate(); err != nil {
		return domain.Product{}, err
	}
	if in.CreatedBy == "" {
		in.CreatedBy = principal.Subject
	}

	productID := ids.New()
	var created domain.Product
	err = s.store.InTx(ctx, func(ctx context.Context, tx CatalogTx) error {
		var err error
		if created, err = tx.CreateProduct(ctx, productID, in); err != nil {
			return err
		}
		if len(in.MediaIDs) > 0 {
			return tx.AttachProductMedia(ctx, productID, in.ArtisanID, in.MediaIDs)
		}
		return nil
	})
	if err != nil {
		return domain.Product{}, err
	}

	s.log.InfoContext(ctx, "product created",
		"product_id", productID, "artisan_id", in.ArtisanID, "craft_id", in.CraftID)
	return created, nil
}

// GetProduct reads one product.
func (s *Catalog) GetProduct(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	if id == uuid.Nil {
		return domain.Product{}, fmt.Errorf("product_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if _, err := auth.RequirePrincipal(ctx); err != nil {
		return domain.Product{}, err
	}
	return s.store.GetProduct(ctx, id)
}

// --- listing -----------------------------------------------------------------

// UpsertListing creates the offer for a product, or replaces the mutable parts
// of an existing one. A new listing starts in DRAFT; an existing listing keeps
// whatever state it is in, and a suspended one is refused outright.
func (s *Catalog) UpsertListing(ctx context.Context, in domain.UpsertListingInput, idempotencyKey string) (domain.Listing, error) {
	if err := in.Validate(); err != nil {
		return domain.Listing{}, err
	}
	if idempotencyKey == "" {
		return domain.Listing{}, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}

	product, err := s.store.GetProduct(ctx, in.ProductID)
	if err != nil {
		return domain.Listing{}, err
	}
	principal, err := s.authoriseFor(ctx, product.ArtisanID)
	if err != nil {
		return domain.Listing{}, err
	}
	if in.CreatedBy == "" {
		in.CreatedBy = principal.Subject
	}

	if in.ListingID != nil {
		existing, err := s.store.GetListing(ctx, *in.ListingID)
		if err != nil {
			return domain.Listing{}, err
		}
		if existing.ProductID != in.ProductID {
			return domain.Listing{}, fmt.Errorf(
				"listing %s offers product %s, not %s: %w",
				existing.ID, existing.ProductID, in.ProductID, pkgdomain.ErrInvalidInput)
		}
		if existing.State == domain.StateSuspended {
			return domain.Listing{}, fmt.Errorf(
				"listing %s is suspended and cannot be edited until it is reinstated: %w",
				existing.ID, pkgdomain.ErrConflict)
		}
	}

	listingID := ids.New()
	if in.ListingID != nil {
		listingID = *in.ListingID
	}

	var stored domain.Listing
	err = s.store.InTx(ctx, func(ctx context.Context, tx CatalogTx) error {
		var err error
		if in.ListingID == nil {
			stored, err = tx.CreateListing(ctx, listingID, product.ArtisanID, in)
		} else {
			stored, err = tx.UpdateListing(ctx, in)
		}
		if err != nil {
			return err
		}
		for _, t := range in.Translations {
			t.ListingID = listingID
			if _, err := tx.UpsertListingTranslation(ctx, t); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.Listing{}, err
	}

	s.log.InfoContext(ctx, "listing upserted",
		"listing_id", listingID, "product_id", in.ProductID, "state", stored.State, "type", in.Type)
	return s.store.GetListingDetail(ctx, listingID)
}

// SubmitForApproval hands a draft to the artisan for sign-off. Field staff may
// submit on an artisan's behalf, which is how a listing assembled at a cluster
// desk reaches the maker's phone.
func (s *Catalog) SubmitForApproval(ctx context.Context, listingID uuid.UUID, idempotencyKey string) (domain.Listing, error) {
	if listingID == uuid.Nil {
		return domain.Listing{}, fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if idempotencyKey == "" {
		return domain.Listing{}, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}

	listing, err := s.store.GetListingDetail(ctx, listingID)
	if err != nil {
		return domain.Listing{}, err
	}
	if _, err := s.authoriseFor(ctx, listing.ArtisanID, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return domain.Listing{}, err
	}
	if err := domain.ValidateTransition(listing.State, domain.StatePendingArtisanApproval); err != nil {
		return domain.Listing{}, err
	}
	// Asking an artisan to approve a listing with no copy in any language is
	// asking them to approve a blank page.
	if len(listing.Translations) == 0 {
		return domain.Listing{}, fmt.Errorf(
			"listing %s has no copy to review: %w", listingID, pkgdomain.ErrInvalidInput)
	}
	// A DRAFT is allowed to still be missing its type and type-specific
	// commercial fields (see migrations/035_listing_draft_type.sql); leaving
	// DRAFT is not.
	if err := listing.ValidateComplete(); err != nil {
		return domain.Listing{}, err
	}

	err = s.store.InTx(ctx, func(ctx context.Context, tx CatalogTx) error {
		_, err := tx.TransitionListingState(ctx, listingID, listing.State, domain.StatePendingArtisanApproval, nil)
		return err
	})
	if err != nil {
		return domain.Listing{}, err
	}

	s.log.InfoContext(ctx, "listing submitted for approval", "listing_id", listingID)
	return s.store.GetListingDetail(ctx, listingID)
}

// ApproveListing records the artisan's sign-off and publishes. Only the owning
// artisan, or the signatory of the self-help group they belong to, may approve:
// the whole point of the state is that a human maker accepted the generated
// copy, so an operator cannot stand in for them.
//
// The state change, the edited copy and the catalog.listing.published outbox row
// all commit in one transaction, so a published listing is never missing from
// the event stream and an event never precedes the row it describes.
func (s *Catalog) ApproveListing(
	ctx context.Context,
	listingID, artisanID uuid.UUID,
	editedTranslations []domain.ListingTranslation,
	idempotencyKey string,
) (domain.Listing, error) {
	if listingID == uuid.Nil {
		return domain.Listing{}, fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if idempotencyKey == "" {
		return domain.Listing{}, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}
	for _, t := range editedTranslations {
		if err := t.Validate(); err != nil {
			return domain.Listing{}, err
		}
	}

	listing, err := s.store.GetListingDetail(ctx, listingID)
	if err != nil {
		return domain.Listing{}, err
	}
	if artisanID != uuid.Nil && artisanID != listing.ArtisanID {
		return domain.Listing{}, fmt.Errorf(
			"listing %s belongs to artisan %s, not %s: %w",
			listingID, listing.ArtisanID, artisanID, pkgdomain.ErrInvalidInput)
	}

	principal, err := s.authoriseFor(ctx, listing.ArtisanID)
	if err != nil {
		return domain.Listing{}, err
	}
	if err := domain.ValidateTransitionFrom(domain.StatePendingArtisanApproval, listing.State, domain.StatePublished); err != nil {
		return domain.Listing{}, err
	}

	product, err := s.store.GetProduct(ctx, listing.ProductID)
	if err != nil {
		return domain.Listing{}, err
	}

	err = s.store.InTx(ctx, func(ctx context.Context, tx CatalogTx) error {
		for _, t := range editedTranslations {
			t.ListingID = listingID
			t.MachineGenerated = false
			editor := principal.Subject
			t.EditedBy = &editor
			if _, err := tx.UpsertListingTranslation(ctx, t); err != nil {
				return err
			}
		}

		published, err := tx.TransitionListingState(ctx, listingID, listing.State, domain.StatePublished, nil)
		if err != nil {
			return err
		}

		return outbox.Enqueue(ctx, tx,
			ids.New().String(),
			listingID.String(),
			topics.CatalogListingPublished,
			idempotencyKey,
			s.newCatalogEvent(listingID, idempotencyKey, listingPublished{
				ListingID:    listingID.String(),
				ProductID:    listing.ProductID.String(),
				ArtisanID:    listing.ArtisanID.String(),
				CraftID:      product.CraftID.String(),
				Type:         published.Type.String(),
				PricePaise:   published.PricePaise,
				CurrencyCode: published.CurrencyCode,
				Languages:    translationLanguages(listing.Translations, editedTranslations),
				GICertified:  published.GICertified,
			}),
		)
	})
	if err != nil {
		return domain.Listing{}, err
	}

	s.log.InfoContext(ctx, "listing published",
		"listing_id", listingID, "artisan_id", listing.ArtisanID, "approved_by", principal.Subject)
	return s.store.GetListingDetail(ctx, listingID)
}

// SuspendListing withdraws a published listing. Suspension is an operator
// action: an artisan who wants to stop selling pauses orders instead.
func (s *Catalog) SuspendListing(ctx context.Context, listingID uuid.UUID, reason, idempotencyKey string) (domain.Listing, error) {
	if listingID == uuid.Nil {
		return domain.Listing{}, fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if reason == "" {
		return domain.Listing{}, fmt.Errorf("a suspension reason is required: %w", pkgdomain.ErrInvalidInput)
	}
	if idempotencyKey == "" {
		return domain.Listing{}, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}
	if _, err := auth.RequireRole(ctx, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return domain.Listing{}, err
	}

	listing, err := s.store.GetListing(ctx, listingID)
	if err != nil {
		return domain.Listing{}, err
	}
	if err := domain.ValidateTransition(listing.State, domain.StateSuspended); err != nil {
		return domain.Listing{}, err
	}

	err = s.store.InTx(ctx, func(ctx context.Context, tx CatalogTx) error {
		_, err := tx.TransitionListingState(ctx, listingID, listing.State, domain.StateSuspended, &reason)
		return err
	})
	if err != nil {
		return domain.Listing{}, err
	}

	s.log.InfoContext(ctx, "listing suspended", "listing_id", listingID, "reason", reason)
	return s.store.GetListingDetail(ctx, listingID)
}

// ReinstateListing returns a suspended listing to PUBLISHED and re-emits
// catalog.listing.published, because search dropped it when it was suspended.
func (s *Catalog) ReinstateListing(ctx context.Context, listingID uuid.UUID, idempotencyKey string) (domain.Listing, error) {
	if listingID == uuid.Nil {
		return domain.Listing{}, fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if idempotencyKey == "" {
		return domain.Listing{}, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}
	if _, err := auth.RequireRole(ctx, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return domain.Listing{}, err
	}

	listing, err := s.store.GetListingDetail(ctx, listingID)
	if err != nil {
		return domain.Listing{}, err
	}
	if err := domain.ValidateTransitionFrom(domain.StateSuspended, listing.State, domain.StatePublished); err != nil {
		return domain.Listing{}, err
	}
	product, err := s.store.GetProduct(ctx, listing.ProductID)
	if err != nil {
		return domain.Listing{}, err
	}

	err = s.store.InTx(ctx, func(ctx context.Context, tx CatalogTx) error {
		published, err := tx.TransitionListingState(ctx, listingID, listing.State, domain.StatePublished, nil)
		if err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx,
			ids.New().String(),
			listingID.String(),
			topics.CatalogListingPublished,
			idempotencyKey,
			s.newCatalogEvent(listingID, idempotencyKey, listingPublished{
				ListingID:    listingID.String(),
				ProductID:    listing.ProductID.String(),
				ArtisanID:    listing.ArtisanID.String(),
				CraftID:      product.CraftID.String(),
				Type:         published.Type.String(),
				PricePaise:   published.PricePaise,
				CurrencyCode: published.CurrencyCode,
				Languages:    translationLanguages(listing.Translations, nil),
				GICertified:  published.GICertified,
				Reinstated:   true,
			}),
		)
	})
	if err != nil {
		return domain.Listing{}, err
	}

	s.log.InfoContext(ctx, "listing reinstated", "listing_id", listingID)
	return s.store.GetListingDetail(ctx, listingID)
}

// GetListing reads one listing with its copy, attributes and media. A listing
// that is not published is visible only to the people who can edit it.
func (s *Catalog) GetListing(ctx context.Context, listingID uuid.UUID) (domain.Listing, error) {
	if listingID == uuid.Nil {
		return domain.Listing{}, fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}

	listing, err := s.store.GetListingDetail(ctx, listingID)
	if err != nil {
		return domain.Listing{}, err
	}
	if listing.State != domain.StatePublished {
		if _, err := s.authoriseFor(ctx, listing.ArtisanID, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
			// Told apart only for callers already entitled to know it exists.
			return domain.Listing{}, fmt.Errorf("listing %s not found: %w", listingID, pkgdomain.ErrNotFound)
		}
	}

	listing.Media, err = s.store.ListListingMedia(ctx, listingID)
	if err != nil {
		return domain.Listing{}, err
	}
	return listing, nil
}

// GetListingAttributes reads a listing's attributes -- what the model
// inferred, and what the artisan has since overridden. Same visibility rule
// as the listing itself: public once PUBLISHED, otherwise only the owning
// artisan or a curator.
func (s *Catalog) GetListingAttributes(ctx context.Context, listingID uuid.UUID) ([]domain.ListingAttribute, error) {
	if listingID == uuid.Nil {
		return nil, fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	listing, err := s.store.GetListing(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if listing.State != domain.StatePublished {
		if _, err := s.authoriseFor(ctx, listing.ArtisanID, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
			return nil, fmt.Errorf("listing %s not found: %w", listingID, pkgdomain.ErrNotFound)
		}
	}
	return s.store.ListListingAttributes(ctx, listingID)
}

// ListListings pages listings under the usual filters. Anything other than a
// published-only query is an operator or owner view and is authorised as one.
func (s *Catalog) ListListings(ctx context.Context, filter domain.ListingFilter, page domain.Page) ([]domain.Listing, error) {
	// An anonymous caller (buyer browsing) is a principal-less Principal{},
	// which HasRole/Subject-equality both fail safely closed on below --
	// falling through to the published-only clamp.
	principal, _ := auth.PrincipalFrom(ctx)

	published := domain.StatePublished
	if filter.State == nil || *filter.State != published {
		if filter.ArtisanID == nil || filter.ArtisanID.String() != principal.Subject {
			if !principal.HasRole(auth.RoleClusterOfficer, auth.RoleMinistry) {
				// A buyer browsing may only ever see live listings.
				filter.State = &published
			}
		}
	}
	return s.store.ListListings(ctx, filter, page)
}

// --- attributes, copy and media ----------------------------------------------

// UpsertListingAttributes stores typed attribute rows for a listing.
//
// Precedence is the rule this method exists for: a value the artisan (or a
// curator acting for them) supplied is never overwritten by the model. A model
// write for an attribute name that already carries an artisan value is dropped;
// an artisan write for a name displaces whatever the model guessed for it.
func (s *Catalog) UpsertListingAttributes(
	ctx context.Context,
	listingID uuid.UUID,
	attrs []domain.ListingAttribute,
	source domain.AttributeSource,
	idempotencyKey string,
) ([]domain.ListingAttribute, error) {
	if listingID == uuid.Nil {
		return nil, fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if len(attrs) == 0 {
		return nil, fmt.Errorf("at least one attribute is required: %w", pkgdomain.ErrInvalidInput)
	}
	if !source.Valid() {
		return nil, fmt.Errorf("unknown attribute source %q: %w", source, pkgdomain.ErrInvalidInput)
	}
	if idempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}
	for i := range attrs {
		attrs[i].ListingID = listingID
		attrs[i].Source = source
		if err := attrs[i].Validate(); err != nil {
			return nil, err
		}
	}

	listing, err := s.store.GetListing(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if _, err := s.authoriseFor(ctx, listing.ArtisanID, auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return nil, err
	}

	var stored []domain.ListingAttribute
	err = s.store.InTx(ctx, func(ctx context.Context, tx CatalogTx) error {
		existing, err := tx.ListListingAttributes(ctx, listingID)
		if err != nil {
			return err
		}

		// Names the artisan or a curator has already spoken for.
		authoritative := make(map[string]struct{}, len(existing))
		for _, a := range existing {
			if a.Source.Authoritative() {
				authoritative[a.Name] = struct{}{}
			}
		}

		displaced := make(map[string]struct{}, len(attrs))
		for _, a := range attrs {
			if !source.Authoritative() {
				if _, held := authoritative[a.Name]; held {
					s.log.DebugContext(ctx, "model attribute ignored, artisan value stands",
						"listing_id", listingID, "name", a.Name, "value", a.Value)
					continue
				}
			} else if _, done := displaced[a.Name]; !done {
				// One delete per name, before the first value for that name is
				// written, clears the model's guesses without touching the
				// artisan's other answers.
				if _, err := tx.DeleteListingAttributesByName(ctx, listingID, a.Name); err != nil {
					return err
				}
				displaced[a.Name] = struct{}{}
			}

			a.ID = ids.New()
			if err := tx.UpsertListingAttribute(ctx, a); err != nil {
				return err
			}
		}

		stored, err = tx.ListListingAttributes(ctx, listingID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return stored, nil
}

// UpsertListingTranslation stores buyer-facing copy for one language. Copy the
// artisan edited is marked as such, so the pipeline never overwrites it silently.
func (s *Catalog) UpsertListingTranslation(
	ctx context.Context,
	listingID uuid.UUID,
	t domain.ListingTranslation,
	idempotencyKey string,
) (domain.ListingTranslation, error) {
	if listingID == uuid.Nil {
		return domain.ListingTranslation{}, fmt.Errorf("listing_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if idempotencyKey == "" {
		return domain.ListingTranslation{}, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}
	t.ListingID = listingID
	if err := t.Validate(); err != nil {
		return domain.ListingTranslation{}, err
	}

	listing, err := s.store.GetListing(ctx, listingID)
	if err != nil {
		return domain.ListingTranslation{}, err
	}
	principal, err := s.authoriseFor(ctx, listing.ArtisanID, auth.RoleClusterOfficer, auth.RoleMinistry)
	if err != nil {
		return domain.ListingTranslation{}, err
	}
	if !t.MachineGenerated && t.EditedBy == nil {
		editor := principal.Subject
		t.EditedBy = &editor
	}

	var stored domain.ListingTranslation
	err = s.store.InTx(ctx, func(ctx context.Context, tx CatalogTx) error {
		var err error
		stored, err = tx.UpsertListingTranslation(ctx, t)
		return err
	})
	if err != nil {
		return domain.ListingTranslation{}, err
	}
	return stored, nil
}

// AttachListingMedia replaces a listing's media set: ordered, with at most one
// primary image and at most one process video. Every asset must already belong
// to the listing's artisan, and the two designated roles must match the asset's
// kind — a process "video" that is actually a JPEG would break the provenance
// review queue in batch 7.
func (s *Catalog) AttachListingMedia(ctx context.Context, in domain.AttachMediaInput, idempotencyKey string) ([]domain.ListingMedia, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if idempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required: %w", pkgdomain.ErrInvalidInput)
	}

	listing, err := s.store.GetListing(ctx, in.ListingID)
	if err != nil {
		return nil, err
	}
	if _, err := s.authoriseFor(ctx, listing.ArtisanID); err != nil {
		return nil, err
	}

	mediaIDs := make([]uuid.UUID, 0, len(in.Items))
	for _, item := range in.Items {
		mediaIDs = append(mediaIDs, item.MediaID)
	}
	owned, err := s.store.ListMediaOwnership(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]domain.MediaOwnership, len(owned))
	for _, m := range owned {
		byID[m.MediaID] = m
	}

	items := make([]domain.ListingMedia, 0, len(in.Items))
	for _, item := range in.Items {
		m, ok := byID[item.MediaID]
		if !ok {
			return nil, fmt.Errorf("media %s not found: %w", item.MediaID, pkgdomain.ErrNotFound)
		}
		if m.ArtisanID != listing.ArtisanID {
			return nil, fmt.Errorf("media %s belongs to another artisan: %w", item.MediaID, pkgdomain.ErrForbidden)
		}
		switch item.Role {
		case domain.MediaRolePrimaryImage:
			if m.Kind != domain.MediaImage {
				return nil, fmt.Errorf("primary image %s is a %s, not an image: %w",
					item.MediaID, m.Kind, pkgdomain.ErrInvalidInput)
			}
		case domain.MediaRoleProcessVideo:
			if m.Kind != domain.MediaVideo {
				return nil, fmt.Errorf("process video %s is a %s, not a video: %w",
					item.MediaID, m.Kind, pkgdomain.ErrInvalidInput)
			}
		}
		item.ListingID = in.ListingID
		item.Kind = m.Kind
		items = append(items, item)
	}

	err = s.store.InTx(ctx, func(ctx context.Context, tx CatalogTx) error {
		if err := tx.ReplaceListingMedia(ctx, in.ListingID, items); err != nil {
			return err
		}
		// Triggers the cataloguing pipeline: enhance every attached photo,
		// extract attributes, draft a description. Fired on every successful
		// attach (not just the first), since re-attaching media is itself a
		// reasonable signal to re-enrich, and every pipeline step downstream
		// is an idempotent upsert.
		return outbox.Enqueue(ctx, tx,
			ids.New().String(),
			in.ListingID.String(),
			topics.CatalogListingMediaAttached,
			idempotencyKey,
			s.newCatalogEvent(in.ListingID, idempotencyKey, listingMediaAttached{
				ListingID: in.ListingID.String(),
				ProductID: listing.ProductID.String(),
				ArtisanID: listing.ArtisanID.String(),
			}))
	})
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "listing media attached", "listing_id", in.ListingID, "items", len(items))
	return s.store.ListListingMedia(ctx, in.ListingID)
}

// --- authorisation -----------------------------------------------------------

// authoriseFor allows the artisan themselves, the signatory of the self-help
// group that artisan belongs to, and any caller holding one of the override
// roles. The SHG signatory is the "member with authority" the collective model
// needs: in most clusters one literate member transacts for the group.
func (s *Catalog) authoriseFor(ctx context.Context, artisanID uuid.UUID, override ...auth.Role) (auth.Principal, error) {
	principal, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return auth.Principal{}, err
	}
	if principal.Subject != "" && principal.Subject == artisanID.String() {
		return principal, nil
	}
	if len(override) > 0 && principal.HasRole(override...) {
		return principal, nil
	}

	group, err := s.store.GetSHGForArtisan(ctx, artisanID)
	if err != nil {
		if errors.Is(err, pkgdomain.ErrNotFound) {
			return auth.Principal{}, fmt.Errorf(
				"only artisan %s may act on this listing: %w", artisanID, pkgdomain.ErrForbidden)
		}
		return auth.Principal{}, err
	}
	if group.SignatoryArtisanID != nil && group.SignatoryArtisanID.String() == principal.Subject {
		return principal, nil
	}
	return auth.Principal{}, fmt.Errorf(
		"neither artisan %s nor the signatory of their self-help group: %w", artisanID, pkgdomain.ErrForbidden)
}

// translationLanguages collects the languages a listing has copy in, with the
// edits applied in this call folded in.
func translationLanguages(stored, edited []domain.ListingTranslation) []string {
	seen := make(map[string]struct{}, len(stored)+len(edited))
	out := make([]string, 0, len(stored)+len(edited))
	for _, set := range [][]domain.ListingTranslation{stored, edited} {
		for _, t := range set {
			if _, dup := seen[t.Language]; dup {
				continue
			}
			seen[t.Language] = struct{}{}
			out = append(out, t.Language)
		}
	}
	return out
}
