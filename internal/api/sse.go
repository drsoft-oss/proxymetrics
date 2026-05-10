package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/broadcaster"
	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// SSEHandler returns an http.HandlerFunc that streams broadcaster events to the
// connected client as Server-Sent Events.
func SSEHandler(b *broadcaster.Broadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		sub := b.Subscribe()
		defer sub.Close()

		ping := time.NewTicker(30 * time.Second)
		defer ping.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ping.C:
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			case e, ok := <-sub.Events():
				if !ok {
					return
				}
				body, err := json.Marshal(eventEnvelope(e))
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "event: event\ndata: %s\n\n", body)
				flusher.Flush()
			}
		}
	}
}

// eventEnvelope converts the internal store.Event to a JSON-friendly map with
// snake_case keys (the dashboard wants snake_case).
func eventEnvelope(e store.Event) map[string]any {
	return map[string]any{
		"ts":               e.TS.UTC().Format(time.RFC3339Nano),
		"request_id":       e.RequestID,
		"profile_id":       e.ProfileID,
		"vendor":           e.Vendor,
		"type":             e.Type,
		"region":           e.Region,
		"target_host":      e.TargetHost,
		"target_path_hash": e.TargetPathHash,
		"status_code":      e.StatusCode,
		"status_class":     e.StatusClass,
		"bytes_in":         e.BytesIn,
		"bytes_out":        e.BytesOut,
		"latency_ms":       e.LatencyMS,
		"cost_usd":         e.CostUSD,
		"team":             e.Team,
		"project":          e.Project,
		"captcha_kind":     e.CaptchaKind,
	}
}
