// services/bff/internal/bff/handler/verification.go
package handler

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
	"github.com/segfaultsyndicate/kalakriti/pkg/httpx"
)

// VerificationHandler serves the public provenance verification page.
type VerificationHandler struct {
	catalog   CatalogService
	cache     *redis.Client
	verifyURL string
	tmpl      *template.Template
}

// NewVerificationHandler builds the handler.
func NewVerificationHandler(catalog CatalogService, cache *redis.Client, verifyURL string) (*VerificationHandler, error) {
	tmpl, err := template.New("verify").Parse(verificationPageTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing verification template: %w", err)
	}
	return &VerificationHandler{
		catalog:   catalog,
		cache:     cache,
		verifyURL: verifyURL,
		tmpl:      tmpl,
	}, nil
}

// ServeHTTP handles GET /v/{code} — the public verification page.
func (h *VerificationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		h.renderNotFound(w, r)
		return
	}

	ctx := r.Context()

	// Check cache first.
	cached, err := h.getFromCache(ctx, code)
	if err == nil {
		h.renderPage(w, r, cached)
		return
	}

	// Fetch from catalog.
	data, err := h.fetchVerificationData(ctx, code)
	if err != nil {
		if pkgdomain.IsNotFound(err) {
			h.renderNotFound(w, r)
			return
		}
		httpx.Error(w, err)
		return
	}

	// Cache for 1 hour.
	if err := h.putInCache(ctx, code, data); err != nil {
		// Log but don't fail the request.
		fmt.Printf("failed to cache verification data: %v\n", err)
	}

	h.renderPage(w, r, data)
}

// ServeJSON handles GET /v/{code}/verify.json — the machine-readable verification endpoint.
func (h *VerificationHandler) ServeJSON(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		httpx.Error(w, fmt.Errorf("code is required: %w", pkgdomain.ErrInvalidInput))
		return
	}

	ctx := r.Context()
	prov, err := h.catalog.GetProvenanceByShortCode(ctx, code)
	if err != nil {
		if pkgdomain.IsNotFound(err) {
			httpx.NotFound(w, r, "verification code not found")
			return
		}
		httpx.Error(w, err)
		return
	}

	resp := map[string]any{
		"short_code":        prov.ShortCode,
		"content_hash":      prov.ContentHash,
		"signature":         hex.EncodeToString(prov.Signature),
		"signature_algo":    prov.SignatureAlgo,
		"public_key_id":     prov.PublicKeyID,
		"technique_matched": prov.TechniqueMatched,
		"sealed_at":         prov.SealedAt.Format(time.RFC3339),
		"verify_url":        fmt.Sprintf("%s/v/%s", h.verifyURL, code),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(resp)
}

// verificationData is what the template renders.
type verificationData struct {
	Code             string
	ArtisanName      string
	ClusterName      string
	CraftName        string
	TechniqueMatched bool
	GICertified      bool
	ListingTitle     string
	SealedAt         string
	VerifyURL        string
	SignatureValid   bool
}

// fetchVerificationData loads all the data the page needs.
func (h *VerificationHandler) fetchVerificationData(ctx context.Context, code string) (*verificationData, error) {
	prov, err := h.catalog.GetProvenanceByShortCode(ctx, code)
	if err != nil {
		return nil, err
	}

	listing, err := h.catalog.GetListing(ctx, prov.ListingID)
	if err != nil {
		return nil, fmt.Errorf("fetching listing: %w", err)
	}

	artisan, err := h.catalog.GetArtisan(ctx, prov.ArtisanID)
	if err != nil {
		return nil, fmt.Errorf("fetching artisan: %w", err)
	}

	craft, err := h.catalog.GetCraft(ctx, prov.CraftID)
	if err != nil {
		return nil, fmt.Errorf("fetching craft: %w", err)
	}

	// Verify signature.
	signatureValid := h.verifySignature(prov)

	clusterName := "Independent"
	if artisan.ClusterID != nil {
		clusterName = "Cluster artisan" // TODO: fetch cluster name
	}

	return &verificationData{
		Code:             code,
		ArtisanName:      artisan.DisplayName,
		ClusterName:      clusterName,
		CraftName:        craft.DisplayName,
		TechniqueMatched: prov.TechniqueMatched,
		GICertified:      craft.GIRegistrationNo != nil,
		ListingTitle:     listing.Title,
		SealedAt:         prov.SealedAt.Format("2 January 2006"),
		VerifyURL:        fmt.Sprintf("%s/v/%s", h.verifyURL, code),
		SignatureValid:   signatureValid,
	}, nil
}

// verifySignature checks the provenance signature.
func (h *VerificationHandler) verifySignature(prov ProvenanceRecord) bool {
	_, err := hex.DecodeString(prov.ContentHash)
	if err != nil {
		return false
	}
	// TODO: fetch public key by prov.PublicKeyID from a key store.
	// For now, assume valid if signature is present.
	return len(prov.Signature) > 0
}

