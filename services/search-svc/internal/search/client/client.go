// services/search-svc/internal/search/client/client.go

// Package client holds search-svc's outbound gRPC and Redis clients: ml-svc for
// vectors, reranking and transcription, core-svc for entity linking, and the
// transliteration cache that keeps a hot path off a rate-limited API.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	inferencev1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/inference/v1"
	pkgredis "github.com/ZoroNewbie00/kalakriti/pkg/redis"

	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/domain"
)

// Inference is search-svc's view of ml-svc: three RPCs, no model types.
type Inference struct {
	stub   inferencev1.InferenceServiceClient
	bucket string
}

// NewInference builds the ml-svc client.
func NewInference(conn grpc.ClientConnInterface, bucket string) *Inference {
	return &Inference{stub: inferencev1.NewInferenceServiceClient(conn), bucket: bucket}
}

// Embed vectorises query or document text. The vectors come back unit-normalised,
// which is what the cosine index assumes.
func (c *Inference) Embed(ctx context.Context, texts []string, language string) ([][]float32, error) {
	resp, err := c.stub.Embed(ctx, &inferencev1.EmbedRequest{
		Texts:    texts,
		Language: languageToProto(language),
	})
	if err != nil {
		return nil, fmt.Errorf("embedding %d texts: %w", len(texts), err)
	}

	out := make([][]float32, 0, len(resp.GetEmbeddings()))
	for _, embedding := range resp.GetEmbeddings() {
		out = append(out, embedding.GetValues())
	}
	return out, nil
}

// Rerank scores query-listing pairs with the cross-encoder, returning a score
// per listing rather than an order: the caller owns the ordering.
func (c *Inference) Rerank(
	ctx context.Context,
	query string,
	candidates []domain.RerankCandidate,
) (map[uuid.UUID]float64, error) {
	request := &inferencev1.RerankRequest{Query: query}
	for _, candidate := range candidates {
		request.Candidates = append(request.Candidates, &inferencev1.RerankCandidate{
			Id: candidate.ListingID.String(), Text: candidate.Text,
		})
	}

	resp, err := c.stub.Rerank(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("reranking %d candidates: %w", len(candidates), err)
	}

	out := make(map[uuid.UUID]float64, len(resp.GetResults()))
	for _, result := range resp.GetResults() {
		id, err := uuid.Parse(result.GetId())
		if err != nil {
			continue // a score we cannot attribute is a score we ignore
		}
		out[id] = float64(result.GetScore())
	}
	return out, nil
}

// Transcribe drains the transcript stream and returns the final text. Partial
// hypotheses are for a live microphone; a search needs the settled sentence.
func (c *Inference) Transcribe(ctx context.Context, objectKey, language string) (string, float32, error) {
	stream, err := c.stub.Transcribe(ctx, &inferencev1.TranscribeRequest{
		Audio:          &commonv1.MediaRef{Bucket: c.bucket, ObjectKey: objectKey, Kind: commonv1.MediaKind_MEDIA_KIND_AUDIO},
		Language:       languageToProto(language),
		InterimResults: false,
	})
	if err != nil {
		return "", 0, fmt.Errorf("opening the transcript stream: %w", err)
	}

	var text string
	var confidence float32
	var chunks int
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		if !chunk.GetIsFinal() {
			continue
		}
		if text != "" {
			text += " "
		}
		text += chunk.GetText()
		confidence += chunk.GetConfidence()
		chunks++
	}
	if chunks > 0 {
		confidence /= float32(chunks)
	}
	return text, confidence, nil
}

// Ontology is core-svc's entity linker.
type Ontology struct {
	stub catalogv1.OntologyServiceClient
}

// NewOntology builds the core-svc ontology client.
func NewOntology(conn grpc.ClientConnInterface) *Ontology {
	return &Ontology{stub: catalogv1.NewOntologyServiceClient(conn)}
}

// ResolveAlias links craft mentions in query text, with the span of each.
func (c *Ontology) ResolveAlias(ctx context.Context, text, language string) ([]domain.Span, error) {
	resp, err := c.stub.ResolveCraftAlias(ctx, &catalogv1.ResolveCraftAliasRequest{
		Text: text, Language: languageToProto(language),
	})
	if err != nil {
		return nil, fmt.Errorf("resolving craft aliases: %w", err)
	}

	out := make([]domain.Span, 0, len(resp.GetMatches()))
	for _, match := range resp.GetMatches() {
		craftID, err := uuid.Parse(match.GetCraftId())
		if err != nil {
			continue
		}
		out = append(out, domain.Span{
			CraftID: craftID, CraftCode: match.GetCraftCode(),
			DisplayName: match.GetDisplayName(), Text: match.GetMatchedText(),
			Start: int(match.GetStartOffset()), End: int(match.GetEndOffset()),
		})
	}
	return out, nil
}

// translitTTL is how long a canonical spelling is kept. Transliteration of a
// craft name does not change, so this is long; it is bounded only so a corrected
// mapping eventually takes effect.
const translitTTL = 30 * 24 * time.Hour

// Transliterator canonicalises a token to one spelling, whatever script it was
// typed in, cached in Redis on the raw token.
//
// The ontology is the transliteration table: it already holds every craft's
// spelling in every script, so resolving a token through it maps "ajrakh",
// "Ajarakh" and "अजरख" onto one craft code. That is the only vocabulary a search
// query needs canonicalised, and it costs no external API at all.
//
// ponytail: ontology lookup rather than a Bhashini round trip. Swap in the ASR
// vendor's transliteration endpoint here when queries start arriving with words
// the ontology has never seen; the cache and the interface do not change.
type Transliterator struct {
	ontology *Ontology
	cache    *pkgredis.Cache[string]
}

// NewTransliterator builds the cached canonicaliser.
func NewTransliterator(ontology *Ontology, client redis.Cmdable) *Transliterator {
	return &Transliterator{ontology: ontology, cache: pkgredis.NewCache[string](client, "translit")}
}

// Canonicalise returns the spelling the index holds for a token. A token the
// ontology cannot place comes back unchanged, and every outcome is cached:
// the misses are the hot path, because most query words are not craft names.
func (t *Transliterator) Canonicalise(ctx context.Context, token, language string) (string, error) {
	key := language + ":" + token
	if cached, err := t.cache.Get(ctx, key); err == nil && cached != "" {
		return cached, nil
	}

	canonical := token
	spans, err := t.ontology.ResolveAlias(ctx, token, language)
	if err != nil {
		// Never fatal: the caller uses the token as typed.
		return token, nil
	}
	// Only a span covering the whole token is a canonicalisation; a partial match
	// is the entity linker's business, not this one's.
	if len(spans) == 1 && spans[0].Start == 0 && spans[0].End == len(token) {
		canonical = spans[0].CraftCode
	}

	if err := t.cache.Set(ctx, key, canonical, translitTTL); err != nil {
		return canonical, nil
	}
	return canonical, nil
}

// languageToProto converts a stored language name to its wire enum.
func languageToProto(name string) commonv1.Language {
	if v, ok := commonv1.Language_value["LANGUAGE_"+name]; ok {
		return commonv1.Language(v)
	}
	return commonv1.Language_LANGUAGE_UNSPECIFIED
}
