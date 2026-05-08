package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

func eventsHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		rows, total, err := s.QueryEvents(r.Context(), f)
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":          eventsEnvelope(rows),
			"limit":          f.Limit,
			"offset":         f.Offset,
			"total_estimate": total,
			"from":           f.From.Format(time.RFC3339),
			"to":             f.To.Format(time.RFC3339),
		})
	}
}

func eventDetailHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/events/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		// Fetch a generous window and scan for the request_id. v1 simplification;
		// a future task can add Store.GetEvent for high-volume deployments.
		rows, _, err := s.QueryEvents(r.Context(), store.EventFilter{
			Limit: 1000,
		})
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		var match *store.Event
		for i := range rows {
			if rows[i].RequestID == id {
				match = &rows[i]
				break
			}
		}
		if match == nil {
			writeProblem(w, http.StatusNotFound, "not_found", "event not found")
			return
		}
		writeJSON(w, http.StatusOK, eventEnvelope(*match))
	}
}

func eventsEnvelope(events []store.Event) []map[string]any {
	out := make([]map[string]any, 0, len(events))
	for _, e := range events {
		out = append(out, eventEnvelope(e))
	}
	return out
}
