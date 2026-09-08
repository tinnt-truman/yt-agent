package api

import "net/http"

func NewRouter(
	h *Handler,
	sh *SettingsHandler,
	th *TrendingHandler,
	oh *OAuthHandler,
	ch *ChannelsHandler,
	vph *VideoPromptHandler,
	ah *AuthHandler,
	corsOrigins []string,
	appPassword string,
) http.Handler {
	// Costs YouTube/DeepSeek quota, can read/change stored API keys, or
	// touches a connected Google account, so it sits behind the shared app
	// password.
	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/analyses", h.CreateAnalysis)
	protected.HandleFunc("GET /api/analyses", h.ListAnalyses)
	protected.HandleFunc("GET /api/analyses/{id}", h.GetAnalysis)
	protected.HandleFunc("POST /api/analyses/{id}/step", h.AdvanceStep)
	protected.HandleFunc("GET /api/settings", sh.GetSettings)
	protected.HandleFunc("PUT /api/settings", sh.UpdateSettings)
	protected.HandleFunc("GET /api/trending", th.GetTrending)
	protected.HandleFunc("GET /api/trending/categories", th.GetCategories)
	protected.HandleFunc("POST /api/trending/insight", th.GenerateTrendingInsight)
	protected.HandleFunc("GET /api/oauth/google/url", oh.GetAuthURL)
	protected.HandleFunc("GET /api/channels", ch.List)
	protected.HandleFunc("DELETE /api/channels/{id}", ch.Disconnect)
	protected.HandleFunc("GET /api/channels/{id}/analytics", ch.GetAnalytics)
	protected.HandleFunc("POST /api/videos/prompt", vph.GeneratePrompt)

	mux := http.NewServeMux()
	// Registered directly on the outer mux (not under requireAuth): login
	// obviously can't require the credential it's meant to grant, healthz is
	// a harmless liveness probe, and the OAuth callback is Google redirecting
	// the browser — no Authorization header is possible on a page navigation,
	// so it's verified via the signed `state` param instead (see
	// googleoauth.VerifyState). Go's ServeMux prefers the more specific exact
	// patterns below over the "/api/" prefix, so none of these are shadowed
	// by the protected catch-all.
	mux.HandleFunc("POST /api/auth/login", ah.Login)
	mux.HandleFunc("GET /api/oauth/google/callback", oh.Callback)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/api/", requireAuth(appPassword, protected))

	return withCORS(mux, corsOrigins)
}

// withCORS allows either any origin (corsOrigins empty, the default) or one
// of a fixed allow-list. A single static origin string doesn't work well
// with Vercel, where one project can be reached from several distinct
// domains (the production alias, the auto-generated project domain, and a
// unique URL per preview deployment) — so a configured list is matched
// dynamically against the request's Origin header and echoed back, rather
// than compared against one hardcoded value.
func withCORS(next http.Handler, allowedOrigins []string) http.Handler {
	allowAll := len(allowedOrigins) == 0
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		switch {
		case allowAll:
			w.Header().Set("Access-Control-Allow-Origin", "*")
		case origin != "" && allowed[origin]:
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
