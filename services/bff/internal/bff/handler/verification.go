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

	"github.com/redis/go-redis/v9"

	"github.com/ZoroNewbie00/kalakriti/pkg/crypto"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
)

// VerificationHandler serves the public provenance verification page.
type VerificationHandler struct {
	catalog   CatalogService
	cache     *redis.Client
	verifyURL string
	tmpl      *template.Template
	publicKey []byte // Ed25519 public key that signed sealed provenance records; nil fails closed.
}

// NewVerificationHandler builds the handler. publicKeyHex is the hex-encoded
// Ed25519 public key matching the provenance signer's private key (see
// scripts/keygen.go); pass "" only in environments with no signer configured,
// which makes every signature check fail closed rather than report valid.
func NewVerificationHandler(catalog CatalogService, cache *redis.Client, verifyURL, publicKeyHex string) (*VerificationHandler, error) {
	tmpl, err := template.New("verify").Parse(verificationPageTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing verification template: %w", err)
	}
	var pubKey []byte
	if publicKeyHex != "" {
		pubKey, err = hex.DecodeString(publicKeyHex)
		if err != nil {
			return nil, fmt.Errorf("decoding provenance public key: %w", err)
		}
	}
	return &VerificationHandler{
		catalog:   catalog,
		cache:     cache,
		verifyURL: verifyURL,
		tmpl:      tmpl,
		publicKey: pubKey,
	}, nil
}

