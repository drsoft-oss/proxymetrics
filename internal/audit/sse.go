package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// sseHandler streams events for a live run. If the run isn't live (already
// finished, or never existed), returns 204.
func sseHandler(w http.ResponseWriter, r *http.Request, mgr *Manager, id string) {
	run, ok := mgr.Get(id)
	if !ok || run.IsFinalized() {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	sub, cancel := run.Subscribe()
	defer cancel()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	summaryTick := time.NewTicker(time.Second)
	defer summaryTick.Stop()

	completed := 0
	locMatches := 0
	typeMatches := 0
	errorCount := 0

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case <-summaryTick.C:
			body, _ := json.Marshal(map[string]any{
				"type":                 "summary",
				"completed_count":      completed,
				"location_match_count": locMatches,
				"type_match_count":     typeMatches,
				"error_count":          errorCount,
			})
			fmt.Fprintf(w, "data: %s\n\n", body)
			flusher.Flush()
		case ev, ok := <-sub:
			if !ok {
				return
			}
			payload := encodeEvent(ev)
			if ev.Type == EventRequest && ev.Request != nil {
				if ev.Request.Error != "" {
					errorCount++
				} else {
					completed++
					if ev.Request.LocationMatch {
						locMatches++
					}
					if ev.Request.TypeMatch {
						typeMatches++
					}
				}
			}
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
			if ev.Type == EventFinished {
				return
			}
		}
	}
}

// encodeEvent marshals an Event to its on-the-wire JSON form.
func encodeEvent(ev Event) []byte {
	switch ev.Type {
	case EventRequest:
		if ev.Request == nil {
			return []byte(`{"type":"request"}`)
		}
		// merge the type tag into the row's JSON
		row, _ := json.Marshal(ev.Request)
		// row is `{...}` — splice the type tag in
		return []byte(`{"type":"request",` + string(row[1:]))
	case EventFinished:
		body, _ := json.Marshal(map[string]any{"type": "finished", "status": string(ev.Status)})
		return body
	default:
		body, _ := json.Marshal(map[string]any{"type": string(ev.Type)})
		return body
	}
}
