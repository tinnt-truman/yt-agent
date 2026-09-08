// Package analytics queries the YouTube Analytics API v2 for a connected
// channel's private metrics (views, watch time, revenue) — data the public
// YouTube Data API key used elsewhere in this app can never see, which is
// the entire reason internal/googleoauth exists.
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ytagent/backend/internal/models"
)

const (
	reportsURL = "https://youtubeanalytics.googleapis.com/v2/reports"
	// WindowDays is the fixed lookback window for every snapshot. YouTube
	// Analytics data typically lags 1-2 days, so the window ends "yesterday".
	WindowDays = 28
)

var trafficSourceLabels = map[string]string{
	"ADVERTISING":                     "Quảng cáo",
	"ANNOTATION":                      "Chú thích video",
	"CAMPAIGN_CARD":                   "Chiến dịch",
	"END_SCREEN":                      "Màn hình kết thúc",
	"EXT_URL":                         "Link bên ngoài",
	"NOTIFICATION":                    "Thông báo",
	"NO_LINK_OTHER":                   "Khác",
	"PLAYLIST":                        "Playlist",
	"PROMOTED":                        "Quảng bá",
	"RELATED_VIDEO":                   "Video đề xuất",
	"SUBSCRIBER":                      "Trang chủ subscriber",
	"YT_CHANNEL":                      "Trang kênh",
	"YT_OTHER_PAGE":                   "Trang khác của YouTube",
	"YT_SEARCH":                       "Tìm kiếm YouTube",
	"SHORTS":                          "Shorts feed",
	"YT_PLAYLIST_PAGE":                "Trang playlist",
	"NOTIFICATION_ANDROID_UPLOAD_END": "Thông báo tải lên",
}

func trafficSourceLabel(code string) string {
	if label, ok := trafficSourceLabels[code]; ok {
		return label
	}
	return code
}

// FetchChannelAnalytics assembles a full snapshot: core metrics, traffic
// sources, top videos by watch time, and a best-effort revenue read. Each
// sub-query fails independently — a channel too new for impressions data,
// or not monetized, still gets a useful (partial) result instead of an
// all-or-nothing error.
func FetchChannelAnalytics(ctx context.Context, client *http.Client, dataAPIKey string) (models.ChannelAnalytics, error) {
	end := time.Now().AddDate(0, 0, -1)
	start := end.AddDate(0, 0, -WindowDays)
	startDate, endDate := start.Format("2006-01-02"), end.Format("2006-01-02")

	result := models.ChannelAnalytics{
		WindowDays:   WindowDays,
		Monetization: models.MonetizationUnknown,
	}

	core, err := queryReport(ctx, client, url.Values{
		"ids":       {"channel==MINE"},
		"startDate": {startDate},
		"endDate":   {endDate},
		"metrics":   {"views,estimatedMinutesWatched,averageViewDuration,subscribersGained,subscribersLost"},
	})
	if err != nil {
		return result, fmt.Errorf("core metrics: %w", err)
	}
	if row := core.firstRow(); row != nil {
		result.Views = row.int64At(core.colIndex("views"))
		result.EstimatedMinutesWatched = row.int64At(core.colIndex("estimatedMinutesWatched"))
		result.AverageViewDurationSecs = row.int64At(core.colIndex("averageViewDuration"))
		result.SubscribersGained = row.int64At(core.colIndex("subscribersGained"))
		result.SubscribersLost = row.int64At(core.colIndex("subscribersLost"))
	}

	// Impressions/CTR: not available for every channel (needs enough
	// homepage/browse traffic) — omit silently on failure.
	if impressions, err := queryReport(ctx, client, url.Values{
		"ids":       {"channel==MINE"},
		"startDate": {startDate},
		"endDate":   {endDate},
		"metrics":   {"impressions,impressionsClickThroughRate"},
	}); err == nil {
		if row := impressions.firstRow(); row != nil {
			result.Impressions = row.int64At(impressions.colIndex("impressions"))
			result.ImpressionsCTR = row.float64At(impressions.colIndex("impressionsClickThroughRate"))
		}
	}

	// Traffic sources, top 5 by views.
	if traffic, err := queryReport(ctx, client, url.Values{
		"ids":        {"channel==MINE"},
		"startDate":  {startDate},
		"endDate":    {endDate},
		"metrics":    {"views"},
		"dimensions": {"insightTrafficSourceType"},
		"sort":       {"-views"},
		"maxResults": {"5"},
	}); err == nil {
		var total int64
		type share struct {
			source string
			views  int64
		}
		var shares []share
		for _, row := range traffic.Rows {
			src := row.stringAt(traffic.colIndex("insightTrafficSourceType"))
			views := row.int64At(traffic.colIndex("views"))
			shares = append(shares, share{src, views})
			total += views
		}
		for _, sh := range shares {
			pct := 0.0
			if total > 0 {
				pct = float64(sh.views) / float64(total) * 100
			}
			result.TrafficSources = append(result.TrafficSources, models.TrafficSourceShare{
				Source:   trafficSourceLabel(sh.source),
				Views:    sh.views,
				SharePct: pct,
			})
		}
	}

	// Top videos by watch time, then enrich with title/thumbnail via the
	// public Data API (a video ID -> metadata lookup needs no OAuth scope).
	if topVideos, err := queryReport(ctx, client, url.Values{
		"ids":        {"channel==MINE"},
		"startDate":  {startDate},
		"endDate":    {endDate},
		"metrics":    {"estimatedMinutesWatched"},
		"dimensions": {"video"},
		"sort":       {"-estimatedMinutesWatched"},
		"maxResults": {"5"},
	}); err == nil {
		videoIDs := make([]string, 0, len(topVideos.Rows))
		minutesByID := map[string]int64{}
		for _, row := range topVideos.Rows {
			id := row.stringAt(topVideos.colIndex("video"))
			videoIDs = append(videoIDs, id)
			minutesByID[id] = row.int64At(topVideos.colIndex("estimatedMinutesWatched"))
		}
		titles, thumbs := fetchVideoMeta(ctx, dataAPIKey, videoIDs)
		for _, id := range videoIDs {
			result.TopVideos = append(result.TopVideos, models.TopVideoByWatchTime{
				VideoID:          id,
				Title:            titles[id],
				Thumbnail:        thumbs[id],
				EstimatedMinutes: minutesByID[id],
			})
		}
	}

	// Revenue: a separate, more sensitive scope. Failure here almost always
	// means "not monetized (yet)" or "no revenue-viewing permission on this
	// account" — both are useful, non-error information for the UI.
	if revenue, err := queryReport(ctx, client, url.Values{
		"ids":       {"channel==MINE"},
		"startDate": {startDate},
		"endDate":   {endDate},
		"metrics":   {"estimatedRevenue"},
	}); err != nil {
		result.Monetization = models.MonetizationDisabled
		result.RevenueNote = "Không lấy được dữ liệu doanh thu — kênh có thể chưa bật kiếm tiền, hoặc tài khoản Google này không có quyền xem doanh thu."
	} else if row := revenue.firstRow(); row != nil {
		v := row.float64At(revenue.colIndex("estimatedRevenue"))
		result.Monetization = models.MonetizationEnabled
		result.EstimatedRevenueUSD = &v
		result.RevenueNote = "Số liệu ước tính từ YouTube Analytics, có thể lệch so với báo cáo AdSense chính thức."
	}

	return result, nil
}

