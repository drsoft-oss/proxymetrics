package api

import "net/http"

// sessionsActiveHandler always returns an empty list. Sessions were dropped in
// sub-project #1; the dashboard's wiring still expects this endpoint to exist.
// Future: wire to a session manager when/if session tracking re-lands.
func sessionsActiveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []any{})
	}
}
