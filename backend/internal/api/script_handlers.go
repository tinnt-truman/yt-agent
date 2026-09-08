package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"ytagent/backend/internal/ai"
	"ytagent/backend/internal/db"
	"ytagent/backend/internal/models"
)

type ScriptHandler struct {
	settings *db.SettingsStore
}

func NewScriptHandler(settings *db.SettingsStore) *ScriptHandler {
	return &ScriptHandler{settings: settings}
}

// GenerateScript turns one content idea (title/description/hook — the
// caller already has this from a generated strategy) into a scene-by-scene
// video script for the requested duration format.
func (h *ScriptHandler) GenerateScript(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	if settings.DeepSeekAPIKey == "" {
		writeError(w, http.StatusPreconditionFailed, "chưa cấu hình DeepSeek API key — vào trang Cài đặt trước")
		return
	}

	var req models.ScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.DurationFormat != "long" && req.DurationFormat != "short" {
		writeError(w, http.StatusBadRequest, "durationFormat must be \"long\" or \"short\"")
		return
	}

	aiClient := ai.NewClient(settings.DeepSeekAPIKey, settings.AIModel)
	script, err := aiClient.GenerateScript(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, script)
}
