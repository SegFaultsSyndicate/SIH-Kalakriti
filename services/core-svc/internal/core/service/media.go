// services/core-svc/internal/core/service/media.go
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/outbox"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// MediaTx is the transactional surface the media service writes through.
type MediaTx interface {
	outbox.Enqueuer

	CreateMediaPending(ctx context.Context, m domain.Media) (domain.Media, error)
	// TransitionMediaState moves a row from the state the caller believes it is
	// in; a zero-row update is reported as ErrConflict, which is what makes a
	// racing second confirmation a no-op rather than a second event.
	TransitionMediaState(ctx context.Context, id uuid.UUID, from, to domain.MediaState, set domain.MediaTransition) (domain.Media, error)
	SetMediaEnhanced(ctx context.Context, id uuid.UUID, from domain.MediaState, enhancedKey string, modelVersion *string) (domain.Media, error)
	DeleteMedia(ctx context.Context, id uuid.UUID) (bool, error)
}

// MediaStore is the media service's read surface plus its transaction entry point.
type MediaStore interface {
	InTx(ctx context.Context, fn func(ctx context.Context, tx MediaTx) error) error

	GetMedia(ctx context.Context, id uuid.UUID) (domain.Media, error)
	GetMediaByHash(ctx context.Context, artisanID uuid.UUID, sha256Hex string) (domain.Media, error)
	ListStalePendingMedia(ctx context.Context, before time.Time, batchSize int32) ([]domain.Media, error)
}

// ObjectStore is the slice of pkg/storage the media service uses. Bytes never
// pass through this process: it mints URLs, asks whether an object is there, and
// deletes. Uploads and downloads go straight to the bucket.
type ObjectStore interface {
	PresignedPutURL(ctx context.Context, objectKey, contentType string, expiry time.Duration) (string, error)
	PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	Stat(ctx context.Context, objectKey string) (ObjectInfo, error)
	Delete(ctx context.Context, objectKey string) error
}

// ObjectInfo is what a stat tells us about a stored object.
type ObjectInfo struct {
	SizeBytes   int64
	ContentType string
	ETag        string
}

// URLCache holds presigned download URLs just under their own expiry, so a
// listing page with twelve images does not mint twelve signatures per viewer.
type URLCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// urlCacheSkew is how far short of the URL's own expiry the cache entry is set
// to live, so a URL handed out at the last moment is still valid on arrival.
const urlCacheSkew = time.Minute

// bucketName is recorded on every row so a later bucket migration can tell old
// objects from new ones without guessing.
type Media struct {
	store   MediaStore
	objects ObjectStore
	cache   URLCache
	bucket  string
	limits  domain.MediaLimits
	log     *slog.Logger
	now     func() time.Time
}

// NewMedia builds the media service. cache may be nil, in which case every
// download URL is minted fresh.
func NewMedia(
	store MediaStore,
	objects ObjectStore,
	cache URLCache,
	bucket string,
	limits domain.MediaLimits,
	log *slog.Logger,
) *Media {
	return &Media{store: store, objects: objects, cache: cache, bucket: bucket, limits: limits, log: log, now: time.Now}
}

// mediaUploaded mirrors events.v1.MediaUploaded's payload fields.
type mediaUploaded struct {
	MediaID   string  `json:"media_id"`
	ArtisanID string  `json:"artisan_id"`
	ProductID *string `json:"product_id,omitempty"`
	Bucket    string  `json:"bucket"`
	ObjectKey string  `json:"object_key"`
	Kind      string  `json:"kind"`
	MimeType  string  `json:"mime_type"`
	SizeBytes int64   `json:"size_bytes"`
	SHA256Hex *string `json:"sha256_hex,omitempty"`
}

// newMediaEvent builds an outbox envelope for one aggregate.
func (s *Media) newMediaEvent(aggregateID uuid.UUID, idempotencyKey string, payload any) event {
	return event{
		Header: eventHeader{
			EventID:        ids.New().String(),
			OccurredAt:     s.now().UTC(),
			AggregateID:    aggregateID.String(),
			IdempotencyKey: idempotencyKey,
			SchemaVersion:  schemaVersion,
			Producer:       producerName,
		},
		Payload: payload,
	}
}

// --- upload ------------------------------------------------------------------

