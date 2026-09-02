// services/bff/internal/bff/client/search.go
package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	searchv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/search/v1"
)

// defaultPageSize bounds an unpaginated buyer search; handler.SearchService's
// signature has no page/cursor parameter to carry a caller's choice.
const defaultPageSize = 20

// Search is bff's view of search-svc's buyer-facing discovery RPCs.
type Search struct {
	search searchv1.SearchServiceClient
}

// NewSearch builds the search client.
func NewSearch(conn grpc.ClientConnInterface) *Search {
	return &Search{search: searchv1.NewSearchServiceClient(conn)}
}

// structuredFilters builds search-svc's filter message from the loose
// craft_id/region query params the HTTP handler collects.
func structuredFilters(filters map[string]any) *searchv1.StructuredFilters {
	sf := &searchv1.StructuredFilters{}
	if craftID, ok := filters["craft_id"].(string); ok && craftID != "" {
		sf.CraftIds = []string{craftID}
	}
	if region, ok := filters["region"].(string); ok && region != "" {
		// ponytail: treated as a bare state code; the ?region= query param has
		// no documented shape beyond that. Extend if a caller ever needs
		// district/block-level narrowing.
		sf.Region = &commonv1.GeoRegion{StateCode: region}
	}
	return sf
}

func hitsToMaps(hits []*searchv1.SearchHit) []map[string]any {
	out := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		out = append(out, map[string]any{
			"listing_id":    h.GetListingId(),
			"product_id":    h.GetProductId(),
			"artisan_id":    h.GetArtisanId(),
			"score":         h.GetScore(),
			"matched_terms": h.GetMatchedTerms(),
			"explanation":   h.GetExplanation(),
		})
	}
	return out
}

// Search runs a hybrid lexical/semantic query.
func (s *Search) Search(ctx context.Context, query string, filters map[string]any) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := s.search.Search(ctx, &searchv1.SearchRequest{
		Query:   query,
		Filters: structuredFilters(filters),
		Page:    &commonv1.PageRequest{PageSize: defaultPageSize},
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	return hitsToMaps(resp.GetHits()), nil
}

// Suggest returns type-ahead completions for a partial query.
func (s *Search) Suggest(ctx context.Context, prefix string) ([]string, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := s.search.Suggest(ctx, &searchv1.SuggestRequest{Prefix: prefix})
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]string, 0, len(resp.GetSuggestions()))
	for _, sug := range resp.GetSuggestions() {
		out = append(out, sug.GetText())
	}
	return out, nil
}

// SearchVoice streams one spoken query to search-svc and returns the
// transcript and its results. handler.SearchService's signature takes the
// whole audio blob at once (the HTTP handler already drained the request
// body), so this opens the RPC's client stream, sends it as a single config
// frame plus a single audio frame, and waits for the one response.
func (s *Search) SearchVoice(ctx context.Context, audioData []byte, language string) (query string, results []map[string]any, err error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	stream, err := s.search.VoiceSearch(ctx)
	if err != nil {
		return "", nil, grpcErr(err)
	}

	// ponytail: 16kHz/LINEAR16 assumed rather than inspected, since the HTTP
	// handler only drains the body into raw bytes and never reads a format
	// header. Read the client's actual capture format here if a real client
	// ever sends something else.
	if err := stream.Send(&searchv1.VoiceSearchRequest{
		Frame: &searchv1.VoiceSearchRequest_Config{
			Config: &searchv1.VoiceSearchConfig{
				Language:     languageToProto(language),
				SampleRateHz: 16000,
				Encoding:     "LINEAR16",
				Page:         &commonv1.PageRequest{PageSize: defaultPageSize},
			},
		},
	}); err != nil {
		return "", nil, fmt.Errorf("sending voice search config: %w", err)
	}

	if err := stream.Send(&searchv1.VoiceSearchRequest{
		Frame: &searchv1.VoiceSearchRequest_Audio{Audio: audioData},
	}); err != nil {
		return "", nil, fmt.Errorf("sending voice search audio: %w", err)
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", nil, grpcErr(err)
	}
	return resp.GetTranscript(), hitsToMaps(resp.GetResults().GetHits()), nil
}

// languageToProto converts a BCP-47-ish language name (as the HTTP layer
// receives it, e.g. from Accept-Language) to the wire enum. Anything it
// doesn't recognise comes back UNSPECIFIED, which asks the server to detect it.
func languageToProto(name string) commonv1.Language {
	if v, ok := commonv1.Language_value["LANGUAGE_"+name]; ok {
		return commonv1.Language(v)
	}
	return commonv1.Language_LANGUAGE_UNSPECIFIED
}
