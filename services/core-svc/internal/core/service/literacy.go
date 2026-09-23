// services/core-svc/internal/core/service/literacy.go

package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// LiteracyStore is the persistence surface of the digital literacy track (F15).
type LiteracyStore interface {
	ListLiteracyProgress(ctx context.Context, artisanID uuid.UUID) (map[string]domain.LessonProgress, error)
	UpsertLiteracyProgress(ctx context.Context, artisanID uuid.UUID, lesson domain.LessonDef, practiceDone, quizPassed bool) (domain.LessonProgress, error)
	CountCompletedLessons(ctx context.Context, artisanID uuid.UUID) (int64, error)
	HasLiteracyCertificate(ctx context.Context, artisanID uuid.UUID) (bool, error)
	GetMedia(ctx context.Context, id uuid.UUID) (domain.Media, error)
}

// ImageQualityChecker is ml-svc's pre-VLM quality gate.
type ImageQualityChecker interface {
	AssessImageQuality(ctx context.Context, objectKey string) (domain.ImageQualityVerdict, error)
}

// LessonBadgeTracker grants the digital_ready badge.
type LessonBadgeTracker interface {
	TrackLessonsCompleted(ctx context.Context, artisanID uuid.UUID, count int64) ([]domain.Badge, error)
}

// Literacy is the digital literacy track. Lesson content lives in the app;
// the server keeps progress and checks practice evidence.
type Literacy struct {
	store   LiteracyStore
	quality ImageQualityChecker // nil when ml-svc is not configured
	badges  LessonBadgeTracker
	log     *slog.Logger
}

// NewLiteracy builds the literacy service.
func NewLiteracy(store LiteracyStore, quality ImageQualityChecker, badges LessonBadgeTracker, log *slog.Logger) *Literacy {
	if log == nil {
		log = slog.Default()
	}
	return &Literacy{store: store, quality: quality, badges: badges, log: log}
}

// LessonsView is the whole track for one artisan.
type LessonsView struct {
	Lessons           []domain.LessonProgress
	CompletedCount    int32
	CertificateIssued bool
}

func (s *Literacy) ListLessons(ctx context.Context) (LessonsView, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return LessonsView{}, err
	}
	progress, err := s.store.ListLiteracyProgress(ctx, artisanID)
	if err != nil {
		return LessonsView{}, err
	}
	view := LessonsView{Lessons: make([]domain.LessonProgress, len(domain.Lessons))}
	for i, def := range domain.Lessons {
		p, ok := progress[def.Code]
		if !ok {
			p = domain.LessonProgress{}
		}
		p.LessonDef = def
		view.Lessons[i] = p
		if p.Completed {
			view.CompletedCount++
		}
	}
	view.CertificateIssued, err = s.store.HasLiteracyCertificate(ctx, artisanID)
	return view, err
}

// RecordLessonProgressInput is one progress report from the app.
type RecordLessonProgressInput struct {
	LessonCode      string
	PracticeDone    bool
	QuizPassed      bool
	PracticeMediaID *uuid.UUID
}

func (s *Literacy) RecordLessonProgress(ctx context.Context, in RecordLessonProgressInput) (domain.LessonProgress, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return domain.LessonProgress{}, err
	}
	lesson, ok := domain.LessonByCode(in.LessonCode)
	if !ok {
		return domain.LessonProgress{}, pkgdomain.InvalidInput("unknown lesson_code")
	}
	practice := in.PracticeDone && lesson.HasPractice
	if practice && lesson.PracticeEvidence != "" {
		if err := s.checkEvidence(ctx, artisanID, lesson, in.PracticeMediaID); err != nil {
			return domain.LessonProgress{}, err
		}
	}
	p, err := s.store.UpsertLiteracyProgress(ctx, artisanID, lesson, practice, in.QuizPassed)
	if err != nil {
		return domain.LessonProgress{}, err
	}
	if p.Completed && s.badges != nil {
		n, err := s.store.CountCompletedLessons(ctx, artisanID)
		if err == nil {
			_, err = s.badges.TrackLessonsCompleted(ctx, artisanID, n)
		}
		if err != nil {
			// The progress row is written; the badge catches up on the next lesson.
			s.log.WarnContext(ctx, "digital_ready badge recompute", "artisan_id", artisanID, "error", err)
		}
	}
	return p, nil
}

// checkEvidence verifies practice: the media must be the artisan's own, of
// the right kind, uploaded (not a dangling ticket), and a photo must pass
// the same quality gate a listing photo would.
func (s *Literacy) checkEvidence(ctx context.Context, artisanID uuid.UUID, lesson domain.LessonDef, mediaID *uuid.UUID) error {
	if mediaID == nil {
		return pkgdomain.InvalidInput("practice_media_id is required for this lesson's practice")
	}
	m, err := s.store.GetMedia(ctx, *mediaID)
	if err != nil {
		return err
	}
	if m.ArtisanID != artisanID || string(m.Kind) != lesson.PracticeEvidence {
		return pkgdomain.InvalidInput("practice_media_id must be your own " + lesson.PracticeEvidence + " upload")
	}
	if m.State == domain.MediaPending || m.State == domain.MediaFailed {
		return pkgdomain.InvalidInput("the practice upload has not finished")
	}
	if lesson.PracticeEvidence != string(domain.MediaImage) || s.quality == nil {
		// ponytail: without ml-svc a confirmed photo is accepted as-is; the gate returns when it is configured.
		return nil
	}
	verdict, err := s.quality.AssessImageQuality(ctx, m.ObjectKey)
	if err != nil {
		return pkgdomain.Unavailable("photo check is unavailable right now, try again shortly")
	}
	if !verdict.Passed {
		msg := "the photo did not pass the quality check"
		if len(verdict.Issues) > 0 {
			msg = verdict.Issues[0].Message
		}
		return pkgdomain.InvalidInput(msg)
	}
	return nil
}
