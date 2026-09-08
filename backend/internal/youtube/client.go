// Package youtube resolves a YouTube video/channel URL and fetches bounded
// channel + video data via the YouTube Data API v3.
package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ytagent/backend/internal/models"
)

const apiBase = "https://www.googleapis.com/youtube/v3"

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ResolveChannelID accepts a video URL, channel URL, handle URL, or shorts
// URL and returns the underlying YouTube channel ID.
func (c *Client) ResolveChannelID(ctx context.Context, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Host == "" {
		// Allow bare handles like "@someone" or channel IDs pasted directly.
		if strings.HasPrefix(rawURL, "@") {
			return c.channelIDForHandle(ctx, rawURL)
		}
		if strings.HasPrefix(rawURL, "UC") {
			return rawURL, nil
		}
		return "", fmt.Errorf("could not parse a YouTube URL from %q", rawURL)
	}

	path := strings.Trim(u.Path, "/")
	segments := strings.Split(path, "/")

	switch {
	case strings.Contains(u.Host, "youtu.be"):
		if len(segments) >= 1 && segments[0] != "" {
			return c.channelIDForVideo(ctx, segments[0])
		}

	case len(segments) >= 2 && segments[0] == "watch":
		// handled below via query param

	case len(segments) >= 2 && (segments[0] == "shorts" || segments[0] == "v" || segments[0] == "embed"):
		return c.channelIDForVideo(ctx, segments[1])

	case len(segments) >= 2 && segments[0] == "channel":
		return segments[1], nil

	case len(segments) >= 2 && segments[0] == "user":
		return c.channelIDForUsername(ctx, segments[1])

	case len(segments) >= 2 && segments[0] == "c":
		return c.channelIDForCustomName(ctx, segments[1])

	case len(segments) >= 1 && strings.HasPrefix(segments[0], "@"):
		return c.channelIDForHandle(ctx, segments[0])
	}

	if v := u.Query().Get("v"); v != "" {
		return c.channelIDForVideo(ctx, v)
	}

	return "", fmt.Errorf("unrecognized YouTube URL format: %s", rawURL)
}

