// services/bff/internal/bff/handler/statement.go
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
)

// StatementClient talks to insight-svc.
type StatementClient interface {
	GenerateStatement(ctx context.Context, artisanID uuid.UUID, year, month int) (pdfURL, code string, err error)
	VerifyStatement(ctx context.Context, code string) (verified bool, details map[string]interface{}, err error)
}

// StatementHandler handles income statement endpoints.
type StatementHandler struct {
	client StatementClient
}

// NewStatementHandler creates the handler.
func NewStatementHandler(client StatementClient) *StatementHandler {
	return &StatementHandler{client: client}
}

// POST /statements - Generate income statement
func (h *StatementHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ArtisanID string `json:"artisan_id"`
		Year      int    `json:"year"`
		Month     int    `json:"month"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, domain.InvalidInput("invalid request body"))
		return
	}

	artisanID, err := uuid.Parse(req.ArtisanID)
	if err != nil {
		httpx.Error(w, domain.InvalidInput("invalid artisan_id"))
		return
	}

	if req.Year < 2020 || req.Year > 2100 || req.Month < 1 || req.Month > 12 {
		httpx.Error(w, domain.InvalidInput("invalid year or month"))
		return
	}

	pdfURL, code, err := h.client.GenerateStatement(r.Context(), artisanID, req.Year, req.Month)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"pdf_url": pdfURL,
		"qr_code": code,
	})
}

// GET /statements/{code}/verify - Verify statement signature
func (h *StatementHandler) Verify(w http.ResponseWriter, r *http.Request) {
	code := httpx.URLParam(r, "code")
	if code == "" {
		httpx.Error(w, domain.InvalidInput("missing code"))
		return
	}

	verified, details, err := h.client.VerifyStatement(r.Context(), code)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	if !verified {
		httpx.JSON(w, http.StatusNotFound, map[string]interface{}{
			"valid": false,
		})
		return
	}

	details["valid"] = true
	details["verified_at"] = time.Now().UTC().Format(time.RFC3339)
	httpx.JSON(w, http.StatusOK, details)
}
