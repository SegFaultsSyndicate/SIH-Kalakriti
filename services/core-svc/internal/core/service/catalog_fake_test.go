// services/core-svc/internal/core/service/catalog_fake_test.go
package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// fakeCatalogStore is an in-memory CatalogStore + CatalogTx. Like fakeStore it
// buffers writes and applies them only on commit, so a failed callback leaves no
// trace, and it enforces the one database behaviour the service depends on: the
// guarded state transition, which updates nothing when the expected state has
// moved under it.
type fakeCatalogStore struct {
	mu sync.Mutex

	products   map[uuid.UUID]domain.Product
	listings   map[uuid.UUID]domain.Listing
	attributes map[uuid.UUID][]domain.ListingAttribute
	medias     map[uuid.UUID][]domain.ListingMedia
	ownership  map[uuid.UUID]domain.MediaOwnership
	// shgByArtisan mirrors GetShgForArtisan: at most one group per artisan.
	shgByArtisan map[uuid.UUID]domain.SelfHelpGroup
	outbox       []outboxRow
}

func newFakeCatalogStore() *fakeCatalogStore {
	return &fakeCatalogStore{
		products:     map[uuid.UUID]domain.Product{},
		listings:     map[uuid.UUID]domain.Listing{},
		attributes:   map[uuid.UUID][]domain.ListingAttribute{},
		medias:       map[uuid.UUID][]domain.ListingMedia{},
		ownership:    map[uuid.UUID]domain.MediaOwnership{},
		shgByArtisan: map[uuid.UUID]domain.SelfHelpGroup{},
	}
}

type fakeCatalogTx struct {
	store  *fakeCatalogStore
	writes []func()
	// pendingAttrs serves read-your-own-writes inside one transaction.
	pendingAttrs map[uuid.UUID][]domain.ListingAttribute
}

func (s *fakeCatalogStore) InTx(ctx context.Context, fn func(ctx context.Context, tx CatalogTx) error) error {
	tx := &fakeCatalogTx{store: s, pendingAttrs: map[uuid.UUID][]domain.ListingAttribute{}}
	if err := fn(ctx, tx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range tx.writes {
		w()
	}
	return nil
}

// --- reads -------------------------------------------------------------------

func (s *fakeCatalogStore) GetProduct(_ context.Context, id uuid.UUID) (domain.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[id]
	if !ok {
		return domain.Product{}, fmt.Errorf("product not found: %w", pkgdomain.ErrNotFound)
	}
	return p, nil
}

func (s *fakeCatalogStore) GetListing(_ context.Context, id uuid.UUID) (domain.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.listings[id]
	if !ok {
		return domain.Listing{}, fmt.Errorf("listing not found: %w", pkgdomain.ErrNotFound)
	}
	return l, nil
}

func (s *fakeCatalogStore) GetListingDetail(ctx context.Context, id uuid.UUID) (domain.Listing, error) {
	l, err := s.GetListing(ctx, id)
	if err != nil {
		return domain.Listing{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	l.Attributes = append([]domain.ListingAttribute(nil), s.attributes[id]...)
	l.Media = append([]domain.ListingMedia(nil), s.medias[id]...)
	return l, nil
}

func (s *fakeCatalogStore) ListListings(_ context.Context, filter domain.ListingFilter, page domain.Page) ([]domain.Listing, error) {
	page = page.Normalise()
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]domain.Listing, 0, len(s.listings))
	for _, l := range s.listings {
		if filter.ArtisanID != nil && l.ArtisanID != *filter.ArtisanID {
			continue
		}
		if filter.State != nil && l.State != *filter.State {
			continue
		}
		if filter.Type != nil && l.Type != *filter.Type {
			continue
		}
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID.String() < out[j].ID.String() })
	if len(out) > int(page.Size) {
		out = out[:page.Size]
	}
	return out, nil
}

func (s *fakeCatalogStore) ListListingMedia(_ context.Context, listingID uuid.UUID) ([]domain.ListingMedia, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.ListingMedia(nil), s.medias[listingID]...), nil
}

func (s *fakeCatalogStore) ListListingAttributes(_ context.Context, listingID uuid.UUID) ([]domain.ListingAttribute, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.ListingAttribute(nil), s.attributes[listingID]...), nil
}

