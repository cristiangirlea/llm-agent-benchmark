package web

import "net/http"

func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Only for local dev
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		h(w, r)
	}
}

func RegisterRoutes() {
	http.HandleFunc("/", RenderIndex)
	http.HandleFunc("/generate", HandleGenerateResponse)
	http.HandleFunc("/healthz", HealthCheck)
	http.HandleFunc("/api", withCORS(Handler))
}
