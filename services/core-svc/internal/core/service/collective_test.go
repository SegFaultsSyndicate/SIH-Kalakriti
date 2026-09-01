// services/core-svc/internal/core/service/collective_test.go
package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

var (
	artisanA = uuid.MustParse("01900000-0000-7000-8000-0000000000a1")
	artisanB = uuid.MustParse("01900000-0000-7000-8000-0000000000a2")
	artisanC = uuid.MustParse("01900000-0000-7000-8000-0000000000a3")
)

func TestValidateSHGSharesMustSumTo100(t *testing.T) {
	tests := []struct {
		name    string
		members []domain.SHGMemberShare
		wantErr string
	}{
		{
			name: "exactly 100 across three members",
			members: []domain.SHGMemberShare{
				{ArtisanID: artisanA, SharePct: 34},
				{ArtisanID: artisanB, SharePct: 33},
				{ArtisanID: artisanC, SharePct: 33},
			},
		},
		{
			name:    "a single member holding everything",
			members: []domain.SHGMemberShare{{ArtisanID: artisanA, SharePct: 100}},
		},
		{
			name: "a zero share is allowed as long as the total is right",
			members: []domain.SHGMemberShare{
				{ArtisanID: artisanA, SharePct: 100},
				{ArtisanID: artisanB, SharePct: 0},
			},
		},
		{
			name: "sums to 99",
			members: []domain.SHGMemberShare{
				{ArtisanID: artisanA, SharePct: 50},
				{ArtisanID: artisanB, SharePct: 49},
			},
			wantErr: "sum to 99",
		},
		{
			name: "sums to 101",
			members: []domain.SHGMemberShare{
				{ArtisanID: artisanA, SharePct: 51},
				{ArtisanID: artisanB, SharePct: 50},
			},
			wantErr: "sum to 101",
		},
		{
			name:    "empty roster",
			members: nil,
			wantErr: "at least one member",
		},
		{
			name: "duplicate artisan",
			members: []domain.SHGMemberShare{
				{ArtisanID: artisanA, SharePct: 50},
				{ArtisanID: artisanA, SharePct: 50},
			},
			wantErr: "appears twice",
		},
		{
			name:    "nil artisan id",
			members: []domain.SHGMemberShare{{ArtisanID: uuid.Nil, SharePct: 100}},
			wantErr: "must not be nil",
		},
		{
			name: "negative share",
			members: []domain.SHGMemberShare{
				{ArtisanID: artisanA, SharePct: -10},
				{ArtisanID: artisanB, SharePct: 110},
			},
			wantErr: "must be between 0 and 100",
		},
		{
			name: "a share above 100 even though the total is right",
			members: []domain.SHGMemberShare{
				{ArtisanID: artisanA, SharePct: 101},
				{ArtisanID: artisanB, SharePct: -1},
			},
			wantErr: "must be between 0 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateSHGShares(tt.members)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected the roster to be valid, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error mentioning %q", tt.wantErr)
			}
			if !errors.Is(err, pkgdomain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not mention %q", err, tt.wantErr)
			}
		})
	}
}

func validSHGInput() domain.CreateSHGInput {
	return domain.CreateSHGInput{
		Name:               "Kutch Weavers Collective",
		RegistrationNo:     "GJ-SHG-1042",
		SignatoryArtisanID: &artisanA,
		Members: []domain.SHGMemberShare{
			{ArtisanID: artisanA, SharePct: 40},
			{ArtisanID: artisanB, SharePct: 35},
			{ArtisanID: artisanC, SharePct: 25},
		},
	}
}

func TestCreateSelfHelpGroupStoresGroupAndRoster(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	group, roster, err := svc.CreateSelfHelpGroup(officerSubjectCtx("officer-1"), validSHGInput())
	if err != nil {
		t.Fatalf("CreateSelfHelpGroup: %v", err)
	}
	if group.ID == uuid.Nil {
		t.Fatal("created group has a nil id")
	}
	if len(roster) != 3 {
		t.Fatalf("roster has %d members, want 3", len(roster))
	}

	var total int32
	for _, m := range roster {
		total += m.SharePct
	}
	if total != domain.TotalSharePct {
		t.Fatalf("stored roster sums to %d, want 100", total)
	}
	if _, ok := store.shgs[group.ID]; !ok {
		t.Fatal("group row was not committed")
	}
}

