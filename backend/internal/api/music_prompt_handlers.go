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

type MusicPromptHandler struct {
	settings *db.SettingsStore
	prompts  *db.MusicPromptStore
}

func NewMusicPromptHandler(settings *db.SettingsStore, prompts *db.MusicPromptStore) *MusicPromptHandler {
	return &MusicPromptHandler{settings: settings, prompts: prompts}
}

type createMusicPromptRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// CreateMusicPrompt queues a new Google Flow Music prompt job and returns it
// immediately at "pending" — the actual AI call runs on the client's next
// AdvanceStep call, the same queued-step pattern as ScriptHandler and
// VideoPromptHandler's series methods (see worker.Worker for why: Vercel's
// Go runtime doesn't keep goroutines alive once the HTTP response is sent).
func (h *MusicPromptHandler) CreateMusicPrompt(w http.ResponseWriter, r *http.Request) {
	var req createMusicPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
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

	id, err := h.prompts.Create(r.Context(), req.Title, req.Description, req.Tags)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create music prompt job")
		return
	}

	job, err := h.prompts.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch created music prompt job")
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

// AdvanceMusicPromptStep runs the (single-stage) generation while the job is
// still pending, and is a safe no-op once it has finished or failed — the
// frontend calls this on every poll tick, mirroring ScriptHandler.AdvanceStep.
func (h *MusicPromptHandler) AdvanceMusicPromptStep(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.prompts.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "music prompt job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch music prompt job")
		return
	}

	if job.Status == models.MusicPromptStatusPending {
		h.runGeneration(r.Context(), job)
		job, err = h.prompts.GetByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not fetch music prompt job")
			return
		}
	}
	writeJSON(w, http.StatusOK, job)
}

// RetryMusicPrompt puts a failed job back at "pending" (clearing its error)
// so the client's next Step call retries generation in place — reusing the
// same history entry instead of creating a duplicate one.
func (h *MusicPromptHandler) RetryMusicPrompt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.prompts.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "music prompt job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch music prompt job")
		return
	}
	if job.Status != models.MusicPromptStatusFailed {
		writeError(w, http.StatusConflict, "only a failed job can be retried")
		return
	}

	if err := h.prompts.ResetToPending(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reset music prompt job")
		return
	}
	job, err = h.prompts.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch music prompt job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *MusicPromptHandler) runGeneration(ctx context.Context, job *models.MusicPromptJob) {
	settings, err := h.settings.Get(ctx)
	if err != nil {
		_ = h.prompts.SetFailed(ctx, job.ID, "could not load settings")
		return
	}

	aiClient := ai.NewClientFromSettings(settings)
	prompt, err := aiClient.GenerateMusicPrompt(ctx, models.MusicPromptRequest{
		Title:       job.VideoTitle,
		Description: job.VideoDescription,
		Tags:        job.VideoTags,
	})
	if err != nil {
		_ = h.prompts.SetFailed(ctx, job.ID, err.Error())
		return
	}
	_ = h.prompts.SetDone(ctx, job.ID, *prompt)
}

func (h *MusicPromptHandler) GetMusicPrompt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.prompts.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "music prompt job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch music prompt job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *MusicPromptHandler) DeleteMusicPrompt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.prompts.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete music prompt job")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ListMusicPrompts returns the generation history (newest first) so past
// prompts can be revisited without regenerating them.
func (h *MusicPromptHandler) ListMusicPrompts(w http.ResponseWriter, r *http.Request) {
	items, err := h.prompts.ListRecent(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list music prompt jobs")
		return
	}
	if items == nil {
		items = []models.MusicPromptJob{}
	}
	writeJSON(w, http.StatusOK, items)
}
