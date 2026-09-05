// services/bff/internal/bff/client/search.go
package client

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc"

	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
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

// structuredFilters builds search-svc's filter message from the loose query
// params the HTTP handler collects. Every field here was already applied
// end-to-end in search-svc's SQL (services/search-svc/internal/search/repo/repo.go)
// before this batch; only craft_id/region ever reached it from the BFF, so
// colour/material/price/gi/sealed/listing_type facets silently did nothing.
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
	if v, ok := filters["colours"].([]string); ok {
		sf.Colours = v
	}
	if v, ok := filters["materials"].([]string); ok {
		sf.Materials = v
	}
	if v, ok := filters["gi_only"].(bool); ok {
		sf.GiOnly = v
	}
	if v, ok := filters["sealed_only"].(bool); ok {
		sf.ProvenanceSealedOnly = v
	}
	if v, ok := filters["listing_type"].(string); ok && v != "" {
		if enum, ok := catalogv1.ListingType_value["LISTING_TYPE_"+v]; ok {
			sf.ListingType = catalogv1.ListingType(enum)
		}
	}
	if v, ok := filters["min_price_paise"].(string); ok && v != "" {
		if amount, err := strconv.ParseInt(v, 10, 64); err == nil {
			sf.MinPrice = &commonv1.Money{AmountPaise: amount, CurrencyCode: "INR"}
		}
	}
	if v, ok := filters["max_price_paise"].(string); ok && v != "" {
		if amount, err := strconv.ParseInt(v, 10, 64); err == nil {
			sf.MaxPrice = &commonv1.Money{AmountPaise: amount, CurrencyCode: "INR"}
		}
	}
	return sf
}

func hitsToMaps(hits []*searchv1.SearchHit) []map[string]any {
	out := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		out = append(out, map[string]any{
			"listing_id":        h.GetListingId(),
			"product_id":        h.GetProductId(),
			"artisan_id":        h.GetArtisanId(),
			"score":             h.GetScore(),
			"matched_terms":     h.GetMatchedTerms(),
			"explanation":       h.GetExplanation(),
			"machine_generated": h.GetMachineGenerated(),
		})
	}
	return out
}

func craftSpansToMaps(spans []*searchv1.CraftSpan) []map[string]any {
	out := make([]map[string]any, 0, len(spans))
	for _, s := range spans {
		out = append(out, map[string]any{
			"craft_id":     s.GetCraftId(),
			"display_name": s.GetDisplayName(),
			"text":         s.GetText(),
		})
	}
	return out
}

// understoodToMap renders the query understander's parsed filters as a plain
// map the buyer client can turn into removable chips. nil when nothing was
// understood, so "no filters parsed" is distinguishable from "every filter
// happens to be its zero value".
func understoodToMap(sf *searchv1.StructuredFilters) map[string]any {
	if sf == nil {
		return nil
	}
	out := map[string]any{
		"craft_ids":              sf.GetCraftIds(),
		"colours":                sf.GetColours(),
		"materials":              sf.GetMaterials(),
		"gi_only":                sf.GetGiOnly(),
		"provenance_sealed_only": sf.GetProvenanceSealedOnly(),
	}
	if sf.GetListingType() != catalogv1.ListingType_LISTING_TYPE_UNSPECIFIED {
		out["listing_type"] = trimEnumPrefix(sf.GetListingType().String(), "LISTING_TYPE_")
	}
	if sf.GetMinPrice() != nil {
		out["min_price"] = moneyMap(sf.GetMinPrice())
	}
	if sf.GetMaxPrice() != nil {
		out["max_price"] = moneyMap(sf.GetMaxPrice())
	}
	if sf.GetRegion() != nil && sf.GetRegion().GetStateCode() != "" {
		out["region"] = sf.GetRegion().GetStateCode()
	}
	return out
}

func responseToMap(resp *searchv1.SearchResponse) map[string]any {
	return map[string]any{
		"results":           hitsToMaps(resp.GetHits()),
		"understood":        understoodToMap(resp.GetUnderstoodFilters()),
		"craft_spans":       craftSpansToMaps(resp.GetCraftSpans()),
		"detected_language": languageToString(resp.GetDetectedLanguage()),
		"did_you_mean":      resp.GetDidYouMean(),
		"query_id":          resp.GetQueryId(),
	}
}

// searchPageSize reads an optional bounded "limit" out of the loose filter
// map -- there is no real cursor from search-svc (SearchResponse.page's
// NextPageToken is always ""), so "load more" on the buyer app re-asks for a
// bigger page rather than paging a cursor that doesn't exist.
func searchPageSize(filters map[string]any) int32 {
	v, ok := filters["limit"].(string)
	if !ok || v == "" {
		return defaultPageSize
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultPageSize
	}
	if n > 100 {
		return 100
	}
	return int32(n)
}

// Search runs a hybrid lexical/semantic query.
func (s *Search) Search(ctx context.Context, query string, filters map[string]any) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := s.search.Search(ctx, &searchv1.SearchRequest{
		Query:   query,
		Filters: structuredFilters(filters),
		Page:    &commonv1.PageRequest{PageSize: searchPageSize(filters)},
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	return responseToMap(resp), nil
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
func (s *Search) SearchVoice(ctx context.Context, audioData []byte, language string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	stream, err := s.search.VoiceSearch(ctx)
	if err != nil {
		return nil, grpcErr(err)
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
		return nil, fmt.Errorf("sending voice search config: %w", err)
	}

	if err := stream.Send(&searchv1.VoiceSearchRequest{
		Frame: &searchv1.VoiceSearchRequest_Audio{Audio: audioData},
	}); err != nil {
		return nil, fmt.Errorf("sending voice search audio: %w", err)
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, grpcErr(err)
	}
	out := responseToMap(resp.GetResults())
	out["query"] = resp.GetTranscript()
	out["detected_language"] = languageToString(resp.GetDetectedLanguage())
	out["transcript_confidence"] = resp.GetTranscriptConfidence()
	return out, nil
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
