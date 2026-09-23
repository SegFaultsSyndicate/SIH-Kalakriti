package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// fakeAssisted implements only what these tests reach; any other method
// panics through the nil embedded interface, which is the point.
type fakeAssisted struct {
	AssistedStore
	staff    domain.StaffAccount
	links    map[uuid.UUID]bool // artisan -> linked to staff
	byPhone  map[string]uuid.UUID
	upserted []domain.AssistedLink
	revoked  int
}

func (f *fakeAssisted) GetStaffAccount(_ context.Context, id uuid.UUID) (domain.StaffAccount, error) {
	if id != f.staff.ID {
		return domain.StaffAccount{}, pkgdomain.NotFound("staff")
	}
	return f.staff, nil
}
func (f *fakeAssisted) HasActiveAssistedLink(_ context.Context, _, artisanID uuid.UUID) (bool, error) {
	return f.links[artisanID], nil
}
func (f *fakeAssisted) GetArtisan(_ context.Context, id uuid.UUID) (domain.Artisan, error) {
	return domain.Artisan{ID: id, DisplayName: "Kamla Devi"}, nil
}
func (f *fakeAssisted) ArtisanExistsByPhone(_ context.Context, phone string) (uuid.UUID, bool, error) {
	id, ok := f.byPhone[phone]
	return id, ok, nil
}
func (f *fakeAssisted) UpsertAssistedLink(_ context.Context, l domain.AssistedLink) (domain.AssistedLink, error) {
	f.upserted = append(f.upserted, l)
	return l, nil
}
func (f *fakeAssisted) InsertAssistedAudit(context.Context, domain.AssistedAudit, []byte) error {
	return nil
}
func (f *fakeAssisted) RevokeAssistedLink(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	f.revoked++
	return true, nil
}

func agentCtx(staff domain.StaffAccount) context.Context {
	return auth.ContextWithPrincipal(context.Background(),
		auth.Principal{Subject: staff.ID.String(), Role: auth.RoleFieldAgent})
}

func newAssistedFixture() (*Assisted, *fakeAssisted) {
	store := &fakeAssisted{
		staff:   domain.StaffAccount{ID: uuid.New(), Role: "FIELD_AGENT", Active: true},
		links:   map[uuid.UUID]bool{},
		byPhone: map[string]uuid.UUID{},
	}
	return NewAssisted(store, &fakeOTP{acceptCode: "123456"}, nil, nil), store
}

func TestResolveOnBehalfNeedsAnActiveLink(t *testing.T) {
	svc, store := newAssistedFixture()
	linked, stranger := uuid.New(), uuid.New()
	store.links[linked] = true

	ok, name, err := svc.ResolveOnBehalf(agentCtx(store.staff), linked)
	if err != nil || !ok || name != "Kamla Devi" {
		t.Fatalf("linked artisan: ok=%v name=%q err=%v", ok, name, err)
	}
	if ok, _, err := svc.ResolveOnBehalf(agentCtx(store.staff), stranger); err != nil || ok {
		t.Fatalf("unlinked artisan must be refused: ok=%v err=%v", ok, err)
	}

	store.staff.Active = false
	if _, _, err := svc.ResolveOnBehalf(agentCtx(store.staff), linked); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("deactivated agent: err = %v, want forbidden", err)
	}
}

func TestResolveOnBehalfRefusesArtisans(t *testing.T) {
	svc, _ := newAssistedFixture()
	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: uuid.NewString(), Role: auth.RoleArtisan})
	if _, _, err := svc.ResolveOnBehalf(ctx, uuid.New()); err == nil {
		t.Fatal("an artisan token must not resolve on-behalf access")
	}
}

func TestLinkArtisanRequiresTheArtisansOTP(t *testing.T) {
	svc, store := newAssistedFixture()
	existing := uuid.New()
	store.byPhone["+919876543210"] = existing

	_, err := svc.LinkArtisan(agentCtx(store.staff), LinkArtisanInput{
		PhoneE164: "+919876543210", ConsentMethod: domain.ConsentArtisanOTP, ChallengeID: "challenge-1", OTP: "000000",
	})
	if !errors.Is(err, pkgdomain.ErrForbidden) || len(store.upserted) != 0 {
		t.Fatalf("wrong OTP: err=%v links=%d", err, len(store.upserted))
	}

	res, err := svc.LinkArtisan(agentCtx(store.staff), LinkArtisanInput{
		PhoneE164: "+919876543210", ConsentMethod: domain.ConsentArtisanOTP, ChallengeID: "challenge-1", OTP: "123456",
	})
	if err != nil || res.ArtisanID != existing || res.RegisteredNow || res.NeedsReview {
		t.Fatalf("valid OTP: res=%+v err=%v", res, err)
	}
}

func TestVoiceConsentLinkIsFlaggedForReview(t *testing.T) {
	svc, store := newAssistedFixture()
	store.byPhone["+919876543210"] = uuid.New()
	res, err := svc.LinkArtisan(agentCtx(store.staff), LinkArtisanInput{
		PhoneE164: "+919876543210", ConsentMethod: domain.ConsentVoiceRecording,
	})
	if err != nil || !res.NeedsReview || !store.upserted[0].NeedsReview {
		t.Fatalf("voice consent: res=%+v err=%v", res, err)
	}
}

func TestOnlyTheArtisanCanRevokeAHelper(t *testing.T) {
	svc, store := newAssistedFixture()
	artisan := uuid.NewString()
	onBehalf := auth.ContextWithPrincipal(context.Background(),
		auth.Principal{Subject: artisan, Role: auth.RoleArtisan, Actor: store.staff.ID.String()})
	if _, err := svc.RevokeHelper(onBehalf, uuid.New()); !errors.Is(err, pkgdomain.ErrForbidden) || store.revoked != 0 {
		t.Fatalf("agent revoking: err=%v revoked=%d", err, store.revoked)
	}
	self := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: artisan, Role: auth.RoleArtisan})
	if ok, err := svc.RevokeHelper(self, uuid.New()); err != nil || !ok {
		t.Fatalf("artisan revoking: ok=%v err=%v", ok, err)
	}
}
