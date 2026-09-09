package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds boot-time settings that must come from the environment
// because they're needed before the database is reachable. Everything else
// (API keys, AI model, sample size) lives in the settings table and is
// editable from the config page — see internal/db.SettingsStore.
type Config struct {
	Port        string
	DatabaseURL string
	// CORSOrigins restricts Access-Control-Allow-Origin to a fixed set of
	// origins (e.g. the frontend's Vercel URL(s)). Empty means "allow any
	// origin" — fine for local dev, and also a reasonable default in
	// production here since the API has no cookie/session auth to protect.
	CORSOrigins []string

	// AppPassword gates every /api/* route (except /api/auth/login and
	// /healthz) behind a single shared password — this app has no user
	// accounts, just one operator, so a shared secret is the right amount of
	// auth. Empty disables the gate entirely (local dev default).
	AppPassword string

	// Seed* values pre-populate the settings table on first boot only, so
	// existing setups that already export these env vars keep working
	// without visiting the config page. They are never read again after that.
	SeedYouTubeAPIKey    string
	SeedDeepSeekAPIKey   string
	SeedOpenRouterAPIKey string
	SeedNineRouterAPIKey string
	SeedAIProvider       string
	SeedAIModel          string
	SeedMaxVideos        int

	// Google OAuth ("Kênh của tôi" — connect a real YouTube channel for
	// private Analytics/revenue data). Optional at boot: the feature just
	// returns a clear error until these are set, same pattern as the app
	// running without AI keys until /config is visited.
	GoogleClientID     string
	GoogleClientSecret string
	// GoogleRedirectURL must exactly match a redirect URI registered on the
	// OAuth client in Google Cloud Console (this backend's own callback URL,
	// e.g. https://yt-agent-backend.vercel.app/api/oauth/google/callback).
	GoogleRedirectURL string
	// FrontendURL is where the browser is sent back to after the OAuth
	// callback completes (e.g. https://yt-agent-fe.vercel.app).
	FrontendURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		CORSOrigins:          parseCSV(os.Getenv("CORS_ORIGIN")),
		AppPassword:          os.Getenv("APP_PASSWORD"),
		SeedYouTubeAPIKey:    os.Getenv("YOUTUBE_API_KEY"),
		SeedDeepSeekAPIKey:   os.Getenv("DEEPSEEK_API_KEY"),
		SeedOpenRouterAPIKey: firstNonEmpty(os.Getenv("OPENROUTER_API_KEY"), os.Getenv("OPENCODE_API_KEY")),
		SeedNineRouterAPIKey: os.Getenv("NINEROUTER_API_KEY"),
		SeedAIProvider:       getEnv("AI_PROVIDER", ""),
		SeedAIModel:          getEnv("AI_MODEL", "deepseek-v4-pro"),
		SeedMaxVideos:        50,

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
		FrontendURL:        os.Getenv("FRONTEND_URL"),
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// parseCSV splits a comma-separated env var into trimmed, non-empty values.
// Lets CORS_ORIGIN carry more than one allowed origin, e.g. both a custom
// domain and the *.vercel.app project domain.
func parseCSV(v string) []string {
	if v == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
