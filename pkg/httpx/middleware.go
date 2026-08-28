// pkg/httpx/middleware.go

// Package httpx provides the chi middleware stack shared by every service's
// REST edge (today, just the bff): request id propagation, structured access
// logging, panic recovery, CORS, and request timeouts.
package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/segfaultsyndicate/kalakriti/pkg/logger"
)

// Config controls the middleware stack Mux assembles.
type Config struct {
	// Logger is the base logger each request's child logger is derived from.
	Logger *slog.Logger
	// AllowedOrigins is the CORS allowlist. "*" allows any origin (credentials
	// are then never reflected, per the CORS spec). Empty disables CORS
	// entirely (no Access-Control-* headers are set).
	AllowedOrigins []string
	// AllowedMethods defaults to a REST-typical set when empty.
	AllowedMethods []string
	// AllowedHeaders defaults to a small safe set when empty.
	AllowedHeaders []string
	// RequestTimeout bounds how long a handler may run before the request
	// context is cancelled and 503 is returned. Zero disables the timeout.
	RequestTimeout time.Duration
}

// Mux builds a *chi.Mux with the standard middleware stack pre-mounted:
// request id -> panic recovery -> request-scoped logger -> access log ->
// CORS -> timeout. Routes are added by the caller.
func Mux(cfg Config) *chi.Mux {
	base := cfg.Logger
	if base == nil {
		base = slog.Default()
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(Recoverer(base))
	r.Use(logger.Middleware(base))
	r.Use(AccessLog(base))
	if len(cfg.AllowedOrigins) > 0 {
		r.Use(CORS(cfg))
	}
	if cfg.RequestTimeout > 0 {
		r.Use(chimw.Timeout(cfg.RequestTimeout))
	}
	return r
}

// Recoverer converts a panic anywhere downstream into a 500 response instead
// of crashing the process, logging the panic value and a stack trace.
func Recoverer(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.FromContext(r.Context(), base).Error("panic recovered",
						"panic", fmt.Sprintf("%v", rec),
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
					)
					Error(w, fmt.Errorf("internal error"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder captures the status code written to a ResponseWriter so
// AccessLog can report it after the handler returns.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.status = http.StatusOK
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(b)
}

// AccessLog emits one structured log line per completed request: method,
// path, status, duration, and the request/trace ids logger.Middleware
// attached to the context.
func AccessLog(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			logger.FromContext(r.Context(), base).Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}

// CORS applies the configured allowlist to preflight and actual requests.
// It is hand-rolled rather than pulled from go-chi/cors, since only
// go-chi/chi/v5 itself is in this project's approved dependency list.
func CORS(cfg Config) func(http.Handler) http.Handler {
	methods := cfg.AllowedMethods
	if len(methods) == 0 {
		methods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions}
	}
	headers := cfg.AllowedHeaders
	if len(headers) == 0 {
		headers = []string{"Authorization", "Content-Type", "X-Trace-Id", "X-Idempotency-Key"}
	}
	allowAll := false
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowAll = true
			continue
		}
		allowed[o] = struct{}{}
	}

	methodsHeader := strings.Join(methods, ", ")
	headersHeader := strings.Join(headers, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				_, ok := allowed[origin]
				switch {
				case allowAll:
					w.Header().Set("Access-Control-Allow-Origin", "*")
				case ok:
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", methodsHeader)
				w.Header().Set("Access-Control-Allow-Headers", headersHeader)
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(int((10 * time.Minute).Seconds())))
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// contextKey namespaces values httpx itself stores in the request context,
// distinct from pkg/logger's own keys.
type contextKey int

const authClaimsKey contextKey = iota

// ContextWithAuthClaims attaches decoded JWT claims to ctx for downstream
// handlers to read. Defined here rather than in a dedicated auth package
// since, so far, only the HTTP edge needs claims off the request context.
func ContextWithAuthClaims(ctx context.Context, claims any) context.Context {
	return context.WithValue(ctx, authClaimsKey, claims)
}

// AuthClaims returns the claims attached by ContextWithAuthClaims, or nil.
func AuthClaims(ctx context.Context) any {
	return ctx.Value(authClaimsKey)
}
