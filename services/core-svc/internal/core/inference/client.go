// services/core-svc/internal/core/inference/client.go

// Package inference is core-svc's client for ml-svc. It converts protobuf to
// domain types so the pipeline never sees a generated message, which is what
// lets the pipeline be tested without a model or a connection.
package inference

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	inferencev1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/inference/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// Client wraps the ml-svc stub.
type Client struct {
	stub   inferencev1.InferenceServiceClient
	bucket string
}

// New builds a client over an existing connection.
func New(conn grpc.ClientConnInterface, bucket string) *Client {
	return &Client{stub: inferencev1.NewInferenceServiceClient(conn), bucket: bucket}
}

// ref is the media reference ml-svc needs: it reads the object itself, so the
// key and the bucket are the whole message.
func (c *Client) ref(objectKey string) *commonv1.MediaRef {
	return &commonv1.MediaRef{Bucket: c.bucket, ObjectKey: objectKey, Kind: commonv1.MediaKind_MEDIA_KIND_IMAGE}
}

// EnhanceImage returns the object key of the enhanced rendition.
func (c *Client) EnhanceImage(ctx context.Context, objectKey string) (string, error) {
	resp, err := c.stub.EnhanceImage(ctx, &inferencev1.EnhanceImageRequest{
		Source:           c.ref(objectKey),
		RemoveBackground: true,
		AutoWhiteBalance: true,
	})
	if err != nil {
		return "", fmt.Errorf("enhancing %s: %w", objectKey, err)
	}
	return resp.GetEnhanced().GetObjectKey(), nil
}

// ExtractAttributes reads a product's attributes out of its imagery.
func (c *Client) ExtractAttributes(
	ctx context.Context,
	objectKeys []string,
	declaredCraftID, hint string,
) (domain.InferredAttributes, error) {
	media := make([]*commonv1.MediaRef, 0, len(objectKeys))
	for _, key := range objectKeys {
		media = append(media, c.ref(key))
	}

	req := &inferencev1.ExtractAttributesRequest{Media: media}
	if declaredCraftID != "" {
		req.DeclaredCraftId = &declaredCraftID
	}
	if hint != "" {
		req.ArtisanHint = &hint
	}

	resp, err := c.stub.ExtractAttributes(ctx, req)
	if err != nil {
		return domain.InferredAttributes{}, fmt.Errorf("extracting attributes: %w", err)
	}

	set := resp.GetAttributes()
	return domain.InferredAttributes{
		CraftCode:       set.GetCraftId().GetValue(),
		CraftConfidence: set.GetCraftId().GetConfidence(),
		Material:        set.GetMaterial().GetValue(),
		Technique:       set.GetTechnique().GetValue(),
		Colours:         set.GetColours().GetValues(),
		Motifs:          set.GetMotifs().GetValues(),
		Confidence: map[string]float32{
			"craft":     set.GetCraftId().GetConfidence(),
			"material":  set.GetMaterial().GetConfidence(),
			"technique": set.GetTechnique().GetConfidence(),
			"colour":    set.GetColours().GetConfidence(),
			"motif":     set.GetMotifs().GetConfidence(),
		},
		ModelVersion: set.GetModelVersion(),
	}, nil
}

// GenerateDescription writes buyer-facing copy from an attribute set.
func (c *Client) GenerateDescription(ctx context.Context, in domain.CopyRequest) (domain.GeneratedCopy, error) {
	req := &inferencev1.GenerateDescriptionRequest{
		Attributes: attributesToProto(in.Attributes),
		CraftId:    in.CraftID.String(),
		Language:   languageToProto(in.Language),
	}
	if in.ArtisanNote != "" {
		req.ArtisanNote = &in.ArtisanNote
	}
	if in.MaxChars > 0 {
		req.MaxChars = &in.MaxChars
	}

	resp, err := c.stub.GenerateDescription(ctx, req)
	if err != nil {
		return domain.GeneratedCopy{}, fmt.Errorf("generating copy in %s: %w", in.Language, err)
	}
	return domain.GeneratedCopy{
		Title:             resp.GetTitle(),
		Description:       resp.GetDescription(),
		Highlights:        resp.GetHighlights(),
		Keywords:          resp.GetKeywords(),
		AttributeKeysUsed: resp.GetAttributeKeysUsed(),
		ModelVersion:      resp.GetModelVersion(),
	}, nil
}

func attributesToProto(a domain.InferredAttributes) *inferencev1.AttributeSet {
	return &inferencev1.AttributeSet{
		CraftId:      &inferencev1.ScoredString{Value: a.CraftCode, Confidence: a.CraftConfidence},
		Material:     &inferencev1.ScoredString{Value: a.Material, Confidence: a.Confidence["material"]},
		Technique:    &inferencev1.ScoredString{Value: a.Technique, Confidence: a.Confidence["technique"]},
		Colours:      &inferencev1.ScoredStrings{Values: a.Colours, Confidence: a.Confidence["colour"]},
		Motifs:       &inferencev1.ScoredStrings{Values: a.Motifs, Confidence: a.Confidence["motif"]},
		ModelVersion: a.ModelVersion,
	}
}

// languageToProto converts a stored language name to its wire enum, defaulting
// to unspecified so ml-svc picks rather than failing.
func languageToProto(name string) commonv1.Language {
	if v, ok := commonv1.Language_value["LANGUAGE_"+name]; ok {
		return commonv1.Language(v)
	}
	return commonv1.Language_LANGUAGE_UNSPECIFIED
}
