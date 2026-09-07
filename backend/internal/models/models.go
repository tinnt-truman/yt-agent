package models

import (
	"encoding/json"
	"time"
)

type AnalysisStatus string

const (
	StatusPending    AnalysisStatus = "pending"
	StatusFetching   AnalysisStatus = "fetching"
	StatusAnalyzing  AnalysisStatus = "analyzing"
	StatusGenerating AnalysisStatus = "generating"
	StatusDone       AnalysisStatus = "done"
	StatusFailed     AnalysisStatus = "failed"
)

// Analysis is the persisted row for one analyze-a-channel job.
type Analysis struct {
	ID           string          `json:"id"`
	InputURL     string          `json:"inputUrl"`
	ChannelID    string          `json:"channelId,omitempty"`
	ChannelTitle string          `json:"channelTitle,omitempty"`
	Status       AnalysisStatus  `json:"status"`
	StageMessage string          `json:"stageMessage,omitempty"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	AnalysisJSON json.RawMessage `json:"analysis,omitempty"`
	AIOutputJSON json.RawMessage `json:"aiOutput,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

// Settings holds the app's single row of runtime configuration: API
// credentials and analysis/AI tunables, editable from the config page.
type Settings struct {
	YouTubeAPIKey   string    `json:"-"`
	AnthropicAPIKey string    `json:"-"`
	AIModel         string    `json:"aiModel"`
	MaxVideos       int       `json:"maxVideos"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// IsConfigured reports whether both required API keys have been set.
func (s Settings) IsConfigured() bool {
	return s.YouTubeAPIKey != "" && s.AnthropicAPIKey != ""
}

// ChannelInfo is the basic metadata fetched for a channel.
type ChannelInfo struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	CustomURL       string `json:"customUrl,omitempty"`
	PublishedAt     string `json:"publishedAt"`
	Country         string `json:"country,omitempty"`
	Thumbnail       string `json:"thumbnail,omitempty"`
	SubscriberCount int64  `json:"subscriberCount"`
	ViewCount       int64  `json:"viewCount"`
	VideoCount      int64  `json:"videoCount"`
	UploadsPlaylist string `json:"-"`
}

// VideoInfo is per-video data used for analysis.
type VideoInfo struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	PublishedAt  string   `json:"publishedAt"`
	Thumbnail    string   `json:"thumbnail,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	ViewCount    int64    `json:"viewCount"`
	LikeCount    int64    `json:"likeCount"`
	CommentCount int64    `json:"commentCount"`
	DurationSecs int64    `json:"durationSeconds"`
	IsShort      bool     `json:"isShort"`
}

// AnalysisResult is the deterministic (non-AI) statistical analysis of a channel.
type AnalysisResult struct {
	Channel       ChannelInfo   `json:"channel"`
	VideosSampled int           `json:"videosSampled"`
	Stats         ChannelStats  `json:"stats"`
	TopVideos     []VideoInfo   `json:"topVideos"`
	RecentVideos  []VideoInfo   `json:"recentVideos"`
	CommonTags    []TagCount    `json:"commonTags"`
	TitlePatterns TitlePatterns `json:"titlePatterns"`
}

type ChannelStats struct {
	AvgViews               float64        `json:"avgViews"`
	MedianViews            float64        `json:"medianViews"`
	AvgLikes               float64        `json:"avgLikes"`
	AvgComments            float64        `json:"avgComments"`
	AvgDurationSeconds     float64        `json:"avgDurationSeconds"`
	ShortsRatio            float64        `json:"shortsRatio"`
	UploadFrequencyPerWeek float64        `json:"uploadFrequencyPerWeek"`
	UploadDaysOfWeek       map[string]int `json:"uploadDaysOfWeek"`
	EngagementRate         float64        `json:"engagementRate"`
}

type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type TitlePatterns struct {
	AvgLength       float64  `json:"avgLength"`
	CommonWords     []string `json:"commonWords"`
	UsesNumbersPct  float64  `json:"usesNumbersPct"`
	UsesQuestionPct float64  `json:"usesQuestionPct"`
	UsesBracketsPct float64  `json:"usesBracketsPct"`
}

// StrategyOutput is the AI-generated content strategy for the new channel.
type StrategyOutput struct {
	Positioning     Positioning     `json:"positioning"`
	ContentPillars  []ContentPillar `json:"contentPillars"`
	ContentIdeas    []ContentIdea   `json:"contentIdeas"`
	PostingSchedule PostingSchedule `json:"postingSchedule"`
	TitleTemplates  []string        `json:"titleTemplates"`
	SeoKeywords     []string        `json:"seoKeywords"`
	HashtagSets     []HashtagSet    `json:"hashtagSets"`
	GrowthTactics   []string        `json:"growthTactics"`
	Risks           []string        `json:"risks"`
}

type Positioning struct {
	Niche            string   `json:"niche"`
	TargetAudience   string   `json:"targetAudience"`
	UniqueAngle      string   `json:"uniqueAngle"`
	ChannelNameIdeas []string `json:"channelNameIdeas"`
}

type ContentPillar struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Percentage  int    `json:"percentage"`
}

type ContentIdea struct {
	Title           string `json:"title"`
	Format          string `json:"format"`
	Hook            string `json:"hook"`
	Description     string `json:"description"`
	EstimatedLength string `json:"estimatedLength"`
}

type PostingSchedule struct {
	FrequencyPerWeek int      `json:"frequencyPerWeek"`
	BestDays         []string `json:"bestDays"`
	BestTimeOfDay    string   `json:"bestTimeOfDay"`
}

type HashtagSet struct {
	Theme    string   `json:"theme"`
	Hashtags []string `json:"hashtags"`
}
