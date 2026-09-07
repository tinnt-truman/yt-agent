package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ytagent/backend/internal/db"
	"ytagent/backend/internal/models"
	"ytagent/backend/internal/worker"
)

type Handler struct {
	store    *db.Store
	settings *db.SettingsStore
	worker   *worker.Worker
}

func NewHandler(store *db.Store, settings *db.SettingsStore, w *worker.Worker) *Handler {
	return &Handler{store: store, settings: settings, worker: w}
}

type createAnalysisRequest struct {
	URL string `json:"url"`
}

type createAnalysisResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (h *Handler) CreateAnalysis(w http.ResponseWriter, r *http.Request) {
	var req createAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	settings, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	if !settings.IsConfigured() {
		writeError(w, http.StatusPreconditionFailed, "chưa cấu hình API key — vào trang Cài đặt trước")
		return
	}

	id, err := h.store.CreateAnalysis(r.Context(), req.URL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create analysis")
		return
	}

	// Run the first (fast) stage inline instead of in a background goroutine:
	// the target deployment (Vercel's Go runtime) does not keep goroutines
	// alive once the HTTP response is sent. The slow stage (the AI call) is
	// left for the client to advance via AdvanceStep.
	h.worker.Step(r.Context(), id)

	analysis, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch created analysis")
		return
	}
	writeJSON(w, http.StatusAccepted, analysis)
}

// AdvanceStep pushes one analysis job forward by exactly one pipeline stage.
// The frontend calls this on every poll tick while a job is still in
// progress — see worker.Step for what each stage does.
func (h *Handler) AdvanceStep(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.worker.Step(r.Context(), id)

	analysis, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "analysis not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch analysis")
		return
	}
	writeJSON(w, http.StatusOK, analysis)
}

func (h *Handler) GetAnalysis(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	analysis, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "analysis not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch analysis")
		return
	}
	writeJSON(w, http.StatusOK, analysis)
}

func (h *Handler) ListAnalyses(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.ListRecent(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list analyses")
		return
	}
	if items == nil {
		items = []models.Analysis{}
	}
	writeJSON(w, http.StatusOK, items)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
