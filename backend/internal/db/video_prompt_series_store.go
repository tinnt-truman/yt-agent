package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"ytagent/backend/internal/models"
)

type VideoPromptSeriesStore struct {
	db *sql.DB
}

func NewVideoPromptSeriesStore(db *sql.DB) *VideoPromptSeriesStore {
	return &VideoPromptSeriesStore{db: db}
}

func (s *VideoPromptSeriesStore) Create(
	ctx context.Context, title, description string, tags []string, episodeCount int,
) (string, error) {
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
		`INSERT INTO video_prompt_series (video_title, video_description, video_tags, episode_count, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		title, description, tagsJSON, episodeCount, models.VideoPromptSeriesStatusPending,
	).Scan(&id)
	return id, err
}

func (s *VideoPromptSeriesStore) SetDone(ctx context.Context, id string, series models.VideoPromptSeries) error {
	raw, err := json.Marshal(series)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE video_prompt_series SET status = $1, series_json = $2, updated_at = now() WHERE id = $3`,
		models.VideoPromptSeriesStatusDone, raw, id,
	)
	return err
}

func (s *VideoPromptSeriesStore) SetFailed(ctx context.Context, id, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE video_prompt_series SET status = $1, error_message = $2, updated_at = now() WHERE id = $3`,
		models.VideoPromptSeriesStatusFailed, errMsg, id,
	)
	return err
}

// ResetToPending clears a failed job's error and puts it back at "pending"
// so the client's next Step call retries generation in place, instead of
// creating a new history entry for the same request.
func (s *VideoPromptSeriesStore) ResetToPending(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE video_prompt_series SET status = $1, error_message = NULL, updated_at = now() WHERE id = $2`,
		models.VideoPromptSeriesStatusPending, id,
	)
	return err
}

func (s *VideoPromptSeriesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM video_prompt_series WHERE id = $1`, id)
	return err
}

func (s *VideoPromptSeriesStore) GetByID(ctx context.Context, id string) (*models.VideoPromptSeriesJob, error) {
	var j models.VideoPromptSeriesJob
	var description, errorMessage sql.NullString
	var tagsJSON, seriesJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, video_title, video_description, video_tags, episode_count, status, error_message,
		        series_json, created_at, updated_at
		 FROM video_prompt_series WHERE id = $1`, id,
	).Scan(&j.ID, &j.VideoTitle, &description, &tagsJSON, &j.EpisodeCount, &j.Status, &errorMessage,
		&seriesJSON, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}

	j.VideoDescription = description.String
	j.ErrorMessage = errorMessage.String
	if len(tagsJSON) > 0 {
		_ = json.Unmarshal(tagsJSON, &j.VideoTags)
	}
	j.SeriesJSON = seriesJSON
	return &j, nil
}

// ListRecent includes each job's full series_json (unlike Analysis's
// equivalent list, which omits its larger blobs) — series results are small
// and this lets the frontend's history list show past results without a
// second round-trip per item.
func (s *VideoPromptSeriesStore) ListRecent(ctx context.Context, limit int) ([]models.VideoPromptSeriesJob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, video_title, video_description, video_tags, episode_count, status, error_message,
		        series_json, created_at, updated_at
		 FROM video_prompt_series ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.VideoPromptSeriesJob
	for rows.Next() {
		var j models.VideoPromptSeriesJob
		var description, errorMessage sql.NullString
		var tagsJSON, seriesJSON []byte
		if err := rows.Scan(&j.ID, &j.VideoTitle, &description, &tagsJSON, &j.EpisodeCount, &j.Status,
			&errorMessage, &seriesJSON, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		j.VideoDescription = description.String
		j.ErrorMessage = errorMessage.String
		if len(tagsJSON) > 0 {
			_ = json.Unmarshal(tagsJSON, &j.VideoTags)
		}
		j.SeriesJSON = seriesJSON
		results = append(results, j)
	}
	return results, rows.Err()
}
