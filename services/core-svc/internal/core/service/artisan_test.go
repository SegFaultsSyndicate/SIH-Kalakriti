// services/core-svc/internal/core/service/artisan_test.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

const testPhone = "+919876543210"

func validRegisterInput() domain.RegisterArtisanInput {
	return domain.RegisterArtisanInput{
		DisplayName: "Rukmini Ben",
		PhoneE164:   testPhone,
		CraftIDs:    []uuid.UUID{uuid.MustParse("01900000-0000-7000-8000-0000000000c1")},
		Languages:   []string{"GUJARATI", "HINDI"},
		Region:      domain.Region{StateCode: "IN-GJ", District: ptr("Kutch")},
	}
}

func TestRegisterArtisanWritesArtisanAndOutboxEventInOneTx(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	artisan, _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "idem-1")
	if err != nil {
		t.Fatalf("RegisterArtisan: %v", err)
	}
	if artisan.ID == uuid.Nil {
		t.Fatal("registered artisan has a nil id")
	}
	if artisan.DisplayName != "Rukmini Ben" {
		t.Errorf("DisplayName = %q", artisan.DisplayName)
	}

	// The artisan row landed.
	if _, ok := store.artisans[artisan.ID]; !ok {
		t.Fatal("artisan row was not committed")
	}

	// ...and exactly one outbox row landed with it, in the same transaction.
	if len(store.outbox) != 1 {
		t.Fatalf("expected 1 outbox row, got %d", len(store.outbox))
	}
	row := store.outbox[0]
	if row.Topic != topics.ArtisanRegistered {
		t.Errorf("topic = %q, want %q", row.Topic, topics.ArtisanRegistered)
	}
	if row.AggregateID != artisan.ID.String() {
		t.Errorf("aggregate_id = %q, want the artisan id %q", row.AggregateID, artisan.ID)
	}
	if row.IdempotencyKey != "idem-1" {
		t.Errorf("idempotency_key = %q, want idem-1", row.IdempotencyKey)
	}

	// The payload must be the event shape a consumer expects.
	var envelope struct {
		Header struct {
			AggregateID    string `json:"aggregate_id"`
			IdempotencyKey string `json:"idempotency_key"`
			Producer       string `json:"producer"`
			SchemaVersion  int32  `json:"schema_version"`
			EventID        string `json:"event_id"`
		} `json:"header"`
		Payload artisanRegistered `json:"payload"`
	}
	if err := json.Unmarshal(row.Payload, &envelope); err != nil {
		t.Fatalf("outbox payload is not the expected JSON: %v", err)
	}
	if envelope.Header.Producer != producerName {
		t.Errorf("producer = %q, want %q", envelope.Header.Producer, producerName)
	}
	if envelope.Header.SchemaVersion != schemaVersion {
		t.Errorf("schema_version = %d, want %d", envelope.Header.SchemaVersion, schemaVersion)
	}
	if envelope.Header.EventID == "" {
		t.Error("event_id should be set")
	}
	if envelope.Payload.ArtisanID != artisan.ID.String() {
		t.Errorf("payload artisan_id = %q, want %q", envelope.Payload.ArtisanID, artisan.ID)
	}
	if envelope.Payload.DisplayName != "Rukmini Ben" {
		t.Errorf("payload display_name = %q", envelope.Payload.DisplayName)
	}
	if len(envelope.Payload.Languages) != 2 {
		t.Errorf("payload languages = %v", envelope.Payload.Languages)
	}
}

// A second self-registration with the same phone is a recovery path, not an
// error: the phone-ownership check earlier in RegisterArtisan already proves
// this exact caller controls testPhone, so the existing profile under that
// phone can only be their own -- e.g. a client that registered once, lost
// the response, and is retrying while still holding its now-useless
// pre-registration token. It must come back as success with fresh tokens
// for the artisan that already exists, not ErrConflict, or that client has
// no way back to a working session.
func TestRegisterArtisanTwiceWithSamePhoneIsIdempotent(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})
	ctx := artisanPhoneCtx("", testPhone)

	first, firstTokens, err := svc.RegisterArtisan(ctx, validRegisterInput(), "idem-1")
	if err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}
	if firstTokens.AccessToken == "" {
		t.Fatal("first registration should mint tokens")
	}

	second, secondTokens, err := svc.RegisterArtisan(ctx, validRegisterInput(), "idem-2")
	if err != nil {
		t.Fatalf("second registration with the same self-verified phone should succeed, got %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("second registration returned a different artisan: %v vs %v", second.ID, first.ID)
	}
	if secondTokens.AccessToken == "" {
		t.Fatal("second registration should also mint tokens, so the caller has a way back to a working session")
	}

	// The recovered attempt must not have written a second outbox row --
	// nothing was actually (re-)created.
	if len(store.outbox) != 1 {
		t.Fatalf("expected 1 outbox row after a recovered duplicate, got %d", len(store.outbox))
	}
}

