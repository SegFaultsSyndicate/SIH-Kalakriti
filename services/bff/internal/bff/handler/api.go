package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
	"github.com/ZoroNewbie00/kalakriti/pkg/webhook"
	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/middleware"
)

// APIHandler aggregates all REST /api/v1 endpoints, bridging HTTP/JSON to gRPC.
type APIHandler struct {
	authSvc       AuthService
	artisanSvc    ArtisanService
	mediaSvc      MediaService
	listingSvc    ListingService
	searchSvc     SearchService
	pricingSvc    PricingService
	orderSvc      OrderService
	followSvc     FollowService
	stmtSvc       StatementService
	insightSvc    InsightService
	catalogSvc    CatalogService
	redis         *redis.Client
	logger        *slog.Logger
	webhookSecret string
}

// SetSecurity configures Redis, structured logging and webhook credentials for security enforcement.
func (h *APIHandler) SetSecurity(rdb *redis.Client, logger *slog.Logger, webhookSecret string) {
	h.redis = rdb
	h.logger = logger
	h.webhookSecret = webhookSecret
}

// AuthService is the auth-svc gRPC client interface.
type AuthService interface {
	RequestOTP(ctx context.Context, phone string) error
	VerifyOTP(ctx context.Context, phone, otp string) (accessToken, refreshToken string, err error)
	RefreshToken(ctx context.Context, refreshToken string) (accessToken string, err error)
}

// ArtisanService is the artisan-svc gRPC client interface.
type ArtisanService interface {
	Register(ctx context.Context, phone, idempotencyKey string, fields map[string]any) (artisanID string, err error)
	GetProfile(ctx context.Context, artisanID string) (map[string]any, error)
	UpdateProfile(ctx context.Context, artisanID string, updates map[string]any) error
}

// MediaService is the media-svc gRPC client interface.
type MediaService interface {
	GenerateUploadURL(ctx context.Context, artisanID, contentType string, sizeBytes int64) (mediaID, uploadURL string, err error)
	ConfirmUpload(ctx context.Context, mediaID string) error
}

// ListingService is the listing-svc gRPC client interface.
type ListingService interface {
	CreateListing(ctx context.Context, artisanID, idempotencyKey string, listing map[string]any) (listingID string, err error)
	UpdateListing(ctx context.Context, listingID, idempotencyKey string, updates map[string]any) error
	SubmitForReview(ctx context.Context, listingID, idempotencyKey string) error
	ApproveListing(ctx context.Context, listingID, reviewerID, idempotencyKey string, editedTranslations []map[string]any) error
	GetListing(ctx context.Context, listingID string) (map[string]any, error)
	// GetListingSummary is GetListing plus craft name, materials, colours and
	// resolved media URLs -- the buyer-marketplace card shape.
	GetListingSummary(ctx context.Context, listingID string) (map[string]any, error)
	ListListings(ctx context.Context, filters map[string]any) ([]map[string]any, error)
	// SealProvenance freezes process evidence for a PUBLISHED listing. fields
	// carries media (array of confirmed media ids), claimed_technique
	// (string, required) and skip_loom_check (bool, optional -- non-textile
	// crafts skip the weave check).
	SealProvenance(ctx context.Context, listingID, idempotencyKey string, fields map[string]any) (map[string]any, error)
}

// SearchService is the search-svc gRPC client interface. Search and
// SearchVoice both return the full response map: results, understood (the
// structured filters the query understander parsed out, for removable
// chips), craft_spans, detected_language, did_you_mean and query_id.
type SearchService interface {
	Search(ctx context.Context, query string, filters map[string]any) (map[string]any, error)
	Suggest(ctx context.Context, prefix string) ([]string, error)
	SearchVoice(ctx context.Context, audioData []byte, language string) (map[string]any, error)
}

// PricingService is the pricing-svc gRPC client interface. inputs carries
// material_cost ({amount_paise, currency_code?}), hours (number) and the
// optional chosen_price ({amount_paise, currency_code?}) — GetAdvisory's own
// request shape, not a craft/region lookup: the advisory is computed from an
// existing listing, not from a craft in the abstract.
type PricingService interface {
	AdvisePricing(ctx context.Context, listingID string, inputs map[string]any) (map[string]any, error)
}

