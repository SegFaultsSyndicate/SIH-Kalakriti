// services/core-svc/internal/core/service/catalog_test.go
package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// catalogFixture is one artisan, one craft, one product and one listing, with
// the service wired to fakes.
type catalogFixture struct {
	svc     *Catalog
	store   *fakeCatalogStore
	artisan uuid.UUID
	craft   uuid.UUID
	product uuid.UUID
	listing uuid.UUID
}

func newCatalogFixture(t *testing.T, state domain.ListingState) catalogFixture {
	t.Helper()

	f := catalogFixture{
		store:   newFakeCatalogStore(),
		artisan: ids.New(),
		craft:   ids.New(),
		product: ids.New(),
		listing: ids.New(),
	}
	f.svc = NewCatalog(f.store,
		fakeCraftIndex{crafts: map[uuid.UUID]domain.Craft{
			f.craft: {ID: f.craft, Code: "ajrakh-block-printing", DisplayName: "Ajrakh Block Printing"},
		}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))

	leadTime := int32(21)
	capacity := int32(8)
	f.store.products[f.product] = domain.Product{
		ID: f.product, ArtisanID: f.artisan, CraftID: f.craft, WorkingTitle: "Ajrakh stole",
	}
	f.store.listings[f.listing] = domain.Listing{
		ID:               f.listing,
		ProductID:        f.product,
		ArtisanID:        f.artisan,
		Type:             domain.ListingMadeToOrder,
		State:            state,
		PricePaise:       450000,
		CurrencyCode:     "INR",
		MinOrderQuantity: 1,
		LeadTimeDays:     &leadTime,
		CapacityPerMonth: &capacity,
		AcceptingOrders:  true,
		Translations: []domain.ListingTranslation{
			{ListingID: f.listing, Language: "ENGLISH", Title: "Ajrakh stole", Description: "Hand block printed.", MachineGenerated: true},
		},
	}
	if state == domain.StateSuspended {
		reason := "reported by a buyer"
		l := f.store.listings[f.listing]
		l.SuspensionReason = &reason
		l.PublishedAt = &fakeNow
		f.store.listings[f.listing] = l
	}
	return f
}

// artisanCtx is a context carrying the owning artisan's token.
func artisanCtx(artisanID uuid.UUID) context.Context {
	return auth.ContextWithPrincipal(context.Background(), auth.Principal{
		Subject: artisanID.String(), Role: auth.RoleArtisan, Language: "ENGLISH",
	})
}

func officerCtx() context.Context {
	return auth.ContextWithPrincipal(context.Background(), auth.Principal{
		Subject: ids.New().String(), Role: auth.RoleClusterOfficer,
	})
}

// TestListingStateMachine walks every ordered pair of states through the domain
// table that the service is the only enforcer of, then checks that the service
// itself refuses an illegal move with the attempted pair named.
func TestListingStateMachine(t *testing.T) {
	t.Parallel()

	states := []domain.ListingState{
		domain.StateDraft,
		domain.StatePendingArtisanApproval,
		domain.StatePublished,
		domain.StateSuspended,
	}
	legal := map[domain.ListingState]domain.ListingState{
		domain.StateDraft:                  domain.StatePendingArtisanApproval,
		domain.StatePendingArtisanApproval: domain.StatePublished,
		domain.StatePublished:              domain.StateSuspended,
		domain.StateSuspended:              domain.StatePublished,
	}

	for _, from := range states {
		for _, to := range states {
			from, to := from, to
			t.Run(from.String()+"_to_"+to.String(), func(t *testing.T) {
				t.Parallel()
				err := domain.ValidateTransition(from, to)
				if legal[from] == to {
					require.NoError(t, err)
					require.True(t, from.CanTransitionTo(to))
					return
				}
				require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
				require.Contains(t, err.Error(), from.String()+" -> "+to.String())
				require.False(t, from.CanTransitionTo(to))
			})
		}
	}
}

