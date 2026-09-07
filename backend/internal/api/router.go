package api

import "net/http"

func NewRouter(h *Handler, sh *SettingsHandler, corsOrigins []string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/analyses", h.CreateAnalysis)
	mux.HandleFunc("GET /api/analyses", h.ListAnalyses)
	mux.HandleFunc("GET /api/analyses/{id}", h.GetAnalysis)
	mux.HandleFunc("POST /api/analyses/{id}/step", h.AdvanceStep)
	mux.HandleFunc("GET /api/settings", sh.GetSettings)
	mux.HandleFunc("PUT /api/settings", sh.UpdateSettings)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