// OrderService is fulfilment-svc's gRPC client interface. CreateBulkOrder
// registers a quantity+listing requirement — fields carries listing_id,
// quantity, required_by (RFC3339), customisations (map of string to
// string) and optional notes — allocation into per-artisan lots happens
// separately and asynchronously; there's no "lots" input at order creation.
// RespondToLot's fields carries promised_ship_date (RFC3339, required when
// accept is true) or decline_reason (required when accept is false).
type OrderService interface {
	CreateBulkOrder(ctx context.Context, buyerID, idempotencyKey string, fields map[string]any) (orderID string, err error)
	GetOrder(ctx context.Context, orderID string) (map[string]any, error)
	RespondToLot(ctx context.Context, lotID, artisanID, idempotencyKey string, accept bool, fields map[string]any) error
	// ReportProgress records production progress (or a rework resubmission
	// from QC_FAILED) against an artisan's own accepted lot. fields carries
	// progress_pct (number, required), media (array of confirmed media ids,
	// optional) and note (string, optional).
	ReportProgress(ctx context.Context, lotID, artisanID, idempotencyKey string, fields map[string]any) (map[string]any, error)
	// RequestReallocation gives up an artisan's own accepted lot they cannot
	// complete. fields carries reason (string, required).
	RequestReallocation(ctx context.Context, lotID, artisanID, idempotencyKey string, fields map[string]any) (map[string]any, error)
	// WatchOrder streams order events. since, when non-nil, replays events
	// after that instant before streaming live ones -- how a reconnected SSE
	// client backfills without duplicate rows.
	WatchOrder(ctx context.Context, orderID string, since *time.Time) (<-chan map[string]any, error)
}

// FollowService is the follow-svc gRPC client interface.
type FollowService interface {
	FollowArtisan(ctx context.Context, followerID, artisanID string) error
	UnfollowArtisan(ctx context.Context, followerID, artisanID string) error
	GetFeed(ctx context.Context, userID string, limit, offset int32) ([]map[string]any, error)
	MarkFeedItemRead(ctx context.Context, notificationID, userID string) error
	GetFollowerCount(ctx context.Context, artisanID string) (int32, error)
}

// StatementService is the statement-svc gRPC client interface.
type StatementService interface {
	GenerateStatement(ctx context.Context, artisanID string, start, end string) (map[string]any, error)
	GetStatement(ctx context.Context, statementID string) (map[string]any, error)
	// ListIncomeStatements lists an artisan's own past statements, newest first.
	ListIncomeStatements(ctx context.Context, artisanID string, limit, offset int32) ([]map[string]any, error)
}

// InsightService is the insight-svc gRPC client interface.
type InsightService interface {
	GetArtisansByCategory(ctx context.Context, filters map[string]any) ([]map[string]any, error)
	GetListingsByCraftMonth(ctx context.Context, filters map[string]any) ([]map[string]any, error)
	GetEarningsByDistrict(ctx context.Context, filters map[string]any) ([]map[string]any, error)
	GetIncomeComparison(ctx context.Context, filters map[string]any) ([]map[string]any, error)
	GetDyingCrafts(ctx context.Context, limit int32) ([]map[string]any, error)
	RefreshMaterializedViews(ctx context.Context) (map[string]any, error)
}

// NewAPIHandler constructs the handler with all service clients.
func NewAPIHandler(
	authSvc AuthService,
	artisanSvc ArtisanService,
	mediaSvc MediaService,
	listingSvc ListingService,
	searchSvc SearchService,
	pricingSvc PricingService,
	orderSvc OrderService,
	followSvc FollowService,
	stmtSvc StatementService,
	insightSvc InsightService,
	catalogSvc CatalogService,
) *APIHandler {
	return &APIHandler{
		authSvc:    authSvc,
		artisanSvc: artisanSvc,
		mediaSvc:   mediaSvc,
		listingSvc: listingSvc,
		searchSvc:  searchSvc,
		pricingSvc: pricingSvc,
		orderSvc:   orderSvc,
		followSvc:  followSvc,
		stmtSvc:    stmtSvc,
		insightSvc: insightSvc,
		catalogSvc: catalogSvc,
	}
}

// Auth endpoints

