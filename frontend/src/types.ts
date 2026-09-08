export type AnalysisStatus =
  | 'pending'
  | 'fetching'
  | 'analyzing'
  | 'generating'
  | 'done'
  | 'failed'

export interface Analysis {
  id: string
  inputUrl: string
  channelId?: string
  channelTitle?: string
  status: AnalysisStatus
  stageMessage?: string
  errorMessage?: string
  analysis?: AnalysisResult
  aiOutput?: StrategyOutput
  createdAt: string
  updatedAt: string
}

export interface ChannelInfo {
  id: string
  title: string
  description: string
  customUrl?: string
  publishedAt: string
  country?: string
  thumbnail?: string
  subscriberCount: number
  viewCount: number
  videoCount: number
}

export interface VideoInfo {
  id: string
  title: string
  description?: string
  publishedAt: string
  thumbnail?: string
  tags?: string[]
  viewCount: number
  likeCount: number
  commentCount: number
  durationSeconds: number
  isShort: boolean
}

export interface ChannelStats {
  avgViews: number
  medianViews: number
  avgLikes: number
  avgComments: number
  avgDurationSeconds: number
  shortsRatio: number
  uploadFrequencyPerWeek: number
  uploadDaysOfWeek: Record<string, number>
  engagementRate: number
}

export interface TagCount {
  tag: string
  count: number
}

export interface TitlePatterns {
  avgLength: number
  commonWords: string[]
  usesNumbersPct: number
  usesQuestionPct: number
  usesBracketsPct: number
}

export interface AnalysisResult {
  channel: ChannelInfo
  videosSampled: number
  stats: ChannelStats
  topVideos: VideoInfo[]
  recentVideos: VideoInfo[]
  commonTags: TagCount[]
  titlePatterns: TitlePatterns
}

export interface Positioning {
  niche: string
  targetAudience: string
  uniqueAngle: string
  channelNameIdeas: string[]
}

export interface ContentPillar {
  name: string
  description: string
  percentage: number
}

export interface ContentIdea {
  title: string
  format: string
  hook: string
  description: string
  estimatedLength: string
}

export interface PostingSchedule {
  frequencyPerWeek: number
  bestDays: string[]
  bestTimeOfDay: string
}

export interface HashtagSet {
  theme: string
  hashtags: string[]
}

export interface Settings {
  configured: boolean
  youtubeApiKeySet: boolean
  youtubeApiKeyPreview?: string
  deepseekApiKeySet: boolean
  deepseekApiKeyPreview?: string
  aiModel: string
  maxVideos: number
  updatedAt: string
}

export interface UpdateSettingsRequest {
  youtubeApiKey?: string
  deepseekApiKey?: string
  aiModel?: string
  maxVideos?: number
}

export interface StrategyOutput {
  positioning: Positioning
  contentPillars: ContentPillar[]
  contentIdeas: ContentIdea[]
  postingSchedule: PostingSchedule
  titleTemplates: string[]
  seoKeywords: string[]
  hashtagSets: HashtagSet[]
  growthTactics: string[]
  risks: string[]
}

export interface TrendingVideo {
  id: string
  title: string
  thumbnail?: string
  channelId: string
  channelTitle: string
  viewCount: number
  likeCount: number
  commentCount: number
  publishedAt: string
  categoryId?: string
}

export interface TrendingChannel {
  channelId: string
  channelTitle: string
  channelThumbnail?: string
  subscriberCount: number
  trendingVideoCount: number
  totalViews: number
  videos: TrendingVideo[]
}

export interface TrendingReport {
  region: string
  channels: TrendingChannel[]
  generatedAt: string
}

export interface VideoCategory {
  id: string
  title: string
}

export interface TrendingInsight {
  summary: string
  opportunities: string[]
}

export interface ConnectedChannel {
  id: string
  channelId: string
  channelTitle: string
  channelThumbnail?: string
  subscriberCount: number
  googleEmail?: string
  createdAt: string
  updatedAt: string
}

export type MonetizationStatus = 'enabled' | 'disabled' | 'unknown'

export interface TrafficSourceShare {
  source: string
  views: number
  sharePct: number
}

export interface TopVideoByWatchTime {
  videoId: string
  title: string
  thumbnail?: string
  estimatedMinutesWatched: number
}

export interface ChannelAnalytics {
  windowDays: number
  views: number
  estimatedMinutesWatched: number
  averageViewDurationSeconds: number
  subscribersGained: number
  subscribersLost: number
  impressions?: number
  impressionsCtr?: number
  trafficSources?: TrafficSourceShare[]
  topVideos?: TopVideoByWatchTime[]
  monetization: MonetizationStatus
  estimatedRevenueUsd?: number
  revenueNote?: string
}
