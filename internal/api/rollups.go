package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

func rollupsHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		level := strings.TrimPrefix(r.URL.Path, "/api/v1/rollups/")
		if level == "" || strings.Contains(level, "/") {
			http.NotFound(w, r)
			return
		}
		f, err := ParseRollupFilter(r.URL.Query(), level, time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		rows, err := s.QueryRollups(r.Context(), f)
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":  rollupsEnvelope(rows),
			"limit":  f.Limit,
			"offset": f.Offset,
			"from":   f.From.Format(time.RFC3339),
			"to":     f.To.Format(time.RFC3339),
		})
	}
}

func rollupsEnvelope(rows []store.RollupRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"ts_bucket":       r.TSBucket.UTC().Format(time.RFC3339),
			"profile_id":      r.ProfileID,
			"vendor":          r.Vendor,
			"type":            r.Type,
			"region":          r.Region,
			"status_class":    r.StatusClass,
			"target_host":     r.TargetHost,
			"team":            r.Team,
			"project":         r.Project,
			"request_count":   r.RequestCount,
			"bytes_in_total":  r.BytesInTotal,
			"bytes_out_total": r.BytesOutTotal,
			"latency_ms_avg":  r.LatencyMSAvg,
			"latency_ms_p50":  r.LatencyMSP50,
			"latency_ms_p95":  r.LatencyMSP95,
			"latency_ms_p99":  r.LatencyMSP99,
			"cost_usd_total":  r.CostUSDTotal,
			"success_count":   r.SuccessCount,
			"failure_count":   r.FailureCount,
		})
	}
	return out
}
