// services/core-svc/internal/core/handler/literacy.go

package handler

import (
	"context"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	literacyv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/literacy/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Literacy implements literacy.v1.LiteracyService.
type Literacy struct {
	literacyv1.UnimplementedLiteracyServiceServer
	svc *service.Literacy
}

// NewLiteracy builds the literacy handler.
func NewLiteracy(svc *service.Literacy) *Literacy { return &Literacy{svc: svc} }

func (h *Literacy) ListLessons(ctx context.Context, _ *literacyv1.ListLessonsRequest) (*literacyv1.ListLessonsResponse, error) {
	v, err := h.svc.ListLessons(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*literacyv1.Lesson, len(v.Lessons))
	for i, l := range v.Lessons {
		out[i] = lessonToProto(l)
	}
	return &literacyv1.ListLessonsResponse{Lessons: out, CompletedCount: v.CompletedCount, CertificateIssued: v.CertificateIssued}, nil
}

func (h *Literacy) RecordLessonProgress(ctx context.Context, req *literacyv1.RecordLessonProgressRequest) (*literacyv1.RecordLessonProgressResponse, error) {
	mediaID, err := parseOptionalUUID("practice_media_id", req.PracticeMediaId)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	p, err := h.svc.RecordLessonProgress(ctx, service.RecordLessonProgressInput{
		LessonCode: req.GetLessonCode(), PracticeDone: req.GetPracticeDone(),
		QuizPassed: req.GetQuizPassed(), PracticeMediaID: mediaID,
	})
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &literacyv1.RecordLessonProgressResponse{Lesson: lessonToProto(p)}, nil
}

func lessonToProto(l domain.LessonProgress) *literacyv1.Lesson {
	return &literacyv1.Lesson{
		Code: l.Code, Order: l.Order, HasPractice: l.HasPractice,
		PracticeDone: l.PracticeDone, QuizPassed: l.QuizPassed, Completed: l.Completed,
	}
}
