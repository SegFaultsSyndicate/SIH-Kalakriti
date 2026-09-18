// pkg/httpx/middleware.go

// Package httpx provides the gin middleware stack shared by every service's
// REST edge (today, just the bff): request id propagation, structured access
// logging, panic recovery, CORS, and request timeouts.
package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/logger"
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
	// DisableHSTS disables the Strict-Transport-Security header when true (e.g. local plaintext HTTP testing).
	DisableHSTS bool
	// DisableCSRF disables CSRF protection when true (e.g. testing).
	DisableCSRF bool
}

// Mux builds a *gin.Engine with the standard middleware stack pre-mounted:
// request id -> security headers (HSTS) -> panic recovery -> request-scoped logger -> access log ->
// CORS -> CSRF -> timeout.
func Mux(cfg Config) *gin.Engine {
	base := cfg.Logger
	if base == nil {
		base = slog.Default()
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(Wrap(RequestID(base)))
	r.Use(Wrap(SecurityHeaders(!cfg.DisableHSTS)))
	r.Use(Wrap(Recoverer(base)))
	r.Use(Wrap(logger.Middleware(base)))
	// AccessLogGin runs natively on gin, not through Wrap: it must read the
	// status actually written through c.Writer, and Wrap's inner handler
	// always calls c.Next() rather than writing through whatever decorated
	// writer it was given -- see Wrap's doc comment. A net/http-style
	// ResponseWriter decorator (the old statusRecorder) is silently inert
	// there, which is why every access-log line used to say 200.
	r.Use(AccessLogGin(base))
	if len(cfg.AllowedOrigins) > 0 {
		r.Use(Wrap(CORS(cfg)))
		if !cfg.DisableCSRF {
			r.Use(Wrap(CSRFProtection(cfg, base)))
		}
	}
	if cfg.RequestTimeout > 0 {
		// Also native, for the same reason: http.TimeoutHandler's writer
		// wrapper would be discarded by Wrap and never actually bound
		// anything.
		r.Use(TimeoutGin(cfg.RequestTimeout))
	}
	return r
}

// Wrap adapts a plain net/http middleware (func(http.Handler) http.Handler)
// into gin's middleware shape, so the stack above — written once, before this
// package took a router dependency — needs no rewrite for gin: the inner
// handler c.Next()s to continue gin's own chain, letting mw's post-next logic
// (status logging, recover()) still run after the rest of the chain returns.
func Wrap(mw func(http.Handler) http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	}
}

// urlParamKey namespaces the path-param values WrapHandler stashes on the
// request context, distinct from any other context key in play.
type urlParamKey string

// WrapHandler adapts a plain http.HandlerFunc into a gin.HandlerFunc,
// copying gin's path params onto the request context first so the handler
// can read them via URLParam without taking a *gin.Context directly — every
// existing REST handler keeps its net/http signature across the chi->gin
// swap, only the URLParam call sites change.
func WrapHandler(h http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		for _, p := range c.Params {
			ctx = context.WithValue(ctx, urlParamKey(p.Key), p.Value)
		}
		h(c.Writer, c.Request.WithContext(ctx))
	}
}

// URLParam returns the value of a named path parameter previously stashed by
// WrapHandler, mirroring chi.URLParam's signature so handlers written
// against chi needed only an import and callsite swap, not a rewrite.
func URLParam(r *http.Request, name string) string {
	v, _ := r.Context().Value(urlParamKey(name)).(string)
	return v
}

// RequestID reads an inbound X-Request-Id header (letting a caller preserve
// its own id across a retry) or generates a fresh uuid, stashes it in the
// request context via logger.ContextWithRequestID, and echoes it back in
// X-Request-Id so a client can correlate its request across services. This
// replaces chi/v5/middleware.RequestID, which did the same job keyed off
// chi's own request context.
func RequestID(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if id == "" {
				id = uuid.NewString()
			}
			w.Header().Set("X-Request-Id", id)
			next.ServeHTTP(w, r.WithContext(logger.ContextWithRequestID(r.Context(), id)))
		})
	}
}

// timeoutWriter guards a gin.ResponseWriter with a "timed out" latch so a
// handler goroutine that keeps running past its deadline can never write
// conflicting bytes alongside the 503 TimeoutGin already wrote. Mirrors
// http.TimeoutHandler's own internal writer, reimplemented natively for gin
// because http.TimeoutHandler's writer is discarded by Wrap (see Mux).
type timeoutWriter struct {
	gin.ResponseWriter
	mu       sync.Mutex
	timedOut bool
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return
	}
	tw.ResponseWriter.WriteHeader(code)
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return len(b), nil
	}
	return tw.ResponseWriter.Write(b)
}

// markTimedOut latches the writer closed and reports whether it won the
// race -- false means the real handler had already started writing a real
// response, so the deadline should not also try to write a 503 on top of it.
func (tw *timeoutWriter) markTimedOut() bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.ResponseWriter.Written() {
		return false
	}
	tw.timedOut = true
	return true
}

