// services/core-svc/internal/core/domain/media.go
package domain

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
)

// MediaState is where an asset sits between the app asking for an upload URL and
// the buyer being able to see it. Values match the media_state Postgres enum.
type MediaState string

const (
	// MediaPending is a row whose bytes have not arrived yet.
	MediaPending MediaState = "PENDING"
	// MediaUploaded means the object exists in the bucket and matches what was declared.
	MediaUploaded MediaState = "UPLOADED"
	// MediaProcessing means the enhancement pipeline has the asset.
	MediaProcessing MediaState = "PROCESSING"
	// MediaReady means the asset is servable to buyers.
	MediaReady MediaState = "READY"
	// MediaFailed means the asset cannot be used and says why.
	MediaFailed MediaState = "FAILED"
)

// String returns the state's wire and database representation.
func (s MediaState) String() string { return string(s) }

// Valid reports whether s is a known media state.
func (s MediaState) Valid() bool {
	_, ok := mediaTransitions[s]
	return ok
}

// mediaTransitions is the whole media lifecycle, in one place:
//
//	PENDING -> UPLOADED -> PROCESSING -> READY | FAILED
//
// UPLOADED may also fail directly, because an object that turns out to be
// unreadable is discovered before enhancement ever starts. READY may fail too:
// the asset is servable from the moment it is enhanced, but the cataloguing
// pipeline runs on past that point, and an artisan whose photograph produced
// nothing needs to be told why rather than left watching a dead upload.
var mediaTransitions = map[MediaState]map[MediaState]struct{}{
	MediaPending:    {MediaUploaded: {}},
	MediaUploaded:   {MediaProcessing: {}, MediaFailed: {}},
	MediaProcessing: {MediaReady: {}, MediaFailed: {}},
	MediaReady:      {MediaFailed: {}},
	MediaFailed:     {},
}

// CanTransitionTo reports whether next is a legal successor of s.
func (s MediaState) CanTransitionTo(next MediaState) bool {
	_, ok := mediaTransitions[s][next]
	return ok
}

// Servable reports whether an asset in this state may be handed to a buyer.
func (s MediaState) Servable() bool { return s == MediaReady }