// A cluster officer proxy-registering someone else still gets the real
// conflict: "already registered" is actionable information for them (don't
// re-onboard this person), not a race to silently recover from.
func TestRegisterArtisanByProxyTwiceWithSamePhoneReturnsConflict(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})
	in := validRegisterInput()

	if _, _, err := svc.RegisterArtisan(officerSubjectCtx("officer-1"), in, "idem-1"); err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}

	_, _, err := svc.RegisterArtisan(officerSubjectCtx("officer-1"), in, "idem-2")
	if err == nil {
		t.Fatal("second proxy registration with the same phone should fail")
	}
	if !errors.Is(err, pkgdomain.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if !strings.Contains(err.Error(), testPhone) {
		t.Errorf("error should name the phone number, got %q", err)
	}

	if len(store.outbox) != 1 {
		t.Fatalf("expected 1 outbox row after a rejected duplicate, got %d", len(store.outbox))
	}
}

func TestRegisterArtisanConflictDetectedAtTheDatabaseAlsoSurfaces(t *testing.T) {
	// Proves the service does not rely solely on its pre-check: when two callers
	// race past it, the constraint violation the repo translates still surfaces
	// as ErrConflict.
	store := newFakeStore()
	store.createArtisanErr = pkgdomain.ErrConflict
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	_, _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "idem-1")
	if !errors.Is(err, pkgdomain.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if len(store.outbox) != 0 {
		t.Fatalf("a failed registration must not commit an outbox row, got %d", len(store.outbox))
	}
}

func TestRegisterArtisanRollsBackEverythingWhenTheCommitFails(t *testing.T) {
	store := newFakeStore()
	store.failInTxAfterCallback = errors.New("commit failed")
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	if _, _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "idem-1"); err == nil {
		t.Fatal("expected the commit failure to surface")
	}
	if len(store.artisans) != 0 {
		t.Errorf("artisan should not exist after a failed commit, got %d", len(store.artisans))
	}
	if len(store.outbox) != 0 {
		t.Errorf("outbox row should not exist after a failed commit, got %d", len(store.outbox))
	}
}

func TestRegisterArtisanJoinsClusterWhenGiven(t *testing.T) {
	store := newFakeStore()
	clusterID := uuid.MustParse("01900000-0000-7000-8000-000000000001")
	store.clusters[clusterID] = domain.Cluster{ID: clusterID, Name: "Kutch"}

	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})
	in := validRegisterInput()
	in.ClusterID = &clusterID

	artisan, _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), in, "idem-1")
	if err != nil {
		t.Fatalf("RegisterArtisan: %v", err)
	}
	if _, ok := store.clusterMem[clusterID][artisan.ID]; !ok {
		t.Fatal("the new artisan was not added to the cluster roster")
	}
	if got := store.clusterMem[clusterID][artisan.ID].Role; got != domain.ClusterRoleMember {
		t.Errorf("joined with role %q, want MEMBER", got)
	}
}

func TestRegisterArtisanValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.RegisterArtisanInput)
		want   string
	}{
		{"blank name", func(in *domain.RegisterArtisanInput) { in.DisplayName = "  " }, "display_name is required"},
		{"bad phone", func(in *domain.RegisterArtisanInput) { in.PhoneE164 = "9876543210" }, "must be E.164"},
		{"no crafts", func(in *domain.RegisterArtisanInput) { in.CraftIDs = nil }, "at least one craft_id"},
		{"duplicate craft", func(in *domain.RegisterArtisanInput) {
			in.CraftIDs = append(in.CraftIDs, in.CraftIDs[0])
		}, "twice"},
		{"no languages", func(in *domain.RegisterArtisanInput) { in.Languages = nil }, "at least one language"},
		{"no state code", func(in *domain.RegisterArtisanInput) { in.Region.StateCode = "" }, "state_code is required"},
		{"bad pincode", func(in *domain.RegisterArtisanInput) { in.Region.Pincode = ptr("12") }, "six-digit"},
		{"negative experience", func(in *domain.RegisterArtisanInput) {
			in.YearsOfExperience = ptr(int32(-1))
		}, "must not be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})
			in := validRegisterInput()
			tt.mutate(&in)

			_, _, err := svc.RegisterArtisan(artisanPhoneCtx("", in.PhoneE164), in, "idem-1")
			if err == nil {
				t.Fatalf("expected a validation error mentioning %q", tt.want)
			}
			if !errors.Is(err, pkgdomain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not mention %q", err, tt.want)
			}
			if len(store.artisans) != 0 || len(store.outbox) != 0 {
				t.Fatal("a rejected registration must not write anything")
			}
		})
	}
}

