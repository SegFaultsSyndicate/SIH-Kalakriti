// pkg/logger/logger.go

// Package logger builds the JSON slog.Logger every service uses, and carries
// request_id / trace_id through context so a log line can always be tied back
// to the call that produced it.
package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	traceIDKey
	loggerKey
)

// New builds the root JSON logger. level is case-insensitive; an unrecognised
// value defaults to info rather than failing startup over a typo.
func New(level string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(h)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ContextWithRequestID attaches a request id to ctx.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// ContextWithTraceID attaches a trace id to ctx.
func ContextWithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// RequestID returns the request id carried by ctx, or "" if none was attached.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// TraceID returns the trace id carried by ctx, or "" if none was attached.
func TraceID(ctx context.Context) string {
	id, _ := ctx.Value(traceIDKey).(string)
	return id
}

// With returns base annotated with whatever request_id and trace_id ctx carries.
// Safe to call with a ctx that carries neither: base is returned unchanged.
func With(ctx context.Context, base *slog.Logger) *slog.Logger {
	l := base
	if id := RequestID(ctx); id != "" {
		l = l.With("request_id", id)
	}
	if id := TraceID(ctx); id != "" {
		l = l.With("trace_id", id)
	}
	return l
}

// FromContext returns the request-scoped logger Middleware injected, or base if
// none was (a background job, a test, anything outside an HTTP request).
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return base
}

// Middleware injects a request-scoped logger into the request context, annotated
// with a request id (chi's, generated if chi's own middleware is not mounted)
// and a trace id (the X-Trace-Id header, when a caller sets one).
func Middleware(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			if reqID == "" {
				reqID = uuid.NewString()
			}
			traceID := r.Header.Get("X-Trace-Id")

			ctx := ContextWithRequestID(r.Context(), reqID)
			if traceID != "" {
				ctx = ContextWithTraceID(ctx, traceID)
			}
			l := With(ctx, base)
			ctx = context.WithValue(ctx, loggerKey, l)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
