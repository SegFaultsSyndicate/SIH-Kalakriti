// services/core-svc/cmd/create-staff/main.go

// Command create-staff creates one staff account directly in Postgres. It
// exists to bootstrap the FIRST MINISTRY account -- every later staff account
// (field agents, cluster officers, more ministry users) is created by that
// ministry user through the admin /staff page. The staffer then logs in with
// ordinary phone OTP and receives their role; no dev_role needed.
//
//	go run ./cmd/create-staff -phone +919800000001 -name "Ministry Admin" -role MINISTRY
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	pkgpostgres "github.com/ZoroNewbie00/kalakriti/pkg/postgres"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo"
)

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("create-staff failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		dsn      = flag.String("dsn", os.Getenv("POSTGRES_DSN"), "postgres connection string")
		phone    = flag.String("phone", "", "staff phone, E.164 (e.g. +919800000001)")
		name     = flag.String("name", "", "display name")
		role     = flag.String("role", "MINISTRY", "FIELD_AGENT | CLUSTER_OFFICER | MINISTRY")
		state    = flag.String("state", "", "scope: ISO 3166-2:IN state code, e.g. IN-UP (optional)")
		district = flag.String("district", "", "scope: district name (optional; needs -state)")
		cscID    = flag.String("csc", "", "CSC VLE id for a field agent (optional)")
	)
	flag.Parse()

	if *dsn == "" {
		return errors.New("-dsn is required (or set POSTGRES_DSN)")
	}
	if *phone == "" || *name == "" {
		return errors.New("-phone and -name are required")
	}
	switch *role {
	case "FIELD_AGENT", "CLUSTER_OFFICER", "MINISTRY":
	default:
		return fmt.Errorf("-role %q must be FIELD_AGENT, CLUSTER_OFFICER or MINISTRY", *role)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pkgpostgres.New(ctx, pkgpostgres.Config{DSN: *dsn, MaxConns: 2, MinConns: 1})
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	staff, err := repo.New(pool).CreateStaffAccount(ctx, domain.StaffAccount{
		ID: ids.New(), PhoneE164: *phone, DisplayName: *name, Role: *role,
		StateCode: optional(*state), District: optional(*district), CSCID: optional(*cscID),
		CreatedBy: "cli:create-staff",
	})
	if err != nil {
		return err
	}
	fmt.Printf("created %s staff account %s for %s\n", staff.Role, staff.ID, staff.DisplayName)
	return nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
