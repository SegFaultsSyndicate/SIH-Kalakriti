// services/bff/internal/bff/handler/export.go
package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
)

// ExportHandler proxies channel-svc's catalog export feeds through the
// public edge. bff is meant to be the only service the outside world talks
// to; channel-svc's own HTTP port stays internal, and this handler is the
// one place that boundary is crossed for an outbound feed like
// IndiaHandmade.
type ExportHandler struct {
	channelSvcAddr string
	client         *http.Client
}

// NewExportHandler builds the proxy. channelSvcAddr is channel-svc's
// internal HTTP base URL (e.g. "http://channel-svc:8083").
func NewExportHandler(channelSvcAddr string) *ExportHandler {
	return &ExportHandler{
		channelSvcAddr: channelSvcAddr,
		client:         &http.Client{Timeout: 30 * time.Second},
	}
}

// IndiaHandmade proxies GET /export/indiahandmade to channel-svc, passing
// the query string through unchanged (?format=csv and friends) and
// forwarding channel-svc's Content-Type and status code as-is.
func (h *ExportHandler) IndiaHandmade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := h.channelSvcAddr + "/export/indiahandmade"
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		httpx.Error(w, fmt.Errorf("building export request: %w", err))
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		httpx.Error(w, fmt.Errorf("fetching export from channel-svc: %w", err))
		return
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