func TestRegisterArtisanRequiresIdempotencyKey(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{})
	_, _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "")
	if !errors.Is(err, pkgdomain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestRegisterArtisanRejectsAPhoneTheCallerDidNotVerify(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{})

	// The caller verified a different number than the one they are registering.
	ctx := artisanPhoneCtx("", "+919000000000")
	_, _, err := svc.RegisterArtisan(ctx, validRegisterInput(), "idem-1")
	if !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}

	// An unauthenticated caller is rejected too.
	if _, _, err := svc.RegisterArtisan(context.Background(), validRegisterInput(), "idem-1"); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden for an unauthenticated caller, got %v", err)
	}
}

func TestRegisterArtisanAllowsAnOfficerToRegisterByProxy(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	// An officer has no verified phone of their own, but may register others.
	artisan, _, err := svc.RegisterArtisan(officerSubjectCtx("officer-1"), validRegisterInput(), "idem-1")
	if err != nil {
		t.Fatalf("officer proxy registration should be allowed: %v", err)
	}
	if artisan.CreatedBy != "officer-1" {
		t.Errorf("created_by = %q, want the officer's subject", artisan.CreatedBy)
	}
}

func TestGetArtisanByPhoneHidesOtherArtisansProfiles(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	created, _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "idem-1")
	if err != nil {
		t.Fatalf("RegisterArtisan: %v", err)
	}

	// The artisan themselves can look their own number up.
	if _, err := svc.GetArtisanByPhone(artisanPhoneCtx(created.ID.String(), testPhone), testPhone); err != nil {
		t.Fatalf("self lookup should be allowed: %v", err)
	}
	// An officer can look anyone up.
	if _, err := svc.GetArtisanByPhone(officerSubjectCtx("officer-1"), testPhone); err != nil {
		t.Fatalf("officer lookup should be allowed: %v", err)
	}
	// Another artisan cannot, and is told not-found rather than forbidden, so
	// the endpoint cannot be used to test whether a number is registered.
	other := artisanPhoneCtx(uuid.New().String(), "+919000000001")
	_, err = svc.GetArtisanByPhone(other, testPhone)
	if !errors.Is(err, pkgdomain.ErrNotFound) {
		t.Fatalf("want ErrNotFound for an unrelated artisan, got %v", err)
	}
}

func TestUpdateArtisanProfileAuthorisation(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})
	created, _, err := svc.RegisterArtisan(artisanPhoneCtx("", testPhone), validRegisterInput(), "idem-1")
	if err != nil {
		t.Fatalf("RegisterArtisan: %v", err)
	}

	in := domain.UpdateArtisanInput{ArtisanID: created.ID, DisplayName: ptr("Rukmini B.")}

	// Self is allowed.
	if _, err := svc.UpdateArtisanProfile(artisanPhoneCtx(created.ID.String(), testPhone), in); err != nil {
		t.Fatalf("self update should be allowed: %v", err)
	}
	// An officer is allowed.
	if _, err := svc.UpdateArtisanProfile(officerSubjectCtx("officer-1"), in); err != nil {
		t.Fatalf("officer update should be allowed: %v", err)
	}
	// A different artisan is not.
	other := artisanPhoneCtx(uuid.New().String(), "+919000000001")
	if _, err := svc.UpdateArtisanProfile(other, in); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	// A buyer is not.
	buyer := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: "b1", Role: auth.RoleBuyer})
	if _, err := svc.UpdateArtisanProfile(buyer, in); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden for a buyer, got %v", err)
	}
}

func TestListArtisansByClusterRejectsAnUnknownCluster(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{})
	_, err := svc.ListArtisansByCluster(officerSubjectCtx("o1"), uuid.New(), domain.Page{})
	if !errors.Is(err, pkgdomain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestPageNormalise(t *testing.T) {
	tests := []struct {
		in   int32
		want int32
	}{
		{0, domain.DefaultPageSize},
		{-5, domain.DefaultPageSize},
		{10, 10},
		{domain.MaxPageSize, domain.MaxPageSize},
		{domain.MaxPageSize + 1, domain.MaxPageSize},
		{1_000_000, domain.MaxPageSize},
	}
	for _, tt := range tests {
		if got := (domain.Page{Size: tt.in}).Normalise().Size; got != tt.want {
			t.Errorf("Page{Size:%d}.Normalise() = %d, want %d", tt.in, got, tt.want)
		}
	}
}
