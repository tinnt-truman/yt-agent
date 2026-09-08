// Package googleoauth handles the Google OAuth2 flow that lets the operator
// connect a real YouTube channel — granting access to private YouTube
// Analytics data (views, watch time, revenue) that the public Data API key
// used elsewhere in this app can never see.
package googleoauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"ytagent/backend/internal/config"
)

// Scopes requested at consent time. youtube.readonly identifies which
// channel(s) the signed-in account manages; the two yt-analytics scopes
// unlock the private Analytics reports (yt-analytics-monetary.readonly is
// specifically required for revenue metrics — Google treats it as a
// separate, more sensitive grant from general analytics).
var scopes = []string{
	"https://www.googleapis.com/auth/youtube.readonly",
	"https://www.googleapis.com/auth/yt-analytics.readonly",
	"https://www.googleapis.com/auth/yt-analytics-monetary.readonly",
	"https://www.googleapis.com/auth/userinfo.email",
}

// var, not const, so tests can shrink it to exercise the expiry path
// without sleeping ten real minutes.
var stateTTL = 10 * time.Minute

// NewOAuthConfig is nil (with an error) if the operator hasn't configured
// Google OAuth yet — callers surface that as a clear "not configured" error
// rather than the app failing to boot over an optional feature.
func NewOAuthConfig(cfg *config.Config) (*oauth2.Config, error) {
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" || cfg.GoogleRedirectURL == "" {
		return nil, fmt.Errorf("Google OAuth chưa được cấu hình (thiếu GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET/GOOGLE_OAUTH_REDIRECT_URL)")
	}
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}, nil
}

type statePayload struct {
	Nonce string `json:"n"`
	Exp   int64  `json:"e"`
}

// NewState produces a signed, self-verifying value for the OAuth "state"
// parameter — this app has no server-side session to stash a CSRF token in
// (it must also run statelessly on serverless hosts), so the state itself
// carries an expiry and an HMAC signature keyed on the OAuth client secret.
func NewState(clientSecret string) (string, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}
	payload := statePayload{
		Nonce: base64.RawURLEncoding.EncodeToString(nonceBytes),
		Exp:   time.Now().Add(stateTTL).Unix(),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig := sign(clientSecret, payloadB64)
	return payloadB64 + "." + sig, nil
}

// VerifyState checks the signature and expiry produced by NewState.
func VerifyState(clientSecret, state string) error {
	i := strings.IndexByte(state, '.')
	if i < 0 {
		return fmt.Errorf("invalid state")
	}
	payloadB64, sig := state[:i], state[i+1:]
	if sig != sign(clientSecret, payloadB64) {
		return fmt.Errorf("invalid state signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return fmt.Errorf("invalid state encoding")
	}
	var payload statePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("invalid state payload")
	}
	if time.Now().Unix() > payload.Exp {
		return fmt.Errorf("state expired, hãy thử kết nối lại")
	}
	return nil
}

func sign(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// HTTPClient returns an *http.Client that attaches the given token and
// transparently refreshes it via the refresh token when expired. It does
// NOT persist a refreshed token on its own — call PersistIfRefreshed after
// using it (see below) so the next request doesn't need to refresh again.
func HTTPClient(ctx context.Context, oauthCfg *oauth2.Config, token *oauth2.Token) *http.Client {
	return oauthCfg.Client(ctx, token)
}

// RefreshIfNeeded returns a token guaranteed valid for at least a short
// margin, refreshing eagerly (rather than relying on the http.Client's
// internal lazy refresh) so the caller can persist the new access token
// before making the API calls that need it.
func RefreshIfNeeded(ctx context.Context, oauthCfg *oauth2.Config, token *oauth2.Token) (*oauth2.Token, bool, error) {
	if token.Valid() && time.Until(token.Expiry) > time.Minute {
		return token, false, nil
	}
	src := oauthCfg.TokenSource(ctx, &oauth2.Token{RefreshToken: token.RefreshToken})
	fresh, err := src.Token()
	if err != nil {
		return nil, false, fmt.Errorf("refresh Google token: %w", err)
	}
	return fresh, true, nil
}
