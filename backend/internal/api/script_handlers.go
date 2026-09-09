package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ytagent/backend/internal/ai"
	"ytagent/backend/internal/db"
	"ytagent/backend/internal/models"
)

type ScriptHandler struct {
	settings *db.SettingsStore
	store    *db.ScriptStore
}

func NewScriptHandler(settings *db.SettingsStore, store *db.ScriptStore) *ScriptHandler {
	return &ScriptHandler{settings: settings, store: store}
}

type createScriptRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	Hook           string `json:"hook,omitempty"`
	DurationFormat string `json:"durationFormat"`
}

// CreateScript queues a new script-generation job and returns it immediately
// at "pending" — the actual AI call runs on the client's next
// AdvanceStep call, the same queued-step pattern the analyze pipeline uses
// (see worker.Worker) and for the same reason: Vercel's Go runtime doesn't
// keep goroutines alive once the HTTP response is sent, so nothing can run
// "in the background" from inside this handler.
func (h *ScriptHandler) CreateScript(w http.ResponseWriter, r *http.Request) {
	var req createScriptRequest
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

	settings, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	if settings.ActiveAIKey() == "" {
		writeError(w, http.StatusPreconditionFailed, "chưa cấu hình API key cho AI provider đang chọn — vào trang Cài đặt trước")
		return
	}

	id, err := h.store.Create(r.Context(), req.Title, req.Description, req.Hook, req.DurationFormat)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create script job")
		return
	}

	job, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch created script job")
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

// AdvanceStep runs the (single-stage) generation while the job is still
// pending, and is a safe no-op once it has finished or failed — the frontend
// calls this on every poll tick, mirroring AdvanceStep for analyses.
func (h *ScriptHandler) AdvanceStep(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "script job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch script job")
		return
	}

	if job.Status == models.ScriptStatusPending {
		h.runGeneration(r.Context(), job)
		job, err = h.store.GetByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not fetch script job")
			return
		}
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *ScriptHandler) runGeneration(ctx context.Context, job *models.ScriptJob) {
	settings, err := h.settings.Get(ctx)
	if err != nil {
		_ = h.store.SetFailed(ctx, job.ID, "could not load settings")
		return
	}

	aiClient := ai.NewClientFromSettings(settings)
	script, err := aiClient.GenerateScript(ctx, models.ScriptRequest{
		Title:          job.IdeaTitle,
		Description:    job.IdeaDescription,
		Hook:           job.IdeaHook,
		DurationFormat: job.DurationFormat,
	})
	if err != nil {
		_ = h.store.SetFailed(ctx, job.ID, err.Error())
		return
	}
	_ = h.store.SetDone(ctx, job.ID, *script)
}

func (h *ScriptHandler) GetScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "script job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch script job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *ScriptHandler) DeleteScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete script job")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ListScripts returns the generation history (newest first) so past scripts
// can be revisited without regenerating them.
func (h *ScriptHandler) ListScripts(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.ListRecent(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list script jobs")
		return
	}
	if items == nil {
		items = []models.ScriptJob{}
	}
	writeJSON(w, http.StatusOK, items)
}