// RequestUpload mints a presigned PUT URL and records a PENDING row for it. The
// bytes go from the artisan's phone straight to the bucket; this service only
// ever sees the metadata.
//
// When the client supplied a content hash and an identical asset is already
// stored for this artisan, no URL is minted at all: the existing media id comes
// back with Deduplicated set, which is what keeps a flaky rural upload from
// filling the bucket with copies of the same photograph.
func (s *Media) RequestUpload(ctx context.Context, in domain.RequestUploadInput) (domain.UploadTicket, error) {
	kind, extension, err := in.Validate(s.limits)
	if err != nil {
		return domain.UploadTicket{}, err
	}
	if _, err := auth.RequireSelfOrRole(ctx, in.ArtisanID.String(),
		auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return domain.UploadTicket{}, err
	}
	if in.Source == "" {
		in.Source = "artisan-app"
	}

	if in.SHA256Hex != nil {
		existing, err := s.store.GetMediaByHash(ctx, in.ArtisanID, *in.SHA256Hex)
		switch {
		case err == nil:
			s.log.InfoContext(ctx, "upload deduplicated by content hash",
				"artisan_id", in.ArtisanID, "media_id", existing.ID)
			return domain.UploadTicket{
				MediaID:      existing.ID,
				ObjectKey:    existing.ObjectKey,
				Deduplicated: true,
			}, nil
		case errors.Is(err, pkgdomain.ErrNotFound):
			// First time these bytes have been seen; carry on and mint a URL.
		default:
			return domain.UploadTicket{}, fmt.Errorf("checking for an identical upload: %w", err)
		}
	}

	mediaID := ids.New()
	objectKey := domain.ObjectKeyFor(in.ArtisanID, mediaID, extension)

	uploadURL, err := s.objects.PresignedPutURL(ctx, objectKey, in.ContentType, s.limits.UploadURLTTL)
	if err != nil {
		return domain.UploadTicket{}, fmt.Errorf("presigning an upload for %s: %w", objectKey, err)
	}

	row := domain.Media{
		ID:        mediaID,
		ArtisanID: in.ArtisanID,
		ProductID: in.ProductID,
		Bucket:    s.bucket,
		ObjectKey: objectKey,
		Kind:      kind,
		MimeType:  in.ContentType,
		SizeBytes: in.SizeBytes,
		SHA256Hex: in.SHA256Hex,
		Source:    in.Source,
		State:     domain.MediaPending,
	}
	err = s.store.InTx(ctx, func(ctx context.Context, tx MediaTx) error {
		_, err := tx.CreateMediaPending(ctx, row)
		return err
	})
	if err != nil {
		return domain.UploadTicket{}, err
	}

	s.log.InfoContext(ctx, "upload ticket issued",
		"media_id", mediaID, "artisan_id", in.ArtisanID, "kind", kind, "declared_bytes", in.SizeBytes)
	return domain.UploadTicket{
		MediaID:   mediaID,
		ObjectKey: objectKey,
		UploadURL: uploadURL,
		ExpiresAt: s.now().UTC().Add(s.limits.UploadURLTTL),
	}, nil
}

// ConfirmUpload verifies the object actually landed, flips the row to UPLOADED
// and enqueues media.uploaded in the same transaction.
//
// It is idempotent in two independent ways, because the app on a bad connection
// will call it twice: a row that has already left PENDING is returned unchanged
// without an event, and the outbox row is keyed on the media id, so even two
// callers racing through the guarded transition cannot enqueue the fact twice.
func (s *Media) ConfirmUpload(ctx context.Context, mediaID uuid.UUID) (domain.Media, error) {
	if mediaID == uuid.Nil {
		return domain.Media{}, fmt.Errorf("media_id is required: %w", pkgdomain.ErrInvalidInput)
	}

	media, err := s.store.GetMedia(ctx, mediaID)
	if err != nil {
		return domain.Media{}, err
	}
	if _, err := auth.RequireSelfOrRole(ctx, media.ArtisanID.String(),
		auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		return domain.Media{}, err
	}

	switch media.State {
	case domain.MediaPending:
		// The work of this call is below.
	case domain.MediaFailed:
		return domain.Media{}, fmt.Errorf(
			"media %s was rejected and cannot be confirmed: %w", mediaID, pkgdomain.ErrConflict)
	default:
		// Already confirmed by an earlier call: same answer, no second event.
		return media, nil
	}

	info, err := s.objects.Stat(ctx, media.ObjectKey)
	if err != nil {
		return domain.Media{}, fmt.Errorf(
			"no object has been uploaded for media %s: %w", mediaID, pkgdomain.ErrInvalidInput)
	}
	if info.SizeBytes != media.SizeBytes {
		return domain.Media{}, fmt.Errorf(
			"media %s declared %d bytes but the bucket holds %d: %w",
			mediaID, media.SizeBytes, info.SizeBytes, pkgdomain.ErrInvalidInput)
	}

	// One key per media id: a replay of this call, from any caller, collides on
	// the outbox's (topic, idempotency_key) unique constraint and is dropped.
	idempotencyKey := "media.uploaded:" + mediaID.String()

	var confirmed domain.Media
	err = s.store.InTx(ctx, func(ctx context.Context, tx MediaTx) error {
		var err error
		confirmed, err = tx.TransitionMediaState(ctx, mediaID, domain.MediaPending, domain.MediaUploaded,
			domain.MediaTransition{SizeBytes: &info.SizeBytes})
		if err != nil {
			return err
		}

		payload := mediaUploaded{
			MediaID:   confirmed.ID.String(),
			ArtisanID: confirmed.ArtisanID.String(),
			Bucket:    confirmed.Bucket,
			ObjectKey: confirmed.ObjectKey,
			Kind:      string(confirmed.Kind),
			MimeType:  confirmed.MimeType,
			SizeBytes: confirmed.SizeBytes,
			SHA256Hex: confirmed.SHA256Hex,
		}
		if confirmed.ProductID != nil {
			productID := confirmed.ProductID.String()
			payload.ProductID = &productID
		}

		return outbox.Enqueue(ctx, tx,
			ids.New().String(),
			mediaID.String(),
			topics.MediaUploaded,
			idempotencyKey,
			s.newMediaEvent(mediaID, idempotencyKey, payload),
		)
	})
	if err != nil {
		// Another caller won the guarded transition between our read and our
		// write: their result is the answer, and they emitted the one event.
		if errors.Is(err, pkgdomain.ErrConflict) {
			return s.store.GetMedia(ctx, mediaID)
		}
		return domain.Media{}, err
	}

	s.log.InfoContext(ctx, "upload confirmed",
		"media_id", mediaID, "artisan_id", confirmed.ArtisanID, "bytes", confirmed.SizeBytes)
	return confirmed, nil
}