func (s *fakeCatalogStore) ListMediaOwnership(_ context.Context, ids []uuid.UUID) ([]domain.MediaOwnership, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.MediaOwnership, 0, len(ids))
	for _, id := range ids {
		if m, ok := s.ownership[id]; ok {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *fakeCatalogStore) GetSHGForArtisan(_ context.Context, artisanID uuid.UUID) (domain.SelfHelpGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.shgByArtisan[artisanID]
	if !ok {
		return domain.SelfHelpGroup{}, fmt.Errorf("self-help group not found: %w", pkgdomain.ErrNotFound)
	}
	return g, nil
}

// --- writes ------------------------------------------------------------------

func (t *fakeCatalogTx) CreateProduct(_ context.Context, id uuid.UUID, in domain.CreateProductInput) (domain.Product, error) {
	p := domain.Product{
		ID:           id,
		ArtisanID:    in.ArtisanID,
		CraftID:      in.CraftID,
		WorkingTitle: in.WorkingTitle,
		Dimensions:   in.Dimensions,
		Materials:    in.Materials,
		Techniques:   in.Techniques,
		CreatedBy:    in.CreatedBy,
		CreatedAt:    fakeNow,
		UpdatedAt:    fakeNow,
	}
	t.writes = append(t.writes, func() { t.store.products[id] = p })
	return p, nil
}

func (t *fakeCatalogTx) AttachProductMedia(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID) error {
	return nil
}

func (t *fakeCatalogTx) CreateListing(_ context.Context, id, artisanID uuid.UUID, in domain.UpsertListingInput) (domain.Listing, error) {
	l := listingFromInput(id, artisanID, in)
	t.writes = append(t.writes, func() { t.store.listings[id] = l })
	return l, nil
}

func (t *fakeCatalogTx) UpdateListing(_ context.Context, in domain.UpsertListingInput) (domain.Listing, error) {
	t.store.mu.Lock()
	existing, ok := t.store.listings[*in.ListingID]
	t.store.mu.Unlock()
	if !ok {
		return domain.Listing{}, fmt.Errorf("listing not found: %w", pkgdomain.ErrNotFound)
	}

	updated := listingFromInput(existing.ID, existing.ArtisanID, in)
	updated.State = existing.State
	updated.PublishedAt = existing.PublishedAt
	t.writes = append(t.writes, func() { t.store.listings[updated.ID] = updated })
	return updated, nil
}

func listingFromInput(id, artisanID uuid.UUID, in domain.UpsertListingInput) domain.Listing {
	currency := in.CurrencyCode
	if currency == "" {
		currency = "INR"
	}
	return domain.Listing{
		ID:               id,
		ProductID:        in.ProductID,
		ArtisanID:        artisanID,
		Type:             in.Type,
		State:            domain.StateDraft,
		PricePaise:       in.PricePaise,
		CurrencyCode:     currency,
		StockQuantity:    in.StockQuantity,
		MinOrderQuantity: in.MinOrderQuantity,
		LeadTimeDays:     in.LeadTimeDays,
		CapacityPerMonth: in.CapacityPerMonth,
		AcceptingOrders:  in.AcceptingOrders,
		AdvancePct:       in.AdvancePct,
		Packaging:        in.Packaging,
		GICertified:      in.GICertified,
		Translations:     in.Translations,
		CreatedBy:        in.CreatedBy,
		CreatedAt:        fakeNow,
		UpdatedAt:        fakeNow,
	}
}

// TransitionListingState mirrors the guarded UPDATE: it moves the row only if it
// is still in the state the caller read, and reports a conflict otherwise.
func (t *fakeCatalogTx) TransitionListingState(
	_ context.Context,
	listingID uuid.UUID,
	from, to domain.ListingState,
	reason *string,
) (domain.Listing, error) {
	t.store.mu.Lock()
	defer t.store.mu.Unlock()

	l, ok := t.store.listings[listingID]
	if !ok {
		return domain.Listing{}, fmt.Errorf("listing not found: %w", pkgdomain.ErrNotFound)
	}
	if l.State != from {
		return domain.Listing{}, fmt.Errorf(
			"listing %s is in state %s, not %s: %w", listingID, l.State, from, pkgdomain.ErrConflict)
	}

	l.State = to
	l.SuspensionReason = nil
	if to == domain.StateSuspended {
		l.SuspensionReason = reason
	}
	if to == domain.StatePublished && l.PublishedAt == nil {
		published := fakeNow
		l.PublishedAt = &published
	}

	// Applied as a delta rather than a whole-row overwrite, so a translation
	// written earlier in the same transaction is not lost at commit.
	t.writes = append(t.writes, func() {
		stored := t.store.listings[listingID]
		stored.State = l.State
		stored.SuspensionReason = l.SuspensionReason
		stored.PublishedAt = l.PublishedAt
		t.store.listings[listingID] = stored
	})
	return l, nil
}

func (t *fakeCatalogTx) UpsertListingTranslation(_ context.Context, tr domain.ListingTranslation) (domain.ListingTranslation, error) {
	tr.UpdatedAt = fakeNow
	t.writes = append(t.writes, func() {
		l := t.store.listings[tr.ListingID]
		for i, existing := range l.Translations {
			if existing.Language == tr.Language {
				l.Translations[i] = tr
				t.store.listings[tr.ListingID] = l
				return
			}
		}
		l.Translations = append(l.Translations, tr)
		t.store.listings[tr.ListingID] = l
	})
	return tr, nil
}

func (t *fakeCatalogTx) ListListingAttributes(_ context.Context, listingID uuid.UUID) ([]domain.ListingAttribute, error) {
	if pending, ok := t.pendingAttrs[listingID]; ok {
		return append([]domain.ListingAttribute(nil), pending...), nil
	}
	t.store.mu.Lock()
	defer t.store.mu.Unlock()
	return append([]domain.ListingAttribute(nil), t.store.attributes[listingID]...), nil
}

func (t *fakeCatalogTx) UpsertListingAttribute(ctx context.Context, a domain.ListingAttribute) error {
	current, err := t.ListListingAttributes(ctx, a.ListingID)
	if err != nil {
		return err
	}
	replaced := false
	for i, existing := range current {
		if existing.Name == a.Name && existing.Value == a.Value {
			current[i] = a
			replaced = true
			break
		}
	}
	if !replaced {
		current = append(current, a)
	}
	t.pendingAttrs[a.ListingID] = current

	listingID := a.ListingID
	t.writes = append(t.writes, func() { t.store.attributes[listingID] = t.pendingAttrs[listingID] })
	return nil
}

func (t *fakeCatalogTx) DeleteListingAttributesByName(ctx context.Context, listingID uuid.UUID, name string) (int64, error) {
	current, err := t.ListListingAttributes(ctx, listingID)
	if err != nil {
		return 0, err
	}
	kept := make([]domain.ListingAttribute, 0, len(current))
	var removed int64
	for _, a := range current {
		if a.Name == name && !a.Source.Authoritative() {
			removed++
			continue
		}
		kept = append(kept, a)
	}
	t.pendingAttrs[listingID] = kept
	t.writes = append(t.writes, func() { t.store.attributes[listingID] = t.pendingAttrs[listingID] })
	return removed, nil
}

func (t *fakeCatalogTx) ReplaceListingMedia(_ context.Context, listingID uuid.UUID, items []domain.ListingMedia) error {
	stored := append([]domain.ListingMedia(nil), items...)
	sort.Slice(stored, func(i, j int) bool { return stored[i].Ordinal < stored[j].Ordinal })
	t.writes = append(t.writes, func() { t.store.medias[listingID] = stored })
	return nil
}

func (t *fakeCatalogTx) InsertOutbox(_ context.Context, id, aggregateID, topic, idempotencyKey string, payload []byte) error {
	row := outboxRow{
		ID: id, AggregateID: aggregateID, Topic: topic,
		IdempotencyKey: idempotencyKey, Payload: payload,
	}
	t.writes = append(t.writes, func() {
		// Mirrors ON CONFLICT (topic, idempotency_key) DO NOTHING.
		for _, existing := range t.store.outbox {
			if existing.Topic == row.Topic && existing.IdempotencyKey == row.IdempotencyKey {
				return
			}
		}
		t.store.outbox = append(t.store.outbox, row)
	})
	return nil
}

// fakeNow is the fixed timestamp every fake write is stamped with.
var fakeNow = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

// fakeCraftIndex is a CraftIndex over a fixed set of crafts.
type fakeCraftIndex struct {
	crafts map[uuid.UUID]domain.Craft
}

func (f fakeCraftIndex) Craft(id uuid.UUID) (domain.Craft, bool) {
	c, ok := f.crafts[id]
	return c, ok
}
