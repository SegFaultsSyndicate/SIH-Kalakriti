// services/search-svc/internal/search/service/codec.go
package service

import (
	"encoding/json"
	"fmt"

	"github.com/segfaultsyndicate/kalakriti/services/search-svc/internal/search/domain"
)

// cachedResult is the on-the-wire form of a cached page. Only what the buyer
// surface renders is stored: the understood query is rebuilt cheaply and the
// transcript belongs to the request, not the page.
type cachedResult struct {
	Hits       []domain.Hit `json:"hits"`
	DidYouMean []string     `json:"did_you_mean,omitempty"`
	Language   string       `json:"language"`
	QueryID    string       `json:"query_id"`
}

func encodeResult(result Result) ([]byte, error) {
	raw, err := json.Marshal(cachedResult{
		Hits:       result.Hits,
		DidYouMean: result.DidYouMean,
		Language:   result.DetectedLanguage,
		QueryID:    result.QueryID,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding a search page: %w", err)
	}
	return raw, nil
}

func decodeResult(raw []byte) (Result, error) {
	var cached cachedResult
	if err := json.Unmarshal(raw, &cached); err != nil {
		return Result{}, fmt.Errorf("decoding a search page: %w", err)
	}
	return Result{
		Hits:             cached.Hits,
		DidYouMean:       cached.DidYouMean,
		DetectedLanguage: cached.Language,
		QueryID:          cached.QueryID,
	}, nil
}
