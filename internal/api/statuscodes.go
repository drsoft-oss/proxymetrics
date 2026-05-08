package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

func statusCodesHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		source := PickSource(f.To.Sub(f.From))

		matrix := map[string]map[string]map[string]any{}

		err = aggregateBy(r.Context(), s, source,
			eventFilterToRollup(f, PickLevel(source)),
			f,
			func(profileID, vendor, typ, region, statusClass string, ts time.Time,
				req, succ, fail, bytesTotal int64, spend, latencyAvg float64, p50, p95, p99 int) {
				if matrix[profileID] == nil {
					matrix[profileID] = map[string]map[string]any{}
				}
				cell := matrix[profileID][statusClass]
				if cell == nil {
					cell = map[string]any{
						"request_count": int64(0),
						"bytes_total":   int64(0),
						"spend_usd":     0.0,
					}
					matrix[profileID][statusClass] = cell
				}
				cell["request_count"] = cell["request_count"].(int64) + req
				cell["bytes_total"] = cell["bytes_total"].(int64) + bytesTotal
				cell["spend_usd"] = cell["spend_usd"].(float64) + spend
				_ = vendor
				_ = typ
				_ = region
				_ = ts
				_ = latencyAvg
				_ = succ
				_ = fail
				_ = p50
				_ = p95
				_ = p99
			})
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		profiles, _ := s.ListProfiles(r.Context())
		labels := map[string]string{}
		for _, p := range profiles {
			labels[p.ID] = p.Label
		}

		rows := make([]map[string]any, 0, len(matrix))
		for profileID, byStatus := range matrix {
			rows = append(rows, map[string]any{
				"profile_id": profileID,
				"label":      labels[profileID],
				"by_status":  byStatus,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"rows":   rows,
			"from":   f.From.Format(time.RFC3339),
			"to":     f.To.Format(time.RFC3339),
			"source": source,
		})
	}
}

func statusCodeDistributionHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		rows, err := s.StatusCodeDistribution(r.Context(), f)
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, map[string]any{
				"code":                r.Code,
				"class":               r.Class,
				"requests":            r.Requests,
				"wasted_usd":          r.WastedUSD,
				"spend_usd":           r.SpendUSD,
				"top_provider_vendor": r.TopProviderVendor,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":  items,
			"from":   f.From.Format(time.RFC3339),
			"to":     f.To.Format(time.RFC3339),
			"source": "events",
		})
	}
}

func statusCodeDetailHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.URL.Path, "/api/v1/status-codes/")
		if raw == "" || strings.Contains(raw, "/") {
			http.NotFound(w, r)
			return
		}
		code, err := strconv.Atoi(raw)
		if err != nil || code < 100 || code > 599 {
			writeProblem(w, http.StatusBadRequest, "bad_request", "code must be a 3-digit HTTP status")
			return
		}

		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		got, err := s.StatusCodeDetail(r.Context(), code, f)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeProblem(w, http.StatusNotFound, "not_found", "status code not seen in the period")
				return
			}
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		topProviders := make([]map[string]any, 0, len(got.TopProviders))
		for _, p := range got.TopProviders {
			topProviders = append(topProviders, map[string]any{
				"vendor":     p.Vendor,
				"requests":   p.Requests,
				"wasted_usd": p.WastedUSD,
				"spend_usd":  p.SpendUSD,
			})
		}
		topTargets := make([]map[string]any, 0, len(got.TopTargets))
		for _, t := range got.TopTargets {
			topTargets = append(topTargets, map[string]any{
				"host":       t.Host,
				"requests":   t.Requests,
				"wasted_usd": t.WastedUSD,
				"spend_usd":  t.SpendUSD,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"code":          got.Code,
			"class":         got.Class,
			"requests":      got.Requests,
			"wasted_usd":    got.WastedUSD,
			"spend_usd":     got.SpendUSD,
			"top_providers": topProviders,
			"top_targets":   topTargets,
			"from":          f.From.Format(time.RFC3339),
			"to":            f.To.Format(time.RFC3339),
		})
	}
}
