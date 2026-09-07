package api

import "net/http"

func NewRouter(h *Handler, sh *SettingsHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/analyses", h.CreateAnalysis)
	mux.HandleFunc("GET /api/analyses", h.ListAnalyses)
	mux.HandleFunc("GET /api/analyses/{id}", h.GetAnalysis)
	mux.HandleFunc("GET /api/settings", sh.GetSettings)
	mux.HandleFunc("PUT /api/settings", sh.UpdateSettings)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