func TestCreateSelfHelpGroupRejectsABadShareSplit(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	in := validSHGInput()
	in.Members[0].SharePct = 41 // now sums to 101

	_, _, err := svc.CreateSelfHelpGroup(officerSubjectCtx("officer-1"), in)
	if err == nil {
		t.Fatal("expected a roster summing to 101 to be rejected")
	}
	if !errors.Is(err, pkgdomain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if len(store.shgs) != 0 {
		t.Fatal("no group should exist after a rejected roster")
	}
}

func TestCreateSelfHelpGroupRejectsANonMemberSignatory(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{})
	in := validSHGInput()
	outsider := uuid.New()
	in.SignatoryArtisanID = &outsider

	_, _, err := svc.CreateSelfHelpGroup(officerSubjectCtx("officer-1"), in)
	if err == nil || !strings.Contains(err.Error(), "not one of the group's members") {
		t.Fatalf("expected the signatory check to fire, got %v", err)
	}
}

func TestSetSelfHelpGroupMembersReplacesRosterWholesale(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	group, _, err := svc.CreateSelfHelpGroup(officerSubjectCtx("officer-1"), validSHGInput())
	if err != nil {
		t.Fatalf("CreateSelfHelpGroup: %v", err)
	}

	roster, err := svc.SetSelfHelpGroupMembers(officerSubjectCtx("officer-1"), group.ID, []domain.SHGMemberShare{
		{ArtisanID: artisanA, SharePct: 60},
		{ArtisanID: artisanB, SharePct: 40},
	})
	if err != nil {
		t.Fatalf("SetSelfHelpGroupMembers: %v", err)
	}
	if len(roster) != 2 {
		t.Fatalf("roster has %d members after replacement, want 2", len(roster))
	}
	// artisanC must be gone: this is a replacement, not a merge.
	for _, m := range roster {
		if m.ArtisanID == artisanC {
			t.Fatal("artisanC should have been removed by the wholesale replacement")
		}
	}
}

func TestSetSelfHelpGroupMembersRejectsABadSplitWithoutTouchingTheStoredRoster(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	group, _, err := svc.CreateSelfHelpGroup(officerSubjectCtx("officer-1"), validSHGInput())
	if err != nil {
		t.Fatalf("CreateSelfHelpGroup: %v", err)
	}

	_, err = svc.SetSelfHelpGroupMembers(officerSubjectCtx("officer-1"), group.ID, []domain.SHGMemberShare{
		{ArtisanID: artisanA, SharePct: 60},
		{ArtisanID: artisanB, SharePct: 50},
	})
	if !errors.Is(err, pkgdomain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}

	// The original roster must be untouched.
	stored, err := svc.ListSelfHelpGroupMembers(officerSubjectCtx("officer-1"), group.ID)
	if err != nil {
		t.Fatalf("ListSelfHelpGroupMembers: %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("stored roster has %d members, want the original 3", len(stored))
	}
}

func TestSetSelfHelpGroupMembersAuthorisation(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})

	group, _, err := svc.CreateSelfHelpGroup(officerSubjectCtx("officer-1"), validSHGInput())
	if err != nil {
		t.Fatalf("CreateSelfHelpGroup: %v", err)
	}
	newRoster := []domain.SHGMemberShare{
		{ArtisanID: artisanA, SharePct: 60},
		{ArtisanID: artisanB, SharePct: 40},
	}

	// The signatory (artisanA) may change the split.
	signatory := artisanPhoneCtx(artisanA.String(), testPhone)
	if _, err := svc.SetSelfHelpGroupMembers(signatory, group.ID, newRoster); err != nil {
		t.Fatalf("the signatory should be allowed: %v", err)
	}

	// An ordinary member may not: these numbers divide real money.
	member := artisanPhoneCtx(artisanB.String(), "+919000000002")
	if _, err := svc.SetSelfHelpGroupMembers(member, group.ID, newRoster); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden for a non-signatory member, got %v", err)
	}

	// An unauthenticated caller may not.
	if _, err := svc.SetSelfHelpGroupMembers(context.Background(), group.ID, newRoster); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden for an unauthenticated caller, got %v", err)
	}
}

