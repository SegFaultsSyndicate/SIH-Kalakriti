package bff

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
	"github.com/ZoroNewbie00/kalakriti/pkg/i18n"
	"github.com/ZoroNewbie00/kalakriti/pkg/webhook"
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
	B2BSvc     handler.B2BService
	TrendSvc   handler.TrendService
	BadgeSvc   handler.BadgeService
	SchemeSvc  handler.SchemeService

	// WebhookMgr backs buyer/seller-facing subscribe/list/unsubscribe
	// endpoints; delivery itself runs out-of-process (cmd/webhook-worker).
	WebhookMgr *webhook.Manager

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
		SelfOrigin:     selfOrigin(cfg.BaseURL),
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

	// A path that exists under a different verb should be distinguishable
	// from one that doesn't exist at all -- without this, gin sends both to
	// NoRoute (the SPA fallback, guarded against /api/* above but still not
	// the same signal as "wrong method").
	r.HandleMethodNotAllowed = true
	r.NoMethod(httpx.WrapHandler(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusMethodNotAllowed, domain.HTTPErrorBody{
			Error:   "method_not_allowed",
			Message: fmt.Sprintf("method %s not allowed for %s", r.Method, r.URL.Path),
		})
	}))

	// Liveness only -- bff has no single dependency whose failure should flip
	// this (unlike core-svc's /readyz, which checks its own Postgres pool).
	// Documented in QUICKSTART.md's smoke test; without this it fell through
	// to NoRoute -> the SPA handler -> "index.html not found", since bff's
	// own container never has a built frontend at cfg.WebDist (that's the
	// separate `web` NGINX container's job in docker-compose.yml).
	r.GET("/healthz", func(c *gin.Context) {
		httpx.JSON(c.Writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Build handlers.
	apiH := handler.NewAPIHandler(
		cfg.AuthSvc, cfg.ArtisanSvc, cfg.MediaSvc, cfg.ListingSvc,
		cfg.SearchSvc, cfg.PricingSvc, cfg.OrderSvc, cfg.FollowSvc,
		cfg.StmtSvc, cfg.InsightSvc, cfg.CatalogSvc,
		cfg.B2BSvc, cfg.TrendSvc, cfg.BadgeSvc, cfg.SchemeSvc,
	)
	apiH.SetSecurity(cfg.Redis, cfg.Logger, "kalakriti-production-webhook-hmac-key")
	apiH.SetWebhookManager(cfg.WebhookMgr)

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

	// Public auth routes (no JWT required) with strict OTP rate limiting (max 5 per 10min).
	api.POST("/auth/otp/request", httpx.Wrap(middleware.RateLimitEndpoint(cfg.Redis, "otp", 5, 10*time.Minute)), httpx.WrapHandler(apiH.RequestOTP))
	api.POST("/auth/otp/verify", httpx.WrapHandler(apiH.VerifyOTP))
	api.POST("/auth/refresh", httpx.WrapHandler(apiH.RefreshToken))

	// Payment webhook callback (HMAC-SHA256 verified)
	api.POST("/payments/webhook", httpx.WrapHandler(apiH.HandlePaymentWebhook))

	// Public search and listing reads.
	api.GET("/search", httpx.WrapHandler(apiH.Search))
	api.GET("/search/suggest", httpx.WrapHandler(apiH.Suggest))
	api.POST("/search/voice", httpx.WrapHandler(apiH.SearchVoice))
	api.GET("/listings", httpx.WrapHandler(apiH.ListListings))
	api.GET("/listings/summaries", httpx.WrapHandler(apiH.BatchGetListingSummaries))
	api.GET("/listings/:id", httpx.WrapHandler(apiH.GetListing))
	api.GET("/listings/:id/summary", httpx.WrapHandler(apiH.GetListingSummary))
	api.GET("/listings/:id/attributes", httpx.WrapHandler(apiH.GetListingAttributes))

	// Public craft ontology, artisan storefront and process feed reads.
	api.GET("/crafts", httpx.WrapHandler(apiH.ListCrafts))
	api.GET("/crafts/:slug", httpx.WrapHandler(apiH.GetCraft))
	api.GET("/artisans/:id/storefront", httpx.WrapHandler(apiH.GetArtisanStorefront))
	api.GET("/artisans/:id/follower-count", httpx.WrapHandler(apiH.GetFollowerCount))
	api.GET("/feed/process", httpx.WrapHandler(apiH.GetProcessFeed))

	// Public company, boutique and trend reads/writes.
	api.POST("/companies", httpx.WrapHandler(withIdempotency(apiH.RegisterCompany, cfg.IdempStore)))
	api.GET("/companies", httpx.WrapHandler(apiH.ListCompanies))
	api.GET("/companies/:id", httpx.WrapHandler(apiH.GetCompany))
	api.GET("/trends", httpx.WrapHandler(apiH.ListTrendLinks))
	api.GET("/badges", httpx.WrapHandler(apiH.ListBadgeCatalog))
	api.GET("/schemes", httpx.WrapHandler(apiH.ListSchemes))
	api.GET("/artisans/:id/badges", httpx.WrapHandler(apiH.ListArtisanBadges))
	api.GET("/boutiques/nearby", httpx.WrapHandler(apiH.ListNearbyBoutiques))

	// Protected routes group (JWT required).
	authed := api.Group("")
	authed.Use(httpx.Wrap(middleware.Auth(cfg.Issuer)))

	// B2B companies, boutiques, leads, partnerships, and market trends.
	authed.GET("/companies/me", httpx.WrapHandler(apiH.GetMyCompany))
	authed.POST("/companies/:id/verify", httpx.WrapHandler(apiH.VerifyCompany))
	authed.GET("/companies/commission-stats", httpx.WrapHandler(apiH.GetPlatformCommissionStats))
	authed.POST("/companies/sales/settle", httpx.WrapHandler(withIdempotency(apiH.RecordCompanySale, cfg.IdempStore)))
	authed.GET("/companies/:id/sales", httpx.WrapHandler(apiH.ListCompanySales))
	authed.POST("/companies/:id/interest", httpx.WrapHandler(withIdempotency(apiH.ExpressInterest, cfg.IdempStore)))
	authed.POST("/leads/:id/respond", httpx.WrapHandler(apiH.RespondToInterest))
	authed.GET("/artisans/me/leads", httpx.WrapHandler(apiH.ListArtisanLeads))
	authed.GET("/artisans/me/boutique-matches", httpx.WrapHandler(apiH.ListBoutiqueMatches))
	authed.POST("/partnerships", httpx.WrapHandler(withIdempotency(apiH.CreatePartnership, cfg.IdempStore)))
	authed.GET("/partnerships", httpx.WrapHandler(apiH.ListPartnerships))
	authed.POST("/trends", httpx.WrapHandler(withIdempotency(apiH.CreateTrendLink, cfg.IdempStore)))
	authed.DELETE("/trends/:id", httpx.WrapHandler(apiH.DeleteTrendLink))
	authed.POST("/trends/:id/pin", httpx.WrapHandler(apiH.PinTrendLink))
	authed.GET("/badges/me/progress", httpx.WrapHandler(apiH.GetBadgeProgress))
	authed.GET("/schemes/match", httpx.WrapHandler(apiH.MatchSchemes))
	authed.POST("/schemes", httpx.WrapHandler(withIdempotency(apiH.UpsertScheme, cfg.IdempStore)))
	authed.PATCH("/schemes/:id", httpx.WrapHandler(withIdempotency(apiH.UpsertScheme, cfg.IdempStore)))
	authed.DELETE("/schemes/:id", httpx.WrapHandler(apiH.DeleteScheme))
	authed.POST("/artisans/:id/badges", httpx.WrapHandler(withIdempotency(apiH.GrantBadge, cfg.IdempStore)))
	authed.DELETE("/artisans/:id/badges/:code", httpx.WrapHandler(apiH.RevokeBadge))

	// Outbound webhook subscriptions (delivery runs in cmd/webhook-worker).
	authed.POST("/webhooks/subscriptions", httpx.WrapHandler(withIdempotency(apiH.CreateWebhookSubscription, cfg.IdempStore)))
	authed.GET("/webhooks/subscriptions", httpx.WrapHandler(apiH.ListWebhookSubscriptions))
	authed.DELETE("/webhooks/subscriptions/:id", httpx.WrapHandler(apiH.DeleteWebhookSubscription))

	// Artisan endpoints.
	authed.POST("/artisans", httpx.WrapHandler(withIdempotency(apiH.RegisterArtisan, cfg.IdempStore)))
	authed.GET("/artisans/me", httpx.WrapHandler(apiH.GetArtisanProfile))
	authed.PATCH("/artisans/me", httpx.WrapHandler(apiH.UpdateArtisanProfile))
	authed.POST("/auth/phone/change/request", httpx.WrapHandler(apiH.RequestPhoneChangeOTP))
	authed.POST("/auth/phone/change/verify", httpx.WrapHandler(apiH.VerifyPhoneChangeOTP))

	// Media endpoints.
	authed.POST("/media/upload-url", httpx.WrapHandler(apiH.GenerateUploadURL))
	authed.POST("/media/:id/confirm", httpx.WrapHandler(apiH.ConfirmUpload))

	// Listing mutations.
	authed.POST("/listings", httpx.WrapHandler(withIdempotency(apiH.CreateListing, cfg.IdempStore)))
	authed.PATCH("/listings/:id", httpx.WrapHandler(apiH.UpdateListing))
	authed.POST("/listings/:id/media", httpx.WrapHandler(withIdempotency(apiH.AttachListingMedia, cfg.IdempStore)))
	authed.POST("/listings/:id/submit", httpx.WrapHandler(apiH.SubmitListing))
	authed.POST("/listings/:id/approve", httpx.WrapHandler(apiH.ApproveListing))
	authed.POST("/listings/:id/seal-provenance", httpx.WrapHandler(apiH.SealProvenance))

	// Pricing.
	authed.POST("/pricing/advise", httpx.WrapHandler(apiH.AdvisePricing))

	// Orders.
	authed.POST("/orders/bulk", httpx.WrapHandler(withIdempotency(apiH.CreateBulkOrder, cfg.IdempStore)))
	authed.GET("/orders/:id", httpx.WrapHandler(apiH.GetOrder))
	authed.GET("/orders/:id/events", httpx.WrapHandler(apiH.WatchOrder))
	authed.POST("/orders/lots/:id/respond", httpx.WrapHandler(apiH.RespondToLot))
	authed.POST("/orders/lots/:id/progress", httpx.WrapHandler(apiH.ReportProgress))
	authed.POST("/orders/lots/:id/reallocate", httpx.WrapHandler(apiH.RequestReallocation))

	// Follows and feed.
	authed.POST("/artisans/:id/follow", httpx.WrapHandler(apiH.FollowArtisan))
	authed.DELETE("/artisans/:id/follow", httpx.WrapHandler(apiH.UnfollowArtisan))
	authed.GET("/feed", httpx.WrapHandler(apiH.GetFeed))
	authed.POST("/feed/:id/read", httpx.WrapHandler(apiH.MarkNotificationRead))

	// Statements.
	authed.POST("/statements", httpx.WrapHandler(withIdempotency(apiH.GenerateStatement, cfg.IdempStore)))
	authed.GET("/statements/:id", httpx.WrapHandler(apiH.GetStatement))
	authed.GET("/statements", httpx.WrapHandler(apiH.ListIncomeStatements))

	// Insights (MINISTRY role only, enforced in handlers).
	authed.GET("/insights/artisans-by-category", httpx.WrapHandler(apiH.GetArtisansByCategory))
	authed.GET("/insights/listings-by-craft-month", httpx.WrapHandler(apiH.GetListingsByCraftMonth))
	authed.GET("/insights/earnings-by-district", httpx.WrapHandler(apiH.GetEarningsByDistrict))
	authed.GET("/insights/income-comparison", httpx.WrapHandler(apiH.GetIncomeComparison))
	authed.GET("/insights/dying-crafts", httpx.WrapHandler(apiH.GetDyingCrafts))
	authed.POST("/insights/refresh", httpx.WrapHandler(apiH.RefreshInsights))

	// Cluster and self-help-group administration (CLUSTER_OFFICER/MINISTRY, enforced in handlers).
	authed.POST("/clusters", httpx.WrapHandler(withIdempotency(apiH.CreateCluster, cfg.IdempStore)))
	authed.GET("/clusters/:id", httpx.WrapHandler(apiH.GetCluster))
	authed.GET("/clusters/:id/members", httpx.WrapHandler(apiH.ListClusterMembers))
	authed.POST("/clusters/:id/members", httpx.WrapHandler(withIdempotency(apiH.AddClusterMember, cfg.IdempStore)))
	authed.DELETE("/clusters/:id/members/:artisanId", httpx.WrapHandler(apiH.RemoveClusterMember))
	authed.POST("/clusters/:id/onboard", httpx.WrapHandler(withIdempotency(apiH.OnboardClusterArtisan, cfg.IdempStore)))
	authed.POST("/self-help-groups", httpx.WrapHandler(withIdempotency(apiH.CreateSelfHelpGroup, cfg.IdempStore)))
	authed.GET("/self-help-groups/:id", httpx.WrapHandler(apiH.GetSelfHelpGroup))
	authed.PUT("/self-help-groups/:id/members", httpx.WrapHandler(withIdempotency(apiH.SetSelfHelpGroupMembers, cfg.IdempStore)))

	// Moderation (CLUSTER_OFFICER/MINISTRY, enforced in handlers).
	authed.POST("/listings/:id/suspend", httpx.WrapHandler(withIdempotency(apiH.SuspendListing, cfg.IdempStore)))
	authed.POST("/listings/:id/reinstate", httpx.WrapHandler(withIdempotency(apiH.ReinstateListing, cfg.IdempStore)))

	// Craft ontology administration (MINISTRY only, enforced in handler).
	authed.POST("/crafts/refresh-index", httpx.WrapHandler(apiH.RefreshCraftIndex))

	// OpenAPI spec, embedded at build time so it serves regardless of the
	// process's working directory.
	api.GET("/openapi.json", httpx.WrapHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(assets.OpenAPIJSON)
	}))

	// SPA fallback for everything else.
	r.NoRoute(httpx.WrapHandler(spaH.ServeHTTP))
}

// selfOrigin extracts the scheme+host Origin header this server's own
// public address would present, from its configured BaseURL, for
// httpx.Config.SelfOrigin. An Origin header never carries a path, so
// "http://localhost:8000/foo" and "http://localhost:8000" must compare
// equal to it -- BaseURL is documented as just scheme+host today, but this
// normalises defensively rather than assuming that never changes. An
// unparseable or empty BaseURL yields "", the same as never setting it.
func selfOrigin(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
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
