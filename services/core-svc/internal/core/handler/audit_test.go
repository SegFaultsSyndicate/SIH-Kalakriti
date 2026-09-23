package handler

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

type recordingAudit struct{ rows []domain.AssistedAudit }

func (r *recordingAudit) InsertAssistedAudit(_ context.Context, a domain.AssistedAudit, _ []byte) error {
	r.rows = append(r.rows, a)
	return nil
}

func TestAuditInterceptor(t *testing.T) {
	agent, artisan, listing := uuid.New(), uuid.New(), uuid.New()
	onBehalf := auth.Principal{Subject: artisan.String(), Role: auth.RoleArtisan, Actor: agent.String()}
	self := auth.Principal{Subject: artisan.String(), Role: auth.RoleArtisan}
	resp := &catalogv1.UpsertListingResponse{Listing: &catalogv1.Listing{Id: listing.String()}}

	cases := []struct {
		name   string
		p      auth.Principal
		method string
		want   int
	}{
		{"agent write is audited", onBehalf, "/catalog.v1.CatalogService/UpsertListing", 1},
		{"agent read is not", onBehalf, "/catalog.v1.CatalogService/GetListing", 0},
		{"artisan's own write is not", self, "/catalog.v1.CatalogService/UpsertListing", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &recordingAudit{}
			ic := AuditInterceptor(w, slog.New(slog.NewTextHandler(io.Discard, nil)))
			ctx := auth.ContextWithPrincipal(context.Background(), tc.p)
			_, err := ic(ctx, nil, &grpc.UnaryServerInfo{FullMethod: tc.method},
				func(context.Context, any) (any, error) { return resp, nil })
			if err != nil {
				t.Fatal(err)
			}
			if len(w.rows) != tc.want {
				t.Fatalf("audit rows = %d, want %d", len(w.rows), tc.want)
			}
			if tc.want == 1 {
				got := w.rows[0]
				if got.ActorID != agent || got.SubjectID != artisan || got.ResourceID != listing || got.Action != tc.method {
					t.Fatalf("audit row = %+v", got)
				}
			}
		})
	}
}
