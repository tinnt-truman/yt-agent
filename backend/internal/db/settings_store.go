package db

import (
	"context"
	"database/sql"

	"ytagent/backend/internal/models"
)

type SettingsStore struct {
	db *sql.DB
}

func NewSettingsStore(db *sql.DB) *SettingsStore {
	return &SettingsStore{db: db}
}

func (s *SettingsStore) Get(ctx context.Context) (models.Settings, error) {
	var out models.Settings
	err := s.db.QueryRowContext(ctx,
		`SELECT youtube_api_key, anthropic_api_key, ai_model, max_videos, updated_at FROM settings WHERE id = 1`,
	).Scan(&out.YouTubeAPIKey, &out.AnthropicAPIKey, &out.AIModel, &out.MaxVideos, &out.UpdatedAt)
	return out, err
}

// SettingsPatch carries only the fields the client wants to change. A nil
// pointer means "leave as-is" — in particular, an empty API key field in the
// UI must never silently wipe out a previously saved key.
type SettingsPatch struct {
	YouTubeAPIKey   *string
	AnthropicAPIKey *string
	AIModel         *string
	MaxVideos       *int
}

func (s *SettingsStore) Update(ctx context.Context, patch SettingsPatch) (models.Settings, error) {
	current, err := s.Get(ctx)
	if err != nil {
		return models.Settings{}, err
	}

	if patch.YouTubeAPIKey != nil {
		current.YouTubeAPIKey = *patch.YouTubeAPIKey
	}
	if patch.AnthropicAPIKey != nil {
		current.AnthropicAPIKey = *patch.AnthropicAPIKey
	}
	if patch.AIModel != nil {
		current.AIModel = *patch.AIModel
	}
	if patch.MaxVideos != nil {
		current.MaxVideos = *patch.MaxVideos
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE settings SET youtube_api_key = $1, anthropic_api_key = $2, ai_model = $3, max_videos = $4, updated_at = now() WHERE id = 1`,
		current.YouTubeAPIKey, current.AnthropicAPIKey, current.AIModel, current.MaxVideos,
	)
	if err != nil {
		return models.Settings{}, err
	}
	return s.Get(ctx)
}

// SeedFromEnv fills in empty credential/model fields from environment values
// on first boot only — it never overwrites a value already saved via the
// config page.
func (s *SettingsStore) SeedFromEnv(ctx context.Context, youtubeKey, anthropicKey, aiModel string, maxVideos int) error {
	current, err := s.Get(ctx)
	if err != nil {
		return err
	}

	patch := SettingsPatch{}
	if current.YouTubeAPIKey == "" && youtubeKey != "" {
		patch.YouTubeAPIKey = &youtubeKey
	}
	if current.AnthropicAPIKey == "" && anthropicKey != "" {
		patch.AnthropicAPIKey = &anthropicKey
	}
	if current.AIModel == "" && aiModel != "" {
		patch.AIModel = &aiModel
	}
	if current.MaxVideos == 0 && maxVideos != 0 {
		patch.MaxVideos = &maxVideos
	}
	if patch.YouTubeAPIKey == nil && patch.AnthropicAPIKey == nil && patch.AIModel == nil && patch.MaxVideos == nil {
		return nil
	}

	_, err = s.Update(ctx, patch)
	return err
}
