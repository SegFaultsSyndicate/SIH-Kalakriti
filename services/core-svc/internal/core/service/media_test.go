// services/core-svc/internal/core/service/media_test.go
package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/topics"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// testMediaLimits are the caps from .env.example: 15 MB of image, 100 MB of video.
func testMediaLimits() domain.MediaLimits {
	return domain.MediaLimits{
		MaxImageBytes:  15 << 20,
		MaxVideoBytes:  100 << 20,
		UploadURLTTL:   15 * time.Minute,
		DownloadURLTTL: time.Hour,
	}
}

type mediaFixture struct {
	svc     *Media
	store   *fakeMediaStore
	objects *fakeObjectStore
	cache   *fakeURLCache
	artisan uuid.UUID
}

func newMediaFixture(t *testing.T) mediaFixture {
	t.Helper()
	f := mediaFixture{
		store:   newFakeMediaStore(),
		objects: newFakeObjectStore(),
		cache:   newFakeURLCache(),
		artisan: ids.New(),
	}
	f.svc = NewMedia(f.store, f.objects, f.cache, "kalakriti-media", testMediaLimits(),
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	f.svc.now = func() time.Time { return fakeNow }
	return f
}

// upload runs the whole happy path: ticket, PUT to the bucket, confirm.
func (f mediaFixture) upload(t *testing.T, contentType string, size int64) domain.Media {
	t.Helper()
	ticket, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
		ArtisanID:   f.artisan,
		ContentType: contentType,
		SizeBytes:   size,
	})
	require.NoError(t, err)
	f.objects.put(ticket.ObjectKey, size)

	media, err := f.svc.ConfirmUpload(artisanCtx(f.artisan), ticket.MediaID)
	require.NoError(t, err)
	return media
}

func TestRequestUploadContentTypeAllowlist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		wantKind    domain.MediaKind
		wantErr     bool
	}{
		{name: "jpeg", contentType: "image/jpeg", wantKind: domain.MediaImage},
		{name: "png", contentType: "image/png", wantKind: domain.MediaImage},
		{name: "webp", contentType: "image/webp", wantKind: domain.MediaImage},
		{name: "mp4", contentType: "video/mp4", wantKind: domain.MediaVideo},
		{name: "webm video", contentType: "video/webm", wantKind: domain.MediaVideo},
		{name: "pdf", contentType: "application/pdf", wantKind: domain.MediaDocument},
		{name: "webm audio, a story-step voice note's real MediaRecorder default", contentType: "audio/webm", wantKind: domain.MediaAudio},
		{name: "ogg audio", contentType: "audio/ogg", wantKind: domain.MediaAudio},
		{name: "m4a audio", contentType: "audio/mp4", wantKind: domain.MediaAudio},
		{name: "mp3 audio", contentType: "audio/mpeg", wantKind: domain.MediaAudio},
		{name: "wav audio", contentType: "audio/wav", wantKind: domain.MediaAudio},
		{name: "uppercase is normalised", contentType: "IMAGE/JPEG", wantKind: domain.MediaImage},
		{name: "heic is not accepted", contentType: "image/heic", wantErr: true},
		{name: "quicktime is not accepted", contentType: "video/quicktime", wantErr: true},
		{name: "gif is not accepted", contentType: "image/gif", wantErr: true},
		{name: "svg is not accepted", contentType: "image/svg+xml", wantErr: true},
		{name: "empty", contentType: "", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newMediaFixture(t)

			ticket, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
				ArtisanID:   f.artisan,
				ContentType: tt.contentType,
				SizeBytes:   1024,
			})
			if tt.wantErr {
				require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
				require.Empty(t, f.store.media)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, ticket.UploadURL)
			require.Equal(t, tt.wantKind, f.store.media[ticket.MediaID].Kind)
			require.Equal(t, domain.MediaPending, f.store.media[ticket.MediaID].State)
		})
	}
}

