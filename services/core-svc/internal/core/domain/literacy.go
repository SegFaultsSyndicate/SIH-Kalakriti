// services/core-svc/internal/core/domain/literacy.go

package domain

// LessonDef is one lesson of the digital literacy track. The catalog is code,
// not data: lesson content is i18n-keyed in the app (lesson.<code_lower>.*)
// and versioned with it.
type LessonDef struct {
	Code  string
	Order int32
	// HasPractice lessons complete only once their in-app practice is done.
	HasPractice bool
	// PracticeEvidence is what the server checks before accepting practice:
	// "IMAGE" (a confirmed photo that passes the quality gate), "AUDIO" (a
	// confirmed voice note), or "" (client-attested, e.g. the scam-call drill).
	PracticeEvidence string
}

// Lessons is the literacy track, in order.
var Lessons = []LessonDef{
	{Code: "PHOTO", Order: 1, HasPractice: true, PracticeEvidence: "IMAGE"},
	{Code: "STORY", Order: 2, HasPractice: true, PracticeEvidence: "AUDIO"},
	{Code: "PRICE", Order: 3},
	{Code: "ORDERS", Order: 4},
	{Code: "PAYMENTS", Order: 5},
	{Code: "SAFETY", Order: 6, HasPractice: true},
	{Code: "WHATSAPP", Order: 7},
	{Code: "LOAN", Order: 8},
}

// LessonByCode finds a lesson definition.
func LessonByCode(code string) (LessonDef, bool) {
	for _, l := range Lessons {
		if l.Code == code {
			return l, true
		}
	}
	return LessonDef{}, false
}

// LessonProgress is one artisan's progress on one lesson.
type LessonProgress struct {
	LessonDef
	PracticeDone bool
	QuizPassed   bool
	Completed    bool
}
