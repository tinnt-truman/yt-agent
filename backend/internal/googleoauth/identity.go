package googleoauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type MyChannel struct {
	ChannelID       string
	Title           string
	Thumbnail       string
	SubscriberCount int64
}

// FetchMyChannel resolves the YouTube channel(s) managed by the
// authenticated Google account (channels.list?mine=true) and returns the
// first one — most individual creator accounts manage exactly one channel;
// Brand Account switching for multiple channels is a Google-account-level
// concept outside what this OAuth grant exposes per call.
func FetchMyChannel(ctx context.Context, client *http.Client) (*MyChannel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/youtube/v3/channels?part=snippet,statistics&mine=true", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("channels.list mine=true: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("channels.list mine=true (status %d): %s", resp.StatusCode, string(body))
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
			Statistics struct {
				SubscriberCount string `json:"subscriberCount"`
			} `json:"statistics"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse channels.list response: %w", err)
	}
	if len(parsed.Items) == 0 {
		return nil, fmt.Errorf("tài khoản Google này không quản lý kênh YouTube nào")
	}

	item := parsed.Items[0]
	var subs int64
	_, _ = fmt.Sscanf(item.Statistics.SubscriberCount, "%d", &subs)

	return &MyChannel{
		ChannelID:       item.ID,
		Title:           item.Snippet.Title,
		Thumbnail:       item.Snippet.Thumbnails.High.URL,
		SubscriberCount: subs,
	}, nil
}

// FetchUserEmail reads the connected Google account's email so the UI can
// show which account a channel is linked through.
func FetchUserEmail(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("userinfo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo (status %d): %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	return parsed.Email, nil
}
