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

type VideoPromptHandler struct {
	settings *db.SettingsStore
	series   *db.VideoPromptSeriesStore
}

func NewVideoPromptHandler(settings *db.SettingsStore, series *db.VideoPromptSeriesStore) *VideoPromptHandler {
	return &VideoPromptHandler{settings: settings, series: series}
}

// GeneratePrompt turns one video's metadata (title/description/tags — the
// caller already has this from an analysis result or a trending report, so
// no YouTube API call happens here) into an AI text-to-video prompt for
// producing a new video on the same topic. Synchronous and unpersisted on
// purpose: this is a single quick clip prompt used internally (e.g. by the
// script modal's "open Kling/Google Flow" action), unlike the multi-episode
// series below, which is queued and tracked.
func (h *VideoPromptHandler) GeneratePrompt(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	if settings.ActiveAIKey() == "" {
		writeError(w, http.StatusPreconditionFailed, "chưa cấu hình API key cho AI provider đang chọn — vào trang Cài đặt trước")
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

	aiClient := ai.NewClientFromSettings(settings)
	prompt, err := aiClient.GenerateVideoPrompt(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prompt)
}

type createVideoPromptSeriesRequest struct {
	Title            string   `json:"title"`
	Description      string   `json:"description,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	EpisodeCount     int      `json:"episodeCount,omitempty"`
	ScenesPerEpisode int      `json:"scenesPerEpisode,omitempty"`
}

// CreateVideoPromptSeries queues a new multi-episode-story job and returns
// it immediately at "pending" — the actual AI call runs on the client's next
// AdvanceStep call, the same queued-step pattern as ScriptHandler (see
// worker.Worker for why: Vercel's Go runtime doesn't keep goroutines alive
// once the HTTP response is sent).
func (h *VideoPromptHandler) CreateVideoPromptSeries(w http.ResponseWriter, r *http.Request) {
	var req createVideoPromptSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	episodeCount, scenesPerEpisode := ai.ClampVideoPromptSeriesDimensions(req.EpisodeCount, req.ScenesPerEpisode)

	settings, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	if settings.ActiveAIKey() == "" {
		writeError(w, http.StatusPreconditionFailed, "chưa cấu hình API key cho AI provider đang chọn — vào trang Cài đặt trước")
		return
	}

	id, err := h.series.Create(r.Context(), req.Title, req.Description, req.Tags, episodeCount, scenesPerEpisode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create video prompt series job")
		return
	}

	job, err := h.series.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch created video prompt series job")
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

// AdvanceVideoPromptSeriesStep runs the (single-stage) generation while the
// job is still pending, and is a safe no-op once it has finished or failed —
// the frontend calls this on every poll tick, mirroring ScriptHandler's
// AdvanceStep.
func (h *VideoPromptHandler) AdvanceVideoPromptSeriesStep(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.series.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "video prompt series job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch video prompt series job")
		return
	}

	if job.Status == models.VideoPromptSeriesStatusPending {
		h.runSeriesGeneration(r.Context(), job)
		job, err = h.series.GetByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not fetch video prompt series job")
			return
		}
	}
	writeJSON(w, http.StatusOK, job)
}

// RetryVideoPromptSeries puts a job back at "pending" (clearing its error)
// so the client's next Step call regenerates it in place — same history
// entry, fresh AI call — instead of creating a duplicate one. Works from
// "done" (re-roll a result the user doesn't like) as well as "failed"
// (retry after an error); a job still "pending" is left alone since it's
// already generating.
func (h *VideoPromptHandler) RetryVideoPromptSeries(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.series.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "video prompt series job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch video prompt series job")
		return
	}
	if job.Status == models.VideoPromptSeriesStatusPending {
		writeError(w, http.StatusConflict, "video prompt series job is already generating")
		return
	}

	if err := h.series.ResetToPending(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reset video prompt series job")
		return
	}
	job, err = h.series.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch video prompt series job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *VideoPromptHandler) runSeriesGeneration(ctx context.Context, job *models.VideoPromptSeriesJob) {
	settings, err := h.settings.Get(ctx)
	if err != nil {
		_ = h.series.SetFailed(ctx, job.ID, "could not load settings")
		return
	}

	aiClient := ai.NewClientFromSettings(settings)
	series, err := aiClient.GenerateVideoPromptSeries(ctx, models.VideoPromptSeriesRequest{
		Title:            job.VideoTitle,
		Description:      job.VideoDescription,
		Tags:             job.VideoTags,
		EpisodeCount:     job.EpisodeCount,
		ScenesPerEpisode: job.ScenesPerEpisode,
	})
	if err != nil {
		_ = h.series.SetFailed(ctx, job.ID, err.Error())
		return
	}
	_ = h.series.SetDone(ctx, job.ID, *series)
}

func (h *VideoPromptHandler) GetVideoPromptSeries(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.series.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "video prompt series job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch video prompt series job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *VideoPromptHandler) DeleteVideoPromptSeries(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.series.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete video prompt series job")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ListVideoPromptSeries returns the generation history (newest first) so
// past series can be revisited without regenerating them.
func (h *VideoPromptHandler) ListVideoPromptSeries(w http.ResponseWriter, r *http.Request) {
	items, err := h.series.ListRecent(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list video prompt series jobs")
		return
	}
	if items == nil {
		items = []models.VideoPromptSeriesJob{}
	}
	writeJSON(w, http.StatusOK, items)
}