func TestRequestUploadSizeCaps(t *testing.T) {
	t.Parallel()
	limits := testMediaLimits()

	tests := []struct {
		name        string
		contentType string
		size        int64
		wantErr     bool
	}{
		{name: "image just under the cap", contentType: "image/jpeg", size: limits.MaxImageBytes - 1},
		{name: "image at the cap", contentType: "image/jpeg", size: limits.MaxImageBytes},
		{name: "image over the cap", contentType: "image/jpeg", size: limits.MaxImageBytes + 1, wantErr: true},
		{name: "video at the image cap is fine", contentType: "video/mp4", size: limits.MaxImageBytes * 2},
		{name: "video at the cap", contentType: "video/mp4", size: limits.MaxVideoBytes},
		{name: "video over the cap", contentType: "video/mp4", size: limits.MaxVideoBytes + 1, wantErr: true},
		{name: "zero bytes", contentType: "image/png", size: 0, wantErr: true},
		{name: "negative bytes", contentType: "image/png", size: -1, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newMediaFixture(t)

			_, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
				ArtisanID:   f.artisan,
				ContentType: tt.contentType,
				SizeBytes:   tt.size,
			})
			if tt.wantErr {
				require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
				require.Empty(t, f.store.media)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRequestUploadAuthorisation(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)

	_, err := f.svc.RequestUpload(artisanCtx(ids.New()), domain.RequestUploadInput{
		ArtisanID:   f.artisan,
		ContentType: "image/jpeg",
		SizeBytes:   1024,
	})
	require.ErrorIs(t, err, pkgdomain.ErrForbidden)

	// Field staff upload on an artisan's behalf from the cluster desk.
	_, err = f.svc.RequestUpload(officerCtx(), domain.RequestUploadInput{
		ArtisanID:   f.artisan,
		ContentType: "image/jpeg",
		SizeBytes:   1024,
	})
	require.NoError(t, err)
}

func TestRequestUploadDeduplicatesByContentHash(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)

	hash := "4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865"
	first, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
		ArtisanID:   f.artisan,
		ContentType: "image/jpeg",
		SizeBytes:   2048,
		SHA256Hex:   &hash,
	})
	require.NoError(t, err)
	require.NotEmpty(t, first.UploadURL)

	f.objects.put(first.ObjectKey, 2048)
	_, err = f.svc.ConfirmUpload(artisanCtx(f.artisan), first.MediaID)
	require.NoError(t, err)

	second, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
		ArtisanID:   f.artisan,
		ContentType: "image/jpeg",
		SizeBytes:   2048,
		SHA256Hex:   &hash,
	})
	require.NoError(t, err)
	require.True(t, second.Deduplicated)
	require.Equal(t, first.MediaID, second.MediaID)
	require.Empty(t, second.UploadURL)
	require.Len(t, f.store.media, 1)
}

func TestRequestUploadRejectsAMalformedHash(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)

	for _, bad := range []string{"nothex", "ABCDEF", "4355a46b"} {
		bad := bad
		_, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
			ArtisanID:   f.artisan,
			ContentType: "image/jpeg",
			SizeBytes:   1024,
			SHA256Hex:   &bad,
		})
		require.ErrorIs(t, err, pkgdomain.ErrInvalidInput, bad)
	}
}

// TestConfirmUploadRoundTrip is the acceptance path: a presigned URL, bytes in
// the bucket, then a confirmation that flips the row and emits exactly one event.
func TestConfirmUploadRoundTrip(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)

	ticket, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
		ArtisanID:   f.artisan,
		ContentType: "image/jpeg",
		SizeBytes:   4096,
	})
	require.NoError(t, err)
	require.Contains(t, ticket.ObjectKey, f.artisan.String())
	require.Equal(t, fakeNow.Add(15*time.Minute), ticket.ExpiresAt)
	require.Empty(t, f.store.outbox)

	// The phone PUTs the bytes straight to the bucket.
	f.objects.put(ticket.ObjectKey, 4096)

	media, err := f.svc.ConfirmUpload(artisanCtx(f.artisan), ticket.MediaID)
	require.NoError(t, err)
	require.Equal(t, domain.MediaUploaded, media.State)
	require.NotNil(t, media.ConfirmedAt)

	require.Len(t, f.store.outbox, 1)
	require.Equal(t, topics.MediaUploaded, f.store.outbox[0].Topic)
	require.Equal(t, ticket.MediaID.String(), f.store.outbox[0].AggregateID)
}

func TestConfirmUploadWithoutAnObject(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)

	ticket, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
		ArtisanID:   f.artisan,
		ContentType: "image/jpeg",
		SizeBytes:   4096,
	})
	require.NoError(t, err)

	// Nothing was ever PUT.
	_, err = f.svc.ConfirmUpload(artisanCtx(f.artisan), ticket.MediaID)
	require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
	require.Equal(t, domain.MediaPending, f.store.media[ticket.MediaID].State)
	require.Empty(t, f.store.outbox)
}

func TestConfirmUploadSizeMismatch(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)

	ticket, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
		ArtisanID:   f.artisan,
		ContentType: "video/mp4",
		SizeBytes:   1 << 20,
	})
	require.NoError(t, err)

	// The client declared one megabyte and sent eighty.
	f.objects.put(ticket.ObjectKey, 80<<20)

	_, err = f.svc.ConfirmUpload(artisanCtx(f.artisan), ticket.MediaID)
	require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
	require.Empty(t, f.store.outbox)
}