// TestApproveListingRejectsIllegalTransitions drives the same table through the
// service: approving is legal only out of PENDING_ARTISAN_APPROVAL, and the
// refusal names the pair that was attempted.
func TestApproveListingRejectsIllegalTransitions(t *testing.T) {
	t.Parallel()

	for _, from := range []domain.ListingState{domain.StateDraft, domain.StatePublished, domain.StateSuspended} {
		from := from
		t.Run(from.String(), func(t *testing.T) {
			t.Parallel()
			f := newCatalogFixture(t, from)

			_, err := f.svc.ApproveListing(artisanCtx(f.artisan), f.listing, f.artisan, nil, "key-1")
			require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
			require.Contains(t, err.Error(), from.String()+" -> "+domain.StatePublished.String())
			require.Empty(t, f.store.outbox)
			require.Equal(t, from, f.store.listings[f.listing].State)
		})
	}
}

// TestApproveListingPublishesWithExactlyOneOutboxRow is the acceptance criterion:
// the state change and one event row, in one transaction, and a replay of the
// same idempotency key adds nothing.
func TestApproveListingPublishesWithExactlyOneOutboxRow(t *testing.T) {
	t.Parallel()
	f := newCatalogFixture(t, domain.StatePendingArtisanApproval)

	published, err := f.svc.ApproveListing(artisanCtx(f.artisan), f.listing, f.artisan,
		[]domain.ListingTranslation{{
			Language: "HINDI", Title: "अजरख स्टोल", Description: "हाथ से छपा हुआ।",
		}},
		"approve-key-1")
	require.NoError(t, err)
	require.Equal(t, domain.StatePublished, published.State)
	require.NotNil(t, published.PublishedAt)

	require.Len(t, f.store.outbox, 1)
	require.Equal(t, topics.CatalogListingPublished, f.store.outbox[0].Topic)
	require.Equal(t, f.listing.String(), f.store.outbox[0].AggregateID)
	require.Equal(t, "approve-key-1", f.store.outbox[0].IdempotencyKey)

	// The artisan's edit is stored as human-edited copy, attributed to them.
	stored := f.store.listings[f.listing]
	var hindi domain.ListingTranslation
	for _, tr := range stored.Translations {
		if tr.Language == "HINDI" {
			hindi = tr
		}
	}
	require.False(t, hindi.MachineGenerated)
	require.NotNil(t, hindi.EditedBy)
	require.Equal(t, f.artisan.String(), *hindi.EditedBy)
}

func TestApproveListingRequiresOwnerOrSHGSignatory(t *testing.T) {
	t.Parallel()

	t.Run("a stranger may not approve", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StatePendingArtisanApproval)

		_, err := f.svc.ApproveListing(artisanCtx(ids.New()), f.listing, uuid.Nil, nil, "key-1")
		require.ErrorIs(t, err, pkgdomain.ErrForbidden)
		require.Empty(t, f.store.outbox)
	})

	t.Run("an officer may not approve for the artisan", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StatePendingArtisanApproval)

		_, err := f.svc.ApproveListing(officerCtx(), f.listing, uuid.Nil, nil, "key-1")
		require.ErrorIs(t, err, pkgdomain.ErrForbidden)
	})

	t.Run("the group signatory may approve", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StatePendingArtisanApproval)

		signatory := ids.New()
		f.store.shgByArtisan[f.artisan] = domain.SelfHelpGroup{
			ID: ids.New(), Name: "Bhuj Weavers SHG", RegistrationNo: "GJ-001",
			SignatoryArtisanID: &signatory,
		}

		published, err := f.svc.ApproveListing(artisanCtx(signatory), f.listing, uuid.Nil, nil, "key-2")
		require.NoError(t, err)
		require.Equal(t, domain.StatePublished, published.State)
		require.Len(t, f.store.outbox, 1)
	})

	t.Run("a request naming a different owner is rejected", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StatePendingArtisanApproval)

		_, err := f.svc.ApproveListing(artisanCtx(f.artisan), f.listing, ids.New(), nil, "key-3")
		require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
	})
}

