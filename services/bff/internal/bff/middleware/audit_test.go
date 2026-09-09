// services/bff/internal/bff/middleware/audit_test.go
package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZoroNewbie00/kalakriti/pkg/audit"
	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
)

type fakeAuditLogger struct {
	events []audit.Event
}

func (f *fakeAuditLogger) Log(_ context.Context, event audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestAuditLogRecordsSuccessfulMutation(t *testing.T) {
	logger := &fakeAuditLogger{}
	handler := AuditLog(logger, testLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings", nil)
	ctx := auth.ContextWithPrincipal(req.Context(), auth.Principal{Subject: "550e8400-e29b-41d4-a716-446655440000", Role: auth.RoleArtisan})
	req = req.WithContext(ctx)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if len(logger.events) != 1 {
		t.Fatalf("got %d audit events, want 1", len(logger.events))
	}
	ev := logger.events[0]
	if ev.ActorID == nil || ev.ActorID.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ActorID = %v, want the principal's subject", ev.ActorID)
	}
	if ev.ResourceType != "listings" {
		t.Errorf("ResourceType = %q, want listings", ev.ResourceType)
	}
}

func TestAuditLogSkipsReads(t *testing.T) {
	logger := &fakeAuditLogger{}
	handler := AuditLog(logger, testLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/listings", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if len(logger.events) != 0 {
		t.Fatalf("GET should not be audited, got %d events", len(logger.events))
	}
}

func TestAuditLogSkipsFailedMutation(t *testing.T) {
	logger := &fakeAuditLogger{}
	handler := AuditLog(logger, testLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings", nil)
	ctx := auth.ContextWithPrincipal(req.Context(), auth.Principal{Subject: "550e8400-e29b-41d4-a716-446655440000", Role: auth.RoleArtisan})
	req = req.WithContext(ctx)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if len(logger.events) != 0 {
		t.Fatalf("a failed mutation should not be audited, got %d events", len(logger.events))
	}
}

func TestAuditLogNilLoggerIsNoop(t *testing.T) {
	handler := AuditLog(nil, testLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("nil logger should still call through to the handler, got status %d", rec.Code)
	}
}
