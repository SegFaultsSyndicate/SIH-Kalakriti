// services/bff/internal/bff/handler/export_test.go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportHandlerIndiaHandmadeProxiesChannelSvc(t *testing.T) {
	var gotQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		assert.Equal(t, "/export/indiahandmade", r.URL.Path)
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("artisan_id,artisan_name\nart-1,Lakshmi Devi\n"))
	}))
	defer backend.Close()

	h := NewExportHandler(backend.URL)

	req := httptest.NewRequest(http.MethodGet, "/export/indiahandmade?format=csv", nil)
	w := httptest.NewRecorder()
	h.IndiaHandmade(w, req)

	assert.Equal(t, "format=csv", gotQuery)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "Lakshmi Devi")
}

func TestExportHandlerIndiaHandmadeRejectsNonGet(t *testing.T) {
	h := NewExportHandler("http://unused.invalid")

	req := httptest.NewRequest(http.MethodPost, "/export/indiahandmade", nil)
	w := httptest.NewRecorder()
	h.IndiaHandmade(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestExportHandlerIndiaHandmadeSurfacesUpstreamError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fetching catalog", http.StatusBadGateway)
	}))
	defer backend.Close()

	h := NewExportHandler(backend.URL)

	req := httptest.NewRequest(http.MethodGet, "/export/indiahandmade", nil)
	w := httptest.NewRecorder()
	h.IndiaHandmade(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestExportHandlerIndiaHandmadeReportsUnreachableBackend(t *testing.T) {
	h := NewExportHandler("http://127.0.0.1:1") // nothing listens here

	req := httptest.NewRequest(http.MethodGet, "/export/indiahandmade", nil)
	w := httptest.NewRecorder()
	h.IndiaHandmade(w, req)

	require.NotEqual(t, http.StatusOK, w.Code)
}