func TestSetSelfHelpGroupMembersRejectsAnUnknownGroup(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{})
	_, err := svc.SetSelfHelpGroupMembers(officerSubjectCtx("o1"), uuid.New(), []domain.SHGMemberShare{
		{ArtisanID: artisanA, SharePct: 100},
	})
	if !errors.Is(err, pkgdomain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// --- cluster -----------------------------------------------------------------

func TestCreateClusterRequiresAnOfficer(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{})
	in := domain.CreateClusterInput{Name: "Kutch", Region: domain.Region{StateCode: "IN-GJ"}}

	if _, err := svc.CreateCluster(officerSubjectCtx("o1"), in); err != nil {
		t.Fatalf("officer should be allowed: %v", err)
	}

	artisan := artisanPhoneCtx(uuid.New().String(), testPhone)
	if _, err := svc.CreateCluster(artisan, in); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden for an artisan, got %v", err)
	}

	ministry := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: "m1", Role: auth.RoleMinistry})
	if _, err := svc.CreateCluster(ministry, in); err != nil {
		t.Fatalf("ministry should be allowed: %v", err)
	}
}

func TestCreateClusterValidation(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{})
	ctx := officerSubjectCtx("o1")

	tests := []struct {
		name string
		in   domain.CreateClusterInput
		want string
	}{
		{"blank name", domain.CreateClusterInput{Name: " ", Region: domain.Region{StateCode: "IN-GJ"}}, "name is required"},
		{"no state", domain.CreateClusterInput{Name: "K"}, "state_code is required"},
		{"bad coordinator phone", domain.CreateClusterInput{
			Name: "K", Region: domain.Region{StateCode: "IN-GJ"}, CoordinatorPhoneE164: ptr("12345"),
		}, "must be E.164"},
		{"bad pincode", domain.CreateClusterInput{
			Name: "K", Region: domain.Region{StateCode: "IN-GJ", Pincode: ptr("000000")},
		}, "six-digit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateCluster(ctx, tt.in)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected an error mentioning %q, got %v", tt.want, err)
			}
		})
	}
}

func TestAddAndRemoveClusterMember(t *testing.T) {
	store := newFakeStore()
	svc := newTestIdentity(store, newFakeTokens(), &fakeOTP{})
	ctx := officerSubjectCtx("o1")

	cluster, err := svc.CreateCluster(ctx, domain.CreateClusterInput{
		Name: "Kutch", Region: domain.Region{StateCode: "IN-GJ"},
	})
	if err != nil {
		t.Fatalf("CreateCluster: %v", err)
	}

	if _, err := svc.AddClusterMember(ctx, cluster.ID, artisanA, domain.ClusterRoleCoordinator); err != nil {
		t.Fatalf("AddClusterMember: %v", err)
	}
	if got := store.clusterMem[cluster.ID][artisanA].Role; got != domain.ClusterRoleCoordinator {
		t.Errorf("role = %q, want COORDINATOR", got)
	}

	// An unspecified role defaults to MEMBER rather than being rejected.
	if _, err := svc.AddClusterMember(ctx, cluster.ID, artisanB, ""); err != nil {
		t.Fatalf("AddClusterMember with a default role: %v", err)
	}
	if got := store.clusterMem[cluster.ID][artisanB].Role; got != domain.ClusterRoleMember {
		t.Errorf("default role = %q, want MEMBER", got)
	}

	// An unknown role is rejected.
	if _, err := svc.AddClusterMember(ctx, cluster.ID, artisanC, domain.ClusterMemberRole("PRESIDENT")); !errors.Is(err, pkgdomain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for an unknown role, got %v", err)
	}

	removed, err := svc.RemoveClusterMember(ctx, cluster.ID, artisanA)
	if err != nil {
		t.Fatalf("RemoveClusterMember: %v", err)
	}
	if !removed {
		t.Error("removing a present member should report true")
	}

	// Removing again is a no-op, not an error.
	removed, err = svc.RemoveClusterMember(ctx, cluster.ID, artisanA)
	if err != nil {
		t.Fatalf("repeated RemoveClusterMember: %v", err)
	}
	if removed {
		t.Error("removing an absent member should report false")
	}
}

func TestClusterMemberMutationsRequireAnOfficer(t *testing.T) {
	svc := newTestIdentity(newFakeStore(), newFakeTokens(), &fakeOTP{})
	artisan := artisanPhoneCtx(artisanA.String(), testPhone)

	if _, err := svc.AddClusterMember(artisan, uuid.New(), artisanB, domain.ClusterRoleMember); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	if _, err := svc.RemoveClusterMember(artisan, uuid.New(), artisanB); !errors.Is(err, pkgdomain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}
