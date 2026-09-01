// services/search-svc/internal/search/handler/search.go

// Package handler exposes search-svc's gRPC surface and its indexing consumer.
// Protobuf is converted here and nowhere else.
package handler

import (
	"context"
	"io"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	searchv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/search/v1"

	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/domain"
	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/service"
)

// Search implements search.v1.SearchService.
type Search struct {
	searchv1.UnimplementedSearchServiceServer
	svc *service.Search
}

// NewSearch builds the search handler.
func NewSearch(svc *service.Search) *Search { return &Search{svc: svc} }

// Search runs one hybrid query.
func (h *Search) Search(ctx context.Context, req *searchv1.SearchRequest) (*searchv1.SearchResponse, error) {
	filters, err := filtersFromProto(req.GetFilters())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	result, err := h.svc.Do(ctx, service.Request{
		Query:    req.GetQuery(),
		Language: languageName(req.GetLanguage()),
		Filters:  filters,
		Limit:    req.GetPage().GetPageSize(),
		Mode:     req.GetMode().String(),
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return responseToProto(result), nil
}

// Suggest is type-ahead.
func (h *Search) Suggest(ctx context.Context, req *searchv1.SuggestRequest) (*searchv1.SuggestResponse, error) {
	suggestions, err := h.svc.Suggest(ctx, req.GetPrefix(), languageName(req.GetLanguage()), req.GetLimit())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	out := make([]*searchv1.Suggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		item := &searchv1.Suggestion{
			Text:  suggestion.Text,
			Kind:  suggestionKindToProto(suggestion.Kind),
			Score: float32(suggestion.Score),
		}
		if suggestion.EntityID != nil {
			id := suggestion.EntityID.String()
			item.EntityId = &id
		}
		out = append(out, item)
	}
	return &searchv1.SuggestResponse{Suggestions: out}, nil
}

// VoiceSearch takes a spoken query and returns one result set. The audio frames
// are written to the media bucket by the client before the stream opens; this
// server needs only the key, which arrives in the config frame's filters.
//
// ponytail: the client uploads and sends a key rather than streaming bytes
// through search-svc. Streaming audio into a search service would put it on the
// upload path, which batch 7 deliberately kept out of Go.
func (h *Search) VoiceSearch(stream searchv1.SearchService_VoiceSearchServer) error {
	var config *searchv1.VoiceSearchConfig
	var audioKey string

	for {
		frame, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if got := frame.GetConfig(); got != nil {
			config = got
			continue
		}
		// Audio frames carry the object key the client already uploaded.
		if raw := frame.GetAudio(); len(raw) > 0 {
			audioKey = string(raw)
		}
	}
	if config == nil {
		return pkgdomain.GRPCError(pkgdomain.InvalidInput("the first frame must be the stream config"))
	}

	filters, err := filtersFromProto(config.GetFilters())
	if err != nil {
		return pkgdomain.GRPCError(err)
	}

	result, err := h.svc.Do(stream.Context(), service.Request{
		Language:       languageName(config.GetLanguage()),
		Filters:        filters,
		Limit:          config.GetPage().GetPageSize(),
		AudioObjectKey: audioKey,
	})
	if err != nil {
		return pkgdomain.GRPCError(err)
	}

	return stream.SendAndClose(&searchv1.VoiceSearchResponse{
		Transcript:           result.Transcript,
		DetectedLanguage:     languageToProto(result.DetectedLanguage),
		TranscriptConfidence: result.TranscriptConfidence,
		Results:              responseToProto(result),
	})
}

// --- conversion ---------------------------------------------------------------

func responseToProto(result service.Result) *searchv1.SearchResponse {
	hits := make([]*searchv1.SearchHit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		item := &searchv1.SearchHit{
			ListingId:    hit.ListingID.String(),
			ProductId:    hit.ProductID.String(),
			ArtisanId:    hit.ArtisanID.String(),
			Score:        float32(hit.Score),
			MatchedTerms: hit.MatchedTerms,
		}
		if hit.Label != "" {
			label := hit.Label
			item.Explanation = &label
		}
		hits = append(hits, item)
	}

	return &searchv1.SearchResponse{
		Hits:             hits,
		DetectedLanguage: languageToProto(result.DetectedLanguage),
		QueryId:          result.QueryID,
		DidYouMean:       result.DidYouMean,
		Page:             &commonv1.PageResponse{NextPageToken: ""},
	}
}

func filtersFromProto(filters *searchv1.StructuredFilters) (domain.Filters, error) {
	if filters == nil {
		return domain.Filters{}, nil
	}

	out := domain.Filters{
		Colours:    filters.GetColours(),
		Materials:  filters.GetMaterials(),
		GIOnly:     filters.GetGiOnly(),
		SealedOnly: filters.GetProvenanceSealedOnly(),
	}
	for _, raw := range filters.GetCraftIds() {
		id, err := uuid.Parse(raw)
		if err != nil {
			return domain.Filters{}, pkgdomain.InvalidInputf("craft_id %q is not a uuid", raw)
		}
		out.CraftIDs = append(out.CraftIDs, id)
	}
	if price := filters.GetMinPrice(); price != nil {
		amount := price.GetAmountPaise()
		out.MinPricePaise = &amount
	}
	if price := filters.GetMaxPrice(); price != nil {
		amount := price.GetAmountPaise()
		out.MaxPricePaise = &amount
	}
	if region := filters.GetRegion(); region != nil && region.GetStateCode() != "" {
		state := region.GetStateCode()
		out.StateCode = &state
	}
	if filters.GetListingType() != catalogv1.ListingType_LISTING_TYPE_UNSPECIFIED {
		listingType := trimEnumPrefix(filters.GetListingType().String(), "LISTING_TYPE_")
		out.ListingType = &listingType
	}
	if days := filters.GetMaxLeadTimeDays(); days > 0 {
		out.MaxLeadTimeDays = &days
	}
	return out, nil
}

func suggestionKindToProto(kind domain.SuggestionKind) searchv1.SuggestionKind {
	if v, ok := searchv1.SuggestionKind_value["SUGGESTION_KIND_"+string(kind)]; ok {
		return searchv1.SuggestionKind(v)
	}
	return searchv1.SuggestionKind_SUGGESTION_KIND_UNSPECIFIED
}

func languageName(language commonv1.Language) string {
	if language == commonv1.Language_LANGUAGE_UNSPECIFIED {
		return ""
	}
	return trimEnumPrefix(language.String(), "LANGUAGE_")
}

func languageToProto(name string) commonv1.Language {
	if v, ok := commonv1.Language_value["LANGUAGE_"+name]; ok {
		return commonv1.Language(v)
	}
	return commonv1.Language_LANGUAGE_UNSPECIFIED
}

func trimEnumPrefix(value, prefix string) string {
	if len(value) > len(prefix) && value[:len(prefix)] == prefix {
		return value[len(prefix):]
	}
	return value
}
