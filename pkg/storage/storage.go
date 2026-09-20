// pkg/storage/storage.go
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrContentTypeNotAllowed is returned when a caller requests an upload URL
// for a MIME type outside the configured allowlist.
var ErrContentTypeNotAllowed = errors.New("storage: content type not allowed")

// allowedContentTypes is the set of MIME types Kalakriti accepts for
// artisan-uploaded media (product photography and short process videos) and
// server-generated documents presigned for upload the same way (insight-svc's
// income-statement PDFs). Keep in sync with bff's own allowedMIMETypes
// (services/bff/internal/bff/handler/api.go) -- that one gates the
// artisan-facing upload flow first, this one gates every PresignedPutURL
// caller, including insight-svc's hardcoded "application/pdf", which had no
// audio types and was missing "application/pdf" itself until this pass:
// insight-svc's own PDF presigning had never worked, and neither had a
// voice-note upload from any browser, since neither type was allowed here
// at all.
var allowedContentTypes = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"image/webp":      {},
	"image/heic":      {},
	"video/mp4":       {},
	"video/quicktime": {},
	"video/webm":      {},
	"application/pdf": {},
	"audio/webm":      {},
	"audio/ogg":       {},
	"audio/mp4":       {},
	"audio/mpeg":      {},
	"audio/wav":       {},
}

// IsAllowedContentType reports whether contentType may be uploaded to
// object storage.
func IsAllowedContentType(contentType string) bool {
	_, ok := allowedContentTypes[contentType]
	return ok
}

// Config holds MinIO/S3 connection settings.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
	// PublicURL, when set, overrides Endpoint as the host used to build
	// presigned URLs returned to browser/mobile clients (useful when the
	// server reaches MinIO over a different address than the public one,
	// e.g. a docker-compose service name vs. a host-mapped port).
	PublicURL string
}

// Client wraps a MinIO client scoped to one bucket.
type Client struct {
	mc     *minio.Client
	bucket string
	// presign is a second client, identical except for its Endpoint, used
	// only to build presigned URLs. Non-nil when PublicURL differs from
	// Endpoint -- see New's doc comment for why this can't just be one
	// client with the host swapped after the fact.
	presign *minio.Client
}

// New builds a Client and verifies the configured bucket exists.
//
// A presigned URL's signature covers its Host header (AWS SigV4 with
// SignedHeaders=host), so it can only ever be valid for the exact host it
// was computed against. When the server reaches MinIO over one address
// (e.g. a docker-compose service name) but PublicURL names a different one
// a browser can resolve, swapping the URL's host string after presigning --
// the previous approach -- produces a URL whose signature no longer matches
// the Host the browser will actually send, and MinIO rejects the PUT with
// SignatureDoesNotMatch. Confirmed live: every real upload failed this way
// once anything (browser, curl) actually followed through on a presigned
// URL, which nothing had before this was caught. The fix is a second
// *minio.Client constructed with PublicURL as its Endpoint, used only for
// presigning -- minio-go computes a presigned URL by pure local signing, no
// network round trip to that endpoint, so this needs no connectivity to the
// public address from inside the container.
func New(ctx context.Context, cfg Config) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("constructing minio client for %s: %w", cfg.Endpoint, err)
	}

	presign := mc
	if cfg.PublicURL != "" {
		publicHost, err := url.Parse(cfg.PublicURL)
		if err != nil {
			return nil, fmt.Errorf("parsing storage public URL %q: %w", cfg.PublicURL, err)
		}
		presign, err = minio.New(publicHost.Host, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: publicHost.Scheme == "https",
			Region: cfg.Region,
		})
		if err != nil {
			return nil, fmt.Errorf("constructing minio presign client for %s: %w", publicHost.Host, err)
		}
	}

	exists, err := mc.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("checking bucket %s exists: %w", cfg.Bucket, err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %s does not exist", cfg.Bucket)
	}

	return &Client{mc: mc, bucket: cfg.Bucket, presign: presign}, nil
}

// PresignedPutURL returns a time-limited URL the caller can PUT an object's
// bytes to directly, bypassing the application server. contentType must be
// on the configured allowlist.
func (c *Client) PresignedPutURL(ctx context.Context, objectKey, contentType string, expiry time.Duration) (string, error) {
	if !IsAllowedContentType(contentType) {
		return "", fmt.Errorf("presigning PUT for %s (content-type %s): %w", objectKey, contentType, ErrContentTypeNotAllowed)
	}
	u, err := c.presign.PresignedPutObject(ctx, c.bucket, objectKey, expiry)
	if err != nil {
		return "", fmt.Errorf("presigning PUT for %s: %w", objectKey, err)
	}
	return u.String(), nil
}

// PresignedGetURL returns a time-limited URL the caller can GET an object's
// bytes from directly.
func (c *Client) PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	u, err := c.presign.PresignedGetObject(ctx, c.bucket, objectKey, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presigning GET for %s: %w", objectKey, err)
	}
	return u.String(), nil
}

// ObjectInfo describes a stored object's basic metadata.
type ObjectInfo struct {
	Key          string
	SizeBytes    int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

// Stat returns metadata for an existing object.
func (c *Client) Stat(ctx context.Context, objectKey string) (ObjectInfo, error) {
	info, err := c.mc.StatObject(ctx, c.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("stat object %s: %w", objectKey, err)
	}
	return ObjectInfo{
		Key:          objectKey,
		SizeBytes:    info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified,
	}, nil
}

// Delete removes an object. Deleting a missing object is not an error, to
// keep callers idempotent.
func (c *Client) Delete(ctx context.Context, objectKey string) error {
	if err := c.mc.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("deleting object %s: %w", objectKey, err)
	}
	return nil
}
