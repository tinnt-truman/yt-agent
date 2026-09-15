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

// RetryScript regenerates a script in place — same history entry, fresh AI
// call with the original idea/title/hook. Works for a "done" job (re-roll a
// result the user doesn't like) as well as a "failed" one (retry after an
// error); a job still "pending" is left alone since it's already generating.
func (h *ScriptHandler) RetryScript(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusConflict, "script job is already generating")
		return
	}

	if err := h.store.ResetToPending(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reset script job")
		return
	}
	job, err = h.store.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch script job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// ReviseScript queues a human-written revision note against an already
// generated script — the next Step call asks the AI to edit the existing
// script accordingly instead of generating a fresh one (see runGeneration).
// Unlike RetryScript (from-scratch regeneration), this requires a script to
// already exist, since there's nothing to revise otherwise.
func (h *ScriptHandler) ReviseScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req reviseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Feedback = strings.TrimSpace(req.Feedback)
	if req.Feedback == "" {
		writeError(w, http.StatusBadRequest, "feedback is required")
		return
	}

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
		writeError(w, http.StatusConflict, "script job is already generating")
		return
	}
	if len(job.ScriptJSON) == 0 {
		writeError(w, http.StatusConflict, "script must be generated at least once before it can be revised")
		return
	}

	if err := h.store.SetRevisionPending(r.Context(), id, req.Feedback); err != nil {
		writeError(w, http.StatusInternalServerError, "could not queue script revision")
		return
	}
	job, err = h.store.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch script job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// UpdateScript overwrites a script with one a human edited directly in the
// UI — saved synchronously (no AI call, no pending step) since there's
// nothing to generate. Works from any status: fixing up a "failed" job this
// way is as valid as touching up a "done" one.
func (h *ScriptHandler) UpdateScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var script models.Script
	if err := json.NewDecoder(r.Body).Decode(&script); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(script.Scenes) == 0 {
		writeError(w, http.StatusBadRequest, "script must have at least one scene")
		return
	}

	if err := h.store.SetDone(r.Context(), id, script); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save script")
		return
	}
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

// runGeneration either revises the existing script per a queued
// RevisionFeedback note, or — the common case — generates a fresh one from
// the job's idea/title/hook. Set by ReviseScript vs. CreateScript/
// RetryScript respectively; cleared again by ScriptStore.SetDone once
// applied, so it never lingers into an unrelated later regeneration.
func (h *ScriptHandler) runGeneration(ctx context.Context, job *models.ScriptJob) {
	settings, err := h.settings.Get(ctx)
	if err != nil {
		_ = h.store.SetFailed(ctx, job.ID, "could not load settings")
		return
	}
	aiClient := ai.NewClientFromSettings(settings)

	if job.RevisionFeedback != "" && len(job.ScriptJSON) > 0 {
		var existing models.Script
		if err := json.Unmarshal(job.ScriptJSON, &existing); err != nil {
			_ = h.store.SetFailed(ctx, job.ID, "could not read existing script for revision")
			return
		}
		revised, err := aiClient.ReviseScript(ctx, existing, job.RevisionFeedback)
		if err != nil {
			_ = h.store.SetFailed(ctx, job.ID, err.Error())
			return
		}
		_ = h.store.SetDone(ctx, job.ID, *revised)
		return
	}

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