// TimeoutGin bounds how long a handler may run before the request context is
// cancelled and a 503 is returned. Native gin.HandlerFunc, not run through
// Wrap -- see Mux's comment; http.TimeoutHandler's writer wrapper is
// silently discarded there and never actually bounds anything.
//
// SSE routes (path ending "/events") are exempt: a long-lived event stream
// is meant to outlive this deadline, not be cut by it.
//
// Same caveat as http.TimeoutHandler and every other goroutine-based Go
// HTTP timeout: the handler goroutine is not killed on deadline, only
// disconnected from the response. It keeps running (and c is not safe for
// concurrent access by it and the timeout path at once, mirroring gin's own
// documented c.Copy() caveat for such goroutines) until it returns on its
// own; TimeoutGin waits for it before returning, so it never outlives the
// request, but a handler that ignores context cancellation can still hold
// the connection's resources for longer than d.
func TimeoutGin(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasSuffix(c.FullPath(), "/events") {
			c.Next()
			return
		}

		tw := &timeoutWriter{ResponseWriter: c.Writer}
		c.Writer = tw

		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		panicked := make(chan any, 1)
		go func() {
			defer close(done)
			defer func() {
				// Recoverer runs earlier in the chain, in the ORIGINAL
				// goroutine -- its own recover() only catches panics on its
				// own call stack, which this goroutine is not on. Without
				// this, a panic here would be unrecovered and would crash
				// the whole process instead of becoming a 500.
				if p := recover(); p != nil {
					panicked <- p
				}
			}()
			c.Next()
		}()

		select {
		case <-done:
			select {
			case p := <-panicked:
				panic(p)
			default:
			}
		case <-ctx.Done():
			if tw.markTimedOut() {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, domain.HTTPErrorBody{
					Error:   "unavailable",
					Message: "request timed out",
				})
			}
			<-done
			select {
			case p := <-panicked:
				panic(p)
			default:
			}
		}
	}
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

// AccessLogGin emits one structured log line per completed request: method,
// path, status, duration, and the request/trace ids logger.Middleware
// attached to the context. It is a native gin.HandlerFunc (not run through
// Wrap) because gin's own c.Writer.Status() is the only reliable source of
// the real status code here -- see Mux's comment on why a net/http-style
// ResponseWriter decorator doesn't work through Wrap.
func AccessLogGin(base *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.FromContext(c.Request.Context(), base).Info("http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", c.Request.RemoteAddr,
		)
	}
}

// CORS applies the configured allowlist to preflight and actual requests.
// It is hand-rolled rather than pulled from gin-contrib/cors, since only
// gin-gonic/gin itself is in this project's approved dependency list.
func CORS(cfg Config) func(http.Handler) http.Handler {
	methods := cfg.AllowedMethods
	if len(methods) == 0 {
		methods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions}
	}
	headers := cfg.AllowedHeaders
	if len(headers) == 0 {
		headers = []string{"Authorization", "Content-Type", "X-Trace-Id", "Idempotency-Key"}
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

// SecurityHeaders injects defense-in-depth HTTP security headers into every response.
// HSTS enforces modern browser TLS, X-Content-Type-Options blocks MIME-sniffing attacks,
// and X-Frame-Options blocks clickjacking.
func SecurityHeaders(enableHSTS bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if enableHSTS {
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			}
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "camera=(self), microphone=(self), geolocation=()")
			w.Header().Set("X-XSS-Protection", "0")
			next.ServeHTTP(w, r)
		})
	}
}

// CSRFProtection guards mutating requests (POST, PUT, PATCH, DELETE) against Cross-Site Request Forgery.
// It verifies that cross-site requests have an origin matching the configured allowlist or a valid
// custom X-CSRF-Token header.
func CSRFProtection(cfg Config, base *slog.Logger) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Safe methods don't mutate state
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Check Origin or Referer for mutating requests
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; !ok && len(allowed) > 0 {
					logger.FromContext(r.Context(), base).Warn("csrf validation failure: untrusted origin", "origin", origin, "path", r.URL.Path)
					http.Error(w, `{"error":"forbidden: cross-site request forgery protection"}`, http.StatusForbidden)
					return
				}
			} else if referer := r.Header.Get("Referer"); referer != "" {
				if u, err := url.Parse(referer); err == nil && u.Host != "" {
					refOrigin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
					if _, ok := allowed[refOrigin]; !ok && len(allowed) > 0 {
						logger.FromContext(r.Context(), base).Warn("csrf validation failure: untrusted referer", "referer", referer, "path", r.URL.Path)
						http.Error(w, `{"error":"forbidden: cross-site request forgery protection"}`, http.StatusForbidden)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SetSecureCookie configures an HTTP cookie with strict security flags: HttpOnly, Secure, SameSite=Strict.
func SetSecureCookie(w http.ResponseWriter, name, value string, maxAge int, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

