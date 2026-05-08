package profile

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

// testEndpointHandler is set by RegisterTestEndpoint and dispatched from the
// /api/v1/profiles/ catch-all when the path ends in /test. Nil means the /test
// route returns 404 (the test endpoint isn't registered).
var testEndpointHandler http.HandlerFunc

// RegisterHTTP installs the read-only profile routes on mux:
//   - GET /api/v1/profiles            → list
//   - GET /api/v1/profiles/{id}       → show
//   - POST /api/v1/profiles/{id}/test → dispatched to the handler installed by RegisterTestEndpoint
//
// Profiles are observed-only: there are no create/update/delete endpoints. New
// profile rows appear automatically when the proxy hot path sees a fresh
// (provider, type) combo (see Registry.UpsertObserved).
func RegisterHTTP(mux *http.ServeMux, r *Registry) {
	mux.HandleFunc("/api/v1/profiles", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleList(w, req, r)
	})
	mux.HandleFunc("/api/v1/profiles/", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/api/v1/profiles/")
		if strings.HasSuffix(path, "/test") {
			if testEndpointHandler != nil {
				testEndpointHandler(w, req)
				return
			}
			http.NotFound(w, req)
			return
		}
		if path == "" {
			http.NotFound(w, req)
			return
		}
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p, ok := r.Lookup(path)
		if !ok {
			writeProblem(w, http.StatusNotFound, "not_found", "profile not found")
			return
		}
		writeJSON(w, http.StatusOK, p)
	})
}

func handleList(w http.ResponseWriter, req *http.Request, r *Registry) {
	if req.URL.Query().Get("with") != "usage" {
		writeJSON(w, http.StatusOK, r.All())
		return
	}
	window := 24 * time.Hour
	if v := req.URL.Query().Get("window"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < time.Minute || d > 168*time.Hour {
			writeProblem(w, http.StatusBadRequest, "bad_request", "window must be a duration between 1m and 168h")
			return
		}
		window = d
	}
	rows, err := r.ListWithUsage(req.Context(), window)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m := profileToMap(row.Profile)
		m["requests"] = row.Requests
		m["spend_usd"] = row.SpendUSD
		out = append(out, m)
	}
	writeJSON(w, http.StatusOK, out)
}

func profileToMap(p store.Profile) map[string]any {
	b, _ := json.Marshal(p)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, map[string]string{"error": code, "detail": detail})
}
