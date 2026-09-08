package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"ytagent/backend/internal/ai"
	"ytagent/backend/internal/db"
	"ytagent/backend/internal/models"
)

type VideoPromptHandler struct {
	settings *db.SettingsStore
}

func NewVideoPromptHandler(settings *db.SettingsStore) *VideoPromptHandler {
	return &VideoPromptHandler{settings: settings}
}

// GeneratePrompt turns one video's metadata (title/description/tags — the
// caller already has this from an analysis result or a trending report, so
// no YouTube API call happens here) into an AI text-to-video prompt for
// producing a new video on the same topic.
func (h *VideoPromptHandler) GeneratePrompt(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	if settings.DeepSeekAPIKey == "" {
		writeError(w, http.StatusPreconditionFailed, "chưa cấu hình DeepSeek API key — vào trang Cài đặt trước")
		return
	}

	var req models.VideoPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	aiClient := ai.NewClient(settings.DeepSeekAPIKey, settings.AIModel)
	prompt, err := aiClient.GenerateVideoPrompt(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prompt)
}
