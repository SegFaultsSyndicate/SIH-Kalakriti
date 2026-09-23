-- migrations/queries/literacy.sql
-- F15 artisan-side: lesson progress. Certificates live in insight.sql.

-- name: ListLiteracyProgress :many
SELECT * FROM literacy_progress WHERE artisan_id = @artisan_id;

-- name: UpsertLiteracyProgress :one
-- Flags only ever go false -> true: a retried or out-of-order outbox write
-- can never un-complete a lesson. A lesson completes (once) when its quiz is
-- passed and, if it has a practice task, that practice is done.
INSERT INTO literacy_progress AS lp (artisan_id, lesson_code, practice_done, quiz_passed, completed_at, updated_at)
VALUES (
    @artisan_id, @lesson_code, @practice_done, @quiz_passed,
    CASE WHEN @quiz_passed AND (@practice_done OR NOT @requires_practice::boolean) THEN now() END,
    now()
)
ON CONFLICT (artisan_id, lesson_code) DO UPDATE SET
    practice_done = lp.practice_done OR EXCLUDED.practice_done,
    quiz_passed   = lp.quiz_passed OR EXCLUDED.quiz_passed,
    completed_at  = COALESCE(lp.completed_at, CASE
        WHEN (lp.quiz_passed OR EXCLUDED.quiz_passed)
         AND (lp.practice_done OR EXCLUDED.practice_done OR NOT @requires_practice::boolean)
        THEN now() END),
    updated_at    = now()
RETURNING *;

-- name: CountCompletedLessons :one
SELECT count(*) FROM literacy_progress WHERE artisan_id = @artisan_id AND completed_at IS NOT NULL;

-- name: HasLiteracyCertificate :one
SELECT EXISTS (SELECT 1 FROM literacy_certificate WHERE artisan_id = @artisan_id)::boolean AS issued;