// TestConfirmUploadIsIdempotent is the second acceptance criterion: the app on a
// bad connection calls twice, and the second call changes nothing.
func TestConfirmUploadIsIdempotent(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)

	media := f.upload(t, "image/png", 8192)
	require.Len(t, f.store.outbox, 1)

	again, err := f.svc.ConfirmUpload(artisanCtx(f.artisan), media.ID)
	require.NoError(t, err)
	require.Equal(t, media.State, again.State)
	require.Len(t, f.store.outbox, 1)

	third, err := f.svc.ConfirmUpload(artisanCtx(f.artisan), media.ID)
	require.NoError(t, err)
	require.Equal(t, media.ID, third.ID)
	require.Len(t, f.store.outbox, 1)
}

func TestConfirmUploadAuthorisation(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)

	ticket, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
		ArtisanID:   f.artisan,
		ContentType: "image/jpeg",
		SizeBytes:   1024,
	})
	require.NoError(t, err)
	f.objects.put(ticket.ObjectKey, 1024)

	_, err = f.svc.ConfirmUpload(artisanCtx(ids.New()), ticket.MediaID)
	require.ErrorIs(t, err, pkgdomain.ErrForbidden)
}

func TestMediaStateMachine(t *testing.T) {
	t.Parallel()

	states := []domain.MediaState{
		domain.MediaPending, domain.MediaUploaded,
		domain.MediaProcessing, domain.MediaReady, domain.MediaFailed,
	}
	legal := map[domain.MediaState]map[domain.MediaState]bool{
		domain.MediaPending:    {domain.MediaUploaded: true},
		domain.MediaUploaded:   {domain.MediaProcessing: true, domain.MediaFailed: true},
		domain.MediaProcessing: {domain.MediaReady: true, domain.MediaFailed: true},
		domain.MediaReady:      {domain.MediaFailed: true},
		domain.MediaFailed:     {},
	}

	for _, from := range states {
		for _, to := range states {
			from, to := from, to
			t.Run(from.String()+"_to_"+to.String(), func(t *testing.T) {
				t.Parallel()
				err := domain.ValidateMediaTransition(from, to)
				if legal[from][to] {
					require.NoError(t, err)
					require.True(t, from.CanTransitionTo(to))
					return
				}
				require.ErrorIs(t, err, pkgdomain.ErrInvalidInput)
				require.Contains(t, err.Error(), from.String()+" -> "+to.String())
				require.False(t, from.CanTransitionTo(to))
			})
		}
	}
}

func TestApplyEnhancement(t *testing.T) {
	t.Parallel()

	t.Run("promotes an uploaded asset to ready", func(t *testing.T) {
		t.Parallel()
		f := newMediaFixture(t)
		media := f.upload(t, "image/jpeg", 4096)

		ready, err := f.svc.ApplyEnhancement(context.Background(), EnhancementResult{
			MediaID:           media.ID,
			EnhancedObjectKey: media.ObjectKey + ".enhanced.webp",
			IdempotencyKey:    "evt-1",
		})
		require.NoError(t, err)
		require.Equal(t, domain.MediaReady, ready.State)
		require.NotNil(t, ready.EnhancedObjectKey)
		require.Equal(t, media.ObjectKey+".enhanced.webp", ready.ServableObjectKey())

		// Only the upload event; what happens next is the pipeline's business.
		require.Len(t, f.store.outbox, 1)
	})

	t.Run("a redelivered event changes nothing", func(t *testing.T) {
		t.Parallel()
		f := newMediaFixture(t)
		media := f.upload(t, "image/jpeg", 4096)

		result := EnhancementResult{
			MediaID:           media.ID,
			EnhancedObjectKey: "enhanced/" + media.ID.String() + ".webp",
			IdempotencyKey:    "evt-1",
		}
		_, err := f.svc.ApplyEnhancement(context.Background(), result)
		require.NoError(t, err)
		require.Len(t, f.store.outbox, 1)

		_, err = f.svc.ApplyEnhancement(context.Background(), result)
		require.NoError(t, err)
		require.Len(t, f.store.outbox, 1)
	})

	t.Run("a failure is recorded with its reason", func(t *testing.T) {
		t.Parallel()
		f := newMediaFixture(t)
		media := f.upload(t, "image/jpeg", 4096)

		reason := "the image could not be decoded"
		failed, err := f.svc.ApplyEnhancement(context.Background(), EnhancementResult{
			MediaID:        media.ID,
			FailureReason:  &reason,
			IdempotencyKey: "evt-1",
		})
		require.NoError(t, err)
		require.Equal(t, domain.MediaFailed, failed.State)
		require.NotNil(t, failed.FailureReason)
		require.Equal(t, reason, *failed.FailureReason)
	})

	t.Run("an unconfirmed asset cannot be enhanced", func(t *testing.T) {
		t.Parallel()
		f := newMediaFixture(t)
		ticket, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
			ArtisanID:   f.artisan,
			ContentType: "image/jpeg",
			SizeBytes:   1024,
		})
		require.NoError(t, err)

		_, err = f.svc.ApplyEnhancement(context.Background(), EnhancementResult{
			MediaID:           ticket.MediaID,
			EnhancedObjectKey: "enhanced/x.webp",
			IdempotencyKey:    "evt-1",
		})
		require.ErrorIs(t, err, pkgdomain.ErrConflict)
	})
}

