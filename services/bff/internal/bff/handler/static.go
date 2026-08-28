package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SPAHandler serves the built Svelte SPA from distDir with correct cache headers
// and SPA fallback routing: hashed assets immutable, index.html no-cache, unknown
// non-API paths fall back to index.html.
type SPAHandler struct {
	distDir string
	fs      http.FileSystem
}

// NewSPAHandler creates a handler serving static files from distDir.
func NewSPAHandler(distDir string) *SPAHandler {
	return &SPAHandler{
		distDir: distDir,
		fs:      http.Dir(distDir),
	}
}

// ServeHTTP implements the SPA serving logic.
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Try to open the requested file.
	file, err := h.fs.Open(path)
	if err != nil {
		// Not found → fall back to index.html for SPA routing.
		h.serveIndex(w, r)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		h.serveIndex(w, r)
		return
	}

	// Set cache headers based on filename.
	if strings.Contains(path, ".") && isHashed(path) {
		// Hashed asset (e.g., main-abc123.js) → immutable, 1 year.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else if path == "/index.html" {
		// index.html → no-cache so updates reach the browser immediately.
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	} else {
		// Other static files → short cache.
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}

	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}

// serveIndex serves index.html as the SPA fallback.
func (h *SPAHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(h.distDir, "index.html")
	file, err := os.Open(indexPath)
	if err != nil {
		http.Error(w, "index.html not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "cannot stat index.html", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	http.ServeContent(w, r, "index.html", stat.ModTime(), file)
}

// isHashed returns true if the filename contains a hash suffix (e.g., main-abc123.js).
// Heuristic: a segment between the basename and extension that looks like a hash.
func isHashed(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext == "" {
		return false
	}
	nameWithoutExt := strings.TrimSuffix(base, ext)
	// Common pattern: filename-[hash].ext where hash is hex or base64url-like.
	parts := strings.Split(nameWithoutExt, "-")
	if len(parts) < 2 {
		return false
	}
	lastPart := parts[len(parts)-1]
	// Hash suffix is typically 8+ alphanumeric characters.
	if len(lastPart) >= 8 && isAlphanumeric(lastPart) {
		return true
	}
	return false
}

func isAlphanumeric(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return len(s) > 0
}
