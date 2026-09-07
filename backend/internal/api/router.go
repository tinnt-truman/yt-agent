package api

import "net/http"

func NewRouter(h *Handler, sh *SettingsHandler, corsOrigin string) http.Handler {
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

	return withCORS(mux, corsOrigin)
}

func withCORS(next http.Handler, origin string) http.Handler {
	if origin == "" {
		origin = "*"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