func TestGetMediaURL(t *testing.T) {
	t.Parallel()

	t.Run("caches the signature and serves the enhanced object", func(t *testing.T) {
		t.Parallel()
		f := newMediaFixture(t)
		media := f.upload(t, "image/jpeg", 4096)

		enhancedKey := "enhanced/" + media.ID.String() + ".webp"
		_, err := f.svc.ApplyEnhancement(context.Background(), EnhancementResult{
			MediaID:           media.ID,
			EnhancedObjectKey: enhancedKey,
			IdempotencyKey:    "evt-1",
		})
		require.NoError(t, err)

		first, expiresAt, err := f.svc.GetMediaURL(artisanCtx(f.artisan), media.ID)
		require.NoError(t, err)
		require.Contains(t, first, enhancedKey)
		require.Equal(t, fakeNow.Add(time.Hour), expiresAt)

		second, _, err := f.svc.GetMediaURL(artisanCtx(ids.New()), media.ID)
		require.NoError(t, err)
		require.Equal(t, first, second)
		require.Equal(t, 1, f.objects.getCalls)
		require.Equal(t, 1, f.cache.sets)
	})

	t.Run("an unready asset is private to its artisan", func(t *testing.T) {
		t.Parallel()
		f := newMediaFixture(t)
		media := f.upload(t, "image/jpeg", 4096)

		_, _, err := f.svc.GetMediaURL(artisanCtx(ids.New()), media.ID)
		require.ErrorIs(t, err, pkgdomain.ErrForbidden)

		_, _, err = f.svc.GetMediaURL(artisanCtx(f.artisan), media.ID)
		require.NoError(t, err)
	})
}

func TestReapStalePending(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)

	// One abandoned ticket from two days ago, one from a minute ago, and one
	// that was actually uploaded.
	stale, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
		ArtisanID: f.artisan, ContentType: "image/jpeg", SizeBytes: 1024,
	})
	require.NoError(t, err)
	fresh, err := f.svc.RequestUpload(artisanCtx(f.artisan), domain.RequestUploadInput{
		ArtisanID: f.artisan, ContentType: "image/jpeg", SizeBytes: 1024,
	})
	require.NoError(t, err)
	confirmed := f.upload(t, "image/png", 2048)

	// A half-finished upload left an orphan object behind.
	f.objects.put(stale.ObjectKey, 512)

	aged := f.store.media[stale.MediaID]
	aged.CreatedAt = fakeNow.Add(-48 * time.Hour)
	f.store.media[stale.MediaID] = aged

	reaped, err := f.svc.ReapStalePending(context.Background(), 24*time.Hour, 100)
	require.NoError(t, err)
	require.Equal(t, 1, reaped)
	require.Equal(t, []string{stale.ObjectKey}, f.objects.deleted)

	_, err = f.store.GetMedia(context.Background(), stale.MediaID)
	require.ErrorIs(t, err, pkgdomain.ErrNotFound)

	// Neither the recent ticket nor the confirmed asset was touched.
	_, err = f.store.GetMedia(context.Background(), fresh.MediaID)
	require.NoError(t, err)
	_, err = f.store.GetMedia(context.Background(), confirmed.ID)
	require.NoError(t, err)
}

// principalless calls are refused: the reaper is the only path that runs
// without one, and it does not go through these methods.
func TestMediaCallsRequireAPrincipal(t *testing.T) {
	t.Parallel()
	f := newMediaFixture(t)
	media := f.upload(t, "image/jpeg", 1024)

	_, err := f.svc.RequestUpload(context.Background(), domain.RequestUploadInput{
		ArtisanID: f.artisan, ContentType: "image/jpeg", SizeBytes: 1024,
	})
	require.ErrorIs(t, err, pkgdomain.ErrForbidden)

	_, err = f.svc.ConfirmUpload(context.Background(), media.ID)
	require.ErrorIs(t, err, pkgdomain.ErrForbidden)

	_, _, err = f.svc.GetMediaURL(context.Background(), media.ID)
	require.ErrorIs(t, err, pkgdomain.ErrForbidden)
}

var _ = auth.RoleArtisan
