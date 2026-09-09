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
	YouTubeAPIKey    string    `json:"-"`
	DeepSeekAPIKey   string    `json:"-"`
	OpenRouterAPIKey string    `json:"-"`
	AIProvider       string    `json:"aiProvider"`
	AIModel          string    `json:"aiModel"`
	MaxVideos        int       `json:"maxVideos"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// AIProviderResolved normalizes the provider, defaulting old rows to deepseek.
func (s Settings) AIProviderResolved() string {
	if s.AIProvider == "openrouter" {
		return "openrouter"
	}
	return "deepseek"
}

// ActiveAIKey returns the API key for the currently selected provider.
func (s Settings) ActiveAIKey() string {
	if s.AIProviderResolved() == "openrouter" {
		return s.OpenRouterAPIKey
	}
	return s.DeepSeekAPIKey
}

// IsConfigured reports whether YouTube plus the active AI provider's key
// have been set. A key for the inactive provider alone is not enough.
func (s Settings) IsConfigured() bool {
	return s.YouTubeAPIKey != "" && s.ActiveAIKey() != ""
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

// VideoCategory is one assignable YouTube video category (e.g. "Gaming",
// "Music") for a region — used to populate the trending report's topic
// filter, since category IDs are stable but titles are localized per region.
type VideoCategory struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// TrendingVideo is one video from YouTube's "mostPopular" chart for a region.
type TrendingVideo struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Thumbnail    string `json:"thumbnail,omitempty"`
	ChannelID    string `json:"channelId"`
	ChannelTitle string `json:"channelTitle"`
	ViewCount    int64  `json:"viewCount"`
	LikeCount    int64  `json:"likeCount"`
	CommentCount int64  `json:"commentCount"`
	PublishedAt  string `json:"publishedAt"`
	CategoryID   string `json:"categoryId,omitempty"`
}

// TrendingChannel aggregates the trending videos that belong to one channel,
// enriched with the channel's own public stats.
type TrendingChannel struct {
	ChannelID          string          `json:"channelId"`
	ChannelTitle       string          `json:"channelTitle"`
	ChannelThumbnail   string          `json:"channelThumbnail,omitempty"`
	SubscriberCount    int64           `json:"subscriberCount"`
	TrendingVideoCount int             `json:"trendingVideoCount"`
	TotalViews         int64           `json:"totalViews"`
	Videos             []TrendingVideo `json:"videos"`
}

// TrendingReport is the full trending-channels report for one region.
type TrendingReport struct {
	Region      string            `json:"region"`
	Channels    []TrendingChannel `json:"channels"`
	GeneratedAt time.Time         `json:"generatedAt"`
}

// TrendingInsight is an optional AI-generated commentary over a TrendingReport.
type TrendingInsight struct {
	Summary       string   `json:"summary"`
	Opportunities []string `json:"opportunities"`
}

// ConnectedChannel is a YouTube channel the operator has linked via Google
// OAuth, granting access to that channel's private YouTube Analytics data
// (not just what the public Data API exposes). Tokens are never serialized
// to JSON — see the API layer for the redacted response shape.
type ConnectedChannel struct {
	ID               string    `json:"id"`
	ChannelID        string    `json:"channelId"`
	ChannelTitle     string    `json:"channelTitle"`
	ChannelThumbnail string    `json:"channelThumbnail,omitempty"`
	SubscriberCount  int64     `json:"subscriberCount"`
	GoogleEmail      string    `json:"googleEmail,omitempty"`
	AccessToken      string    `json:"-"`
	RefreshToken     string    `json:"-"`
	TokenExpiry      time.Time `json:"-"`
	Scopes           string    `json:"-"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// MonetizationStatus reflects what we could actually determine about a
// channel's monetization — YouTube's APIs expose no direct "is monetized"
// field, so this is inferred from whether the revenue report query succeeds.
type MonetizationStatus string

