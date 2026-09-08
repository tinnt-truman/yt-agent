package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"ytagent/backend/internal/ai"
	"ytagent/backend/internal/db"
	"ytagent/backend/internal/models"
	"ytagent/backend/internal/trending"
	"ytagent/backend/internal/youtube"
)

type TrendingHandler struct {
	settings *db.SettingsStore
}

func NewTrendingHandler(settings *db.SettingsStore) *TrendingHandler {
	return &TrendingHandler{settings: settings}
}

// GetTrending builds a live trending-channels report for one region from
// YouTube's "mostPopular" chart. Stateless — nothing is persisted, so
// re-requesting always reflects the current chart. Query params: region
// (default VN), category (video category ID, from GetCategories — empty
// means all categories), max (trending videos to sample, default 25, capped
// at 50).
func (h *TrendingHandler) GetTrending(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	if settings.YouTubeAPIKey == "" {
		writeError(w, http.StatusPreconditionFailed, "chưa cấu hình YouTube API key — vào trang Cài đặt trước")
		return
	}

	region := r.URL.Query().Get("region")
	if region == "" {
		region = "VN"
	}
	category := r.URL.Query().Get("category")
	maxResults := 25
	if v := r.URL.Query().Get("max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxResults = n
		}
	}

	ytClient := youtube.NewClient(settings.YouTubeAPIKey)

	videos, err := ytClient.FetchTrendingVideos(r.Context(), region, category, maxResults)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	channelIDs := make([]string, 0, len(videos))
	seen := make(map[string]bool, len(videos))
	for _, v := range videos {
		if !seen[v.ChannelID] {
			seen[v.ChannelID] = true
			channelIDs = append(channelIDs, v.ChannelID)
		}
	}

	channelInfo, err := ytClient.FetchChannelsBasic(r.Context(), channelIDs)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	report := trending.BuildReport(region, videos, channelInfo)
	writeJSON(w, http.StatusOK, report)
}

// GetCategories lists assignable video categories for a region, to populate
// the trending report's topic filter. Query param: region (default VN).
func (h *TrendingHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	if settings.YouTubeAPIKey == "" {
		writeError(w, http.StatusPreconditionFailed, "chưa cấu hình YouTube API key — vào trang Cài đặt trước")
		return
	}

	region := r.URL.Query().Get("region")
	if region == "" {
		region = "VN"
	}

	ytClient := youtube.NewClient(settings.YouTubeAPIKey)
	categories, err := ytClient.FetchVideoCategories(r.Context(), region)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if categories == nil {
		categories = []models.VideoCategory{}
	}
	writeJSON(w, http.StatusOK, categories)
}

// GenerateTrendingInsight takes a TrendingReport the client already fetched
// (via GetTrending) and asks DeepSeek to summarize it — kept separate from
// GetTrending so viewing the report never spends AI budget, only clicking
// "insight" does.
func (h *TrendingHandler) GenerateTrendingInsight(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}
	if settings.DeepSeekAPIKey == "" {
		writeError(w, http.StatusPreconditionFailed, "chưa cấu hình DeepSeek API key — vào trang Cài đặt trước")
		return
	}

	var report models.TrendingReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(report.Channels) == 0 {
		writeError(w, http.StatusBadRequest, "report has no channels")
		return
	}

	aiClient := ai.NewClient(settings.DeepSeekAPIKey, settings.AIModel)
	insight, err := aiClient.GenerateTrendingInsight(r.Context(), report)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, insight)
}