func (c *Client) channelIDForVideo(ctx context.Context, videoID string) (string, error) {
	var resp struct {
		Items []struct {
			Snippet struct {
				ChannelID string `json:"channelId"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := c.get(ctx, "/videos", url.Values{
		"part": {"snippet"},
		"id":   {videoID},
	}, &resp); err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", fmt.Errorf("video %q not found", videoID)
	}
	return resp.Items[0].Snippet.ChannelID, nil
}

func (c *Client) channelIDForHandle(ctx context.Context, handle string) (string, error) {
	if !strings.HasPrefix(handle, "@") {
		handle = "@" + handle
	}
	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := c.get(ctx, "/channels", url.Values{
		"part":      {"id"},
		"forHandle": {handle},
	}, &resp); err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", fmt.Errorf("channel handle %q not found", handle)
	}
	return resp.Items[0].ID, nil
}

func (c *Client) channelIDForUsername(ctx context.Context, username string) (string, error) {
	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := c.get(ctx, "/channels", url.Values{
		"part":        {"id"},
		"forUsername": {username},
	}, &resp); err != nil {
		return "", err
	}
	if len(resp.Items) > 0 {
		return resp.Items[0].ID, nil
	}
	// Legacy /user/ names often are also handles nowadays.
	return c.channelIDForHandle(ctx, username)
}

// channelIDForCustomName handles the legacy /c/CustomName URLs, which have no
// direct API lookup. We fall back to search.list, which is fuzzy: it returns
// the best-matching channel, not a guaranteed exact match.
func (c *Client) channelIDForCustomName(ctx context.Context, name string) (string, error) {
	// Custom URLs are frequently also valid handles today.
	if id, err := c.channelIDForHandle(ctx, name); err == nil {
		return id, nil
	}

	var resp struct {
		Items []struct {
			Snippet struct {
				ChannelID string `json:"channelId"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := c.get(ctx, "/search", url.Values{
		"part":       {"snippet"},
		"q":          {name},
		"type":       {"channel"},
		"maxResults": {"1"},
	}, &resp); err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", fmt.Errorf("could not resolve custom channel name %q", name)
	}
	return resp.Items[0].Snippet.ChannelID, nil
}

// FetchChannel returns the channel's metadata and its uploads playlist ID.
func (c *Client) FetchChannel(ctx context.Context, channelID string) (models.ChannelInfo, error) {
	var resp struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				CustomURL   string `json:"customUrl"`
				PublishedAt string `json:"publishedAt"`
				Country     string `json:"country"`
				Thumbnails  struct {
					High struct {
						URL string `json:"url"`
					} `json:"high"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			Statistics struct {
				SubscriberCount string `json:"subscriberCount"`
				ViewCount       string `json:"viewCount"`
				VideoCount      string `json:"videoCount"`
			} `json:"statistics"`
			ContentDetails struct {
				RelatedPlaylists struct {
					Uploads string `json:"uploads"`
				} `json:"relatedPlaylists"`
			} `json:"contentDetails"`
		} `json:"items"`
	}

	if err := c.get(ctx, "/channels", url.Values{
		"part": {"snippet,statistics,contentDetails"},
		"id":   {channelID},
	}, &resp); err != nil {
		return models.ChannelInfo{}, err
	}
	if len(resp.Items) == 0 {
		return models.ChannelInfo{}, fmt.Errorf("channel %q not found", channelID)
	}
	item := resp.Items[0]

	return models.ChannelInfo{
		ID:              item.ID,
		Title:           item.Snippet.Title,
		Description:     item.Snippet.Description,
		CustomURL:       item.Snippet.CustomURL,
		PublishedAt:     item.Snippet.PublishedAt,
		Country:         item.Snippet.Country,
		Thumbnail:       item.Snippet.Thumbnails.High.URL,
		SubscriberCount: parseInt64(item.Statistics.SubscriberCount),
		ViewCount:       parseInt64(item.Statistics.ViewCount),
		VideoCount:      parseInt64(item.Statistics.VideoCount),
		UploadsPlaylist: item.ContentDetails.RelatedPlaylists.Uploads,
	}, nil
}

// FetchRecentVideos returns up to maxVideos of the channel's most recent
// uploads with full statistics, newest first. Bounded to protect API quota.
func (c *Client) FetchRecentVideos(ctx context.Context, uploadsPlaylistID string, maxVideos int) ([]models.VideoInfo, error) {
	videoIDs, err := c.listPlaylistVideoIDs(ctx, uploadsPlaylistID, maxVideos)
	if err != nil {
		return nil, err
	}
	return c.fetchVideoDetails(ctx, videoIDs)
}

// FetchTrendingVideos returns YouTube's "mostPopular" chart for a region,
// optionally narrowed to one video category (categoryID from
// FetchVideoCategories; empty means all categories) — the closest thing the
// Data API offers to "what's trending right now". There is no separate
// "trending channels" endpoint; the report is built by grouping these
// videos by channel (see internal/trending). Capped at 50 (one page, one
// quota unit) since that's plenty for a report.
func (c *Client) FetchTrendingVideos(ctx context.Context, regionCode, categoryID string, maxResults int) ([]models.TrendingVideo, error) {
	if maxResults <= 0 || maxResults > 50 {
		maxResults = 50
	}

	var resp struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				ChannelID    string `json:"channelId"`
				ChannelTitle string `json:"channelTitle"`
				PublishedAt  string `json:"publishedAt"`
				CategoryID   string `json:"categoryId"`
				Thumbnails   struct {
					High struct {
						URL string `json:"url"`
					} `json:"high"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			Statistics struct {
				ViewCount    string `json:"viewCount"`
				LikeCount    string `json:"likeCount"`
				CommentCount string `json:"commentCount"`
			} `json:"statistics"`
		} `json:"items"`
	}

	params := url.Values{
		"part":       {"snippet,statistics"},
		"chart":      {"mostPopular"},
		"regionCode": {regionCode},
		"maxResults": {strconv.Itoa(maxResults)},
	}
	if categoryID != "" {
		params.Set("videoCategoryId", categoryID)
	}

	if err := c.get(ctx, "/videos", params, &resp); err != nil {
		return nil, err
	}

	videos := make([]models.TrendingVideo, 0, len(resp.Items))
	for _, item := range resp.Items {
		videos = append(videos, models.TrendingVideo{
			ID:           item.ID,
			Title:        item.Snippet.Title,
			Thumbnail:    item.Snippet.Thumbnails.High.URL,
			ChannelID:    item.Snippet.ChannelID,
			ChannelTitle: item.Snippet.ChannelTitle,
			ViewCount:    parseInt64(item.Statistics.ViewCount),
			LikeCount:    parseInt64(item.Statistics.LikeCount),
			CommentCount: parseInt64(item.Statistics.CommentCount),
			PublishedAt:  item.Snippet.PublishedAt,
			CategoryID:   item.Snippet.CategoryID,
		})
	}
	return videos, nil
}

// FetchVideoCategories lists YouTube's assignable video categories for a
// region (names are localized per region, and not every category is
// assignable in every region — e.g. some regions omit "Shows"), so the
// frontend's filter dropdown is built from this instead of a hardcoded list.
func (c *Client) FetchVideoCategories(ctx context.Context, regionCode string) ([]models.VideoCategory, error) {
	var resp struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title      string `json:"title"`
				Assignable bool   `json:"assignable"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := c.get(ctx, "/videoCategories", url.Values{
		"part":       {"snippet"},
		"regionCode": {regionCode},
	}, &resp); err != nil {
		return nil, err
	}

	categories := make([]models.VideoCategory, 0, len(resp.Items))
	for _, item := range resp.Items {
		if !item.Snippet.Assignable {
			continue
		}
		categories = append(categories, models.VideoCategory{
			ID:    item.ID,
			Title: item.Snippet.Title,
		})
	}
	return categories, nil
}

// FetchChannelsBasic returns basic public info (thumbnail, subscriber count)
// for a batch of channel IDs, keyed by channel ID. Used to enrich a trending
// report — one call for up to 50 channels.
func (c *Client) FetchChannelsBasic(ctx context.Context, channelIDs []string) (map[string]models.ChannelInfo, error) {
	result := make(map[string]models.ChannelInfo, len(channelIDs))
	if len(channelIDs) == 0 {
		return result, nil
	}

	for i := 0; i < len(channelIDs); i += 50 {
		end := i + 50
		if end > len(channelIDs) {
			end = len(channelIDs)
		}
		batch := channelIDs[i:end]

		var resp struct {
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
				Statistics struct {
					SubscriberCount string `json:"subscriberCount"`
				} `json:"statistics"`
			} `json:"items"`
		}

		if err := c.get(ctx, "/channels", url.Values{
			"part": {"snippet,statistics"},
			"id":   {strings.Join(batch, ",")},
		}, &resp); err != nil {
			return nil, err
		}

		for _, item := range resp.Items {
			result[item.ID] = models.ChannelInfo{
				ID:              item.ID,
				Title:           item.Snippet.Title,
				Thumbnail:       item.Snippet.Thumbnails.High.URL,
				SubscriberCount: parseInt64(item.Statistics.SubscriberCount),
			}
		}
	}
	return result, nil
}

func (c *Client) listPlaylistVideoIDs(ctx context.Context, playlistID string, maxVideos int) ([]string, error) {
	var ids []string
	pageToken := ""
	for len(ids) < maxVideos {
		pageSize := maxVideos - len(ids)
		if pageSize > 50 {
			pageSize = 50
		}
		params := url.Values{
			"part":       {"contentDetails"},
			"playlistId": {playlistID},
			"maxResults": {strconv.Itoa(pageSize)},
		}
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}

		var resp struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				ContentDetails struct {
					VideoID string `json:"videoId"`
				} `json:"contentDetails"`
			} `json:"items"`
		}
		if err := c.get(ctx, "/playlistItems", params, &resp); err != nil {
			return nil, err
		}
		for _, item := range resp.Items {
			ids = append(ids, item.ContentDetails.VideoID)
		}
		if resp.NextPageToken == "" || len(resp.Items) == 0 {
			break
		}
		pageToken = resp.NextPageToken
	}
	return ids, nil
}

func (c *Client) fetchVideoDetails(ctx context.Context, videoIDs []string) ([]models.VideoInfo, error) {
	var videos []models.VideoInfo
	for i := 0; i < len(videoIDs); i += 50 {
		end := i + 50
		if end > len(videoIDs) {
			end = len(videoIDs)
		}
		batch := videoIDs[i:end]

		var resp struct {
			Items []struct {
				ID      string `json:"id"`
				Snippet struct {
					Title       string   `json:"title"`
					Description string   `json:"description"`
					PublishedAt string   `json:"publishedAt"`
					Tags        []string `json:"tags"`
					Thumbnails  struct {
						High struct {
							URL string `json:"url"`
						} `json:"high"`
					} `json:"thumbnails"`
				} `json:"snippet"`
				Statistics struct {
					ViewCount    string `json:"viewCount"`
					LikeCount    string `json:"likeCount"`
					CommentCount string `json:"commentCount"`
				} `json:"statistics"`
				ContentDetails struct {
					Duration string `json:"duration"`
				} `json:"contentDetails"`
			} `json:"items"`
		}

		if err := c.get(ctx, "/videos", url.Values{
			"part": {"snippet,statistics,contentDetails"},
			"id":   {strings.Join(batch, ",")},
		}, &resp); err != nil {
			return nil, err
		}

		for _, item := range resp.Items {
			durationSecs := parseISO8601Duration(item.ContentDetails.Duration)
			videos = append(videos, models.VideoInfo{
				ID:           item.ID,
				Title:        item.Snippet.Title,
				Description:  item.Snippet.Description,
				PublishedAt:  item.Snippet.PublishedAt,
				Thumbnail:    item.Snippet.Thumbnails.High.URL,
				Tags:         item.Snippet.Tags,
				ViewCount:    parseInt64(item.Statistics.ViewCount),
				LikeCount:    parseInt64(item.Statistics.LikeCount),
				CommentCount: parseInt64(item.Statistics.CommentCount),
				DurationSecs: durationSecs,
				IsShort:      durationSecs > 0 && durationSecs <= 60,
			})
		}
	}
	return videos, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out interface{}) error {
	params.Set("key", c.apiKey)
	reqURL := apiBase + path + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("youtube api request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &apiErr)
		msg := apiErr.Error.Message
		if msg == "" {
			msg = string(body)
		}
		return fmt.Errorf("youtube api error (status %d): %s", resp.StatusCode, msg)
	}

	return json.Unmarshal(body, out)
}

func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

var iso8601DurationRe = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// parseISO8601Duration parses YouTube's contentDetails.duration (e.g. "PT4M13S").
func parseISO8601Duration(d string) int64 {
	matches := iso8601DurationRe.FindStringSubmatch(d)
	if matches == nil {
		return 0
	}
	hours := parseIntOrZero(matches[1])
	minutes := parseIntOrZero(matches[2])
	seconds := parseIntOrZero(matches[3])
	return int64(hours*3600 + minutes*60 + seconds)
}

func parseIntOrZero(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}
