package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"ytagent/backend/internal/models"
)

type ScriptStore struct {
	db *sql.DB
}

func NewScriptStore(db *sql.DB) *ScriptStore {
	return &ScriptStore{db: db}
}

func (s *ScriptStore) Create(ctx context.Context, title, description, hook, durationFormat string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO scripts (idea_title, idea_description, idea_hook, duration_format, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		title, description, hook, durationFormat, models.ScriptStatusPending,
	).Scan(&id)
	return id, err
}

// SetDone also clears revision_feedback: whether this call came from a
// from-scratch generation or a revision, the feedback that led here has now
// been applied (or was never set), so it shouldn't linger and be reapplied
// on some later unrelated regeneration.
func (s *ScriptStore) SetDone(ctx context.Context, id string, script models.Script) error {
	raw, err := json.Marshal(script)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE scripts SET status = $1, script_json = $2, revision_feedback = NULL, updated_at = now() WHERE id = $3`,
		models.ScriptStatusDone, raw, id,
	)
	return err
}

func (s *ScriptStore) SetFailed(ctx context.Context, id, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE scripts SET status = $1, error_message = $2, updated_at = now() WHERE id = $3`,
		models.ScriptStatusFailed, errMsg, id,
	)
	return err
}

// ResetToPending clears a job's error and puts it back at "pending" so the
// client's next Step call regenerates it in place — same history entry,
// fresh AI call — instead of creating a new one. Works from "done" (re-roll
// a result the user doesn't like) as well as "failed" (retry after an
// error). Also clears any leftover revision_feedback: a plain "Tạo lại" is a
// from-scratch regeneration, not a revision, and shouldn't reapply a stale
// note left over from an earlier, unrelated revise attempt.
func (s *ScriptStore) ResetToPending(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE scripts SET status = $1, error_message = NULL, revision_feedback = NULL, updated_at = now() WHERE id = $2`,
		models.ScriptStatusPending, id,
	)
	return err
}

// SetRevisionPending puts a job back at "pending" like ResetToPending, but
// stores the human feedback alongside it instead of clearing it — the next
// Step call reads RevisionFeedback and asks the AI to revise the existing
// ScriptJSON accordingly (see ai.Client.ReviseScript) instead of generating
// from scratch.
func (s *ScriptStore) SetRevisionPending(ctx context.Context, id, feedback string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE scripts SET status = $1, error_message = NULL, revision_feedback = $2, updated_at = now() WHERE id = $3`,
		models.ScriptStatusPending, feedback, id,
	)
	return err
}

func (s *ScriptStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM scripts WHERE id = $1`, id)
	return err
}

func (s *ScriptStore) GetByID(ctx context.Context, id string) (*models.ScriptJob, error) {
	var j models.ScriptJob
	var description, hook, errorMessage, revisionFeedback sql.NullString
	var scriptJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, idea_title, idea_description, idea_hook, duration_format, status, error_message,
		        script_json, revision_feedback, created_at, updated_at
		 FROM scripts WHERE id = $1`, id,
	).Scan(&j.ID, &j.IdeaTitle, &description, &hook, &j.DurationFormat, &j.Status, &errorMessage,
		&scriptJSON, &revisionFeedback, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}

	j.IdeaDescription = description.String
	j.IdeaHook = hook.String
	j.ErrorMessage = errorMessage.String
	j.ScriptJSON = scriptJSON
	j.RevisionFeedback = revisionFeedback.String
	return &j, nil
}

// ListRecent includes each job's full script_json (unlike Analysis's
// equivalent list, which omits its larger blobs) — scripts are small and
// this lets the frontend's history list show past results without a second
// round-trip per item.
func (s *ScriptStore) ListRecent(ctx context.Context, limit int) ([]models.ScriptJob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, idea_title, idea_description, idea_hook, duration_format, status, error_message,
		        script_json, revision_feedback, created_at, updated_at
		 FROM scripts ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.ScriptJob
	for rows.Next() {
		var j models.ScriptJob
		var description, hook, errorMessage, revisionFeedback sql.NullString
		var scriptJSON []byte
		if err := rows.Scan(&j.ID, &j.IdeaTitle, &description, &hook, &j.DurationFormat, &j.Status,
			&errorMessage, &scriptJSON, &revisionFeedback, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		j.IdeaDescription = description.String
		j.IdeaHook = hook.String
		j.ErrorMessage = errorMessage.String
		j.ScriptJSON = scriptJSON
		j.RevisionFeedback = revisionFeedback.String
		results = append(results, j)
	}
	return results, rows.Err()
}