const (
	MonetizationEnabled  MonetizationStatus = "enabled"
	MonetizationDisabled MonetizationStatus = "disabled"
	MonetizationUnknown  MonetizationStatus = "unknown"
)

// TrafficSourceShare is one row of the traffic-source breakdown.
type TrafficSourceShare struct {
	Source   string  `json:"source"`
	Views    int64   `json:"views"`
	SharePct float64 `json:"sharePct"`
}

// TopVideoByWatchTime is one row of the "top content" list, ranked by
// estimated minutes watched over the report window.
type TopVideoByWatchTime struct {
	VideoID          string `json:"videoId"`
	Title            string `json:"title"`
	Thumbnail        string `json:"thumbnail,omitempty"`
	EstimatedMinutes int64  `json:"estimatedMinutesWatched"`
}

// ChannelAnalytics is a private YouTube Analytics snapshot for one connected
// channel over a fixed recent window (see analytics.WindowDays).
type ChannelAnalytics struct {
	WindowDays              int                   `json:"windowDays"`
	Views                   int64                 `json:"views"`
	EstimatedMinutesWatched int64                 `json:"estimatedMinutesWatched"`
	AverageViewDurationSecs int64                 `json:"averageViewDurationSeconds"`
	SubscribersGained       int64                 `json:"subscribersGained"`
	SubscribersLost         int64                 `json:"subscribersLost"`
	Impressions             int64                 `json:"impressions,omitempty"`
	ImpressionsCTR          float64               `json:"impressionsCtr,omitempty"`
	TrafficSources          []TrafficSourceShare  `json:"trafficSources,omitempty"`
	TopVideos               []TopVideoByWatchTime `json:"topVideos,omitempty"`
	Monetization            MonetizationStatus    `json:"monetization"`
	EstimatedRevenueUSD     *float64              `json:"estimatedRevenueUsd,omitempty"`
	RevenueNote             string                `json:"revenueNote,omitempty"`
}

// VideoPromptRequest carries the metadata of one existing video (from an
// analysis result or a trending report — both cover the same fields) used
// as the reference for generating a new text-to-video prompt.
type VideoPromptRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// VideoPrompt is an AI-generated prompt for text-to-video tools (Kling,
// Runway, Sora, ...) meant to produce a new video on the same topic/theme as
// the reference video — not a copy of it.
type VideoPrompt struct {
	// Prompt is written in English on purpose: current text-to-video models
	// are trained overwhelmingly on English captions and follow English
	// prompts more reliably than Vietnamese ones.
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negativePrompt,omitempty"`
	Style          string `json:"style"`
	DurationHint   string `json:"durationHint"`
}

// VideoPromptSeriesRequest carries the metadata of one existing video plus
// how many episodes to break the new, original story into.
type VideoPromptSeriesRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	EpisodeCount int      `json:"episodeCount,omitempty"`
}

// VideoPromptEpisode is one episode's text-to-video prompt within a
// multi-episode story. Each episode is meant to become its own
// separately-generated, separately-uploaded video (Tập 1, Tập 2, ...) —
// current text-to-video tools only produce short clips per call, so a
// longer serialized story is told across several of them rather than one.
type VideoPromptEpisode struct {
	EpisodeNumber  int    `json:"episodeNumber"`
	Title          string `json:"title"`
	PlotSummary    string `json:"plotSummary"`
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negativePrompt,omitempty"`
}

// Character is one recurring character in a multi-episode story. Appearance
// is in English and detailed enough to paste into a text-to-video prompt at
// each episode so the character stays visually consistent across
// separately-generated clips; the rest are narrative attributes (in
// Vietnamese) to keep characterization consistent too.
type Character struct {
	Name         string   `json:"name"`
	Role         string   `json:"role"` // vd: "Nhân vật chính diện", "Phản diện", "Nhân vật phụ"
	Appearance   string   `json:"appearance"`
	CoreTags     []string `json:"coreTags"`
	PersonalInfo string   `json:"personalInfo"`
	Personality  string   `json:"personality"`
}

