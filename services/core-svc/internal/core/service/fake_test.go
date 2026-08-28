// services/core-svc/internal/core/service/fake_test.go
package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// outboxRow is what the fake records when the service enqueues an event.
type outboxRow struct {
	ID             string
	AggregateID    string
	Topic          string
	IdempotencyKey string
	Payload        []byte
}

// fakeStore is an in-memory Store + Tx. It models the two behaviours the tests
// care about that a map alone would not: the phone-uniqueness constraint, and
// the all-or-nothing transaction boundary — a failed InTx callback discards
// every write the callback made, exactly as a ROLLBACK would.
type fakeStore struct {
	mu sync.Mutex

	artisans   map[uuid.UUID]domain.Artisan
	byPhone    map[string]uuid.UUID
	clusters   map[uuid.UUID]domain.Cluster
	clusterMem map[uuid.UUID]map[uuid.UUID]domain.ClusterMember
	shgs       map[uuid.UUID]domain.SelfHelpGroup
	shgMem     map[uuid.UUID][]domain.SHGMember
	outbox     []outboxRow

	// failInTxAfterCallback simulates a COMMIT that fails, to prove the service
	// surfaces the error and reports nothing as created.
	failInTxAfterCallback error
	// createArtisanErr forces CreateArtisan to fail, e.g. to simulate the
	// database's own unique-constraint rejection.
	createArtisanErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		artisans:   map[uuid.UUID]domain.Artisan{},
		byPhone:    map[string]uuid.UUID{},
		clusters:   map[uuid.UUID]domain.Cluster{},
		clusterMem: map[uuid.UUID]map[uuid.UUID]domain.ClusterMember{},
		shgs:       map[uuid.UUID]domain.SelfHelpGroup{},
		shgMem:     map[uuid.UUID][]domain.SHGMember{},
	}
}

// fakeTx buffers every write and applies them to the store only on commit, so a
// failed transaction leaves no trace.
type fakeTx struct {
	store  *fakeStore
	writes []func()
	// pendingSHGMem serves read-your-own-writes for rosters written in this tx.
	pendingSHGMem map[uuid.UUID][]domain.SHGMember
}

