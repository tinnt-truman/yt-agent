// Package worker runs the analyze-channel pipeline one step per call so it
// works the same on a persistent server and on a serverless platform (e.g.
// Vercel's Go runtime), which does not keep goroutines alive after an HTTP
// handler returns. The caller (an HTTP handler) drives progress by calling
// Step repeatedly — see internal/api's CreateAnalysis and AdvanceStep.
package worker

import (
	"context"
	"encoding/json"
	"fmt"

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

// Step advances one analysis job by exactly one stage and returns. It is
// idempotent to call on a job that has already finished or failed — it's a
// no-op in that case. On any error it records the failure on the row itself
// rather than returning it, so a caller can always just re-fetch the row
// afterwards to see what happened.
//
//   - pending    -> resolves the channel, fetches videos, computes the
//     deterministic analysis, and leaves the job at "generating".
//     This step does real network I/O but no LLM call, so it
//     normally finishes in a few seconds.
//   - generating -> calls the configured AI provider to produce the strategy
//     at "done". Isolated in its own step because it's the one
//     stage whose duration depends on the model, not our code.
//   - fetching / analyzing / done / failed -> no-op (already advanced, or
//     terminal).
func (w *Worker) Step(ctx context.Context, id string) {
	analysis, err := w.store.GetByID(ctx, id)
	if err != nil {
		return
	}

	switch analysis.Status {
	case models.StatusPending:
		w.stepFetchAndAnalyze(ctx, id, analysis.InputURL)
	case models.StatusGenerating:
		w.stepGenerateStrategy(ctx, id)
	}
}

func (w *Worker) stepFetchAndAnalyze(ctx context.Context, id, inputURL string) {
	settings, err := w.settings.Get(ctx)
	if err != nil {
		w.fail(ctx, id, fmt.Errorf("load settings: %w", err))
		return
	}
	if !settings.IsConfigured() {
		w.fail(ctx, id, fmt.Errorf("thiếu YouTube API key hoặc API key cho AI provider đang chọn — vào trang Cài đặt để cấu hình"))
		return
	}
	maxVideos := settings.MaxVideos
	if maxVideos <= 0 {
		maxVideos = 50
	}
	ytClient := youtube.NewClient(settings.YouTubeAPIKey)

	if err := w.store.UpdateStage(ctx, id, models.StatusFetching, "Đang xác định kênh YouTube..."); err != nil {
		return
	}

	channelID, err := ytClient.ResolveChannelID(ctx, inputURL)
	if err != nil {
		w.fail(ctx, id, err)
		return
	}

	channel, err := ytClient.FetchChannel(ctx, channelID)
	if err != nil {
		w.fail(ctx, id, err)
		return
	}
	if err := w.store.SetChannelInfo(ctx, id, channel.ID, channel.Title); err != nil {
		w.fail(ctx, id, err)
		return
	}

	if err := w.store.UpdateStage(ctx, id, models.StatusFetching, "Đang lấy danh sách video..."); err != nil {
		return
	}
	videos, err := ytClient.FetchRecentVideos(ctx, channel.UploadsPlaylist, maxVideos)
	if err != nil {
		w.fail(ctx, id, err)
		return
	}

	if err := w.store.UpdateStage(ctx, id, models.StatusAnalyzing, "Đang phân tích số liệu kênh..."); err != nil {
		return
	}
	result := analyzer.Analyze(channel, videos)
	if err := w.store.SetAnalysisJSON(ctx, id, result); err != nil {
		w.fail(ctx, id, err)
		return
	}

	w.store.UpdateStage(ctx, id, models.StatusGenerating, "Đang tạo chiến lược nội dung với AI...")
}

func (w *Worker) stepGenerateStrategy(ctx context.Context, id string) {
	settings, err := w.settings.Get(ctx)
	if err != nil {
		w.fail(ctx, id, fmt.Errorf("load settings: %w", err))
		return
	}

	analysis, err := w.store.GetByID(ctx, id)
	if err != nil {
		return
	}
	var result models.AnalysisResult
	if err := decodeJSON(analysis.AnalysisJSON, &result); err != nil {
		w.fail(ctx, id, fmt.Errorf("decode stored analysis: %w", err))
		return
	}

	aiClient := ai.NewClientFromSettings(settings)
	strategy, err := aiClient.GenerateStrategy(ctx, result)
	if err != nil {
		w.fail(ctx, id, err)
		return
	}
	if err := w.store.SetAIOutput(ctx, id, *strategy); err != nil {
		w.fail(ctx, id, err)
		return
	}

	w.store.UpdateStage(ctx, id, models.StatusDone, "Hoàn tất")
}

func (w *Worker) fail(ctx context.Context, id string, err error) {
	_ = w.store.SetFailed(ctx, id, err.Error())
}

func decodeJSON(raw json.RawMessage, out any) error {
	if len(raw) == 0 {
		return fmt.Errorf("no data stored")
	}
	return json.Unmarshal(raw, out)
}