// GetMedia reads one media row.
func (s *Media) GetMedia(ctx context.Context, mediaID uuid.UUID) (domain.Media, error) {
	if mediaID == uuid.Nil {
		return domain.Media{}, fmt.Errorf("media_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if _, err := auth.RequirePrincipal(ctx); err != nil {
		return domain.Media{}, err
	}
	return s.store.GetMedia(ctx, mediaID)
}

// GetMediaURL returns a presigned GET URL for an asset, cached just under its
// own expiry. An asset that is not yet servable is visible only to the artisan
// who uploaded it and to field staff.
func (s *Media) GetMediaURL(ctx context.Context, mediaID uuid.UUID) (string, time.Time, error) {
	if mediaID == uuid.Nil {
		return "", time.Time{}, fmt.Errorf("media_id is required: %w", pkgdomain.ErrInvalidInput)
	}

	media, err := s.store.GetMedia(ctx, mediaID)
	if err != nil {
		return "", time.Time{}, err
	}
	if !media.State.Servable() {
		if _, err := auth.RequireSelfOrRole(ctx, media.ArtisanID.String(),
			auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
			return "", time.Time{}, err
		}
	} else if _, err := auth.RequirePrincipal(ctx); err != nil {
		return "", time.Time{}, err
	}

	objectKey := media.ServableObjectKey()
	cacheKey := mediaID.String() + ":" + objectKey
	expiresAt := s.now().UTC().Add(s.limits.DownloadURLTTL)

	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
			return cached, expiresAt, nil
		}
	}

	url, err := s.objects.PresignedGetURL(ctx, objectKey, s.limits.DownloadURLTTL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("presigning a download for %s: %w", objectKey, err)
	}

	if s.cache != nil {
		// Held just under the signature's own lifetime, so a cached URL is never
		// handed out after it has expired.
		ttl := s.limits.DownloadURLTTL - urlCacheSkew
		if ttl > 0 {
			if err := s.cache.Set(ctx, cacheKey, url, ttl); err != nil {
				s.log.WarnContext(ctx, "caching a media URL", "media_id", mediaID, "error", err)
			}
		}
	}
	return url, expiresAt, nil
}

// --- enhancement ---------------------------------------------------------------

// EnhancementResult is what the media.enhanced consumer hands the service.
type EnhancementResult struct {
	MediaID           uuid.UUID
	EnhancedObjectKey string
	ModelVersion      *string
	FailureReason     *string
	// IdempotencyKey is the incoming event's own key, carried onto anything this
	// handler enqueues so a redelivery cannot fan out twice.
	IdempotencyKey string
}

