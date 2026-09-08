package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"ytagent/backend/internal/models"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateAnalysis(ctx context.Context, inputURL string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO analyses (input_url, status) VALUES ($1, $2) RETURNING id`,
		inputURL, models.StatusPending,
	).Scan(&id)
	return id, err
}

func (s *Store) UpdateStage(ctx context.Context, id string, status models.AnalysisStatus, stageMessage string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE analyses SET status = $1, stage_message = $2, updated_at = now() WHERE id = $3`,
		status, stageMessage, id,
	)
	return err
}

func (s *Store) SetChannelInfo(ctx context.Context, id, channelID, channelTitle string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE analyses SET channel_id = $1, channel_title = $2, updated_at = now() WHERE id = $3`,
		channelID, channelTitle, id,
	)
	return err
}

func (s *Store) SetAnalysisJSON(ctx context.Context, id string, result models.AnalysisResult) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE analyses SET analysis_json = $1, updated_at = now() WHERE id = $2`,
		raw, id,
	)
	return err
}

func (s *Store) SetAIOutput(ctx context.Context, id string, output models.StrategyOutput) error {
	raw, err := json.Marshal(output)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE analyses SET ai_output_json = $1, updated_at = now() WHERE id = $2`,
		raw, id,
	)
	return err
}

func (s *Store) SetFailed(ctx context.Context, id, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE analyses SET status = $1, error_message = $2, updated_at = now() WHERE id = $3`,
		models.StatusFailed, errMsg, id,
	)
	return err
}

func (s *Store) DeleteAnalysis(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM analyses WHERE id = $1`, id)
	return err
}

func (s *Store) GetByID(ctx context.Context, id string) (*models.Analysis, error) {
	var a models.Analysis
	var channelID, channelTitle, stageMessage, errorMessage sql.NullString
	var analysisJSON, aiOutputJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, input_url, channel_id, channel_title, status, stage_message, error_message,
		        analysis_json, ai_output_json, created_at, updated_at
		 FROM analyses WHERE id = $1`, id,
	).Scan(&a.ID, &a.InputURL, &channelID, &channelTitle, &a.Status, &stageMessage, &errorMessage,
		&analysisJSON, &aiOutputJSON, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}

	a.ChannelID = channelID.String
	a.ChannelTitle = channelTitle.String
	a.StageMessage = stageMessage.String
	a.ErrorMessage = errorMessage.String
	a.AnalysisJSON = analysisJSON
	a.AIOutputJSON = aiOutputJSON
	return &a, nil
}

func (s *Store) ListRecent(ctx context.Context, limit int) ([]models.Analysis, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, input_url, channel_id, channel_title, status, stage_message, error_message,
		        created_at, updated_at
		 FROM analyses ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.Analysis
	for rows.Next() {
		var a models.Analysis
		var channelID, channelTitle, stageMessage, errorMessage sql.NullString
		if err := rows.Scan(&a.ID, &a.InputURL, &channelID, &channelTitle, &a.Status, &stageMessage,
			&errorMessage, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.ChannelID = channelID.String
		a.ChannelTitle = channelTitle.String
		a.StageMessage = stageMessage.String
		a.ErrorMessage = errorMessage.String
		results = append(results, a)
	}
	return results, rows.Err()
}
