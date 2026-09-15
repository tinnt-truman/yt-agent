package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"ytagent/backend/internal/models"
)

type MusicPromptStore struct {
	db *sql.DB
}

func NewMusicPromptStore(db *sql.DB) *MusicPromptStore {
	return &MusicPromptStore{db: db}
}

func (s *MusicPromptStore) Create(ctx context.Context, title, description string, tags []string) (string, error) {
	var tagsJSON []byte
	if len(tags) > 0 {
		var err error
		tagsJSON, err = json.Marshal(tags)
		if err != nil {
			return "", err
		}
	}

	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO music_prompts (video_title, video_description, video_tags, status)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		title, description, tagsJSON, models.MusicPromptStatusPending,
	).Scan(&id)
	return id, err
}

func (s *MusicPromptStore) SetDone(ctx context.Context, id string, prompt models.MusicPrompt) error {
	raw, err := json.Marshal(prompt)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE music_prompts SET status = $1, music_prompt_json = $2, updated_at = now() WHERE id = $3`,
		models.MusicPromptStatusDone, raw, id,
	)
	return err
}

func (s *MusicPromptStore) SetFailed(ctx context.Context, id, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE music_prompts SET status = $1, error_message = $2, updated_at = now() WHERE id = $3`,
		models.MusicPromptStatusFailed, errMsg, id,
	)
	return err
}

// ResetToPending clears a failed job's error and puts it back at "pending"
// so the client's next Step call retries generation in place, instead of
// creating a new history entry for the same request.
func (s *MusicPromptStore) ResetToPending(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE music_prompts SET status = $1, error_message = NULL, updated_at = now() WHERE id = $2`,
		models.MusicPromptStatusPending, id,
	)
	return err
}

func (s *MusicPromptStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM music_prompts WHERE id = $1`, id)
	return err
}

func (s *MusicPromptStore) GetByID(ctx context.Context, id string) (*models.MusicPromptJob, error) {
	var j models.MusicPromptJob
	var description, errorMessage sql.NullString
	var tagsJSON, promptJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, video_title, video_description, video_tags, status, error_message,
		        music_prompt_json, created_at, updated_at
		 FROM music_prompts WHERE id = $1`, id,
	).Scan(&j.ID, &j.VideoTitle, &description, &tagsJSON, &j.Status, &errorMessage,
		&promptJSON, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}

	j.VideoDescription = description.String
	j.ErrorMessage = errorMessage.String
	if len(tagsJSON) > 0 {
		_ = json.Unmarshal(tagsJSON, &j.VideoTags)
	}
	j.MusicPromptJSON = promptJSON
	return &j, nil
}

// ListRecent includes each job's full music_prompt_json (unlike Analysis's
// equivalent list, which omits its larger blobs) — prompts are small and
// this lets the frontend's history list show past results without a second
// round-trip per item.
func (s *MusicPromptStore) ListRecent(ctx context.Context, limit int) ([]models.MusicPromptJob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, video_title, video_description, video_tags, status, error_message,
		        music_prompt_json, created_at, updated_at
		 FROM music_prompts ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.MusicPromptJob
	for rows.Next() {
		var j models.MusicPromptJob
		var description, errorMessage sql.NullString
		var tagsJSON, promptJSON []byte
		if err := rows.Scan(&j.ID, &j.VideoTitle, &description, &tagsJSON, &j.Status,
			&errorMessage, &promptJSON, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		j.VideoDescription = description.String
		j.ErrorMessage = errorMessage.String
		if len(tagsJSON) > 0 {
			_ = json.Unmarshal(tagsJSON, &j.VideoTags)
		}
		j.MusicPromptJSON = promptJSON
		results = append(results, j)
	}
	return results, rows.Err()
}