func TestSubmitForApproval(t *testing.T) {
	t.Parallel()

	t.Run("a draft with copy is submitted", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StateDraft)

		moved, err := f.svc.SubmitForApproval(artisanCtx(f.artisan), f.listing, "key-1")
		require.NoError(t, err)
		require.Equal(t, domain.StatePendingArtisanApproval, moved.State)
	})

	t.Run("a draft with no copy is refused", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StateDraft)
		l := f.store.listings[f.listing]
		l.Translations = nil
		f.store.listings[f.listing] = l

		_, err := f.svc.SubmitForApproval(artisanCtx(f.artisan), f.listing, "key-1")
		require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
	})

	t.Run("a cluster officer may submit on the artisan's behalf", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StateDraft)

		moved, err := f.svc.SubmitForApproval(officerCtx(), f.listing, "key-1")
		require.NoError(t, err)
		require.Equal(t, domain.StatePendingArtisanApproval, moved.State)
	})
}

func TestSuspendAndReinstate(t *testing.T) {
	t.Parallel()

	t.Run("an officer suspends a published listing", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StatePublished)

		suspended, err := f.svc.SuspendListing(officerCtx(), f.listing, "counterfeit claim", "key-1")
		require.NoError(t, err)
		require.Equal(t, domain.StateSuspended, suspended.State)
		require.NotNil(t, suspended.SuspensionReason)
		require.Empty(t, f.store.outbox)
	})

	t.Run("the artisan may not suspend their own listing", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StatePublished)

		_, err := f.svc.SuspendListing(artisanCtx(f.artisan), f.listing, "changed my mind", "key-1")
		require.ErrorIs(t, err, pkgdomain.ErrForbidden)
	})

	t.Run("reinstating republishes so search reindexes", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StateSuspended)

		reinstated, err := f.svc.ReinstateListing(officerCtx(), f.listing, "key-2")
		require.NoError(t, err)
		require.Equal(t, domain.StatePublished, reinstated.State)
		require.Nil(t, reinstated.SuspensionReason)
		require.Len(t, f.store.outbox, 1)
		require.Equal(t, topics.CatalogListingPublished, f.store.outbox[0].Topic)
	})
}