// ServeHTTP handles GET /v/{code} — the public verification page.
func (h *VerificationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := httpx.URLParam(r, "code")
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
	code := httpx.URLParam(r, "code")
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

// verifySignature recomputes the signed message from the record's content
// hash and checks it against the stored signature. A missing or malformed
// content hash, or no configured public key, fails closed.
func (h *VerificationHandler) verifySignature(prov ProvenanceRecord) bool {
	if len(h.publicKey) == 0 {
		return false
	}
	message, err := hex.DecodeString(prov.ContentHash)
	if err != nil {
		return false
	}
	return crypto.Verify(h.publicKey, message, prov.Signature)
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

	view := struct {
		*verificationData
		CSS template.CSS
	}{data, template.CSS(verificationPageCSS)}
	if err := h.tmpl.Execute(w, view); err != nil {
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
		Code: httpx.URLParam(r, "code"),
	}

	notFoundTmpl := template.Must(template.New("404").Parse(notFoundPageTemplate))
	notFoundTmpl.Execute(w, notFoundData)
}

// verificationPageCSS is the standalone stylesheet for the public provenance
// page, built from the same tokens web/packages/tokens/src ships (values
// copied literally since a Go html/template has no CSS custom-property
// pipeline of its own -- token names kept in comments so a palette change
// is easy to find and mirror here). No @font-face/@import: the page must
// render correctly with the network otherwise blocked, so it falls back to
// the system stack rather than fetch a webfont.
const verificationPageCSS = `:root{color-scheme:light}
body{margin:0;padding:0;background:#FCFAF6;color:#241E1A;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;font-size:1rem;line-height:1.55}
.k-verify{max-width:34rem;margin:0 auto;padding:2rem 1rem 4rem}
.k-verify__mark{display:flex;align-items:center;gap:.5rem;font-size:.833rem;letter-spacing:.08em;text-transform:uppercase;color:#554B44;margin-block-end:2rem}
.k-verify__status{border-block:2px solid;padding:1rem 0;margin-block-end:2rem}
.k-verify__status--valid{border-color:#254D21;color:#254D21}
.k-verify__status--invalid{border-color:#93011E;color:#93011E}
.k-verify__status-title{font-size:1.2rem;font-weight:600;margin:0 0 .25rem}
.k-verify__status-body{margin:0;color:#241E1A;font-size:.9rem}
.k-verify h1{font-size:1.44rem;font-weight:600;margin:0 0 .25rem;line-height:1.15}
.k-verify__subtitle{color:#554B44;margin:0 0 2rem}
.k-verify__fields{margin:0;padding:0}
.k-verify__field{display:flex;justify-content:space-between;gap:1rem;padding:.75rem 0;border-block-end:1px solid #CECAC6}
.k-verify__field:first-child{border-block-start:1px solid #CECAC6}
.k-verify__label{color:#554B44;font-size:.9rem}
.k-verify__value{color:#241E1A;font-weight:500;text-align:right}
.k-verify__badge{display:inline-block;font-size:.833rem;font-weight:600;padding:.125rem .5rem;border-radius:2px}
.k-verify__badge--yes{background:#B4CCB1;color:#254D21}
.k-verify__badge--no{background:#EFC7B8;color:#7E1300}
.k-verify__code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;background:#F5F0E8;padding:.125rem .375rem;border-radius:2px}
.k-verify__footer{margin-block-start:2rem;color:#554B44;font-size:.833rem}
.k-verify__footer a{color:#305191}
@media (prefers-color-scheme:dark){
  :root{color-scheme:dark}
  body{background:#0F0A07;color:#F5F0E8}
  .k-verify__mark{color:#AFAAA5}
  .k-verify__subtitle,.k-verify__label,.k-verify__footer{color:#AFAAA5}
  .k-verify__field{border-color:#3D3630}
  .k-verify__field:first-child{border-color:#3D3630}
  .k-verify__code{background:#3C332E}
  .k-verify__status--valid{border-color:#8FAB8B;color:#8FAB8B}
  .k-verify__status--invalid{border-color:#E68582;color:#E68582}
}`

const verificationPageTemplate = `{{define "verify"}}<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Provenance verification — {{.ArtisanName}}</title>
    <meta name="description" content="Authenticated {{.CraftName}} by {{.ArtisanName}}">
    <meta property="og:title" content="Provenance: {{.ArtisanName}}">
    <meta property="og:description" content="Authenticated {{.CraftName}} by {{.ArtisanName}}">
    <meta property="og:url" content="{{.VerifyURL}}">
    <style>{{.CSS}}</style>
</head>
<body>
    <main class="k-verify">
        <p class="k-verify__mark">Kalakriti · Provenance</p>

        {{if .SignatureValid}}
        <section class="k-verify__status k-verify__status--valid" role="status">
            <p class="k-verify__status-title">Verified</p>
            <p class="k-verify__status-body">This item's provenance record was cryptographically signed and has not been altered.</p>
        </section>
        {{else}}
        <section class="k-verify__status k-verify__status--invalid" role="status">
            <p class="k-verify__status-title">Signature could not be verified</p>
            <p class="k-verify__status-body">The record's signature did not check out against the signing key on file. Treat this tag as unverified.</p>
        </section>
        {{end}}

        <h1>{{.ListingTitle}}</h1>
        <p class="k-verify__subtitle">by {{.ArtisanName}}</p>

        <dl class="k-verify__fields">
            <div class="k-verify__field"><dt class="k-verify__label">Craft</dt><dd class="k-verify__value">{{.CraftName}}</dd></div>
            <div class="k-verify__field"><dt class="k-verify__label">Cluster</dt><dd class="k-verify__value">{{.ClusterName}}</dd></div>
            <div class="k-verify__field">
                <dt class="k-verify__label">Technique matches claim</dt>
                <dd class="k-verify__value">{{if .TechniqueMatched}}<span class="k-verify__badge k-verify__badge--yes">Confirmed</span>{{else}}<span class="k-verify__badge k-verify__badge--no">Not confirmed</span>{{end}}</dd>
            </div>
            <div class="k-verify__field">
                <dt class="k-verify__label">Geographical Indication</dt>
                <dd class="k-verify__value">{{if .GICertified}}<span class="k-verify__badge k-verify__badge--yes">GI tagged</span>{{else}}<span class="k-verify__badge k-verify__badge--no">Not GI tagged</span>{{end}}</dd>
            </div>
            <div class="k-verify__field"><dt class="k-verify__label">Sealed on</dt><dd class="k-verify__value">{{.SealedAt}}</dd></div>
            <div class="k-verify__field"><dt class="k-verify__label">Verification code</dt><dd class="k-verify__value"><span class="k-verify__code">{{.Code}}</span></dd></div>
        </dl>

        <p class="k-verify__footer">This page renders without JavaScript, so a printed tag verifies the same way it looks here even on a low-end phone or a weak connection.</p>
    </main>
</body>
</html>{{end}}`

const notFoundPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>We cannot verify this tag</title>
    <style>` + verificationPageCSS + `</style>
</head>
<body>
    <main class="k-verify">
        <p class="k-verify__mark">Kalakriti · Provenance</p>
        <section class="k-verify__status k-verify__status--invalid" role="status">
            <p class="k-verify__status-title">We cannot verify this tag</p>
            <p class="k-verify__status-body">
                {{if .Code}}The code <span class="k-verify__code">{{.Code}}</span> does not match a sealed provenance record.{{else}}No verification code was given.{{end}}
            </p>
        </section>
        <p class="k-verify__subtitle">This can honestly mean any of a few things:</p>
        <dl class="k-verify__fields">
            <div class="k-verify__field"><dt class="k-verify__label">Mistyped or damaged code</dt><dd class="k-verify__value">Check the printed tag and try again</dd></div>
            <div class="k-verify__field"><dt class="k-verify__label">Not yet sealed</dt><dd class="k-verify__value">The artisan hasn't frozen provenance for this piece yet</dd></div>
            <div class="k-verify__field"><dt class="k-verify__label">Not genuine</dt><dd class="k-verify__value">The tag doesn't correspond to any Kalakriti-verified item</dd></div>
        </dl>
        <p class="k-verify__footer">An unverifiable tag is not proof of fraud, but it is not proof of anything either — treat it as unverified rather than assume it is real.</p>
    </main>
</body>
</html>`