func (h *APIHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		httpx.Error(w, domain.InvalidInput("phone is required"))
		return
	}

	// Account lockout check to mitigate credential brute force
	if h.redis != nil {
		if locked, remaining := middleware.CheckAccountLockout(r.Context(), h.redis, req.Phone); locked {
			if h.logger != nil {
				h.logger.Warn("security audit: attempt on locked account", "event", "ACCOUNT_LOCKED_ATTEMPT", "phone", maskIdentifier(req.Phone), "remaining_sec", int(remaining.Seconds()))
			}
			httpx.Error(w, domain.Unavailable(fmt.Sprintf("account temporarily locked due to failed attempts; retry in %d seconds", int(remaining.Seconds()))))
			return
		}
	}

	// Always execute and return uniform response to prevent user enumeration
	_ = h.authSvc.RequestOTP(r.Context(), req.Phone)

	if h.logger != nil {
		h.logger.Info("security audit: otp requested", "event", "OTP_REQUEST", "phone", maskIdentifier(req.Phone), "remote_addr", r.RemoteAddr)
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"status":  "otp_sent",
		"message": "If this account exists, an OTP has been dispatched.",
	})
}

func (h *APIHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone string `json:"phone"`
		OTP   string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	req.OTP = strings.TrimSpace(req.OTP)
	if req.Phone == "" || req.OTP == "" {
		httpx.Error(w, domain.InvalidInput("phone and otp are required"))
		return
	}

	// Check lockout
	if h.redis != nil {
		if locked, remaining := middleware.CheckAccountLockout(r.Context(), h.redis, req.Phone); locked {
			httpx.Error(w, domain.Unavailable(fmt.Sprintf("account locked; retry in %d seconds", int(remaining.Seconds()))))
			return
		}
	}

	accessToken, refreshToken, err := h.authSvc.VerifyOTP(r.Context(), req.Phone, req.OTP)
	if err != nil {
		if h.redis != nil {
			locked := middleware.RecordFailedLogin(r.Context(), h.redis, req.Phone)
			if locked && h.logger != nil {
				h.logger.Warn("security audit: account locked after consecutive failures", "event", "ACCOUNT_LOCKED", "phone", maskIdentifier(req.Phone))
			}
		}
		if h.logger != nil {
			h.logger.Warn("security audit: login verification failed", "event", "LOGIN_FAILED", "phone", maskIdentifier(req.Phone), "remote_addr", r.RemoteAddr)
		}
		httpx.Error(w, err)
		return
	}

	// Authentication succeeded: clear failed counter
	if h.redis != nil {
		middleware.ClearFailedLogin(r.Context(), h.redis, req.Phone)
	}
	if h.logger != nil {
		h.logger.Info("security audit: login successful", "event", "LOGIN_SUCCESS", "phone", maskIdentifier(req.Phone), "remote_addr", r.RemoteAddr)
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// RequestPhoneChangeOTP requests an OTP for changing the artisan's registered phone number.
func (h *APIHandler) RequestPhoneChangeOTP(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var req struct {
		NewPhone string `json:"new_phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	req.NewPhone = strings.TrimSpace(req.NewPhone)
	if req.NewPhone == "" {
		httpx.Error(w, domain.InvalidInput("new_phone is required"))
		return
	}

	if err := h.authSvc.RequestOTP(r.Context(), req.NewPhone); err != nil {
		httpx.Error(w, err)
		return
	}

	if h.logger != nil {
		h.logger.Info("security audit: phone change otp requested", "event", "PHONE_CHANGE_OTP_REQUEST", "actor", p.Subject)
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "otp_sent"})
}

// VerifyPhoneChangeOTP verifies the OTP, updates the phone, and invalidates older sessions.
func (h *APIHandler) VerifyPhoneChangeOTP(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var req struct {
		NewPhone string `json:"new_phone"`
		OTP      string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	req.NewPhone = strings.TrimSpace(req.NewPhone)
	req.OTP = strings.TrimSpace(req.OTP)
	if req.NewPhone == "" || req.OTP == "" {
		httpx.Error(w, domain.InvalidInput("new_phone and otp are required"))
		return
	}

	accessToken, refreshToken, err := h.authSvc.VerifyOTP(r.Context(), req.NewPhone, req.OTP)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	// Update phone on profile
	_ = h.artisanSvc.UpdateProfile(r.Context(), p.Subject, map[string]any{"phone": req.NewPhone})

	// Invalidate older sessions by logging revocation epoch in Redis
	if h.redis != nil {
		h.redis.Set(r.Context(), fmt.Sprintf("session_revoked:%s", p.Subject), time.Now().Unix(), 30*24*time.Hour)
	}

	if h.logger != nil {
		h.logger.Info("security audit: phone updated and existing sessions invalidated", "event", "PHONE_CHANGED_SESSIONS_REVOKED", "actor", p.Subject)
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"status":        "phone_updated",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// HandlePaymentWebhook processes payment gateway notifications with strict HMAC-SHA256 signature verification.
func (h *APIHandler) HandlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.Error(w, domain.InvalidInput("cannot read webhook payload"))
		return
	}

	sig := r.Header.Get("X-Webhook-Signature")
	secret := h.webhookSecret
	if secret == "" {
		secret = "kalakriti-production-webhook-hmac-key"
	}

	if !webhook.VerifySignature(body, sig, secret) {
		if h.logger != nil {
			h.logger.Warn("security audit: invalid webhook signature", "event", "PAYMENT_WEBHOOK_SIG_FAILED", "remote_addr", r.RemoteAddr)
		}
		httpx.Error(w, domain.Unauthenticated("invalid webhook signature"))
		return
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	if h.logger != nil {
		h.logger.Info("security audit: verified payment webhook processed", "event", "PAYMENT_WEBHOOK_PROCESSED", "type", event["event"])
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "processed"})
}

func (h *APIHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	accessToken, err := h.authSvc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"access_token": accessToken})
}

// idempotencyKeyFrom reuses the client's X-Idempotency-Key when present, so a
// retried HTTP call also replays at the gRPC layer instead of repeating a
// write core-svc requires a non-empty idempotency_key for; mints one
// otherwise.
func idempotencyKeyFrom(r *http.Request) string {
	if key := r.Header.Get("X-Idempotency-Key"); key != "" {
		return key
	}
	return uuid.NewString()
}

// Artisan endpoints

func (h *APIHandler) RegisterArtisan(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var fields map[string]any
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	artisanID, err := h.artisanSvc.Register(r.Context(), p.PhoneE164, idempotencyKeyFrom(r), fields)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]string{"artisan_id": artisanID})
}

func (h *APIHandler) GetArtisanProfile(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	profile, err := h.artisanSvc.GetProfile(r.Context(), p.Subject)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, profile)
}

func (h *APIHandler) UpdateArtisanProfile(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	if err := h.artisanSvc.UpdateProfile(r.Context(), p.Subject, updates); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

// Media endpoints

var allowedMIMETypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"video/mp4":       true,
	"video/webm":      true,
	"application/pdf": true,
}

func (h *APIHandler) GenerateUploadURL(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var req struct {
		ContentType string `json:"content_type"`
		SizeBytes   int64  `json:"size_bytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	req.ContentType = strings.ToLower(strings.TrimSpace(req.ContentType))
	if !allowedMIMETypes[req.ContentType] {
		if h.logger != nil {
			h.logger.Warn("security audit: rejected unwhitelisted upload MIME type", "type", req.ContentType, "actor", p.Subject)
		}
		httpx.Error(w, domain.InvalidInput(fmt.Sprintf("unsupported media content-type %q; allowed: jpeg, png, webp, mp4, webm, pdf", req.ContentType)))
		return
	}

	// Enforce max upload size limits: 10MB images/PDFs, 100MB videos
	maxBytes := int64(10 << 20)
	if strings.HasPrefix(req.ContentType, "video/") {
		maxBytes = 100 << 20
	}
	if req.SizeBytes <= 0 || req.SizeBytes > maxBytes {
		httpx.Error(w, domain.InvalidInput(fmt.Sprintf("file size %d bytes exceeds maximum allowable limit (%d bytes)", req.SizeBytes, maxBytes)))
		return
	}

	mediaID, uploadURL, err := h.mediaSvc.GenerateUploadURL(r.Context(), p.Subject, req.ContentType, req.SizeBytes)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"media_id":   mediaID,
		"upload_url": uploadURL,
	})
}

func (h *APIHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	mediaID := httpx.URLParam(r, "id")
	if err := h.mediaSvc.ConfirmUpload(r.Context(), mediaID); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

// Listing endpoints

func (h *APIHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var listing map[string]any
	if err := json.NewDecoder(r.Body).Decode(&listing); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	listingID, err := h.listingSvc.CreateListing(r.Context(), p.Subject, idempotencyKeyFrom(r), listing)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]string{"listing_id": listingID})
}