func TestUpsertListingValidation(t *testing.T) {
	t.Parallel()

	i32 := func(v int32) *int32 { return &v }
	base := func(mutate func(*domain.UpsertListingInput)) domain.UpsertListingInput {
		in := domain.UpsertListingInput{
			Type:             domain.ListingMadeToOrder,
			PricePaise:       450000,
			MinOrderQuantity: 1,
			LeadTimeDays:     i32(21),
			CapacityPerMonth: i32(8),
			AcceptingOrders:  true,
			AdvancePct:       i32(30),
		}
		mutate(&in)
		return in
	}

	tests := []struct {
		name    string
		mutate  func(*domain.UpsertListingInput)
		wantErr bool
	}{
		{name: "made to order with terms", mutate: func(*domain.UpsertListingInput) {}},
		{name: "made to order without lead time", wantErr: true,
			mutate: func(in *domain.UpsertListingInput) { in.LeadTimeDays = nil }},
		{name: "made to order with zero lead time", wantErr: true,
			mutate: func(in *domain.UpsertListingInput) { in.LeadTimeDays = i32(0) }},
		{name: "made to order without capacity", wantErr: true,
			mutate: func(in *domain.UpsertListingInput) { in.CapacityPerMonth = nil }},
		{name: "ready stock with quantity", mutate: func(in *domain.UpsertListingInput) {
			in.Type = domain.ListingReadyStock
			in.LeadTimeDays, in.CapacityPerMonth = nil, nil
			in.StockQuantity = i32(4)
		}},
		{name: "ready stock without quantity", wantErr: true, mutate: func(in *domain.UpsertListingInput) {
			in.Type = domain.ListingReadyStock
			in.LeadTimeDays, in.CapacityPerMonth = nil, nil
		}},
		{name: "ready stock with negative quantity", wantErr: true, mutate: func(in *domain.UpsertListingInput) {
			in.Type = domain.ListingReadyStock
			in.LeadTimeDays, in.CapacityPerMonth = nil, nil
			in.StockQuantity = i32(-1)
		}},
		{name: "advance below zero", wantErr: true,
			mutate: func(in *domain.UpsertListingInput) { in.AdvancePct = i32(-1) }},
		{name: "advance above a hundred", wantErr: true,
			mutate: func(in *domain.UpsertListingInput) { in.AdvancePct = i32(101) }},
		{name: "advance at the boundaries", mutate: func(in *domain.UpsertListingInput) { in.AdvancePct = i32(100) }},
		{name: "unknown type", wantErr: true,
			mutate: func(in *domain.UpsertListingInput) { in.Type = domain.ListingType("BARTER") }},
		{name: "negative price", wantErr: true,
			mutate: func(in *domain.UpsertListingInput) { in.PricePaise = -1 }},
		{name: "zero minimum order", wantErr: true,
			mutate: func(in *domain.UpsertListingInput) { in.MinOrderQuantity = 0 }},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newCatalogFixture(t, domain.StateDraft)

			in := base(tt.mutate)
			in.ProductID = f.product
			in.Translations = []domain.ListingTranslation{
				{Language: "ENGLISH", Title: "Ajrakh stole", Description: "Hand block printed."},
			}

			_, err := f.svc.UpsertListing(artisanCtx(f.artisan), in, "key-1")
			if tt.wantErr {
				require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpsertListingRefusesASuspendedListing(t *testing.T) {
	t.Parallel()
	f := newCatalogFixture(t, domain.StateSuspended)

	listingID := f.listing
	_, err := f.svc.UpsertListing(artisanCtx(f.artisan), domain.UpsertListingInput{
		ListingID:        &listingID,
		ProductID:        f.product,
		Type:             domain.ListingReadyStock,
		PricePaise:       1,
		MinOrderQuantity: 1,
		StockQuantity:    func() *int32 { v := int32(2); return &v }(),
	}, "key-1")
	require.ErrorIs(t, err, pkgdomain.ErrConflict)
}

// TestArtisanAttributesBeatModelAttributes is the precedence rule: whatever the
// artisan says stands, whichever order the writes arrive in.
func TestArtisanAttributesBeatModelAttributes(t *testing.T) {
	t.Parallel()

	attr := func(name, value string, confidence float32) domain.ListingAttribute {
		return domain.ListingAttribute{Name: name, Value: value, Confidence: confidence}
	}
	valuesFor := func(attrs []domain.ListingAttribute, name string) map[string]domain.AttributeSource {
		out := map[string]domain.AttributeSource{}
		for _, a := range attrs {
			if a.Name == name {
				out[a.Value] = a.Source
			}
		}
		return out
	}

	t.Run("the model does not overwrite an artisan value", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StateDraft)
		ctx := artisanCtx(f.artisan)

		_, err := f.svc.UpsertListingAttributes(ctx, f.listing,
			[]domain.ListingAttribute{attr("material", "ajrakh cotton", 1)},
			domain.SourceArtisan, "key-1")
		require.NoError(t, err)

		stored, err := f.svc.UpsertListingAttributes(ctx, f.listing,
			[]domain.ListingAttribute{attr("material", "polyester", 0.71), attr("motif", "kalka", 0.66)},
			domain.SourceModel, "key-2")
		require.NoError(t, err)

		materials := valuesFor(stored, "material")
		require.Len(t, materials, 1)
		require.Equal(t, domain.SourceArtisan, materials["ajrakh cotton"])

		// An attribute the artisan never spoke for is still the model's to fill.
		motifs := valuesFor(stored, "motif")
		require.Equal(t, domain.SourceModel, motifs["kalka"])
	})

	t.Run("an artisan value displaces the model's guesses for that name", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StateDraft)
		ctx := artisanCtx(f.artisan)

		_, err := f.svc.UpsertListingAttributes(ctx, f.listing,
			[]domain.ListingAttribute{attr("material", "polyester", 0.71), attr("motif", "kalka", 0.66)},
			domain.SourceModel, "key-1")
		require.NoError(t, err)

		stored, err := f.svc.UpsertListingAttributes(ctx, f.listing,
			[]domain.ListingAttribute{attr("material", "ajrakh cotton", 1)},
			domain.SourceArtisan, "key-2")
		require.NoError(t, err)

		materials := valuesFor(stored, "material")
		require.Len(t, materials, 1)
		require.Equal(t, domain.SourceArtisan, materials["ajrakh cotton"])
		require.Equal(t, domain.SourceModel, valuesFor(stored, "motif")["kalka"])
	})

	t.Run("a curator value also outranks the model", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StateDraft)

		_, err := f.svc.UpsertListingAttributes(officerCtx(), f.listing,
			[]domain.ListingAttribute{attr("craft", "ajrakh-block-printing", 1)},
			domain.SourceCurator, "key-1")
		require.NoError(t, err)

		stored, err := f.svc.UpsertListingAttributes(officerCtx(), f.listing,
			[]domain.ListingAttribute{attr("craft", "bagru-block-printing", 0.52)},
			domain.SourceModel, "key-2")
		require.NoError(t, err)

		crafts := valuesFor(stored, "craft")
		require.Len(t, crafts, 1)
		require.Equal(t, domain.SourceCurator, crafts["ajrakh-block-printing"])
	})

	t.Run("an out of range confidence is rejected", func(t *testing.T) {
		t.Parallel()
		f := newCatalogFixture(t, domain.StateDraft)

		_, err := f.svc.UpsertListingAttributes(artisanCtx(f.artisan), f.listing,
			[]domain.ListingAttribute{attr("material", "cotton", 1.4)},
			domain.SourceModel, "key-1")
		require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
	})
}

