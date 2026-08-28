// services/core-svc/internal/core/repo/repo_integration_test.go

//go:build integration

// This file is behind the `integration` build tag because it starts a real
// PostgreSQL container. Run it with:
//
//	go test -tags=integration ./internal/core/repo/...
//
// or `make test-integration` from the repository root. It is excluded from the
// default `go test ./...` so a plain unit-test run needs no Docker daemon.
package repo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	pkgpostgres "github.com/ZoroNewbie00/kalakriti/pkg/postgres"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// pgImage matches docker-compose.yml so the test runs against the same
// PostgreSQL major version and pgvector build as local development.
const pgImage = "pgvector/pgvector:pg18"

// migrationsDir is the repository's migrations directory, relative to this file.
func migrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "migrations"))
	require.NoError(t, err)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("migrations directory not found at %s: %v", dir, err)
	}
	return dir
}

// startPostgres brings up a throwaway PostgreSQL container, runs every goose
// migration against it, and returns a pool wired the way the service wires it.
func startPostgres(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()

	container, err := postgres.Run(ctx, pgImage,
		postgres.WithDatabase("kalakriti"),
		postgres.WithUsername("kalakriti"),
		postgres.WithPassword("kalakriti"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	require.NoError(t, err, "starting postgres container")
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("terminating postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// goose runs over database/sql; the service's own pool is opened separately
	// below so the test exercises the same pgx path production uses.
	require.NoError(t, goose.SetDialect("postgres"))
	sqlDB, err := goose.OpenDBWithDriver("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, goose.Up(sqlDB, migrationsDir(t)))
	require.NoError(t, sqlDB.Close())

	pool, err := pkgpostgres.New(ctx, pkgpostgres.Config{DSN: dsn, MaxConns: 4, MinConns: 1})
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

// seedCraft inserts a craft row so artisan_craft foreign keys resolve.
func seedCraft(ctx context.Context, t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := ids.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO craft (id, code, display_name) VALUES ($1, $2, $3)`,
		id, "kutch-embroidery-"+id.String()[:8], "Kutch Embroidery")
	require.NoError(t, err)
	return id
}

func registerInput(craftID uuid.UUID, phone string) domain.RegisterArtisanInput {
	district := "Kutch"
	return domain.RegisterArtisanInput{
		DisplayName: "Rukmini Ben",
		PhoneE164:   phone,
		CraftIDs:    []uuid.UUID{craftID},
		Languages:   []string{"GUJARATI", "HINDI"},
		Region:      domain.Region{StateCode: "IN-GJ", District: &district},
		CreatedBy:   "integration-test",
	}
}

func TestRepoRegisterArtisanCommitsArtisanAndOutboxTogether(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)

	craftID := seedCraft(ctx, t, pool)
	artisanID := ids.New()

	err := r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		if _, err := tx.CreateArtisan(ctx, artisanID, registerInput(craftID, "+919876543210")); err != nil {
			return err
		}
		return tx.InsertOutbox(ctx,
			ids.New().String(), artisanID.String(), "artisan.registered", "idem-1", []byte(`{"artisan_id":"x"}`))
	})
	require.NoError(t, err)

	// Both rows must be visible after commit.
	got, err := r.GetArtisan(ctx, artisanID)
	require.NoError(t, err)
	require.Equal(t, "Rukmini Ben", got.DisplayName)
	require.Equal(t, "+919876543210", got.PhoneE164)
	require.Equal(t, []string{"GUJARATI", "HINDI"}, got.Languages)
	require.Equal(t, "IN-GJ", got.Region.StateCode)
	require.Len(t, got.CraftIDs, 1)
	require.Equal(t, craftID, got.CraftIDs[0])

	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM outbox WHERE aggregate_id = $1 AND topic = 'artisan.registered'`,
		artisanID).Scan(&outboxCount))
	require.Equal(t, 1, outboxCount, "the outbox row must have committed with the artisan")
}

func TestRepoRollbackLeavesNoArtisanAndNoOutboxRow(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)

	craftID := seedCraft(ctx, t, pool)
	artisanID := ids.New()
	boom := errors.New("deliberate failure after both writes")

	err := r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		if _, err := tx.CreateArtisan(ctx, artisanID, registerInput(craftID, "+919876543211")); err != nil {
			return err
		}
		if err := tx.InsertOutbox(ctx,
			ids.New().String(), artisanID.String(), "artisan.registered", "idem-2", []byte(`{}`)); err != nil {
			return err
		}
		return boom
	})
	require.ErrorIs(t, err, boom)

	_, err = r.GetArtisan(ctx, artisanID)
	require.ErrorIs(t, err, pkgdomain.ErrNotFound, "the artisan must not survive a rollback")

	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM outbox WHERE aggregate_id = $1`, artisanID).Scan(&outboxCount))
	require.Zero(t, outboxCount, "the outbox row must not survive a rollback")
}

func TestRepoDuplicatePhoneIsTranslatedToErrConflict(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)

	craftID := seedCraft(ctx, t, pool)
	const phone = "+919876543212"

	require.NoError(t, r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		_, err := tx.CreateArtisan(ctx, ids.New(), registerInput(craftID, phone))
		return err
	}))

	err := r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		_, err := tx.CreateArtisan(ctx, ids.New(), registerInput(craftID, phone))
		return err
	})
	require.ErrorIs(t, err, pkgdomain.ErrConflict,
		"the artisan_phone_e164_key unique violation must map to ErrConflict")
	require.Contains(t, err.Error(), "artisan_phone_e164_key",
		"the translated error should name the constraint that fired")
}

func TestRepoUnknownCraftIsTranslatedToErrInvalidInput(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)

	err := r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		_, err := tx.CreateArtisan(ctx, ids.New(), registerInput(uuid.New(), "+919876543213"))
		return err
	})
	require.ErrorIs(t, err, pkgdomain.ErrInvalidInput,
		"a foreign-key violation must map to ErrInvalidInput, not a 500")
}

func TestRepoSHGRosterMustSumTo100AtCommit(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)

	craftID := seedCraft(ctx, t, pool)

	// Two artisans to share the group between.
	a1, a2 := ids.New(), ids.New()
	require.NoError(t, r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		if _, err := tx.CreateArtisan(ctx, a1, registerInput(craftID, "+919876543220")); err != nil {
			return err
		}
		_, err := tx.CreateArtisan(ctx, a2, registerInput(craftID, "+919876543221"))
		return err
	}))

	shgID := ids.New()
	in := domain.CreateSHGInput{Name: "Weavers", RegistrationNo: "GJ-" + shgID.String()[:8]}

	// A roster summing to 100 commits.
	require.NoError(t, r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		if _, err := tx.CreateSHG(ctx, shgID, in); err != nil {
			return err
		}
		return tx.ReplaceSHGMembers(ctx, shgID, []domain.SHGMemberShare{
			{ArtisanID: a1, SharePct: 60},
			{ArtisanID: a2, SharePct: 40},
		})
	}))

	roster, err := r.ListSHGMembers(ctx, shgID)
	require.NoError(t, err)
	require.Len(t, roster, 2)

	// A roster summing to 90 is refused by the deferred trigger at COMMIT, even
	// though every individual statement succeeded.
	err = r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		return tx.ReplaceSHGMembers(ctx, shgID, []domain.SHGMemberShare{
			{ArtisanID: a1, SharePct: 50},
			{ArtisanID: a2, SharePct: 40},
		})
	})
	require.Error(t, err, "a roster summing to 90 must be refused at commit")

	// ...and the original roster is intact.
	roster, err = r.ListSHGMembers(ctx, shgID)
	require.NoError(t, err)
	require.Len(t, roster, 2)
	var total int32
	for _, m := range roster {
		total += m.SharePct
	}
	require.Equal(t, int32(100), total, "the committed roster must still sum to 100")
}

func TestRepoOutboxRelayRoundTrip(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)
	store := NewOutboxStore(r)

	craftID := seedCraft(ctx, t, pool)
	artisanID := ids.New()
	rowID := ids.New().String()

	require.NoError(t, r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		if _, err := tx.CreateArtisan(ctx, artisanID, registerInput(craftID, "+919876543230")); err != nil {
			return err
		}
		return tx.InsertOutbox(ctx, rowID, artisanID.String(), "artisan.registered", "idem-relay", []byte(`{"a":1}`))
	}))

	// The relay claims it...
	batch, err := store.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, batch, 1)
	require.Equal(t, "artisan.registered", batch[0].Topic)
	require.Equal(t, artisanID.String(), batch[0].AggregateID)

	// ...marks it published...
	require.NoError(t, store.MarkPublished(ctx, []string{batch[0].ID}))

	// ...and it is no longer claimable.
	batch, err = store.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, batch, "a published row must not be claimed again")
}

func TestRepoOutboxInsertIsIdempotentPerTopicAndKey(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)

	craftID := seedCraft(ctx, t, pool)
	artisanID := ids.New()

	require.NoError(t, r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		_, err := tx.CreateArtisan(ctx, artisanID, registerInput(craftID, "+919876543240"))
		return err
	}))

	// Two inserts with the same (topic, idempotency_key) must leave one row: the
	// ON CONFLICT DO NOTHING clause is what makes a replayed handler safe.
	for i := 0; i < 2; i++ {
		require.NoError(t, r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
			return tx.InsertOutbox(ctx, ids.New().String(), artisanID.String(),
				"artisan.registered", "same-key", []byte(`{}`))
		}))
	}

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM outbox WHERE topic = 'artisan.registered' AND idempotency_key = 'same-key'`).
		Scan(&count))
	require.Equal(t, 1, count, "a replayed enqueue must not duplicate the event")
}

