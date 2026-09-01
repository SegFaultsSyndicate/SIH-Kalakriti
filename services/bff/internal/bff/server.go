package bff

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
	"github.com/ZoroNewbie00/kalakriti/pkg/i18n"
	assets "github.com/ZoroNewbie00/kalakriti/services/bff"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/handler"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/middleware"
)

// Config holds all dependencies the BFF needs.
type Config struct {
	Addr    string
	BaseURL string
	WebDist string

	// AllowedOrigins is the CORS allowlist for the public API. Empty disables
	// CORS entirely; set explicitly per environment rather than wildcarding.
	AllowedOrigins []string
	// ProvenancePublicKeyHex is the hex-encoded Ed25519 public key matching
	// the provenance signer's private key. Empty makes the /v/{code}
	// signature check fail closed.
	ProvenancePublicKeyHex string

	Logger     *slog.Logger
	Issuer     *auth.Issuer
	Redis      *redis.Client
	IdempStore middleware.IdempotencyStore

	// Service clients (gRPC).
	AuthSvc    handler.AuthService
	ArtisanSvc handler.ArtisanService
	MediaSvc   handler.MediaService
	ListingSvc handler.ListingService
	SearchSvc  handler.SearchService
	PricingSvc handler.PricingService
	OrderSvc   handler.OrderService
	FollowSvc  handler.FollowService
	StmtSvc    handler.StatementService
	InsightSvc handler.InsightService
	CatalogSvc handler.CatalogService

	// Rate limiting.
	RateLimitPerIP        int
	RateLimitPerPrincipal int
	RateLimitWindow       time.Duration
}

// Server is the BFF HTTP server.
type Server struct {
	cfg    Config
	router *chi.Mux
}

// NewServer builds the server with all routes mounted.
func NewServer(cfg Config) (*Server, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// Build the chi router with pkg/httpx base middleware.
	r := httpx.Mux(httpx.Config{
		Logger:         cfg.Logger,
		AllowedOrigins: cfg.AllowedOrigins,
		RequestTimeout: 30 * time.Second,
	})

	// Global middleware: i18n locale detection, body size limit.
	r.Use(i18n.Middleware)
	r.Use(middleware.MaxBodySize(10 << 20)) // 10 MiB

	// Mount routes.
	s := &Server{cfg: cfg, router: r}
	s.mountRoutes()

	return s, nil
}

// mountRoutes wires all handlers to their paths.
func (s *Server) mountRoutes() {
	r := s.router
	cfg := s.cfg

	// Build handlers.
	apiH := handler.NewAPIHandler(
		cfg.AuthSvc, cfg.ArtisanSvc, cfg.MediaSvc, cfg.ListingSvc,
		cfg.SearchSvc, cfg.PricingSvc, cfg.OrderSvc, cfg.FollowSvc,
		cfg.StmtSvc, cfg.InsightSvc,
	)

	verifyH, _ := handler.NewVerificationHandler(cfg.CatalogSvc, cfg.Redis, cfg.BaseURL, cfg.ProvenancePublicKeyHex)
	seoH, _ := handler.NewSEOHandler(cfg.CatalogSvc, cfg.Redis, cfg.BaseURL)
	spaH := handler.NewSPAHandler(cfg.WebDist)

	// Public SEO pages (server-rendered HTML).
	r.Get("/listing/{slug}", seoH.ListingPage)
	r.Get("/artisan/{slug}", seoH.ArtisanPage)
	r.Get("/v/{code}", verifyH.ServeHTTP)
	r.Get("/v/{code}/verify.json", verifyH.ServeJSON)
	r.Get("/sitemap.xml", seoH.Sitemap)
	r.Get("/robots.txt", seoH.RobotsTxt)

	// API routes under /api/v1.
	r.Route("/api/v1", func(api chi.Router) {
		// Rate limiting on all API routes.
		api.Use(middleware.RateLimit(cfg.Redis, middleware.RateLimitConfig{
			PerIPLimit:        cfg.RateLimitPerIP,
			PerPrincipalLimit: cfg.RateLimitPerPrincipal,
			Window:            cfg.RateLimitWindow,
		}))

		// Public auth routes (no JWT required).
		api.Post("/auth/otp/request", apiH.RequestOTP)
		api.Post("/auth/otp/verify", apiH.VerifyOTP)
		api.Post("/auth/refresh", apiH.RefreshToken)

		// Public search and listing reads.
		api.Get("/search", apiH.Search)
		api.Get("/search/suggest", apiH.Suggest)
		api.Get("/listings", apiH.ListListings)
		api.Get("/listings/{id}", apiH.GetListing)

		// Protected routes group (JWT required).
		api.Group(func(authed chi.Router) {
			authed.Use(middleware.Auth(cfg.Issuer))

			// Artisan endpoints.
			authed.Post("/artisans", withIdempotency(apiH.RegisterArtisan, cfg.IdempStore))
			authed.Get("/artisans/me", apiH.GetArtisanProfile)
			authed.Patch("/artisans/me", apiH.UpdateArtisanProfile)

			// Media endpoints.
			authed.Post("/media/upload-url", apiH.GenerateUploadURL)
			authed.Post("/media/{id}/confirm", apiH.ConfirmUpload)

			// Listing mutations.
			authed.Post("/listings", withIdempotency(apiH.CreateListing, cfg.IdempStore))
			authed.Patch("/listings/{id}", apiH.UpdateListing)
			authed.Post("/listings/{id}/submit", apiH.SubmitListing)
			authed.Post("/listings/{id}/approve", apiH.ApproveListing)

			// Search voice.
			authed.Post("/search/voice", apiH.SearchVoice)

			// Pricing.
			authed.Post("/pricing/advise", apiH.AdvisePricing)

			// Orders.
			authed.Post("/orders/bulk", withIdempotency(apiH.CreateBulkOrder, cfg.IdempStore))
			authed.Get("/orders/{id}", apiH.GetOrder)
			authed.Get("/orders/{id}/events", apiH.WatchOrder)
			authed.Post("/orders/lots/{id}/respond", apiH.RespondToLot)

			// Follows and feed.
			authed.Post("/artisans/{id}/follow", apiH.FollowArtisan)
			authed.Delete("/artisans/{id}/follow", apiH.UnfollowArtisan)
			authed.Get("/feed", apiH.GetFeed)

			// Statements.
			authed.Post("/statements", withIdempotency(apiH.GenerateStatement, cfg.IdempStore))
			authed.Get("/statements/{id}", apiH.GetStatement)

			// Insights (MINISTRY role only, enforced in handlers).
			authed.Get("/insights/earnings-by-district", apiH.GetEarningsByDistrict)
			authed.Get("/insights/income-comparison", apiH.GetIncomeComparison)
			authed.Get("/insights/dying-crafts", apiH.GetDyingCrafts)
		})

		// OpenAPI spec, embedded at build time so it serves regardless of the
		// process's working directory.
		api.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(assets.OpenAPIJSON)
		})
	})

	// SPA fallback for everything else.
	r.NotFound(spaH.ServeHTTP)
}

// withIdempotency wraps a handler with idempotency middleware.
func withIdempotency(h http.HandlerFunc, store middleware.IdempotencyStore) http.HandlerFunc {
	return middleware.Idempotency(store, "bff")(h).ServeHTTP
}

// ServeHTTP implements http.Handler for testing.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// HTTPServer returns the underlying http.Server for graceful shutdown.
func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	s.cfg.Logger.Info("bff starting", "addr", s.cfg.Addr)
	return http.ListenAndServe(s.cfg.Addr, s.router)
}
