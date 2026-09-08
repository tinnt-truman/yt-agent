package api

import (
	"net/http"
	"net/url"

	"golang.org/x/oauth2"

	"ytagent/backend/internal/config"
	"ytagent/backend/internal/db"
	"ytagent/backend/internal/googleoauth"
	"ytagent/backend/internal/models"
)

type OAuthHandler struct {
	cfg      *config.Config
	channels *db.ConnectedChannelStore
}

func NewOAuthHandler(cfg *config.Config, channels *db.ConnectedChannelStore) *OAuthHandler {
	return &OAuthHandler{cfg: cfg, channels: channels}
}

// GetAuthURL returns the Google consent URL for the frontend to navigate the
// full browser tab to. It cannot be a redirect the frontend "just links to"
// with our own app password attached — a page navigation carries no
// Authorization header — so the frontend calls this (authenticated, like
// any other API request) first and then does the navigation itself.
func (h *OAuthHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	oauthCfg, err := googleoauth.NewOAuthConfig(h.cfg)
	if err != nil {
		writeError(w, http.StatusPreconditionFailed, err.Error())
		return
	}
	state, err := googleoauth.NewState(h.cfg.GoogleClientSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create oauth state")
		return
	}
	authURL := oauthCfg.AuthCodeURL(state,
		// offline + consent: without both, Google only issues a refresh
		// token on the very first-ever consent for that account, silently
		// omitting it on any later re-auth — and Analytics calls need one
		// indefinitely, not just for the current access token's lifetime.
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	writeJSON(w, http.StatusOK, map[string]string{"url": authURL})
}

// Callback is Google redirecting the user's browser back to us — no
// Authorization header is possible here, so this route is intentionally
// registered outside requireAuth; the signed `state` param is what proves
// the request followed a GetAuthURL call from this backend.
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		h.redirectWithError(w, r, "Google từ chối cấp quyền: "+errMsg)
		return
	}

	oauthCfg, err := googleoauth.NewOAuthConfig(h.cfg)
	if err != nil {
		h.redirectWithError(w, r, err.Error())
		return
	}

	state := r.URL.Query().Get("state")
	if err := googleoauth.VerifyState(h.cfg.GoogleClientSecret, state); err != nil {
		h.redirectWithError(w, r, "Phiên kết nối không hợp lệ hoặc đã hết hạn, hãy thử lại.")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		h.redirectWithError(w, r, "Thiếu mã xác thực từ Google.")
		return
	}

	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		h.redirectWithError(w, r, "Không đổi được mã xác thực: "+err.Error())
		return
	}

	client := googleoauth.HTTPClient(ctx, oauthCfg, token)

	myChannel, err := googleoauth.FetchMyChannel(ctx, client)
	if err != nil {
		h.redirectWithError(w, r, err.Error())
		return
	}
	email, _ := googleoauth.FetchUserEmail(ctx, client) // best-effort; not fatal

	_, err = h.channels.Upsert(ctx, models.ConnectedChannel{
		ChannelID:        myChannel.ChannelID,
		ChannelTitle:     myChannel.Title,
		ChannelThumbnail: myChannel.Thumbnail,
		SubscriberCount:  myChannel.SubscriberCount,
		GoogleEmail:      email,
		AccessToken:      token.AccessToken,
		RefreshToken:     token.RefreshToken,
		TokenExpiry:      token.Expiry,
		Scopes:           "youtube.readonly yt-analytics.readonly yt-analytics-monetary.readonly",
	})
	if err != nil {
		h.redirectWithError(w, r, "Không lưu được kênh đã kết nối: "+err.Error())
		return
	}

	http.Redirect(w, r, h.cfg.FrontendURL+"/channels?connected=1", http.StatusFound)
}

func (h *OAuthHandler) redirectWithError(w http.ResponseWriter, r *http.Request, message string) {
	target := h.cfg.FrontendURL + "/channels?error=" + url.QueryEscape(message)
	http.Redirect(w, r, target, http.StatusFound)
}
