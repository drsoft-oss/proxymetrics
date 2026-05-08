package api

import (
	"net/http"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

func overviewHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		source := PickSource(f.To.Sub(f.From))

		var (
			requestCount, successCount, failCount int64
			bytesTotal                            int64
			spendUSD, wastedUSD                   float64
		)

		if source == "events" {
			// v1 shortcut: events-source overview reads up to 1000 raw events. For windows
			// holding >1000 events the auto-router picks rollups_1hour/1day instead, so the
			// undercount only affects very-recent (≤24h) ranges with high traffic.
			rows, _, err := s.QueryEvents(r.Context(), store.EventFilter{
				From: f.From, To: f.To, Limit: 1000,
			})
			if err != nil {
				writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
				return
			}
			for _, e := range rows {
				requestCount++
				bytesTotal += e.BytesIn + e.BytesOut
				spendUSD += e.CostUSD
				if e.StatusClass == "2xx" {
					successCount++
				} else {
					failCount++
					wastedUSD += e.CostUSD
				}
			}
		} else {
			level := PickLevel(source)
			rows, err := s.QueryRollups(r.Context(), store.RollupFilter{
				Level: level, From: f.From, To: f.To, Limit: 1000,
			})
			if err != nil {
				writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
				return
			}
			for _, rr := range rows {
				requestCount += rr.RequestCount
				bytesTotal += rr.BytesInTotal + rr.BytesOutTotal
				spendUSD += rr.CostUSDTotal
				if rr.StatusClass == "2xx" {
					successCount += rr.SuccessCount
				} else {
					failCount += rr.FailureCount
					wastedUSD += rr.CostUSDTotal
				}
			}
		}

		successRate := 0.0
		if requestCount > 0 {
			successRate = float64(successCount) / float64(requestCount)
		}
		wastedPct := 0.0
		if spendUSD > 0 {
			wastedPct = wastedUSD / spendUSD
		}
		_ = failCount

		writeJSON(w, http.StatusOK, map[string]any{
			"from":                f.From.Format(time.RFC3339),
			"to":                  f.To.Format(time.RFC3339),
			"spend_usd_total":     spendUSD,
			"wasted_usd_total":    wastedUSD,
			"wasted_pct_of_spend": wastedPct,
			"bytes_total":         bytesTotal,
			"request_count":       requestCount,
			"success_rate":        successRate,
			"source":              source,
		})
	}
}
