package audit

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// RegisterRoutes wires the audit endpoints onto the given mux.
func RegisterRoutes(mux *http.ServeMux, mgr *Manager) {
	mux.HandleFunc("/api/v1/audits", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreate(w, r, mgr)
		case http.MethodGet:
			handleList(w, r, mgr)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/audits/providers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleListProviders(w, r, mgr)
	})
	mux.HandleFunc("/api/v1/audits/", func(w http.ResponseWriter, r *http.Request) {
		// Routes:
		//   GET  /api/v1/audits/{id}
		//   POST /api/v1/audits/{id}/cancel
		//   GET  /api/v1/audits/{id}/events     (handled by RegisterSSE)
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/audits/")
		if path == "" {
			http.NotFound(w, r)
			return
		}
		if id, ok := strings.CutSuffix(path, "/cancel"); ok && r.Method == http.MethodPost {
			handleCancel(w, r, mgr, id)
			return
		}
		if id, ok := strings.CutSuffix(path, "/events"); ok {
			handleSSE(w, r, mgr, id)
			return
		}
		if r.Method == http.MethodGet {
			handleGet(w, r, mgr, path)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
}

func handleCreate(w http.ResponseWriter, r *http.Request, mgr *Manager) {
	var spec Spec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeProblem(w, http.StatusBadRequest, "decode", err.Error())
		return
	}
	id, err := mgr.Start(r.Context(), spec)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "status": "running"})
}

func handleList(w http.ResponseWriter, r *http.Request, mgr *Manager) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := mgr.ListSummaries(r.Context(), limit, offset)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "list", err.Error())
		return
	}
	if rows == nil {
		rows = []RunSummary{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func handleGet(w http.ResponseWriter, r *http.Request, mgr *Manager, id string) {
	sum, err := mgr.GetSummary(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeProblem(w, http.StatusNotFound, "not_found", "audit run not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "get", err.Error())
		return
	}
	rows, err := mgr.ListRequests(r.Context(), id)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "requests", err.Error())
		return
	}
	if rows == nil {
		rows = []RequestRow{}
	}
	writeJSON(w, http.StatusOK, RunDetail{Run: sum, Requests: rows})
}

func handleCancel(w http.ResponseWriter, r *http.Request, mgr *Manager, id string) {
	_ = mgr.Cancel(id)
	w.WriteHeader(http.StatusNoContent)
}

// handleSSE is implemented in sse.go.
func handleSSE(w http.ResponseWriter, r *http.Request, mgr *Manager, id string) {
	sseHandler(w, r, mgr, id)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, map[string]string{"error": code, "detail": detail})
}

func handleListProviders(w http.ResponseWriter, r *http.Request, mgr *Manager) {
	historical, err := mgr.ListDistinctProviders(r.Context())
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "providers", err.Error())
		return
	}
	// Merge static + history, dedupe case-insensitively, sort case-folded ASC.
	seen := map[string]string{} // lower -> canonical
	for _, p := range StaticProviders() {
		key := strings.ToLower(p)
		if _, ok := seen[key]; !ok {
			seen[key] = p
		}
	}
	for _, p := range historical {
		key := strings.ToLower(p)
		if _, ok := seen[key]; !ok {
			seen[key] = p
		}
	}
	merged := make([]string, 0, len(seen))
	for _, v := range seen {
		merged = append(merged, v)
	}
	sort.Slice(merged, func(i, j int) bool {
		return strings.ToLower(merged[i]) < strings.ToLower(merged[j])
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"providers":         merged,
		"hostname_suffixes": ProviderHostSuffixes(),
	})
}
