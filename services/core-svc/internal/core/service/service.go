// services/core-svc/internal/core/service/service.go

// Package service holds core-svc's business logic. It takes and returns domain
// types only: protobuf is converted in the handler, sqlc rows in the repo.
// Nothing here parses a JWT — the interceptor has already done that, and the
// caller arrives as an auth.Principal on the context.
package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// Tx is the transactional surface the service writes through. It embeds
// outbox.Enqueuer so a business write and its event can be handed to
// outbox.Enqueue inside the same transaction.
type Tx interface {
	outbox.Enqueuer

	CreateArtisan(ctx context.Context, id uuid.UUID, in domain.RegisterArtisanInput) (domain.Artisan, error)
	CreateCluster(ctx context.Context, id uuid.UUID, in domain.CreateClusterInput) (domain.Cluster, error)
	UpsertClusterMember(ctx context.Context, clusterID, artisanID uuid.UUID, role domain.ClusterMemberRole) error
	RemoveClusterMember(ctx context.Context, clusterID, artisanID uuid.UUID) (bool, error)
	CreateSHG(ctx context.Context, id uuid.UUID, in domain.CreateSHGInput) (domain.SelfHelpGroup, error)
	ReplaceSHGMembers(ctx context.Context, shgID uuid.UUID, members []domain.SHGMemberShare) error
	ListSHGMembers(ctx context.Context, shgID uuid.UUID) ([]domain.SHGMember, error)
}

// Store is the persistence surface the service reads through, plus the
// transaction entry point. Declaring it as an interface here is what lets the
// service tests run against a fake with no database.
type Store interface {
	InTx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error

	GetArtisan(ctx context.Context, id uuid.UUID) (domain.Artisan, error)
	GetArtisanByPhone(ctx context.Context, phone string) (domain.Artisan, error)
	ArtisanExistsByPhone(ctx context.Context, phone string) (uuid.UUID, bool, error)
	UpdateArtisan(ctx context.Context, in domain.UpdateArtisanInput) (domain.Artisan, error)
	ListArtisansByCluster(ctx context.Context, clusterID uuid.UUID, page domain.Page) ([]domain.Artisan, error)

	GetCluster(ctx context.Context, id uuid.UUID) (domain.Cluster, error)
	UpdateCluster(ctx context.Context, in domain.UpdateClusterInput) (domain.Cluster, error)
	GetClusterMember(ctx context.Context, clusterID, artisanID uuid.UUID) (domain.ClusterMember, error)
	ListClusterMembers(ctx context.Context, clusterID uuid.UUID, page domain.Page) ([]domain.ClusterMember, error)

	GetSHG(ctx context.Context, id uuid.UUID) (domain.SelfHelpGroup, error)
	ListSHGMembers(ctx context.Context, shgID uuid.UUID) ([]domain.SHGMember, error)
}

// TokenIssuer is the slice of pkg/auth.Issuer the service needs to mint tokens.
type TokenIssuer interface {
	Issue(sub auth.Subject) (auth.TokenPair, error)
	Verify(token string, want auth.TokenKind) (*auth.Claims, error)
}

// Challenger is the slice of OTPService the identity service needs, declared as
// an interface so tests can drive login without Redis.
type Challenger interface {
	Request(ctx context.Context, phone, language string) (Challenge, error)
	Verify(ctx context.Context, challengeID, phone, code string) error
	DevMode() bool
}

// Revocations is the refresh-token-replay and session-revocation surface the
// identity service checks on every refresh — satisfied by
// *pkg/auth.RevocationStore. A nil Revocations (the interface's zero value)
// skips both checks entirely, so existing tests need not construct one.
type Revocations interface {
	TryBurnJTI(ctx context.Context, jti string, ttl time.Duration) (fresh bool, err error)
	RevokeSubject(ctx context.Context, subject string, ttl time.Duration) error
	RevokedBefore(ctx context.Context, subject string, issuedAt time.Time) (bool, error)
}

// Identity is core-svc's identity and collective-management service.
type Identity struct {
	store       Store
	tokens      TokenIssuer
	otp         Challenger
	revocations Revocations
	log         *slog.Logger
	// now is injected so tests can assert on timestamps deterministically.
	now func() time.Time
}

// NewIdentity builds the identity service. Every dependency is injected; the
// service holds no global state and does no work until a method is called.
// revocations may be nil to skip refresh-token replay and session-revocation
// checks entirely.
func NewIdentity(store Store, tokens TokenIssuer, otp Challenger, revocations Revocations, log *slog.Logger) *Identity {
	return &Identity{store: store, tokens: tokens, otp: otp, revocations: revocations, log: log, now: time.Now}
}

// event is the envelope written to the outbox. It mirrors events.v1.EventHeader
// so a consumer decoding the JSON payload sees the same shape the proto defines.
type event struct {
	Header  eventHeader `json:"header"`
	Payload any         `json:"payload"`
}

// eventHeader mirrors events.v1.EventHeader.
type eventHeader struct {
	EventID        string    `json:"event_id"`
	OccurredAt     time.Time `json:"occurred_at"`
	AggregateID    string    `json:"aggregate_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	SchemaVersion  int32     `json:"schema_version"`
	Producer       string    `json:"producer"`
	CorrelationID  string    `json:"correlation_id"`
}

// producerName identifies this service in every event it emits.
const producerName = "core-svc"

// schemaVersion is bumped when an event payload's shape changes incompatibly.
const schemaVersion int32 = 1

// newEvent builds an outbox envelope for one aggregate.
func (s *Identity) newEvent(aggregateID uuid.UUID, idempotencyKey string, payload any) event {
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
