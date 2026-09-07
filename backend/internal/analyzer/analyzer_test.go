package analyzer

import (
	"testing"

	"ytagent/backend/internal/models"
)

func sampleVideos() []models.VideoInfo {
	return []models.VideoInfo{
		{
			ID: "v1", Title: "Top 10 Go Tips [2024]", ViewCount: 1000, LikeCount: 100, CommentCount: 10,
			DurationSecs: 600, PublishedAt: "2024-01-01T00:00:00Z", Tags: []string{"go", "golang", "tips"},
		},
		{
			ID: "v2", Title: "How to learn Go?", ViewCount: 500, LikeCount: 50, CommentCount: 5,
			DurationSecs: 45, IsShort: true, PublishedAt: "2024-01-08T00:00:00Z", Tags: []string{"go", "beginner"},
		},
		{
			ID: "v3", Title: "Golang Concurrency Explained", ViewCount: 2000, LikeCount: 200, CommentCount: 20,
			DurationSecs: 900, PublishedAt: "2024-01-15T00:00:00Z", Tags: []string{"go", "golang", "concurrency"},
		},
	}
}

func TestComputeStats(t *testing.T) {
	stats := computeStats(sampleVideos())

	if got, want := stats.AvgViews, (1000.0+500.0+2000.0)/3; got != want {
		t.Errorf("AvgViews = %v, want %v", got, want)
	}
	if stats.MedianViews != 1000 {
		t.Errorf("MedianViews = %v, want 1000", stats.MedianViews)
	}
	if stats.ShortsRatio < 0.33 || stats.ShortsRatio > 0.34 {
		t.Errorf("ShortsRatio = %v, want ~0.333", stats.ShortsRatio)
	}
	// 3 uploads spanning 14 days -> (3/14)*7 = 1.5/week
	if got, want := stats.UploadFrequencyPerWeek, 1.5; got != want {
		t.Errorf("UploadFrequencyPerWeek = %v, want %v", got, want)
	}
}

func TestTopTags(t *testing.T) {
	tags := topTags(sampleVideos(), 2)
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	if tags[0].Tag != "go" || tags[0].Count != 3 {
		t.Errorf("top tag = %+v, want {go 3}", tags[0])
	}
}

func TestAnalyzeTitles(t *testing.T) {
	patterns := analyzeTitles(sampleVideos())
	if patterns.UsesNumbersPct == 0 {
		t.Error("expected non-zero UsesNumbersPct (one title has '10' and '2024')")
	}
	if patterns.UsesQuestionPct == 0 {
		t.Error("expected non-zero UsesQuestionPct ('How to learn Go?')")
	}
	if patterns.UsesBracketsPct == 0 {
		t.Error("expected non-zero UsesBracketsPct ('[2024]')")
	}
}

func TestAnalyze_EmptyVideos(t *testing.T) {
	result := Analyze(models.ChannelInfo{ID: "c1"}, nil)
	if result.VideosSampled != 0 {
		t.Errorf("VideosSampled = %d, want 0", result.VideosSampled)
	}
	if result.Stats.UploadDaysOfWeek == nil {
		t.Error("UploadDaysOfWeek should be an empty map, not nil, for stable JSON output")
	}
}