func (h *APIHandler) UpdateListing(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	listingID := httpx.URLParam(r, "id")
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	if err := h.listingSvc.UpdateListing(r.Context(), listingID, idempotencyKeyFrom(r), updates); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

func (h *APIHandler) SubmitListing(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	listingID := httpx.URLParam(r, "id")
	if err := h.listingSvc.SubmitForReview(r.Context(), listingID, idempotencyKeyFrom(r)); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

func (h *APIHandler) ApproveListing(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequireRole(r.Context(), auth.RoleArtisan)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	// The body is optional: approving with no edits needs no request body at
	// all, only edited copy needs one.
	var req struct {
		EditedTranslations []map[string]any `json:"edited_translations"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.Error(w, domain.InvalidInput("invalid JSON"))
			return
		}
	}

	listingID := httpx.URLParam(r, "id")
	if err := h.listingSvc.ApproveListing(r.Context(), listingID, p.Subject, idempotencyKeyFrom(r), req.EditedTranslations); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

func (h *APIHandler) GetListing(w http.ResponseWriter, r *http.Request) {
	listingID := httpx.URLParam(r, "id")
	listing, err := h.listingSvc.GetListing(r.Context(), listingID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, listing)
}

// GetListingSummary handles GET /listings/{id}/summary -- the card shape
// (craft name, materials, colours, resolved image/video URLs) used by search
// results and editorial home sections. Split from GetListing because it costs
// extra joins and signed-URL round trips a bare listing read shouldn't pay.
func (h *APIHandler) GetListingSummary(w http.ResponseWriter, r *http.Request) {
	listingID := httpx.URLParam(r, "id")
	summary, err := h.listingSvc.GetListingSummary(r.Context(), listingID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, summary)
}

func (h *APIHandler) ListListings(w http.ResponseWriter, r *http.Request) {
	filters := map[string]any{
		"craft_id":   r.URL.Query().Get("craft_id"),
		"state":      r.URL.Query().Get("state"),
		"artisan_id": r.URL.Query().Get("artisan_id"),
		"type":       r.URL.Query().Get("type"),
	}

	listings, err := h.listingSvc.ListListings(r.Context(), filters)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"listings": listings})
}

func (h *APIHandler) SealProvenance(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	listingID := httpx.URLParam(r, "id")
	var fields map[string]any
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	record, err := h.listingSvc.SealProvenance(r.Context(), listingID, idempotencyKeyFrom(r), fields)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, record)
}

// Search endpoints

// searchFilters collects every query param /search accepts into search-svc's
// loose filter map. craft_id/region were the only ones ever wired through to
// search-svc; colour/material/price/gi/made_to_order/sealed were accepted by
// the domain and the SQL all along (services/search-svc/internal/search/repo/repo.go)
// but never reached from here, so buyer-facing facets silently did nothing.
func searchFilters(q url.Values) map[string]any {
	filters := map[string]any{
		"craft_id":    q.Get("craft_id"),
		"region":      q.Get("region"),
		"colours":     q["colour"],
		"materials":   q["material"],
		"gi_only":     q.Get("gi_tagged") == "true",
		"sealed_only": q.Get("provenance_verified") == "true",
	}
	if v := q.Get("made_to_order"); v == "true" {
		filters["listing_type"] = "MADE_TO_ORDER"
	} else if v == "false" {
		filters["listing_type"] = "READY_STOCK"
	}
	if v := q.Get("price_min"); v != "" {
		filters["min_price_paise"] = v
	}
	if v := q.Get("price_max"); v != "" {
		filters["max_price_paise"] = v
	}
	if v := q.Get("limit"); v != "" {
		filters["limit"] = v
	}
	return filters
}

func (h *APIHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	result, err := h.searchSvc.Search(r.Context(), query, searchFilters(r.URL.Query()))
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, result)
}

func (h *APIHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("q")
	suggestions, err := h.searchSvc.Suggest(r.Context(), prefix)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"suggestions": suggestions})
}

func (h *APIHandler) SearchVoice(w http.ResponseWriter, r *http.Request) {
	audioData, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.Error(w, domain.InvalidInput("cannot read audio data"))
		return
	}

	language := r.Header.Get("Accept-Language")
	result, err := h.searchSvc.SearchVoice(r.Context(), audioData, language)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, result)
}

// Craft, storefront and process-feed endpoints (public, buyer-facing)

// ListCrafts handles GET /crafts.
func (h *APIHandler) ListCrafts(w http.ResponseWriter, r *http.Request) {
	crafts, err := h.catalogSvc.ListCrafts(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"crafts": craftsToMaps(crafts)})
}

// GetCraft handles GET /crafts/{slug}, the craft landing page: what the
// craft is, its region, GI status, and the artisans who practise it.
func (h *APIHandler) GetCraft(w http.ResponseWriter, r *http.Request) {
	slug := httpx.URLParam(r, "slug")
	craft, err := h.catalogSvc.GetCraftBySlug(r.Context(), slug)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	artisans, err := h.catalogSvc.ListArtisansByCraft(r.Context(), craft.ID, 24)
	if err != nil {
		artisans = nil // an artisan-listing hiccup shouldn't blank the craft page itself
	}

	out := craftToMap(*craft)
	out["artisans"] = artisansToMaps(artisans)
	httpx.JSON(w, http.StatusOK, out)
}

// GetArtisanStorefront handles GET /artisans/{slug}/storefront: portrait,
// story, district, cluster, verification status -- everything the buyer
// storefront page needs beyond the catalog (GET /listings?artisan_id=) and
// follower count (GET /artisans/{id}/follower-count), which the client
// fetches separately.
func (h *APIHandler) GetArtisanStorefront(w http.ResponseWriter, r *http.Request) {
	slug := httpx.URLParam(r, "slug")
	profile, err := h.catalogSvc.GetArtisanBySlug(r.Context(), slug)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, artisanProfileToMap(*profile))
}

// GetProcessFeed handles GET /feed/process: the vertical process-provenance
// feed, most recent first. See client.Catalog.ListProcessClips for how it is
// derived -- there is no dedicated feed store.
func (h *APIHandler) GetProcessFeed(w http.ResponseWriter, r *http.Request) {
	clips, err := h.catalogSvc.ListProcessClips(r.Context(), 20)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"clips": clipsToMaps(clips)})
}

func craftToMap(c Craft) map[string]any {
	m := map[string]any{
		"id":           c.ID,
		"slug":         c.Slug,
		"display_name": c.DisplayName,
		"regions":      c.Regions,
		"techniques":   c.Techniques,
		"materials":    c.Materials,
		"gi_certified": c.GIRegistrationNo != nil && *c.GIRegistrationNo != "",
	}
	if c.GIRegistrationNo != nil {
		m["gi_registration_no"] = *c.GIRegistrationNo
	}
	return m
}

func craftsToMaps(crafts []Craft) []map[string]any {
	out := make([]map[string]any, 0, len(crafts))
	for _, c := range crafts {
		out = append(out, craftToMap(c))
	}
	return out
}

func artisansToMaps(artisans []Artisan) []map[string]any {
	out := make([]map[string]any, 0, len(artisans))
	for _, a := range artisans {
		out = append(out, map[string]any{"id": a.ID, "display_name": a.DisplayName})
	}
	return out
}

func artisanProfileToMap(p ArtisanProfile) map[string]any {
	m := map[string]any{
		"id":           p.ID,
		"slug":         p.Slug,
		"display_name": p.DisplayName,
		"bio":          p.Bio,
		"location":     p.Location,
		"state_code":   p.StateCode,
		"image_url":    p.ImageURL,
		"craft_name":   p.CraftName,
		"craft_slug":   p.CraftSlug,
		"verified":     p.Verified,
	}
	if p.District != nil {
		m["district"] = *p.District
	}
	if p.ClusterID != nil {
		m["cluster_id"] = *p.ClusterID
	}
	if p.YearsExperience != nil {
		m["years_experience"] = *p.YearsExperience
	}
	return m
}

func clipsToMaps(clips []ProcessClip) []map[string]any {
	out := make([]map[string]any, 0, len(clips))
	for _, c := range clips {
		m := map[string]any{
			"listing_id":   c.ListingID,
			"listing_slug": c.ListingSlug,
			"title":        c.Title,
			"artisan_id":   c.ArtisanID,
			"artisan_name": c.ArtisanName,
			"video_url":    c.VideoURL,
		}
		if c.DurationMs != nil {
			m["duration_ms"] = *c.DurationMs
		}
		out = append(out, m)
	}
	return out
}

// Pricing endpoints

func (h *APIHandler) AdvisePricing(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var inputs map[string]any
	if err := json.NewDecoder(r.Body).Decode(&inputs); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	listingID, _ := inputs["listing_id"].(string)
	if listingID == "" {
		httpx.Error(w, domain.InvalidInput("listing_id is required"))
		return
	}

	advice, err := h.pricingSvc.AdvisePricing(r.Context(), listingID, inputs)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, advice)
}

// Order endpoints

func (h *APIHandler) CreateBulkOrder(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var fields map[string]any
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	// Security: delete any client-provided price or currency fields to guarantee
	// server-side pricing from canonical catalog records.
	delete(fields, "price")
	delete(fields, "price_paise")
	delete(fields, "unit_price")
	delete(fields, "total_price")

	// Sanitize any notes or customisation strings to prevent XSS/injection
	if notes, ok := fields["notes"].(string); ok {
		fields["notes"] = sanitizeText(notes)
	}

	orderID, err := h.orderSvc.CreateBulkOrder(r.Context(), p.Subject, idempotencyKeyFrom(r), fields)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	if h.logger != nil {
		h.logger.Info("security audit: bulk order created with server-computed pricing", "event", "ORDER_CREATED", "buyer_id", p.Subject, "order_id", orderID)
	}

	httpx.JSON(w, http.StatusCreated, map[string]string{"order_id": orderID})
}

func (h *APIHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	orderID := httpx.URLParam(r, "id")
	order, err := h.orderSvc.GetOrder(r.Context(), orderID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, order)
}

func (h *APIHandler) RespondToLot(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	lotID := httpx.URLParam(r, "id")
	var req struct {
		Accept bool `json:"accept"`
	}
	var fields map[string]any
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.Error(w, domain.InvalidInput("cannot read request body"))
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}
	if err := json.Unmarshal(body, &fields); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	if err := h.orderSvc.RespondToLot(r.Context(), lotID, p.Subject, idempotencyKeyFrom(r), req.Accept, fields); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

// ReportProgress handles POST /orders/lots/{id}/progress.
func (h *APIHandler) ReportProgress(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	lotID := httpx.URLParam(r, "id")
	var fields map[string]any
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	lot, err := h.orderSvc.ReportProgress(r.Context(), lotID, p.Subject, idempotencyKeyFrom(r), fields)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, lot)
}

// RequestReallocation handles POST /orders/lots/{id}/reallocate — the
// non-punitive give-up path when an artisan cannot complete a lot.
func (h *APIHandler) RequestReallocation(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	lotID := httpx.URLParam(r, "id")
	var fields map[string]any
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	lot, err := h.orderSvc.RequestReallocation(r.Context(), lotID, p.Subject, idempotencyKeyFrom(r), fields)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, lot)
}

// WatchOrder handles GET /orders/{id}/events — SSE bridging gRPC WatchOrder.
// Reconnect/backfill without duplicate rows works like this: each frame's
// `id:` line IS the event's occurred_at timestamp (RFC3339Nano), not an
// opaque counter, so when the browser auto-reconnects it sends that value
// straight back as Last-Event-ID and it's already exactly what
// WatchOrderRequest.since wants -- no separate id-to-timestamp store needed.
// A manual reconnect (or a client managing its own retry) can pass the same
// value as a ?since= query param instead. Either way the client should still
// key rows by event_id, since a boundary event can be replayed once.
func (h *APIHandler) WatchOrder(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	orderID := httpx.URLParam(r, "id")
	since := parseSince(r.Header.Get("Last-Event-ID"))
	if since == nil {
		since = parseSince(r.URL.Query().Get("since"))
	}

	events, err := h.orderSvc.WatchOrder(r.Context(), orderID, since)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, domain.Unavailable("SSE not supported"))
		return
	}

	// Stream events.
	for event := range events {
		data, _ := json.Marshal(event)
		id, _ := event["occurred_at"].(string)
		fmt.Fprintf(w, "id: %s\ndata: %s\n\n", id, data)
		flusher.Flush()
	}
}

func parseSince(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil
	}
	return &t
}

// Follow endpoints

func (h *APIHandler) FollowArtisan(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	artisanID := httpx.URLParam(r, "id")
	if err := h.followSvc.FollowArtisan(r.Context(), p.Subject, artisanID); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

func (h *APIHandler) UnfollowArtisan(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	artisanID := httpx.URLParam(r, "id")
	if err := h.followSvc.UnfollowArtisan(r.Context(), p.Subject, artisanID); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

func (h *APIHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	feed, err := h.followSvc.GetFeed(r.Context(), p.Subject, 50, 0)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"feed": feed})
}

// MarkNotificationRead handles POST /feed/{id}/read.
func (h *APIHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	notificationID := httpx.URLParam(r, "id")
	if err := h.followSvc.MarkFeedItemRead(r.Context(), notificationID, p.Subject); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

// GetFollowerCount handles GET /artisans/{id}/follower-count. Public: a
// follower count is display data on any visitor's view of a storefront, not
// something that needs a signed-in principal to read.
func (h *APIHandler) GetFollowerCount(w http.ResponseWriter, r *http.Request) {
	artisanID := httpx.URLParam(r, "id")
	count, err := h.followSvc.GetFollowerCount(r.Context(), artisanID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"count": count})
}

// Statement endpoints

func (h *APIHandler) GenerateStatement(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var req struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	statement, err := h.stmtSvc.GenerateStatement(r.Context(), p.Subject, req.Start, req.End)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, statement)
}

// ListIncomeStatements handles GET /statements — the artisan's own earnings
// history, newest first.
func (h *APIHandler) ListIncomeStatements(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	statements, err := h.stmtSvc.ListIncomeStatements(r.Context(), p.Subject, 50, 0)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"statements": statements})
}

func (h *APIHandler) GetStatement(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	statementID := httpx.URLParam(r, "id")
	statement, err := h.stmtSvc.GetStatement(r.Context(), statementID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, statement)
}

// Insight endpoints (MINISTRY role only)

func (h *APIHandler) GetEarningsByDistrict(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequireRole(r.Context(), auth.RoleMinistry)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	filters := map[string]any{
		"state_code": r.URL.Query().Get("state_code"),
		"district":   r.URL.Query().Get("district"),
	}

	data, err := h.insightSvc.GetEarningsByDistrict(r.Context(), filters)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *APIHandler) GetIncomeComparison(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequireRole(r.Context(), auth.RoleMinistry)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	filters := map[string]any{
		"state_code": r.URL.Query().Get("state_code"),
		"district":   r.URL.Query().Get("district"),
	}
	data, err := h.insightSvc.GetIncomeComparison(r.Context(), filters)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *APIHandler) GetDyingCrafts(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequireRole(r.Context(), auth.RoleMinistry)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	data, err := h.insightSvc.GetDyingCrafts(r.Context(), 100)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *APIHandler) GetArtisansByCategory(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequireRole(r.Context(), auth.RoleMinistry)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	filters := map[string]any{
		"state_code": r.URL.Query().Get("state_code"),
		"district":   r.URL.Query().Get("district"),
	}
	data, err := h.insightSvc.GetArtisansByCategory(r.Context(), filters)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *APIHandler) GetListingsByCraftMonth(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequireRole(r.Context(), auth.RoleMinistry)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	filters := map[string]any{"craft_id": r.URL.Query().Get("craft_id")}
	if from := r.URL.Query().Get("from_date"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filters["from_date"] = t
		}
	}
	if to := r.URL.Query().Get("to_date"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filters["to_date"] = t
		}
	}

	data, err := h.insightSvc.GetListingsByCraftMonth(r.Context(), filters)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *APIHandler) RefreshInsights(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequireRole(r.Context(), auth.RoleMinistry)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	data, err := h.insightSvc.RefreshMaterializedViews(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, data)
}

var promptInjectionPatterns = regexp.MustCompile(`(?i)(ignore\s+(all\s+)?previous\s+instructions|system\s*:\s*|\[INST\]|<\|im_start\|>|<\|endoftext\|>|jailbreak|prompt\s*injection)`)

// SanitizePromptInput blocks common prompt injection tokens and control characters.
func SanitizePromptInput(input string) string {
	cleaned := promptInjectionPatterns.ReplaceAllString(input, "[REDACTED_PROMPT_CONTROL]")
	cleaned = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, cleaned)
	return strings.TrimSpace(cleaned)
}

func sanitizeText(input string) string {
	htmlStrip := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(htmlStrip.ReplaceAllString(input, ""))
}

func maskIdentifier(id string) string {
	if len(id) <= 4 {
		return "****"
	}
	return id[:2] + "****" + id[len(id)-2:]
}