func (s *fakeStore) InTx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error {
	tx := &fakeTx{store: s}
	if err := fn(ctx, tx); err != nil {
		return err // buffered writes are discarded
	}
	if s.failInTxAfterCallback != nil {
		return s.failInTxAfterCallback // commit failed: still discard
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range tx.writes {
		w()
	}
	return nil
}

func (t *fakeTx) CreateArtisan(_ context.Context, id uuid.UUID, in domain.RegisterArtisanInput) (domain.Artisan, error) {
	if t.store.createArtisanErr != nil {
		return domain.Artisan{}, t.store.createArtisanErr
	}

	t.store.mu.Lock()
	_, taken := t.store.byPhone[in.PhoneE164]
	t.store.mu.Unlock()
	if taken {
		// Mirrors the artisan_phone_e164_key unique constraint as the repo
		// would translate it.
		return domain.Artisan{}, fmt.Errorf("artisan already exists (artisan_phone_e164_key): %w", pkgdomain.ErrConflict)
	}

	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	a := domain.Artisan{
		ID:                id,
		DisplayName:       in.DisplayName,
		PhoneE164:         in.PhoneE164,
		PehchanID:         in.PehchanID,
		PMVishwakarmaID:   in.PMVishwakarmaID,
		PrimaryClusterID:  in.ClusterID,
		Region:            in.Region,
		Languages:         in.Languages,
		CraftIDs:          in.CraftIDs,
		YearsOfExperience: in.YearsOfExperience,
		Bio:               in.Bio,
		CreatedBy:         in.CreatedBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	t.writes = append(t.writes, func() {
		t.store.artisans[id] = a
		t.store.byPhone[a.PhoneE164] = id
	})
	return a, nil
}

func (t *fakeTx) InsertOutbox(_ context.Context, id, aggregateID, topic, idempotencyKey string, payload []byte) error {
	row := outboxRow{ID: id, AggregateID: aggregateID, Topic: topic, IdempotencyKey: idempotencyKey, Payload: payload}
	t.writes = append(t.writes, func() {
		t.store.outbox = append(t.store.outbox, row)
	})
	return nil
}

func (t *fakeTx) CreateCluster(_ context.Context, id uuid.UUID, in domain.CreateClusterInput) (domain.Cluster, error) {
	c := domain.Cluster{
		ID:                   id,
		Name:                 in.Name,
		Region:               in.Region,
		CoordinatorPhoneE164: in.CoordinatorPhoneE164,
	}
	t.writes = append(t.writes, func() { t.store.clusters[id] = c })
	return c, nil
}

func (t *fakeTx) UpsertClusterMember(_ context.Context, clusterID, artisanID uuid.UUID, role domain.ClusterMemberRole) error {
	t.writes = append(t.writes, func() {
		if t.store.clusterMem[clusterID] == nil {
			t.store.clusterMem[clusterID] = map[uuid.UUID]domain.ClusterMember{}
		}
		t.store.clusterMem[clusterID][artisanID] = domain.ClusterMember{
			ClusterID: clusterID, ArtisanID: artisanID, Role: role,
		}
	})
	return nil
}

func (t *fakeTx) RemoveClusterMember(_ context.Context, clusterID, artisanID uuid.UUID) (bool, error) {
	t.store.mu.Lock()
	_, present := t.store.clusterMem[clusterID][artisanID]
	t.store.mu.Unlock()

	t.writes = append(t.writes, func() { delete(t.store.clusterMem[clusterID], artisanID) })
	return present, nil
}

func (t *fakeTx) CreateSHG(_ context.Context, id uuid.UUID, in domain.CreateSHGInput) (domain.SelfHelpGroup, error) {
	g := domain.SelfHelpGroup{
		ID:                 id,
		Name:               in.Name,
		RegistrationNo:     in.RegistrationNo,
		ClusterID:          in.ClusterID,
		SignatoryArtisanID: in.SignatoryArtisanID,
	}
	t.writes = append(t.writes, func() { t.store.shgs[id] = g })
	return g, nil
}

func (t *fakeTx) ReplaceSHGMembers(_ context.Context, shgID uuid.UUID, members []domain.SHGMemberShare) error {
	// Mirrors the deferred database trigger: whatever path a write takes, a
	// roster that does not sum to 100 is refused at commit.
	if err := domain.ValidateSHGShares(members); err != nil {
		return fmt.Errorf("shg_member_shares_sum_to_100: %w", err)
	}
	roster := make([]domain.SHGMember, 0, len(members))
	for _, m := range members {
		roster = append(roster, domain.SHGMember{SHGID: shgID, ArtisanID: m.ArtisanID, SharePct: m.SharePct})
	}
	if t.pendingSHGMem == nil {
		t.pendingSHGMem = map[uuid.UUID][]domain.SHGMember{}
	}
	t.pendingSHGMem[shgID] = roster
	t.writes = append(t.writes, func() { t.store.shgMem[shgID] = roster })
	return nil
}

// ListSHGMembers reads the transaction's own uncommitted writes first, so a
// service that writes a roster then reads it back inside one transaction sees
// what the real database would show it.
func (t *fakeTx) ListSHGMembers(_ context.Context, shgID uuid.UUID) ([]domain.SHGMember, error) {
	if roster, ok := t.pendingSHGMem[shgID]; ok {
		return roster, nil
	}
	t.store.mu.Lock()
	defer t.store.mu.Unlock()
	return t.store.shgMem[shgID], nil
}

// --- read side ---------------------------------------------------------------

func (s *fakeStore) GetArtisan(_ context.Context, id uuid.UUID) (domain.Artisan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.artisans[id]
	if !ok {
		return domain.Artisan{}, fmt.Errorf("artisan not found: %w", pkgdomain.ErrNotFound)
	}
	return a, nil
}

func (s *fakeStore) GetArtisanByPhone(_ context.Context, phone string) (domain.Artisan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byPhone[phone]
	if !ok {
		return domain.Artisan{}, fmt.Errorf("artisan not found: %w", pkgdomain.ErrNotFound)
	}
	return s.artisans[id], nil
}

func (s *fakeStore) ArtisanExistsByPhone(_ context.Context, phone string) (uuid.UUID, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byPhone[phone]
	return id, ok, nil
}

func (s *fakeStore) UpdateArtisan(_ context.Context, in domain.UpdateArtisanInput) (domain.Artisan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.artisans[in.ArtisanID]
	if !ok {
		return domain.Artisan{}, fmt.Errorf("artisan not found: %w", pkgdomain.ErrNotFound)
	}
	if in.DisplayName != nil {
		a.DisplayName = *in.DisplayName
	}
	if len(in.Languages) > 0 {
		a.Languages = in.Languages
	}
	if in.Bio != nil {
		a.Bio = in.Bio
	}
	s.artisans[in.ArtisanID] = a
	return a, nil
}

func (s *fakeStore) ListArtisansByCluster(_ context.Context, clusterID uuid.UUID, page domain.Page) ([]domain.Artisan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.Artisan
	for _, a := range s.artisans {
		if a.PrimaryClusterID != nil && *a.PrimaryClusterID == clusterID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (s *fakeStore) GetCluster(_ context.Context, id uuid.UUID) (domain.Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[id]
	if !ok {
		return domain.Cluster{}, fmt.Errorf("cluster not found: %w", pkgdomain.ErrNotFound)
	}
	return c, nil
}

func (s *fakeStore) UpdateCluster(_ context.Context, in domain.UpdateClusterInput) (domain.Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[in.ClusterID]
	if !ok {
		return domain.Cluster{}, fmt.Errorf("cluster not found: %w", pkgdomain.ErrNotFound)
	}
	if in.Name != nil {
		c.Name = *in.Name
	}
	s.clusters[in.ClusterID] = c
	return c, nil
}

func (s *fakeStore) GetClusterMember(_ context.Context, clusterID, artisanID uuid.UUID) (domain.ClusterMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.clusterMem[clusterID][artisanID]
	if !ok {
		return domain.ClusterMember{}, fmt.Errorf("cluster member not found: %w", pkgdomain.ErrNotFound)
	}
	return m, nil
}

func (s *fakeStore) ListClusterMembers(_ context.Context, clusterID uuid.UUID, page domain.Page) ([]domain.ClusterMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.ClusterMember
	for _, m := range s.clusterMem[clusterID] {
		out = append(out, m)
	}
	return out, nil
}

func (s *fakeStore) GetSHG(_ context.Context, id uuid.UUID) (domain.SelfHelpGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.shgs[id]
	if !ok {
		return domain.SelfHelpGroup{}, fmt.Errorf("self-help group not found: %w", pkgdomain.ErrNotFound)
	}
	return g, nil
}

func (s *fakeStore) ListSHGMembers(_ context.Context, shgID uuid.UUID) ([]domain.SHGMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shgMem[shgID], nil
}

// --- other fakes -------------------------------------------------------------

// fakeTokens issues predictable tokens without cryptography.
type fakeTokens struct {
	issued   []auth.Subject
	issueErr error
	verify   map[string]*auth.Claims
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{verify: map[string]*auth.Claims{}}
}

func (f *fakeTokens) Issue(sub auth.Subject) (auth.TokenPair, error) {
	if f.issueErr != nil {
		return auth.TokenPair{}, f.issueErr
	}
	f.issued = append(f.issued, sub)
	return auth.TokenPair{
		AccessToken:      "access-" + sub.ID,
		RefreshToken:     "refresh-" + sub.ID,
		AccessExpiresAt:  time.Date(2026, 8, 26, 12, 15, 0, 0, time.UTC),
		RefreshExpiresAt: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (f *fakeTokens) Verify(token string, want auth.TokenKind) (*auth.Claims, error) {
	c, ok := f.verify[token]
	if !ok {
		return nil, fmt.Errorf("token rejected: %w", pkgdomain.ErrForbidden)
	}
	return c, nil
}

// fakeOTP records requests and accepts one configured code.
type fakeOTP struct {
	requested  []string
	acceptCode string
	requestErr error
	devMode    bool
}

func (f *fakeOTP) Request(_ context.Context, phone, language string) (Challenge, error) {
	if f.requestErr != nil {
		return Challenge{}, f.requestErr
	}
	f.requested = append(f.requested, phone)
	return Challenge{
		ID:        "challenge-1",
		ExpiresAt: time.Date(2026, 8, 26, 12, 5, 0, 0, time.UTC),
		DevMode:   f.devMode,
	}, nil
}

func (f *fakeOTP) Verify(_ context.Context, challengeID, phone, code string) error {
	if challengeID == "challenge-1" && code == f.acceptCode {
		return nil
	}
	return fmt.Errorf("otp challenge is not valid: %w", pkgdomain.ErrForbidden)
}

func (f *fakeOTP) DevMode() bool { return f.devMode }

// --- helpers -----------------------------------------------------------------

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// newTestIdentity assembles a service over fakes, with a fixed clock so event
// timestamps are deterministic.
func newTestIdentity(store *fakeStore, tokens *fakeTokens, otp *fakeOTP) *Identity {
	svc := NewIdentity(store, tokens, otp, discardLogger())
	svc.now = func() time.Time { return time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC) }
	return svc
}

// artisanCtx builds a context for a logged-in artisan who verified phone.
func artisanCtx(subject, phone string) context.Context {
	return auth.ContextWithPrincipal(context.Background(), auth.Principal{
		Subject:   subject,
		Role:      auth.RoleArtisan,
		PhoneE164: phone,
	})
}

// officerCtx builds a context for a cluster officer.
func officerCtx(subject string) context.Context {
	return auth.ContextWithPrincipal(context.Background(), auth.Principal{
		Subject: subject,
		Role:    auth.RoleClusterOfficer,
	})
}

func ptr[T any](v T) *T { return &v }