// VideoPromptSeries is an AI-generated ORIGINAL multi-episode story arc —
// inspired by, not a reproduction of, the reference video's topic/mood —
// with a cast of recurring Characters so each separately-generated episode
// clip stays visually and narratively consistent, plus one text-to-video
// prompt per episode.
type VideoPromptSeries struct {
	Synopsis     string               `json:"synopsis"`
	Characters   []Character          `json:"characters"`
	Style        string               `json:"style"`
	DurationHint string               `json:"durationHint"`
	Episodes     []VideoPromptEpisode `json:"episodes"`
}

type VideoPromptSeriesJobStatus string

const (
	VideoPromptSeriesStatusPending VideoPromptSeriesJobStatus = "pending"
	VideoPromptSeriesStatusDone    VideoPromptSeriesJobStatus = "done"
	VideoPromptSeriesStatusFailed  VideoPromptSeriesJobStatus = "failed"
)

// VideoPromptSeriesJob is the persisted row for one generate-a-video-prompt-
// series job. Queued and advanced one step at a time — like ScriptJob —
// rather than generated synchronously in the create request, so a slow AI
// call never ties up that request, and so the frontend gets a persisted
// history of past series to revisit without regenerating them.
type VideoPromptSeriesJob struct {
	ID               string                     `json:"id"`
	VideoTitle       string                     `json:"videoTitle"`
	VideoDescription string                     `json:"videoDescription,omitempty"`
	VideoTags        []string                   `json:"videoTags,omitempty"`
	EpisodeCount     int                        `json:"episodeCount"`
	Status           VideoPromptSeriesJobStatus `json:"status"`
	ErrorMessage     string                     `json:"errorMessage,omitempty"`
	SeriesJSON       json.RawMessage            `json:"series,omitempty"`
	CreatedAt        time.Time                  `json:"createdAt"`
	UpdatedAt        time.Time                  `json:"updatedAt"`
}

// ScriptRequest carries one content idea (from a generated strategy) used as
// the basis for a scene-by-scene video script.
type ScriptRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	Hook           string `json:"hook,omitempty"`
	DurationFormat string `json:"durationFormat"` // "long" (5-10 phút) or "short" (60 giây)
}

// ScriptScene is one scene of a generated video script.
type ScriptScene struct {
	Timecode  string `json:"timecode"`
	Visual    string `json:"visual"`
	Voiceover string `json:"voiceover,omitempty"`
}

// Script is an AI-generated scene-by-scene script for producing one video —
// either a long-form video (5-10 minutes) or a Short (60 seconds).
type Script struct {
	Hook           string        `json:"hook"`
	Scenes         []ScriptScene `json:"scenes"`
	CallToAction   string        `json:"callToAction"`
	DurationFormat string        `json:"durationFormat"`
}

type ScriptJobStatus string

const (
	ScriptStatusPending ScriptJobStatus = "pending"
	ScriptStatusDone    ScriptJobStatus = "done"
	ScriptStatusFailed  ScriptJobStatus = "failed"
)

// ScriptJob is the persisted row for one generate-a-script job. Queued and
// advanced one step at a time — like Analysis — rather than generated
// synchronously in the create request, so a slow AI call never ties up
// that request (and matters even more on Vercel's Go runtime, which doesn't
// keep goroutines alive once the HTTP response is sent). Also gives the
// frontend a persisted history of past scripts to revisit without
// regenerating them.
type ScriptJob struct {
	ID              string          `json:"id"`
	IdeaTitle       string          `json:"ideaTitle"`
	IdeaDescription string          `json:"ideaDescription,omitempty"`
	IdeaHook        string          `json:"ideaHook,omitempty"`
	DurationFormat  string          `json:"durationFormat"`
	Status          ScriptJobStatus `json:"status"`
	ErrorMessage    string          `json:"errorMessage,omitempty"`
	ScriptJSON      json.RawMessage `json:"script,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}