// --- reports.query plumbing ---

type reportResponse struct {
	ColumnHeaders []struct {
		Name string `json:"name"`
	} `json:"columnHeaders"`
	Rows []reportRow `json:"rows"`
}

type reportRow []any

func (r reportRow) int64At(i int) int64 {
	if i < 0 || i >= len(r) {
		return 0
	}
	switch v := r[i].(type) {
	case float64:
		return int64(v)
	case string:
		var n int64
		_, _ = fmt.Sscanf(v, "%d", &n)
		return n
	}
	return 0
}

func (r reportRow) float64At(i int) float64 {
	if i < 0 || i >= len(r) {
		return 0
	}
	if v, ok := r[i].(float64); ok {
		return v
	}
	return 0
}

func (r reportRow) stringAt(i int) string {
	if i < 0 || i >= len(r) {
		return ""
	}
	if v, ok := r[i].(string); ok {
		return v
	}
	return ""
}

func (rr *reportResponse) colIndex(name string) int {
	for i, h := range rr.ColumnHeaders {
		if h.Name == name {
			return i
		}
	}
	return -1
}

func (rr *reportResponse) firstRow() reportRow {
	if len(rr.Rows) == 0 {
		return nil
	}
	return rr.Rows[0]
}

func queryReport(ctx context.Context, client *http.Client, params url.Values) (*reportResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reportsURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("youtube analytics request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube analytics error (status %d): %s", resp.StatusCode, string(body))
	}

	var parsed reportResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse analytics response: %w", err)
	}
	return &parsed, nil
}

// fetchVideoMeta looks up title/thumbnail for a batch of video IDs using the
// public Data API key — never fails the caller; missing entries are simply
// left blank.
func fetchVideoMeta(ctx context.Context, apiKey string, videoIDs []string) (titles, thumbs map[string]string) {
	titles, thumbs = map[string]string{}, map[string]string{}
	if len(videoIDs) == 0 || apiKey == "" {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/youtube/v3/videos?part=snippet&id="+strings.Join(videoIDs, ",")+"&key="+url.QueryEscape(apiKey),
		nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	var parsed struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title      string `json:"title"`
				Thumbnails struct {
					High struct {
						URL string `json:"url"`
					} `json:"high"`
				} `json:"thumbnails"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return
	}
	for _, item := range parsed.Items {
		titles[item.ID] = item.Snippet.Title
		thumbs[item.ID] = item.Snippet.Thumbnails.High.URL
	}
	return
}
