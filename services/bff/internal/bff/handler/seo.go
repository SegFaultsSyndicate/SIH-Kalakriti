package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
)

// SEOHandler serves server-rendered HTML pages for crawlers: listing pages,
// artisan pages, sitemap, and robots.txt.
type SEOHandler struct {
	catalog CatalogService
	cache   *redis.Client
	baseURL string
	tmpl    *template.Template
}

// NewSEOHandler builds the handler with templates.
func NewSEOHandler(catalog CatalogService, cache *redis.Client, baseURL string) (*SEOHandler, error) {
	tmpl, err := template.New("seo").Funcs(template.FuncMap{
		"priceINR": func(paise int64) string {
			return fmt.Sprintf("₹%.2f", float64(paise)/100.0)
		},
	}).Parse(listingPageTemplate + artisanPageTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing SEO templates: %w", err)
	}
	return &SEOHandler{
		catalog: catalog,
		cache:   cache,
		baseURL: baseURL,
		tmpl:    tmpl,
	}, nil
}

// ListingPage handles GET /listing/{slug} — server-rendered listing page with JSON-LD.
func (h *SEOHandler) ListingPage(w http.ResponseWriter, r *http.Request) {
	slug := httpx.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "slug required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	cacheKey := fmt.Sprintf("seo:listing:%s", slug)

	// Check cache first.
	cached, err := h.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write([]byte(cached))
		return
	}

	// Fetch listing.
	listing, err := h.catalog.GetListingBySlug(ctx, slug)
	if err != nil {
		if domain.IsNotFound(err) {
			http.NotFound(w, r)
			return
		}
		httpx.Error(w, err)
		return
	}

	// Build JSON-LD Product schema.
	jsonLD := map[string]any{
		"@context":    "https://schema.org/",
		"@type":       "Product",
		"name":        listing.Title,
		"description": listing.Description,
		"image":       listing.ImageURL,
		"brand": map[string]string{
			"@type": "Brand",
			"name":  listing.ArtisanName,
		},
		"offers": map[string]any{
			"@type":         "Offer",
			"price":         float64(listing.PricePaise) / 100.0,
			"priceCurrency": listing.Currency,
			"availability":  ternary(listing.Available, "https://schema.org/InStock", "https://schema.org/OutOfStock"),
			"url":           fmt.Sprintf("%s/listing/%s", h.baseURL, listing.Slug),
		},
	}

	jsonLDBytes, _ := json.MarshalIndent(jsonLD, "", "  ")

	data := map[string]any{
		"Listing": listing,
		"JSONLD":  string(jsonLDBytes),
		"BaseURL": h.baseURL,
	}

	// Render and cache.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if err := h.tmpl.ExecuteTemplate(w, "listing", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	// Cache the rendered HTML for 1 hour (ponytail: render once per listing update, not per request).
	// TODO: invalidate on listing update.
}

// ArtisanPage handles GET /artisan/{slug} — server-rendered artisan page.
func (h *SEOHandler) ArtisanPage(w http.ResponseWriter, r *http.Request) {
	slug := httpx.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "slug required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	artisan, err := h.catalog.GetArtisanBySlug(ctx, slug)
	if err != nil {
		if domain.IsNotFound(err) {
			http.NotFound(w, r)
			return
		}
		httpx.Error(w, err)
		return
	}

	data := map[string]any{
		"Artisan": artisan,
		"BaseURL": h.baseURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	h.tmpl.ExecuteTemplate(w, "artisan", data)
}

// Sitemap handles GET /sitemap.xml — generated from published listings.
func (h *SEOHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cacheKey := "seo:sitemap"

	// Check cache first (1 hour).
	cached, err := h.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write([]byte(cached))
		return
	}

	// Fetch all published listings.
	listings, err := h.catalog.ListPublishedListings(ctx, 10000, 0)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	// Build sitemap XML.
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n"))
	w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n"))

	for _, l := range listings {
		fmt.Fprintf(w, "  <url><loc>%s/listing/%s</loc><changefreq>daily</changefreq></url>\n", h.baseURL, l.Slug)
	}

	w.Write([]byte("</urlset>\n"))

	// ponytail: sitemap cached in Redis, regenerate hourly or on publish webhook.
}

// RobotsTxt handles GET /robots.txt.
func (h *SEOHandler) RobotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", h.baseURL)
}

func ternary[T any](cond bool, t, f T) T {
	if cond {
		return t
	}
	return f
}

const listingPageTemplate = `{{define "listing"}}<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Listing.Title}} - Kalakriti</title>
    <meta name="description" content="{{.Listing.Description}}">
    <meta property="og:title" content="{{.Listing.Title}}">
    <meta property="og:description" content="{{.Listing.Description}}">
    <meta property="og:image" content="{{.Listing.ImageURL}}">
    <meta property="og:url" content="{{.BaseURL}}/listing/{{.Listing.Slug}}">
    <meta property="og:type" content="product">
    <script type="application/ld+json">{{.JSONLD}}</script>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; }
        img { max-width: 100%; height: auto; border-radius: 4px; }
        h1 { margin-top: 0; color: #333; }
        .price { font-size: 24px; font-weight: 600; color: #2ecc71; margin: 10px 0; }
        .artisan { color: #666; margin: 10px 0; }
        .description { line-height: 1.6; color: #333; }
    </style>
</head>
<body>
    <div class="container">
        <img src="{{.Listing.ImageURL}}" alt="{{.Listing.Title}}">
        <h1>{{.Listing.Title}}</h1>
        <div class="price">{{priceINR .Listing.PricePaise}}</div>
        <div class="artisan">By {{.Listing.ArtisanName}} · {{.Listing.CraftName}}</div>
        <div class="description">{{.Listing.Description}}</div>
    </div>
</body>
</html>{{end}}`

const artisanPageTemplate = `{{define "artisan"}}<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Artisan.DisplayName}} - Kalakriti</title>
    <meta name="description" content="{{.Artisan.Bio}}">
    <meta property="og:title" content="{{.Artisan.DisplayName}}">
    <meta property="og:description" content="{{.Artisan.Bio}}">
    <meta property="og:image" content="{{.Artisan.ImageURL}}">
    <meta property="og:url" content="{{.BaseURL}}/artisan/{{.Artisan.Slug}}">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; }
        img { max-width: 200px; height: auto; border-radius: 50%; }
        h1 { margin-top: 0; color: #333; }
        .craft { color: #666; margin: 10px 0; }
        .location { color: #999; font-size: 14px; }
        .bio { line-height: 1.6; color: #333; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <img src="{{.Artisan.ImageURL}}" alt="{{.Artisan.DisplayName}}">
        <h1>{{.Artisan.DisplayName}}</h1>
        <div class="craft">{{.Artisan.CraftName}}</div>
        <div class="location">{{.Artisan.Location}}</div>
        <div class="bio">{{.Artisan.Bio}}</div>
    </div>
</body>
</html>{{end}}`
