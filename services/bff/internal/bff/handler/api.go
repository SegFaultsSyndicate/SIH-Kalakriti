package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
)

// APIHandler aggregates all REST /api/v1 endpoints, bridging HTTP/JSON to gRPC.
type APIHandler struct {
	authSvc     AuthService
	artisanSvc  ArtisanService
	mediaSvc    MediaService
	listingSvc  ListingService
	searchSvc   SearchService
	pricingSvc  PricingService
	orderSvc    OrderService
	followSvc   FollowService
	stmtSvc     StatementService
	insightSvc  InsightService
}

// AuthService is the auth-svc gRPC client interface.
type AuthService interface {
	RequestOTP(phone string) error
	VerifyOTP(phone, otp string) (accessToken, refreshToken string, err error)
	RefreshToken(refreshToken string) (accessToken string, err error)
}

// ArtisanService is the artisan-svc gRPC client interface.
type ArtisanService interface {
	Register(principalID, displayName, phone, language string) (artisanID string, err error)
	GetProfile(artisanID string) (map[string]any, error)
	UpdateProfile(artisanID string, updates map[string]any) error
}

// MediaService is the media-svc gRPC client interface.
type MediaService interface {
	GenerateUploadURL(artisanID, contentType string, sizeBytes int64) (mediaID, uploadURL string, err error)
	ConfirmUpload(mediaID string) error
}

// ListingService is the listing-svc gRPC client interface.
type ListingService interface {
	CreateListing(artisanID string, listing map[string]any) (listingID string, err error)
	UpdateListing(listingID string, updates map[string]any) error
	SubmitForReview(listingID string) error
	ApproveListing(listingID, reviewerID string) error
	GetListing(listingID string) (map[string]any, error)
	ListListings(filters map[string]any) ([]map[string]any, error)
}

// SearchService is the search-svc gRPC client interface.
type SearchService interface {
	Search(query string, filters map[string]any) ([]map[string]any, error)
	Suggest(prefix string) ([]string, error)
	SearchVoice(audioData []byte, language string) (query string, results []map[string]any, err error)
}

// PricingService is the pricing-svc gRPC client interface.
type PricingService interface {
	AdvisePricing(craftID, region string, inputs map[string]any) (map[string]any, error)
}

// OrderService is the order-svc gRPC client interface.
type OrderService interface {
	CreateBulkOrder(buyerID string, lots []map[string]any) (orderID string, err error)
	GetOrder(orderID string) (map[string]any, error)
	RespondToLot(lotID, artisanID, response string) error
	WatchOrder(orderID string) (<-chan map[string]any, error)
}

// FollowService is the follow-svc gRPC client interface.
type FollowService interface {
	FollowArtisan(followerID, artisanID string) error
	UnfollowArtisan(followerID, artisanID string) error
	GetFeed(userID string, limit, offset int32) ([]map[string]any, error)
}

// StatementService is the statement-svc gRPC client interface.
type StatementService interface {
	GenerateStatement(artisanID string, start, end string) (statementID string, err error)
	GetStatement(statementID string) (map[string]any, error)
}

// InsightService is the insight-svc gRPC client interface.
type InsightService interface {
	GetEarningsByDistrict(filters map[string]any) ([]map[string]any, error)
	GetIncomeComparison(filters map[string]any) ([]map[string]any, error)
	GetDyingCrafts(limit int32) ([]map[string]any, error)
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
	if req.Phone == "" {
		httpx.Error(w, domain.InvalidInput("phone is required"))
		return
	}

	if err := h.authSvc.RequestOTP(req.Phone); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "otp_sent"})
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

	accessToken, refreshToken, err := h.authSvc.VerifyOTP(req.Phone, req.OTP)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *APIHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	accessToken, err := h.authSvc.RefreshToken(req.RefreshToken)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"access_token": accessToken})
}

// Artisan endpoints

