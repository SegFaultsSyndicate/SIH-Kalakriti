// services/core-svc/internal/core/repo/literacy.go

package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/repo/db"
)

// ListLiteracyProgress returns the artisan's stored progress keyed by lesson code.
func (r *Repo) ListLiteracyProgress(ctx context.Context, artisanID uuid.UUID) (map[string]domain.LessonProgress, error) {
	rows, err := r.q.ListLiteracyProgress(ctx, artisanID)
	if err != nil {
		return nil, translate(err, "literacy progress")
	}
	out := make(map[string]domain.LessonProgress, len(rows))
	for _, row := range rows {
		out[row.LessonCode] = domain.LessonProgress{
			PracticeDone: row.PracticeDone, QuizPassed: row.QuizPassed, Completed: row.CompletedAt != nil,
		}
	}
	return out, nil
}

func (r *Repo) UpsertLiteracyProgress(ctx context.Context, artisanID uuid.UUID, lesson domain.LessonDef, practiceDone, quizPassed bool) (domain.LessonProgress, error) {
	row, err := r.q.UpsertLiteracyProgress(ctx, db.UpsertLiteracyProgressParams{
		ArtisanID: artisanID, LessonCode: lesson.Code, PracticeDone: practiceDone, QuizPassed: quizPassed,
		RequiresPractice: lesson.HasPractice,
	})
	if err != nil {
		return domain.LessonProgress{}, translate(err, "literacy progress")
	}
	return domain.LessonProgress{
		LessonDef: lesson, PracticeDone: row.PracticeDone, QuizPassed: row.QuizPassed, Completed: row.CompletedAt != nil,
	}, nil
}

func (r *Repo) CountCompletedLessons(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	n, err := r.q.CountCompletedLessons(ctx, artisanID)
	return n, translate(err, "literacy progress")
}

func (r *Repo) HasLiteracyCertificate(ctx context.Context, artisanID uuid.UUID) (bool, error) {
	ok, err := r.q.HasLiteracyCertificate(ctx, artisanID)
	return ok, translate(err, "literacy certificate")
}
