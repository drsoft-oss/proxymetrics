package api

import (
	"encoding/json"
	"net/http"

	"github.com/drsoft-oss/proxymetrics/internal/broadcaster"
	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// Register installs every #2 endpoint on the given mux.
func Register(mux *http.ServeMux, s store.Store, b *broadcaster.Broadcaster, deploymentSecret string) {
	mux.HandleFunc("/api/v1/config", configHandler(deploymentSecret))

	// Events list, detail, and stream.
	mux.HandleFunc("/api/v1/events", eventsHandler(s))
	mux.HandleFunc("/api/v1/events/", func(w http.ResponseWriter, r *http.Request) {
		// Special-case: /api/v1/events/stream lands here too.
		if r.URL.Path == "/api/v1/events/stream" {
			SSEHandler(b)(w, r)
			return
		}
		eventDetailHandler(s)(w, r)
	})

	// Rollups raw rows.
	mux.HandleFunc("/api/v1/rollups/", rollupsHandler(s))

	// Aggregations.
	mux.HandleFunc("/api/v1/overview", overviewHandler(s))
	mux.HandleFunc("/api/v1/providers", providersHandler(s))
	mux.HandleFunc("/api/v1/providers/", providerDetailHandler(s))
	mux.HandleFunc("/api/v1/targets", targetsHandler(s))
	mux.HandleFunc("/api/v1/targets/", targetDetailHandler(s))
	mux.HandleFunc("/api/v1/status-codes", statusCodesHandler(s))
	mux.HandleFunc("/api/v1/status-codes/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/status-codes/distribution" {
			statusCodeDistributionHandler(s)(w, r)
			return
		}
		statusCodeDetailHandler(s)(w, r)
	})
	mux.HandleFunc("/api/v1/sessions/active", sessionsActiveHandler())
	mux.HandleFunc("/api/v1/dimensions", dimensionsHandler(s))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, map[string]string{"error": code, "detail": detail})
}
