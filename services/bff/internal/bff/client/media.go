// services/bff/internal/bff/client/media.go
package client

import (
	"google.golang.org/grpc"

	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
)

// Media is bff's view of core-svc's upload flow.
type Media struct {
	media catalogv1.MediaServiceClient
}

// NewMedia builds the media client, sharing conn with core-svc's other services.
func NewMedia(conn grpc.ClientConnInterface) *Media {
	return &Media{media: catalogv1.NewMediaServiceClient(conn)}
}

// GenerateUploadURL asks for a presigned PUT URL for a new asset.
func (m *Media) GenerateUploadURL(artisanID, contentType string, sizeBytes int64) (mediaID, uploadURL string, err error) {
	ctx, cancel := withTimeout()
	defer cancel()

	resp, err := m.media.RequestUpload(ctx, &catalogv1.RequestUploadRequest{
		ArtisanId:   artisanID,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		Source:      "bff",
	})
	if err != nil {
		return "", "", grpcErr(err)
	}
	return resp.GetMediaId(), resp.GetUploadUrl(), nil
}

// ConfirmUpload tells core-svc the bytes have landed.
func (m *Media) ConfirmUpload(mediaID string) error {
	ctx, cancel := withTimeout()
	defer cancel()

	if _, err := m.media.ConfirmUpload(ctx, &catalogv1.ConfirmUploadRequest{MediaId: mediaID}); err != nil {
		return grpcErr(err)
	}
	return nil
}