// ValidateMediaTransition returns ErrInvalidInput naming the attempted pair when
// the move is not one the lifecycle allows.
func ValidateMediaTransition(from, to MediaState) error {
	if !from.Valid() {
		return fmt.Errorf("unknown media state %q: %w", from, pkgdomain.ErrInvalidInput)
	}
	if !to.Valid() {
		return fmt.Errorf("unknown media state %q: %w", to, pkgdomain.ErrInvalidInput)
	}
	if !from.CanTransitionTo(to) {
		return fmt.Errorf("illegal media state transition %s -> %s: %w", from, to, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// uploadableTypes is the content-type allowlist for artisan uploads, with the
// kind and file extension each maps to. Anything not in this table is refused
// before a presigned URL is minted, because the bucket will happily accept
// whatever bytes a URL is issued for.
var uploadableTypes = map[string]struct {
	kind      MediaKind
	extension string
}{
	"image/jpeg": {MediaImage, ".jpg"},
	"image/png":  {MediaImage, ".png"},
	"image/webp": {MediaImage, ".webp"},
	"video/mp4":  {MediaVideo, ".mp4"},
}

// MediaKindForContentType returns the kind and canonical extension for an
// allowed content type.
func MediaKindForContentType(contentType string) (MediaKind, string, bool) {
	entry, ok := uploadableTypes[strings.ToLower(strings.TrimSpace(contentType))]
	if !ok {
		return "", "", false
	}
	return entry.kind, entry.extension, true
}

// AllowedContentTypes lists what may be uploaded, for an error message that
// tells the caller what to send instead.
func AllowedContentTypes() []string {
	out := make([]string, 0, len(uploadableTypes))
	for ct := range uploadableTypes {
		out = append(out, ct)
	}
	return out
}

// MediaLimits are the size caps and URL lifetimes, injected from config so an
// environment with a slower network can widen the upload window without a
// rebuild.
type MediaLimits struct {
	MaxImageBytes  int64
	MaxVideoBytes  int64
	UploadURLTTL   time.Duration
	DownloadURLTTL time.Duration
}

// MaxBytesFor returns the cap that applies to one kind of asset.
func (l MediaLimits) MaxBytesFor(kind MediaKind) int64 {
	if kind == MediaVideo {
		return l.MaxVideoBytes
	}
	return l.MaxImageBytes
}

// Media is one stored asset.
type Media struct {
	ID                uuid.UUID
	ArtisanID         uuid.UUID
	ProductID         *uuid.UUID
	Bucket            string
	ObjectKey         string
	EnhancedObjectKey *string
	Kind              MediaKind
	MimeType          string
	SizeBytes         int64
	SHA256Hex         *string
	WidthPx           *int32
	HeightPx          *int32
	DurationMs        *int32
	Source            string
	State             MediaState
	ModelVersion      *string
	FailureReason     *string
	UploadedAt        time.Time
	ConfirmedAt       *time.Time
	CreatedAt         time.Time
}

// ServableObjectKey is the object a viewer should be given: the enhanced
// rendition when one exists, and the artisan's original otherwise.
func (m Media) ServableObjectKey() string {
	if m.EnhancedObjectKey != nil && *m.EnhancedObjectKey != "" {
		return *m.EnhancedObjectKey
	}
	return m.ObjectKey
}

// RequestUploadInput is what the service needs to mint an upload ticket.
type RequestUploadInput struct {
	ArtisanID   uuid.UUID
	ContentType string
	SizeBytes   int64
	// SHA256Hex is the client's own hash of the bytes it is about to send. It is
	// optional, but supplying it lets an identical re-upload be deduped without
	// the bytes ever crossing the wire.
	SHA256Hex *string
	ProductID *uuid.UUID
	// Source names the client, e.g. "artisan-app" or "cluster-desk".
	Source string
}

// Validate checks the allowlist and the size cap before any URL is minted.
func (in RequestUploadInput) Validate(limits MediaLimits) (MediaKind, string, error) {
	if in.ArtisanID == uuid.Nil {
		return "", "", fmt.Errorf("artisan_id is required: %w", pkgdomain.ErrInvalidInput)
	}

	kind, extension, ok := MediaKindForContentType(in.ContentType)
	if !ok {
		return "", "", fmt.Errorf("content_type %q is not accepted, use one of %s: %w",
			in.ContentType, strings.Join(AllowedContentTypes(), ", "), pkgdomain.ErrInvalidInput)
	}
	if in.SizeBytes <= 0 {
		return "", "", fmt.Errorf("size_bytes must be positive: %w", pkgdomain.ErrInvalidInput)
	}
	if cap := limits.MaxBytesFor(kind); in.SizeBytes > cap {
		return "", "", fmt.Errorf("size_bytes %d exceeds the %s cap of %d bytes: %w",
			in.SizeBytes, kind, cap, pkgdomain.ErrInvalidInput)
	}
	if in.SHA256Hex != nil && !isSHA256Hex(*in.SHA256Hex) {
		return "", "", fmt.Errorf("sha256_hex must be 64 lowercase hex characters: %w", pkgdomain.ErrInvalidInput)
	}
	return kind, extension, nil
}

// isSHA256Hex mirrors the media_sha256_hex_check constraint.
func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// ObjectKeyFor is where an artisan's asset lives in the bucket. The artisan
// prefix keeps a bucket listing per-maker and makes a lifecycle rule per cluster
// possible later; the id is unguessable, which is what keeps a presigned URL
// from being a directory listing.
func ObjectKeyFor(artisanID, mediaID uuid.UUID, extension string) string {
	return path.Join("artisans", artisanID.String(), mediaID.String()+extension)
}

// UploadTicket is what the app needs to send bytes straight to the bucket. The
// service never sees those bytes.
type UploadTicket struct {
	MediaID   uuid.UUID
	ObjectKey string
	// UploadURL is empty when Deduplicated is true: the bytes are already stored.
	UploadURL string
	ExpiresAt time.Time
	// Deduplicated reports that an identical asset was already uploaded by this
	// artisan, and MediaID names it.
	Deduplicated bool
}

// MediaTransition carries the fields a state change may also record.
type MediaTransition struct {
	SizeBytes     *int64
	SHA256Hex     *string
	FailureReason *string
}
