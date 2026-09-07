// Package analyzer computes deterministic statistics from fetched YouTube
// data. Pure functions, no network I/O.
package analyzer

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"ytagent/backend/internal/models"
)

const topN = 10

func Analyze(channel models.ChannelInfo, videos []models.VideoInfo) models.AnalysisResult {
	result := models.AnalysisResult{
		Channel:       channel,
		VideosSampled: len(videos),
		RecentVideos:  firstN(videos, topN),
		TopVideos:     topByViews(videos, topN),
		CommonTags:    topTags(videos, 20),
		TitlePatterns: analyzeTitles(videos),
		Stats:         computeStats(videos),
	}
	return result
}

func firstN(videos []models.VideoInfo, n int) []models.VideoInfo {
	if len(videos) <= n {
		return videos
	}
	return videos[:n]
}

func topByViews(videos []models.VideoInfo, n int) []models.VideoInfo {
	sorted := make([]models.VideoInfo, len(videos))
	copy(sorted, videos)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ViewCount > sorted[j].ViewCount })
	return firstN(sorted, n)
}

func topTags(videos []models.VideoInfo, n int) []models.TagCount {
	counts := map[string]int{}
	for _, v := range videos {
		for _, tag := range v.Tags {
			counts[strings.ToLower(tag)]++
		}
	}
	tagCounts := make([]models.TagCount, 0, len(counts))
	for tag, count := range counts {
		tagCounts = append(tagCounts, models.TagCount{Tag: tag, Count: count})
	}
	sort.Slice(tagCounts, func(i, j int) bool {
		if tagCounts[i].Count != tagCounts[j].Count {
			return tagCounts[i].Count > tagCounts[j].Count
		}
		return tagCounts[i].Tag < tagCounts[j].Tag
	})
	if len(tagCounts) > n {
		tagCounts = tagCounts[:n]
	}
	return tagCounts
}

var wordRe = regexp.MustCompile(`[a-zA-ZÀ-ỹ0-9]+`)
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "of": true, "to": true, "in": true,
	"is": true, "for": true, "on": true, "with": true, "va": true, "la": true, "cua": true,
}

func analyzeTitles(videos []models.VideoInfo) models.TitlePatterns {
	if len(videos) == 0 {
		return models.TitlePatterns{}
	}

	var totalLen int
	var withNumbers, withQuestion, withBrackets int
	wordFreq := map[string]int{}

	for _, v := range videos {
		title := v.Title
		totalLen += len(title)

		if regexp.MustCompile(`\d`).MatchString(title) {
			withNumbers++
		}
		if strings.Contains(title, "?") {
			withQuestion++
		}
		if strings.ContainsAny(title, "[](){}") {
			withBrackets++
		}

		for _, w := range wordRe.FindAllString(strings.ToLower(title), -1) {
			if len(w) < 3 || stopWords[w] {
				continue
			}
			wordFreq[w]++
		}
	}

	type wc struct {
		word  string
		count int
	}
	words := make([]wc, 0, len(wordFreq))
	for w, c := range wordFreq {
		words = append(words, wc{w, c})
	}
	sort.Slice(words, func(i, j int) bool {
		if words[i].count != words[j].count {
			return words[i].count > words[j].count
		}
		return words[i].word < words[j].word
	})
	commonWords := make([]string, 0, 15)
	for i := 0; i < len(words) && i < 15; i++ {
		commonWords = append(commonWords, words[i].word)
	}

	n := float64(len(videos))
	return models.TitlePatterns{
		AvgLength:       float64(totalLen) / n,
		CommonWords:     commonWords,
		UsesNumbersPct:  100 * float64(withNumbers) / n,
		UsesQuestionPct: 100 * float64(withQuestion) / n,
		UsesBracketsPct: 100 * float64(withBrackets) / n,
	}
}

func computeStats(videos []models.VideoInfo) models.ChannelStats {
	if len(videos) == 0 {
		return models.ChannelStats{UploadDaysOfWeek: map[string]int{}}
	}

	n := float64(len(videos))
	var sumViews, sumLikes, sumComments, sumDuration float64
	var shorts int
	views := make([]float64, 0, len(videos))
	dayCounts := map[string]int{}
	var timestamps []time.Time

	for _, v := range videos {
		sumViews += float64(v.ViewCount)
		sumLikes += float64(v.LikeCount)
		sumComments += float64(v.CommentCount)
		sumDuration += float64(v.DurationSecs)
		views = append(views, float64(v.ViewCount))
		if v.IsShort {
			shorts++
		}
		if t, err := time.Parse(time.RFC3339, v.PublishedAt); err == nil {
			dayCounts[t.Weekday().String()]++
			timestamps = append(timestamps, t)
		}
	}

	sort.Float64s(views)
	median := views[len(views)/2]
	if len(views)%2 == 0 && len(views) > 1 {
		median = (views[len(views)/2-1] + views[len(views)/2]) / 2
	}

	uploadFrequencyPerWeek := 0.0
	if len(timestamps) >= 2 {
		sort.Slice(timestamps, func(i, j int) bool { return timestamps[i].Before(timestamps[j]) })
		spanDays := timestamps[len(timestamps)-1].Sub(timestamps[0]).Hours() / 24
		if spanDays > 0 {
			uploadFrequencyPerWeek = (float64(len(timestamps)) / spanDays) * 7
		}
	}

	engagementRate := 0.0
	if sumViews > 0 {
		engagementRate = (sumLikes + sumComments) / sumViews
	}

	return models.ChannelStats{
		AvgViews:               sumViews / n,
		MedianViews:            median,
		AvgLikes:               sumLikes / n,
		AvgComments:            sumComments / n,
		AvgDurationSeconds:     sumDuration / n,
		ShortsRatio:            float64(shorts) / n,
		UploadFrequencyPerWeek: uploadFrequencyPerWeek,
		UploadDaysOfWeek:       dayCounts,
		EngagementRate:         engagementRate,
	}
}