func TestAttachListingMedia(t *testing.T) {
	t.Parallel()

	newFixtureWithMedia := func(t *testing.T) (catalogFixture, uuid.UUID, uuid.UUID, uuid.UUID) {
		t.Helper()
		f := newCatalogFixture(t, domain.StateDraft)
		primary, gallery, video := ids.New(), ids.New(), ids.New()
		f.store.ownership[primary] = domain.MediaOwnership{MediaID: primary, ArtisanID: f.artisan, Kind: domain.MediaImage}
		f.store.ownership[gallery] = domain.MediaOwnership{MediaID: gallery, ArtisanID: f.artisan, Kind: domain.MediaImage}
		f.store.ownership[video] = domain.MediaOwnership{MediaID: video, ArtisanID: f.artisan, Kind: domain.MediaVideo}
		return f, primary, gallery, video
	}

	t.Run("attaches in order with one primary and one process video", func(t *testing.T) {
		t.Parallel()
		f, primary, gallery, video := newFixtureWithMedia(t)

		stored, err := f.svc.AttachListingMedia(artisanCtx(f.artisan), domain.AttachMediaInput{
			ListingID: f.listing,
			Items: []domain.ListingMedia{
				{MediaID: gallery, Ordinal: 1, Role: domain.MediaRoleGallery},
				{MediaID: primary, Ordinal: 0, Role: domain.MediaRolePrimaryImage},
				{MediaID: video, Ordinal: 2, Role: domain.MediaRoleProcessVideo},
			},
		}, "key-1")
		require.NoError(t, err)
		require.Len(t, stored, 3)
		require.Equal(t, primary, stored[0].MediaID)
		require.Equal(t, domain.MediaRolePrimaryImage, stored[0].Role)
		require.Equal(t, video, stored[2].MediaID)
	})

	t.Run("two primary images are refused", func(t *testing.T) {
		t.Parallel()
		f, primary, gallery, _ := newFixtureWithMedia(t)

		_, err := f.svc.AttachListingMedia(artisanCtx(f.artisan), domain.AttachMediaInput{
			ListingID: f.listing,
			Items: []domain.ListingMedia{
				{MediaID: primary, Ordinal: 0, Role: domain.MediaRolePrimaryImage},
				{MediaID: gallery, Ordinal: 1, Role: domain.MediaRolePrimaryImage},
			},
		}, "key-1")
		require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
	})

	t.Run("a still cannot be the process video", func(t *testing.T) {
		t.Parallel()
		f, primary, _, _ := newFixtureWithMedia(t)

		_, err := f.svc.AttachListingMedia(artisanCtx(f.artisan), domain.AttachMediaInput{
			ListingID: f.listing,
			Items:     []domain.ListingMedia{{MediaID: primary, Ordinal: 0, Role: domain.MediaRoleProcessVideo}},
		}, "key-1")
		require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
	})

	t.Run("another artisan's media is refused", func(t *testing.T) {
		t.Parallel()
		f, _, _, _ := newFixtureWithMedia(t)
		stranger := ids.New()
		f.store.ownership[stranger] = domain.MediaOwnership{MediaID: stranger, ArtisanID: ids.New(), Kind: domain.MediaImage}

		_, err := f.svc.AttachListingMedia(artisanCtx(f.artisan), domain.AttachMediaInput{
			ListingID: f.listing,
			Items:     []domain.ListingMedia{{MediaID: stranger, Ordinal: 0, Role: domain.MediaRoleGallery}},
		}, "key-1")
		require.ErrorIs(t, err, pkgdomain.ErrForbidden)
	})

	t.Run("unknown media is not found", func(t *testing.T) {
		t.Parallel()
		f, _, _, _ := newFixtureWithMedia(t)

		_, err := f.svc.AttachListingMedia(artisanCtx(f.artisan), domain.AttachMediaInput{
			ListingID: f.listing,
			Items:     []domain.ListingMedia{{MediaID: ids.New(), Ordinal: 0, Role: domain.MediaRoleGallery}},
		}, "key-1")
		require.ErrorIs(t, err, pkgdomain.ErrNotFound)
	})
}