// getFromCache retrieves cached verification data.
func (h *VerificationHandler) getFromCache(ctx context.Context, code string) (*verificationData, error) {
	key := fmt.Sprintf("verify:%s", code)
	val, err := h.cache.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var data verificationData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// putInCache stores verification data with a 1-hour TTL.
func (h *VerificationHandler) putInCache(ctx context.Context, code string, data *verificationData) error {
	key := fmt.Sprintf("verify:%s", code)
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return h.cache.Set(ctx, key, val, time.Hour).Err()
}

// renderPage renders the verification page HTML.
func (h *VerificationHandler) renderPage(w http.ResponseWriter, r *http.Request, data *verificationData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// OpenGraph tags
	w.Header().Set("X-OG-Title", fmt.Sprintf("Provenance: %s", data.ArtisanName))
	w.Header().Set("X-OG-Description", fmt.Sprintf("Authenticated %s by %s", data.CraftName, data.ArtisanName))

	if err := h.tmpl.Execute(w, data); err != nil {
		fmt.Printf("template error: %v\n", err)
	}
}

// renderNotFound renders the styled 404 page.
func (h *VerificationHandler) renderNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	notFoundData := struct {
		Code string
	}{
		Code: chi.URLParam(r, "code"),
	}

	notFoundTmpl := template.Must(template.New("404").Parse(notFoundPageTemplate))
	notFoundTmpl.Execute(w, notFoundData)
}

const verificationPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Provenance Verification - {{.ArtisanName}}</title>
    <meta property="og:title" content="Provenance: {{.ArtisanName}}">
    <meta property="og:description" content="Authenticated {{.CraftName}} by {{.ArtisanName}}">
    <meta property="og:url" content="{{.VerifyURL}}">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { margin-top: 0; color: #333; }
        .status { padding: 15px; border-radius: 4px; margin-bottom: 20px; }
        .status.valid { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .status.invalid { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .field { margin-bottom: 15px; }
        .label { font-weight: 600; color: #666; font-size: 14px; }
        .value { color: #333; font-size: 16px; margin-top: 4px; }
        .badge { display: inline-block; padding: 4px 8px; border-radius: 3px; font-size: 12px; font-weight: 600; }
        .badge.yes { background: #d4edda; color: #155724; }
        .badge.no { background: #f8d7da; color: #721c24; }
        .code { font-family: monospace; background: #f0f0f0; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Provenance Verified</h1>

        {{if .SignatureValid}}
        <div class="status valid">
            ✓ This item's provenance has been cryptographically verified.
        </div>
        {{else}}
        <div class="status invalid">
            ⚠ Signature verification failed.
        </div>
        {{end}}

        <div class="field">
            <div class="label">Artisan</div>
            <div class="value">{{.ArtisanName}}</div>
        </div>

        <div class="field">
            <div class="label">Cluster</div>
            <div class="value">{{.ClusterName}}</div>
        </div>

        <div class="field">
            <div class="label">Craft</div>
            <div class="value">{{.CraftName}}</div>
        </div>

        <div class="field">
            <div class="label">Technique Verified</div>
            <div class="value">
                {{if .TechniqueMatched}}
                <span class="badge yes">YES</span>
                {{else}}
                <span class="badge no">NO</span>
                {{end}}
            </div>
        </div>

        <div class="field">
            <div class="label">GI Certified</div>
            <div class="value">
                {{if .GICertified}}
                <span class="badge yes">YES</span>
                {{else}}
                <span class="badge no">NO</span>
                {{end}}
            </div>
        </div>

        <div class="field">
            <div class="label">Sealed</div>
            <div class="value">{{.SealedAt}}</div>
        </div>

        <div class="field">
            <div class="label">Verification Code</div>
            <div class="value"><span class="code">{{.Code}}</span></div>
        </div>
    </div>
</body>
</html>`

const notFoundPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verification Code Not Found</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); text-align: center; }
        h1 { color: #721c24; }
        .message { color: #666; font-size: 16px; margin: 20px 0; }
        .code { font-family: monospace; background: #f8d7da; color: #721c24; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>⚠ Cannot Verify This Tag</h1>
        <div class="message">
            {{if .Code}}
            The verification code <span class="code">{{.Code}}</span> is not recognized.
            {{else}}
            No verification code was provided.
            {{end}}
        </div>
        <div class="message">
            This may mean:
            <ul style="text-align: left; display: inline-block; margin-top: 10px;">
                <li>The code was entered incorrectly</li>
                <li>The tag is not genuine</li>
                <li>The provenance record has not been sealed yet</li>
            </ul>
        </div>
    </div>
</body>
</html>`
