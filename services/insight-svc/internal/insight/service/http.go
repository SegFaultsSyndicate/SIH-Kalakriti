//go:build !stub

package service

import (
	"context"
	"io"
	"net/http"
	"time"
)

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

func httpClient() *http.Client {
	return defaultHTTPClient
}

func httpNewRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}
