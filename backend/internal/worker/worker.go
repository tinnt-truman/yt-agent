// Package worker runs the analyze-channel pipeline in the background so HTTP
// handlers can respond immediately with a job ID.
package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"ytagent/backend/internal/ai"
	"ytagent/backend/internal/analyzer"
	"ytagent/backend/internal/db"
	"ytagent/backend/internal/models"
	"ytagent/backend/internal/youtube"
)

type Worker struct {
	store    *db.Store
	settings *db.SettingsStore
}

func New(store *db.Store, settings *db.SettingsStore) *Worker {
	return &Worker{store: store, settings: settings}
}

// RunAnalysis executes the full pipeline for one job: resolve -> fetch ->
// analyze -> generate strategy -> persist. Runs in its own goroutine with a
// bounded timeout; failures are recorded on the row, never panic the caller.
func (w *Worker) RunAnalysis(id, inputURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := w.run(ctx, id, inputURL); err != nil {
		log.Printf("analysis %s failed: %v", id, err)
		if setErr := w.store.SetFailed(ctx, id, err.Error()); setErr != nil {
			log.Printf("analysis %s: failed to persist error: %v", id, setErr)
		}
	}
}

func (w *Worker) run(ctx context.Context, id, inputURL string) error {
	settings, err := w.settings.Get(ctx)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	if !settings.IsConfigured() {
		return fmt.Errorf("thiếu YouTube API key hoặc Anthropic API key — vào trang Cài đặt để cấu hình")
	}
	ytClient := youtube.NewClient(settings.YouTubeAPIKey)
	aiClient := ai.NewClient(settings.AnthropicAPIKey, settings.AIModel)
	maxVideos := settings.MaxVideos
	if maxVideos <= 0 {
		maxVideos = 50
	}

	if err := w.store.UpdateStage(ctx, id, models.StatusFetching, "Đang xác định kênh YouTube..."); err != nil {
		return err
	}

	channelID, err := ytClient.ResolveChannelID(ctx, inputURL)
	if err != nil {
		return err
	}

	channel, err := ytClient.FetchChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if err := w.store.SetChannelInfo(ctx, id, channel.ID, channel.Title); err != nil {
		return err
	}

	if err := w.store.UpdateStage(ctx, id, models.StatusFetching, "Đang lấy danh sách video..."); err != nil {
		return err
	}
	videos, err := ytClient.FetchRecentVideos(ctx, channel.UploadsPlaylist, maxVideos)
	if err != nil {
		return err
	}

	if err := w.store.UpdateStage(ctx, id, models.StatusAnalyzing, "Đang phân tích số liệu kênh..."); err != nil {
		return err
	}
	result := analyzer.Analyze(channel, videos)
	if err := w.store.SetAnalysisJSON(ctx, id, result); err != nil {
		return err
	}

	if err := w.store.UpdateStage(ctx, id, models.StatusGenerating, "Đang tạo chiến lược nội dung với AI..."); err != nil {
		return err
	}
	strategy, err := aiClient.GenerateStrategy(ctx, result)
	if err != nil {
		return err
	}
	if err := w.store.SetAIOutput(ctx, id, *strategy); err != nil {
		return err
	}

	return w.store.UpdateStage(ctx, id, models.StatusDone, "Hoàn tất")
}