func (h *APIHandler) RegisterArtisan(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Language    string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	artisanID, err := h.artisanSvc.Register(p.Subject, req.DisplayName, p.PhoneE164, req.Language)
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

	profile, err := h.artisanSvc.GetProfile(p.Subject)
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

	if err := h.artisanSvc.UpdateProfile(p.Subject, updates); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

// Media endpoints

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

	mediaID, uploadURL, err := h.mediaSvc.GenerateUploadURL(p.Subject, req.ContentType, req.SizeBytes)
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

	mediaID := chi.URLParam(r, "id")
	if err := h.mediaSvc.ConfirmUpload(mediaID); err != nil {
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

	listingID, err := h.listingSvc.CreateListing(p.Subject, listing)
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

	listingID := chi.URLParam(r, "id")
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	if err := h.listingSvc.UpdateListing(listingID, updates); err != nil {
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

	listingID := chi.URLParam(r, "id")
	if err := h.listingSvc.SubmitForReview(listingID); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

func (h *APIHandler) ApproveListing(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequireRole(r.Context(), auth.RoleMinistry)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	listingID := chi.URLParam(r, "id")
	if err := h.listingSvc.ApproveListing(listingID, p.Subject); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

func (h *APIHandler) GetListing(w http.ResponseWriter, r *http.Request) {
	listingID := chi.URLParam(r, "id")
	listing, err := h.listingSvc.GetListing(listingID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, listing)
}

func (h *APIHandler) ListListings(w http.ResponseWriter, r *http.Request) {
	filters := map[string]any{
		"craft_id": r.URL.Query().Get("craft_id"),
		"state":    r.URL.Query().Get("state"),
	}

	listings, err := h.listingSvc.ListListings(filters)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"listings": listings})
}

// Search endpoints

func (h *APIHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	filters := map[string]any{
		"craft_id": r.URL.Query().Get("craft_id"),
		"region":   r.URL.Query().Get("region"),
	}

	results, err := h.searchSvc.Search(query, filters)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"results": results})
}

func (h *APIHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("q")
	suggestions, err := h.searchSvc.Suggest(prefix)
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
	query, results, err := h.searchSvc.SearchVoice(audioData, language)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"results": results,
	})
}

// Pricing endpoints

func (h *APIHandler) AdvisePricing(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var req struct {
		CraftID string         `json:"craft_id"`
		Region  string         `json:"region"`
		Inputs  map[string]any `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	advice, err := h.pricingSvc.AdvisePricing(req.CraftID, req.Region, req.Inputs)
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

	var req struct {
		Lots []map[string]any `json:"lots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	orderID, err := h.orderSvc.CreateBulkOrder(p.Subject, req.Lots)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]string{"order_id": orderID})
}

func (h *APIHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	orderID := chi.URLParam(r, "id")
	order, err := h.orderSvc.GetOrder(orderID)
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

	lotID := chi.URLParam(r, "id")
	var req struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid JSON"))
		return
	}

	if err := h.orderSvc.RespondToLot(lotID, p.Subject, req.Response); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w)
}

// WatchOrder handles GET /orders/{id}/events — SSE bridging gRPC WatchOrder.
func (h *APIHandler) WatchOrder(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	orderID := chi.URLParam(r, "id")
	events, err := h.orderSvc.WatchOrder(orderID)
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
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// Follow endpoints

func (h *APIHandler) FollowArtisan(w http.ResponseWriter, r *http.Request) {
	p, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	artisanID := chi.URLParam(r, "id")
	if err := h.followSvc.FollowArtisan(p.Subject, artisanID); err != nil {
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

	artisanID := chi.URLParam(r, "id")
	if err := h.followSvc.UnfollowArtisan(p.Subject, artisanID); err != nil {
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

	feed, err := h.followSvc.GetFeed(p.Subject, 50, 0)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"feed": feed})
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

	statementID, err := h.stmtSvc.GenerateStatement(p.Subject, req.Start, req.End)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]string{"statement_id": statementID})
}

func (h *APIHandler) GetStatement(w http.ResponseWriter, r *http.Request) {
	_, err := auth.RequirePrincipal(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}

	statementID := chi.URLParam(r, "id")
	statement, err := h.stmtSvc.GetStatement(statementID)
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

	data, err := h.insightSvc.GetEarningsByDistrict(filters)
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

	filters := map[string]any{}
	data, err := h.insightSvc.GetIncomeComparison(filters)
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

	data, err := h.insightSvc.GetDyingCrafts(100)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"data": data})
}