func TestRepoClusterMembershipRoundTrip(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)

	craftID := seedCraft(ctx, t, pool)
	clusterID := ids.New()
	artisanID := ids.New()

	require.NoError(t, r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		if _, err := tx.CreateCluster(ctx, clusterID, domain.CreateClusterInput{
			Name:   "Bhuj",
			Region: domain.Region{StateCode: "IN-GJ"},
		}); err != nil {
			return err
		}
		if _, err := tx.CreateArtisan(ctx, artisanID, registerInput(craftID, "+919876543250")); err != nil {
			return err
		}
		return tx.UpsertClusterMember(ctx, clusterID, artisanID, domain.ClusterRoleCoordinator)
	}))

	member, err := r.GetClusterMember(ctx, clusterID, artisanID)
	require.NoError(t, err)
	require.Equal(t, domain.ClusterRoleCoordinator, member.Role)
	require.Equal(t, "Rukmini Ben", member.DisplayName)

	cluster, err := r.GetCluster(ctx, clusterID)
	require.NoError(t, err)
	require.Equal(t, int32(1), cluster.ArtisanCount)

	// Upsert must update the role rather than fail on the primary key.
	require.NoError(t, r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		return tx.UpsertClusterMember(ctx, clusterID, artisanID, domain.ClusterRoleMaster)
	}))
	member, err = r.GetClusterMember(ctx, clusterID, artisanID)
	require.NoError(t, err)
	require.Equal(t, domain.ClusterRoleMaster, member.Role)

	var removed bool
	require.NoError(t, r.InTx(ctx, func(ctx context.Context, tx *Tx) error {
		var err error
		removed, err = tx.RemoveClusterMember(ctx, clusterID, artisanID)
		return err
	}))
	require.True(t, removed)

	_, err = r.GetClusterMember(ctx, clusterID, artisanID)
	require.ErrorIs(t, err, pkgdomain.ErrNotFound)
}

func TestRepoGetArtisanNotFound(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)

	_, err := r.GetArtisan(ctx, ids.New())
	require.ErrorIs(t, err, pkgdomain.ErrNotFound)

	_, found, err := r.ArtisanExistsByPhone(ctx, "+919999999999")
	require.NoError(t, err, "a missing phone is not an error, just absent")
	require.False(t, found)
}