func TestCreateProductRejectsAnUnknownCraft(t *testing.T) {
	t.Parallel()
	f := newCatalogFixture(t, domain.StateDraft)

	_, err := f.svc.CreateProduct(artisanCtx(f.artisan), domain.CreateProductInput{
		ArtisanID:    f.artisan,
		CraftID:      ids.New(),
		WorkingTitle: "Ajrakh stole",
	}, "key-1")
	require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
}

func TestGetListingHidesUnpublishedFromStrangers(t *testing.T) {
	t.Parallel()
	f := newCatalogFixture(t, domain.StateDraft)

	_, err := f.svc.GetListing(artisanCtx(ids.New()), f.listing)
	require.ErrorIs(t, err, pkgdomain.ErrNotFound)

	got, err := f.svc.GetListing(artisanCtx(f.artisan), f.listing)
	require.NoError(t, err)
	require.Equal(t, f.listing, got.ID)
}

func TestMutatingCallsRequireAnIdempotencyKey(t *testing.T) {
	t.Parallel()
	f := newCatalogFixture(t, domain.StatePendingArtisanApproval)
	ctx := artisanCtx(f.artisan)

	_, err := f.svc.ApproveListing(ctx, f.listing, f.artisan, nil, "")
	require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)

	_, err = f.svc.SubmitForApproval(ctx, f.listing, "")
	require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)

	_, err = f.svc.UpsertListingAttributes(ctx, f.listing,
		[]domain.ListingAttribute{{Name: "material", Value: "cotton", Confidence: 1}},
		domain.SourceArtisan, "")
	require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
}
