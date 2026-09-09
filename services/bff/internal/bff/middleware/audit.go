package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/audit"
	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/logger"
)

// AuditLogger is the compliance-log write surface AuditLog writes mutating
// requests through — satisfied by *pkg/audit.Logger.
type AuditLogger interface {
	Log(ctx context.Context, event audit.Event) error
}

// AuditLog records one audit_log row for every mutating request (POST, PUT,
// PATCH, DELETE) that completes successfully on an authenticated route.
// Routes mount this behind Auth — it needs the request's principal, and a
// nil AuditLogger (e.g. in tests that don't wire one) makes it a no-op.
// Logging is best-effort: a write failure is logged itself but never fails
// the request it describes, since a mutation that already succeeded must
// not be reported as failed because its audit trail didn't write.
func AuditLog(auditLog AuditLogger, base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if auditLog == nil || !isMutatingMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			rec := &auditStatusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			if rec.status >= 400 {
				// Only a mutation that actually took effect is compliance-relevant.
				return
			}

			principal, ok := auth.PrincipalFrom(r.Context())
			if !ok {
				return
			}
			var actorID *uuid.UUID
			if id, err := uuid.Parse(principal.Subject); err == nil {
				actorID = &id
			}
			resourceType, resourceID := resourceFromPath(r.URL.Path)

			event := audit.Event{
				ActorID:      actorID,
				ActorType:    "user",
				Action:       r.Method + " " + r.URL.Path,
				ResourceType: resourceType,
				ResourceID:   resourceID,
				UserAgent:    r.UserAgent(),
			}
			if err := auditLog.Log(r.Context(), event); err != nil {
				logger.FromContext(r.Context(), base).Error("audit log write failed",
					"error", err, "action", event.Action)
			}
		})
	}
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// resourceFromPath derives a coarse resource type/id pair from a
// /api/v1/<type>/<id>/... request path. resourceID is the zero UUID when the
// path has no id segment (e.g. a collection POST) or that segment isn't a
// UUID (e.g. "me" in /artisans/me) — a deliberately coarse, path-based
// mapping rather than per-route resource extraction.
func resourceFromPath(path string) (string, uuid.UUID) {
	segs := strings.Split(strings.Trim(strings.TrimPrefix(path, "/api/v1"), "/"), "/")
	var resourceType string
	if len(segs) > 0 {
		resourceType = segs[0]
	}
	var resourceID uuid.UUID
	if len(segs) > 1 {
		if id, err := uuid.Parse(segs[1]); err == nil {
			resourceID = id
		}
	}
	return resourceType, resourceID
}

// auditStatusRecorder captures the status code written to a ResponseWriter,
// mirroring pkg/httpx's own unexported statusRecorder — kept local rather
// than exported from there, since only this middleware needs it.
type auditStatusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *auditStatusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *auditStatusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.status = http.StatusOK
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(b)
}
