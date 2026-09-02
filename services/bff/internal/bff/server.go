package bff

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
	// ChannelSvcAddr is channel-svc's internal HTTP base URL, proxied for
	// outbound catalog feeds (IndiaHandmade today) so bff stays the only
	// service the outside world talks to.
	ChannelSvcAddr string

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
	router *gin.Engine
}

// NewServer builds the server with all routes mounted.
func NewServer(cfg Config) (*Server, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// Build the gin router with pkg/httpx base middleware.
	r := httpx.Mux(httpx.Config{
		Logger:         cfg.Logger,
		AllowedOrigins: cfg.AllowedOrigins,
		RequestTimeout: 30 * time.Second,
	})

	// Global middleware: i18n locale detection, body size limit.
	r.Use(httpx.Wrap(i18n.Middleware))
	r.Use(httpx.Wrap(middleware.MaxBodySize(10 << 20))) // 10 MiB

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
	exportH := handler.NewExportHandler(cfg.ChannelSvcAddr)

	// Public SEO pages (server-rendered HTML).
	r.GET("/listing/:slug", httpx.WrapHandler(seoH.ListingPage))
	r.GET("/artisan/:slug", httpx.WrapHandler(seoH.ArtisanPage))
	r.GET("/v/:code", httpx.WrapHandler(verifyH.ServeHTTP))
	r.GET("/export/indiahandmade", httpx.WrapHandler(exportH.IndiaHandmade))
	r.GET("/v/:code/verify.json", httpx.WrapHandler(verifyH.ServeJSON))
	r.GET("/sitemap.xml", httpx.WrapHandler(seoH.Sitemap))
	r.GET("/robots.txt", httpx.WrapHandler(seoH.RobotsTxt))

	// API routes under /api/v1.
	api := r.Group("/api/v1")

	// Rate limiting on all API routes.
	api.Use(httpx.Wrap(middleware.RateLimit(cfg.Redis, middleware.RateLimitConfig{
		PerIPLimit:        cfg.RateLimitPerIP,
		PerPrincipalLimit: cfg.RateLimitPerPrincipal,
		Window:            cfg.RateLimitWindow,
	})))

	// Public auth routes (no JWT required).
	api.POST("/auth/otp/request", httpx.WrapHandler(apiH.RequestOTP))
	api.POST("/auth/otp/verify", httpx.WrapHandler(apiH.VerifyOTP))
	api.POST("/auth/refresh", httpx.WrapHandler(apiH.RefreshToken))

	// Public search and listing reads.
	api.GET("/search", httpx.WrapHandler(apiH.Search))
	api.GET("/search/suggest", httpx.WrapHandler(apiH.Suggest))
	api.GET("/listings", httpx.WrapHandler(apiH.ListListings))
	api.GET("/listings/:id", httpx.WrapHandler(apiH.GetListing))

	// Protected routes group (JWT required).
	authed := api.Group("")
	authed.Use(httpx.Wrap(middleware.Auth(cfg.Issuer)))

	// Artisan endpoints.
	authed.POST("/artisans", httpx.WrapHandler(withIdempotency(apiH.RegisterArtisan, cfg.IdempStore)))
	authed.GET("/artisans/me", httpx.WrapHandler(apiH.GetArtisanProfile))
	authed.PATCH("/artisans/me", httpx.WrapHandler(apiH.UpdateArtisanProfile))

	// Media endpoints.
	authed.POST("/media/upload-url", httpx.WrapHandler(apiH.GenerateUploadURL))
	authed.POST("/media/:id/confirm", httpx.WrapHandler(apiH.ConfirmUpload))

	// Listing mutations.
	authed.POST("/listings", httpx.WrapHandler(withIdempotency(apiH.CreateListing, cfg.IdempStore)))
	authed.PATCH("/listings/:id", httpx.WrapHandler(apiH.UpdateListing))
	authed.POST("/listings/:id/submit", httpx.WrapHandler(apiH.SubmitListing))
	authed.POST("/listings/:id/approve", httpx.WrapHandler(apiH.ApproveListing))

	// Search voice.
	authed.POST("/search/voice", httpx.WrapHandler(apiH.SearchVoice))

	// Pricing.
	authed.POST("/pricing/advise", httpx.WrapHandler(apiH.AdvisePricing))

	// Orders.
	authed.POST("/orders/bulk", httpx.WrapHandler(withIdempotency(apiH.CreateBulkOrder, cfg.IdempStore)))
	authed.GET("/orders/:id", httpx.WrapHandler(apiH.GetOrder))
	authed.GET("/orders/:id/events", httpx.WrapHandler(apiH.WatchOrder))
	authed.POST("/orders/lots/:id/respond", httpx.WrapHandler(apiH.RespondToLot))

	// Follows and feed.
	authed.POST("/artisans/:id/follow", httpx.WrapHandler(apiH.FollowArtisan))
	authed.DELETE("/artisans/:id/follow", httpx.WrapHandler(apiH.UnfollowArtisan))
	authed.GET("/feed", httpx.WrapHandler(apiH.GetFeed))

	// Statements.
	authed.POST("/statements", httpx.WrapHandler(withIdempotency(apiH.GenerateStatement, cfg.IdempStore)))
	authed.GET("/statements/:id", httpx.WrapHandler(apiH.GetStatement))

	// Insights (MINISTRY role only, enforced in handlers).
	authed.GET("/insights/earnings-by-district", httpx.WrapHandler(apiH.GetEarningsByDistrict))
	authed.GET("/insights/income-comparison", httpx.WrapHandler(apiH.GetIncomeComparison))
	authed.GET("/insights/dying-crafts", httpx.WrapHandler(apiH.GetDyingCrafts))

	// OpenAPI spec, embedded at build time so it serves regardless of the
	// process's working directory.
	api.GET("/openapi.json", httpx.WrapHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(assets.OpenAPIJSON)
	}))

	// SPA fallback for everything else.
	r.NoRoute(httpx.WrapHandler(spaH.ServeHTTP))
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
