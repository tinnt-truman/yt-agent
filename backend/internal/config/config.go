package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds boot-time settings that must come from the environment
// because they're needed before the database is reachable. Everything else
// (API keys, AI model, sample size) lives in the settings table and is
// editable from the config page — see internal/db.SettingsStore.
type Config struct {
	Port        string
	DatabaseURL string
	// CORSOrigin restricts Access-Control-Allow-Origin to a single origin
	// (e.g. the frontend's Vercel URL). Empty means "*" — fine for local dev,
	// but worth locking down once frontend and backend are deployed separately.
	CORSOrigin string

	// Seed* values pre-populate the settings table on first boot only, so
	// existing setups that already export these env vars keep working
	// without visiting the config page. They are never read again after that.
	SeedYouTubeAPIKey  string
	SeedDeepSeekAPIKey string
	SeedAIModel        string
	SeedMaxVideos      int
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		CORSOrigin:         os.Getenv("CORS_ORIGIN"),
		SeedYouTubeAPIKey:  os.Getenv("YOUTUBE_API_KEY"),
		SeedDeepSeekAPIKey: os.Getenv("DEEPSEEK_API_KEY"),
		SeedAIModel:        getEnv("AI_MODEL", "deepseek-v4-pro"),
		SeedMaxVideos:      50,
	}

	if v := os.Getenv("MAX_VIDEOS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.SeedMaxVideos = n
		}
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
