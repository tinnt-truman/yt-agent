package trending

import (
	"testing"

	"ytagent/backend/internal/models"
)

func TestBuildReport_GroupsAndSortsByTotalViews(t *testing.T) {
	videos := []models.TrendingVideo{
		{ID: "v1", ChannelID: "c1", ChannelTitle: "Channel One", ViewCount: 1000},
		{ID: "v2", ChannelID: "c2", ChannelTitle: "Channel Two", ViewCount: 5000},
		{ID: "v3", ChannelID: "c1", ChannelTitle: "Channel One", ViewCount: 2000},
	}
	channelInfo := map[string]models.ChannelInfo{
		"c1": {Thumbnail: "thumb1", SubscriberCount: 100},
	}

	report := BuildReport("VN", videos, channelInfo)

	if report.Region != "VN" {
		t.Errorf("Region = %q, want VN", report.Region)
	}
	if len(report.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(report.Channels))
	}

	// c2 has higher total views (5000) than c1 (1000+2000=3000), so it sorts first.
	if report.Channels[0].ChannelID != "c2" {
		t.Errorf("Channels[0].ChannelID = %q, want c2", report.Channels[0].ChannelID)
	}
	if report.Channels[0].TotalViews != 5000 {
		t.Errorf("Channels[0].TotalViews = %d, want 5000", report.Channels[0].TotalViews)
	}

	c1 := report.Channels[1]
	if c1.ChannelID != "c1" || c1.TrendingVideoCount != 2 || c1.TotalViews != 3000 {
		t.Errorf("c1 = %+v, want {ChannelID:c1 TrendingVideoCount:2 TotalViews:3000 ...}", c1)
	}
	if c1.SubscriberCount != 100 || c1.ChannelThumbnail != "thumb1" {
		t.Errorf("c1 enrichment missing: %+v", c1)
	}
}

func TestBuildReport_Empty(t *testing.T) {
	report := BuildReport("US", nil, nil)
	if len(report.Channels) != 0 {
		t.Errorf("expected 0 channels, got %d", len(report.Channels))
	}
}
