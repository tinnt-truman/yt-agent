package api

import (
	"database/sql"
	"errors"
	"net/http"

	"golang.org/x/oauth2"

	"ytagent/backend/internal/analytics"
	"ytagent/backend/internal/config"
	"ytagent/backend/internal/db"
	"ytagent/backend/internal/googleoauth"
	"ytagent/backend/internal/models"
)

type ChannelsHandler struct {
	cfg      *config.Config
	channels *db.ConnectedChannelStore
	settings *db.SettingsStore
}

func NewChannelsHandler(cfg *config.Config, channels *db.ConnectedChannelStore, settings *db.SettingsStore) *ChannelsHandler {
	return &ChannelsHandler{cfg: cfg, channels: channels, settings: settings}
}

func (h *ChannelsHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.channels.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list connected channels")
		return
	}
	if list == nil {
		list = []models.ConnectedChannel{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *ChannelsHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ch, err := h.channels.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load channel")
		return
	}

	// Best-effort revoke at Google — an already-expired or already-revoked
	// token still needs the row gone locally, so a revoke failure here isn't
	// fatal to the disconnect action itself.
	_, _ = http.PostForm("https://oauth2.googleapis.com/revoke", map[string][]string{
		"token": {ch.RefreshToken},
	})

	if err := h.channels.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not disconnect channel")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// GetAnalytics fetches a fresh YouTube Analytics snapshot for a connected
// channel: views, watch time, traffic sources, top videos, and a
// best-effort revenue/monetization read. The token is refreshed (and the
// new access token persisted) first if it's expired or close to it.
func (h *ChannelsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	ch, err := h.channels.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load channel")
		return
	}

	oauthCfg, err := googleoauth.NewOAuthConfig(h.cfg)
	if err != nil {
		writeError(w, http.StatusPreconditionFailed, err.Error())
		return
	}

	token := &oauth2.Token{
		AccessToken:  ch.AccessToken,
		RefreshToken: ch.RefreshToken,
		Expiry:       ch.TokenExpiry,
	}
	fresh, refreshed, err := googleoauth.RefreshIfNeeded(ctx, oauthCfg, token)
	if err != nil {
		writeError(w, http.StatusBadGateway, "phiên kết nối Google đã hết hạn, hãy kết nối lại kênh này: "+err.Error())
		return
	}
	if refreshed {
		if err := h.channels.UpdateTokens(ctx, ch.ID, fresh.AccessToken, fresh.Expiry); err != nil {
			writeError(w, http.StatusInternalServerError, "could not persist refreshed token")
			return
		}
	}

	settings, err := h.settings.Get(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load settings")
		return
	}

	client := googleoauth.HTTPClient(ctx, oauthCfg, fresh)
	result, err := analytics.FetchChannelAnalytics(ctx, client, settings.YouTubeAPIKey)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
