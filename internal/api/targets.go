package api

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

type targetAggregate struct {
	TargetHost   string
	RequestCount int64
	SuccessCount int64
	FailureCount int64
	BytesTotal   int64
	SpendUSD     float64
	WastedUSD    float64
	Class2xx     int64
	Class3xx     int64
	Class4xx     int64
	Class5xx     int64
	LastSeenTS   time.Time
}

func targetsHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		source := PickSource(f.To.Sub(f.From))

		aggs := map[string]*targetAggregate{}
		err = aggregateByTarget(r.Context(), s, source,
			eventFilterToRollup(f, PickLevel(source)),
			f,
			func(host, statusClass string, ts time.Time, req, succ, fail, bytesTotal int64, spend float64) {
				if host == "" {
					return
				}
				a, ok := aggs[host]
				if !ok {
					a = &targetAggregate{TargetHost: host}
					aggs[host] = a
				}
				a.RequestCount += req
				a.SuccessCount += succ
				a.FailureCount += fail
				a.BytesTotal += bytesTotal
				a.SpendUSD += spend
				switch statusClass {
				case "2xx":
					a.Class2xx += req
				case "3xx":
					a.Class3xx += req
				case "4xx":
					a.Class4xx += req
				case "5xx":
					a.Class5xx += req
				}
				if statusClass != "2xx" {
					a.WastedUSD += spend
				}
				if ts.After(a.LastSeenTS) {
					a.LastSeenTS = ts
				}
			})
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		now := time.Now().UTC()
		buckets, _ := s.RequestsByHour(r.Context(), "target_host", now)
		sparkline := buildSparkline(buckets, 24, now)

		items := make([]map[string]any, 0, len(aggs))
		for _, a := range aggs {
			succRate := 0.0
			if a.RequestCount > 0 {
				succRate = float64(a.SuccessCount) / float64(a.RequestCount)
			}
			rbh := sparkline[a.TargetHost]
			if rbh == nil {
				rbh = make([]int64, 24)
			}
			items = append(items, map[string]any{
				"target_host":   a.TargetHost,
				"request_count": a.RequestCount,
				"bytes_total":   a.BytesTotal,
				"success_rate":  succRate,
				"spend_usd":     a.SpendUSD,
				"wasted_usd":    a.WastedUSD,
				"last_seen":     a.LastSeenTS.Format(time.RFC3339),
				"status_mix": map[string]any{
					"class2xx": a.Class2xx,
					"class3xx": a.Class3xx,
					"class4xx": a.Class4xx,
					"class5xx": a.Class5xx,
				},
				"requests_by_hour": rbh,
			})
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i]["spend_usd"].(float64) > items[j]["spend_usd"].(float64)
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"items":  items,
			"from":   f.From.Format(time.RFC3339),
			"to":     f.To.Format(time.RFC3339),
			"source": source,
		})
	}
}

func targetDetailHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := strings.TrimPrefix(r.URL.Path, "/api/v1/targets/")
		if host == "" || strings.Contains(host, "/") {
			http.NotFound(w, r)
			return
		}

		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		f.TargetHost = host
		source := PickSource(f.To.Sub(f.From))

		byProvider := map[string]*providerAggregate{}
		byStatus := map[string]map[string]any{}

		err = aggregateBy(r.Context(), s, source,
			eventFilterToRollup(f, PickLevel(source)),
			f,
			func(profileID, vendor, typ, region, statusClass string, ts time.Time,
				req, succ, fail, bytesTotal int64, spend, latencyAvg float64, p50, p95, p99 int) {
				p := byProvider[profileID]
				if p == nil {
					p = &providerAggregate{ProfileID: profileID, Vendor: vendor, Type: typ, Region: region}
					byProvider[profileID] = p
				}
				p.RequestCount += req
				p.SuccessCount += succ
				p.FailureCount += fail
				p.BytesTotal += bytesTotal
				p.SpendUSD += spend
				if statusClass != "2xx" {
					p.WastedUSD += spend
				}

				row := byStatus[statusClass]
				if row == nil {
					row = map[string]any{
						"status_class":  statusClass,
						"request_count": int64(0),
						"bytes_total":   int64(0),
						"spend_usd":     0.0,
					}
					byStatus[statusClass] = row
				}
				row["request_count"] = row["request_count"].(int64) + req
				row["bytes_total"] = row["bytes_total"].(int64) + bytesTotal
				row["spend_usd"] = row["spend_usd"].(float64) + spend
				_ = ts
				_ = latencyAvg
				_ = p50
				_ = p95
				_ = p99
			})
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"target_host":     host,
			"by_provider":     providerAggregatesAsSlice(byProvider),
			"by_status_class": mapValuesAsSlice(byStatus),
			"from":            f.From.Format(time.RFC3339),
			"to":              f.To.Format(time.RFC3339),
			"source":          source,
		})
	}
}

func providerAggregatesAsSlice(m map[string]*providerAggregate) []map[string]any {
	out := make([]map[string]any, 0, len(m))
	for _, p := range m {
		succRate := 0.0
		if p.RequestCount > 0 {
			succRate = float64(p.SuccessCount) / float64(p.RequestCount)
		}
		out = append(out, map[string]any{
			"profile_id":    p.ProfileID,
			"vendor":        p.Vendor,
			"type":          p.Type,
			"region":        p.Region,
			"request_count": p.RequestCount,
			"bytes_total":   p.BytesTotal,
			"success_rate":  succRate,
			"spend_usd":     p.SpendUSD,
			"wasted_usd":    p.WastedUSD,
		})
	}
	return out
}

// aggregateByTarget is a target-grouped variant of aggregateBy.
func aggregateByTarget(ctx context.Context, s store.Store, source string,
	rf store.RollupFilter, ef store.EventFilter,
	cb func(host, statusClass string, ts time.Time, req, succ, fail, bytesTotal int64, spend float64),
) error {
	if source == "events" {
		rows, _, err := s.QueryEvents(ctx, eventFilterForAggregate(ef))
		if err != nil {
			return err
		}
		for _, e := range rows {
			succ, fail := int64(0), int64(0)
			if e.StatusClass == "2xx" {
				succ = 1
			} else {
				fail = 1
			}
			cb(e.TargetHost, e.StatusClass, e.TS, 1, succ, fail, e.BytesIn+e.BytesOut, e.CostUSD)
		}
		return nil
	}
	rows, err := s.QueryRollups(ctx, rf)
	if err != nil {
		return err
	}
	for _, rr := range rows {
		cb(rr.TargetHost, rr.StatusClass, rr.TSBucket,
			rr.RequestCount, rr.SuccessCount, rr.FailureCount,
			rr.BytesInTotal+rr.BytesOutTotal, rr.CostUSDTotal)
	}
	return nil
}
