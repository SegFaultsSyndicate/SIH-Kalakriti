package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

type fakeFinance struct {
	FinanceStore
	inserted []domain.FinanceLink
	hashes   [][]byte
	deleted  int
}

func (f *fakeFinance) InsertFinanceLink(_ context.Context, l domain.FinanceLink, h []byte, _ domain.FinanceTerms) (domain.FinanceLink, error) {
	f.inserted = append(f.inserted, l)
	f.hashes = append(f.hashes, h)
	return l, nil
}

func (f *fakeFinance) DeleteFinanceLink(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	f.deleted++
	return true, nil
}

func artisanActorCtx(actor string) context.Context {
	return auth.ContextWithPrincipal(context.Background(),
		auth.Principal{Subject: uuid.NewString(), Role: auth.RoleArtisan, Actor: actor})
}

func TestLinkFinanceNeedsConsentAndStoresOnlyLast4AndHash(t *testing.T) {
	store := &fakeFinance{}
	svc := NewFinance(store, []byte("test-salt-test-salt-test-salt-32"), nil)
	in := LinkFinanceInput{Corporation: "NSFDC", Reference: "up-2024-00981234"}

	if _, err := svc.LinkFinance(artisanActorCtx(""), in); !errors.Is(err, pkgdomain.ErrInvalidInput) || len(store.inserted) != 0 {
		t.Fatalf("no consent: err=%v inserted=%d", err, len(store.inserted))
	}

	in.ConsentGiven, in.ConsentVersion = true, domain.FinanceConsentVersion
	link, err := svc.LinkFinance(artisanActorCtx(""), in)
	if err != nil {
		t.Fatal(err)
	}
	if link.ReferenceLast4 != "1234" {
		t.Fatalf("last4 = %q", link.ReferenceLast4)
	}
	if bytes.Contains(store.hashes[0], []byte("UP202400981234")) || len(store.hashes[0]) != 32 {
		t.Fatalf("stored hash must be an HMAC, not the reference: %x", store.hashes[0])
	}
}

func TestOnlyTheArtisanCanWithdrawFinanceConsent(t *testing.T) {
	store := &fakeFinance{}
	svc := NewFinance(store, []byte("test-salt-test-salt-test-salt-32"), nil)
	if _, err := svc.DeleteFinanceLink(artisanActorCtx(uuid.NewString()), uuid.New()); !errors.Is(err, pkgdomain.ErrForbidden) || store.deleted != 0 {
		t.Fatalf("agent delete: err=%v deleted=%d", err, store.deleted)
	}
	if ok, err := svc.DeleteFinanceLink(artisanActorCtx(""), uuid.New()); err != nil || !ok {
		t.Fatalf("artisan delete: ok=%v err=%v", ok, err)
	}
}
