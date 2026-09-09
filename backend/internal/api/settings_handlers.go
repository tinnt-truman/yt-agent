package api

import (
	"encoding/json"
	"net/http"
	"time"

	"ytagent/backend/internal/ai"
	"ytagent/backend/internal/db"
)

type SettingsHandler struct {
	settings *db.SettingsStore
}

func NewSettingsHandler(settings *db.SettingsStore) *SettingsHandler {
	return &SettingsHandler{settings: settings}
}

type settingsResponse struct {
	Configured              bool      `json:"configured"`
	YouTubeAPIKeySet        bool      `json:"youtubeApiKeySet"`
	YouTubeAPIKeyPreview    string    `json:"youtubeApiKeyPreview,omitempty"`
	DeepSeekAPIKeySet       bool      `json:"deepseekApiKeySet"`
	DeepSeekAPIKeyPreview   string    `json:"deepseekApiKeyPreview,omitempty"`
	OpenRouterAPIKeySet     bool      `json:"openrouterApiKeySet"`
	OpenRouterAPIKeyPreview string    `json:"openrouterApiKeyPreview,omitempty"`
	AIProvider              string    `json:"aiProvider"`
	AIModel                 string    `json:"aiModel"`
	MaxVideos               int       `json:"maxVideos"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

func maskKey(key string) string {
	if len(key) <= 4 {
		return "••••"
	}
	return "••••" + key[len(key)-4:]
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}

	resp := settingsResponse{
		Configured: s.IsConfigured(),
		AIProvider: s.AIProviderResolved(),
		AIModel:    s.AIModel,
		MaxVideos:  s.MaxVideos,
		UpdatedAt:  s.UpdatedAt,
	}
	if s.YouTubeAPIKey != "" {
		resp.YouTubeAPIKeySet = true
		resp.YouTubeAPIKeyPreview = maskKey(s.YouTubeAPIKey)
	}
	if s.DeepSeekAPIKey != "" {
		resp.DeepSeekAPIKeySet = true
		resp.DeepSeekAPIKeyPreview = maskKey(s.DeepSeekAPIKey)
	}
	if s.OpenRouterAPIKey != "" {
		resp.OpenRouterAPIKeySet = true
		resp.OpenRouterAPIKeyPreview = maskKey(s.OpenRouterAPIKey)
	}
	writeJSON(w, http.StatusOK, resp)
}

type updateSettingsRequest struct {
	YouTubeAPIKey    *string `json:"youtubeApiKey"`
	DeepSeekAPIKey   *string `json:"deepseekApiKey"`
	OpenRouterAPIKey *string `json:"openrouterApiKey"`
	AIProvider       *string `json:"aiProvider"`
	AIModel          *string `json:"aiModel"`
	MaxVideos        *int    `json:"maxVideos"`
}

func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	patch := db.SettingsPatch{}
	// A blank string means "leave unchanged" — the UI never shows a saved key
	// back in full, so an empty submit must not wipe it out.
	if req.YouTubeAPIKey != nil && *req.YouTubeAPIKey != "" {
		patch.YouTubeAPIKey = req.YouTubeAPIKey
	}
	if req.DeepSeekAPIKey != nil && *req.DeepSeekAPIKey != "" {
		patch.DeepSeekAPIKey = req.DeepSeekAPIKey
	}
	if req.OpenRouterAPIKey != nil && *req.OpenRouterAPIKey != "" {
		patch.OpenRouterAPIKey = req.OpenRouterAPIKey
	}
	if req.AIProvider != nil && (*req.AIProvider == "deepseek" || *req.AIProvider == "openrouter") {
		patch.AIProvider = req.AIProvider
	}
	// An OpenRouter model implies the OpenRouter provider even if the UI only sent aiModel.
	if req.AIModel != nil && *req.AIModel != "" {
		patch.AIModel = req.AIModel
		if ai.ProviderForModel(*req.AIModel) == ai.ProviderOpenRouter {
			or := ai.ProviderOpenRouter
			patch.AIProvider = &or
		}
	}
	if req.MaxVideos != nil && *req.MaxVideos > 0 {
		patch.MaxVideos = req.MaxVideos
	}

	if _, err := h.settings.Update(r.Context(), patch); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update settings")
		return
	}

	h.GetSettings(w, r)
}