// ApplyEnhancement records the enhanced rendition and moves the asset to READY,
// promoting it through PROCESSING when the pipeline never told us it had started.
// What happens next to the enhanced image is the pipeline's business, not this
// method's.
func (s *Media) ApplyEnhancement(ctx context.Context, in EnhancementResult) (domain.Media, error) {
	if in.MediaID == uuid.Nil {
		return domain.Media{}, fmt.Errorf("media_id is required: %w", pkgdomain.ErrInvalidInput)
	}
	if in.IdempotencyKey == "" {
		return domain.Media{}, fmt.Errorf("the event carries no idempotency key: %w", pkgdomain.ErrInvalidInput)
	}

	media, err := s.store.GetMedia(ctx, in.MediaID)
	if err != nil {
		return domain.Media{}, err
	}

	if in.FailureReason != nil {
		return s.failMedia(ctx, media, *in.FailureReason)
	}
	if in.EnhancedObjectKey == "" {
		return domain.Media{}, fmt.Errorf("enhanced_object_key is required: %w", pkgdomain.ErrInvalidInput)
	}

	// A redelivery of an event we already applied: nothing to do, and saying so
	// is what lets the consumer commit the offset.
	if media.State == domain.MediaReady &&
		media.EnhancedObjectKey != nil && *media.EnhancedObjectKey == in.EnhancedObjectKey {
		return media, nil
	}
	if media.State == domain.MediaPending {
		return domain.Media{}, fmt.Errorf(
			"media %s has not been confirmed yet: %w", in.MediaID, pkgdomain.ErrConflict)
	}

	var ready domain.Media
	err = s.store.InTx(ctx, func(ctx context.Context, tx MediaTx) error {
		state := media.State
		if state == domain.MediaUploaded {
			// The enhancer finished without ever announcing that it had started.
			promoted, err := tx.TransitionMediaState(ctx, in.MediaID, domain.MediaUploaded, domain.MediaProcessing,
				domain.MediaTransition{})
			if err != nil {
				return err
			}
			state = promoted.State
		}

		var err error
		ready, err = tx.SetMediaEnhanced(ctx, in.MediaID, state, in.EnhancedObjectKey, in.ModelVersion)
		return err
	})
	if err != nil {
		return domain.Media{}, err
	}

	s.log.InfoContext(ctx, "enhancement applied",
		"media_id", in.MediaID, "enhanced_key", in.EnhancedObjectKey)
	return ready, nil
}

// failMedia records why an asset cannot be used, from whichever state it is in.
func (s *Media) failMedia(ctx context.Context, media domain.Media, reason string) (domain.Media, error) {
	if media.State == domain.MediaFailed {
		return media, nil
	}
	if err := domain.ValidateMediaTransition(media.State, domain.MediaFailed); err != nil {
		return domain.Media{}, err
	}

	var failed domain.Media
	err := s.store.InTx(ctx, func(ctx context.Context, tx MediaTx) error {
		var err error
		failed, err = tx.TransitionMediaState(ctx, media.ID, media.State, domain.MediaFailed,
			domain.MediaTransition{FailureReason: &reason})
		return err
	})
	if err != nil {
		return domain.Media{}, err
	}

	s.log.WarnContext(ctx, "media failed", "media_id", media.ID, "reason", reason)
	return failed, nil
}

// --- reaper --------------------------------------------------------------------

// ReapStalePending deletes media rows whose bytes never arrived, and any orphan
// object a half-finished upload left in the bucket. It reports how many rows it
// removed so the ticker can log a number rather than a shrug.
//
// It takes no principal: it is a background sweep, not a caller's request.
func (s *Media) ReapStalePending(ctx context.Context, olderThan time.Duration, batchSize int32) (int, error) {
	cutoff := s.now().UTC().Add(-olderThan)
	stale, err := s.store.ListStalePendingMedia(ctx, cutoff, batchSize)
	if err != nil {
		return 0, fmt.Errorf("listing stale pending media: %w", err)
	}

	var reaped int
	for _, media := range stale {
		// The object usually is not there at all — that is why the row is stale —
		// and pkg/storage treats deleting a missing object as success. A bucket
		// that is unreachable must not cost us the row: it is unreferenced either
		// way, and the bucket's own lifecycle rule sweeps the orphan.
		if err := s.objects.Delete(ctx, media.ObjectKey); err != nil {
			s.log.WarnContext(ctx, "deleting an orphaned object",
				"media_id", media.ID, "object_key", media.ObjectKey, "error", err)
		}

		err := s.store.InTx(ctx, func(ctx context.Context, tx MediaTx) error {
			_, err := tx.DeleteMedia(ctx, media.ID)
			return err
		})
		if err != nil {
			s.log.ErrorContext(ctx, "deleting a stale pending media row",
				"media_id", media.ID, "error", err)
			continue
		}
		reaped++
	}

	if reaped > 0 {
		s.log.InfoContext(ctx, "reaped stale pending media", "count", reaped, "older_than", olderThan)
	}
	return reaped, nil
}

// RunReaper sweeps on a ticker until ctx is cancelled. A failed sweep is logged
// and retried on the next tick rather than ending the loop.
func (s *Media) RunReaper(ctx context.Context, interval, olderThan time.Duration, batchSize int32) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.ReapStalePending(ctx, olderThan, batchSize); err != nil {
				s.log.ErrorContext(ctx, "media reaper sweep failed", "error", err)
			}
		}
	}
}
