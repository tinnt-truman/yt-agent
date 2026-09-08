// Package trending groups YouTube's "mostPopular" chart videos by channel to
// approximate a "trending channels" report — pure aggregation, no network
// I/O (that lives in internal/youtube).
package trending

import (
	"sort"
	"time"

	"ytagent/backend/internal/models"
)

// BuildReport groups trendingVideos by channel and sorts channels by total
// trending views (descending). channelInfo enriches each channel with its
// thumbnail/subscriber count when available — missing entries are fine, the
// channel just renders without them.
func BuildReport(region string, trendingVideos []models.TrendingVideo, channelInfo map[string]models.ChannelInfo) models.TrendingReport {
	byChannel := map[string]*models.TrendingChannel{}
	order := make([]string, 0)

	for _, v := range trendingVideos {
		ch, ok := byChannel[v.ChannelID]
		if !ok {
			ch = &models.TrendingChannel{
				ChannelID:    v.ChannelID,
				ChannelTitle: v.ChannelTitle,
			}
			if info, found := channelInfo[v.ChannelID]; found {
				ch.ChannelThumbnail = info.Thumbnail
				ch.SubscriberCount = info.SubscriberCount
			}
			byChannel[v.ChannelID] = ch
			order = append(order, v.ChannelID)
		}
		ch.Videos = append(ch.Videos, v)
		ch.TrendingVideoCount++
		ch.TotalViews += v.ViewCount
	}

	channels := make([]models.TrendingChannel, 0, len(order))
	for _, id := range order {
		channels = append(channels, *byChannel[id])
	}
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].TotalViews > channels[j].TotalViews
	})

	return models.TrendingReport{
		Region:      region,
		Channels:    channels,
		GeneratedAt: time.Now().UTC(),
	}
}
