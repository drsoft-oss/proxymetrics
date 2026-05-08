package api

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

type providerAggregate struct {
	ProfileID    string
	Vendor       string
	Type         string
	Region       string
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
	LatencyP50   int
	LatencyP95   int
	LatencyP99   int
	LastSeenTS   time.Time
}

func providersHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		source := PickSource(f.To.Sub(f.From))
		aggs := map[string]*providerAggregate{}

		err = aggregateBy(r.Context(), s, source,
			eventFilterToRollup(f, PickLevel(source)),
			f,
			func(profileID, vendor, typ, region, statusClass string, ts time.Time,
				req, succ, fail, bytesTotal int64, spend, latencyAvg float64, p50, p95, p99 int) {
				a, ok := aggs[profileID]
				if !ok {
					a = &providerAggregate{ProfileID: profileID, Vendor: vendor, Type: typ, Region: region}
					aggs[profileID] = a
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
				if p50 > a.LatencyP50 {
					a.LatencyP50 = p50
				}
				if p95 > a.LatencyP95 {
					a.LatencyP95 = p95
				}
				if p99 > a.LatencyP99 {
					a.LatencyP99 = p99
				}
				_ = latencyAvg
			})
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		// Trailing-24h sparkline data, decoupled from the filter window.
		now := time.Now().UTC()
		buckets, _ := s.RequestsByHour(r.Context(), "profile_id", now)
		sparkline := buildSparkline(buckets, 24, now)

		profiles, _ := s.ListProfiles(r.Context())
		labels := map[string]string{}
		for _, p := range profiles {
			labels[p.ID] = p.Label
		}

		items := make([]map[string]any, 0, len(aggs))
		for _, a := range aggs {
			succRate := 0.0
			if a.RequestCount > 0 {
				succRate = float64(a.SuccessCount) / float64(a.RequestCount)
			}
			rbh := sparkline[a.ProfileID]
			if rbh == nil {
				rbh = make([]int64, 24)
			}
			items = append(items, map[string]any{
				"profile_id":     a.ProfileID,
				"label":          labels[a.ProfileID],
				"vendor":         a.Vendor,
				"type":           a.Type,
				"region":         a.Region,
				"request_count":  a.RequestCount,
				"bytes_total":    a.BytesTotal,
				"success_rate":   succRate,
				"latency_ms_p50": a.LatencyP50,
				"latency_ms_p95": a.LatencyP95,
				"spend_usd":      a.SpendUSD,
				"wasted_usd":     a.WastedUSD,
				"last_seen":      a.LastSeenTS.Format(time.RFC3339),
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

func providerDetailHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/providers/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		profile, err := s.GetProfile(r.Context(), id)
		if err != nil {
			writeProblem(w, http.StatusNotFound, "not_found", "profile not found")
			return
		}

		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		source := PickSource(f.To.Sub(f.From))

		byStatus := map[string]map[string]any{}

		err = aggregateBy(r.Context(), s, source,
			store.RollupFilter{
				Level: PickLevel(source), From: f.From, To: f.To,
				ProfileIDs: []string{id}, Limit: 100000,
			},
			store.EventFilter{
				From: f.From, To: f.To, ProfileIDs: []string{id}, Limit: 1000,
			},
			func(profileID, vendor, typ, region, statusClass string, ts time.Time,
				req, succ, fail, bytesTotal int64, spend, latencyAvg float64, p50, p95, p99 int) {
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
				_ = succ
				_ = fail
				_ = ts
				_ = vendor
				_ = typ
				_ = region
				_ = latencyAvg
				_ = p50
				_ = p95
				_ = p99
			})
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		// Top 10 targets for this profile.
		type topTarget struct {
			Host         string
			RequestCount int64
			SuccessCount int64
			BytesTotal   int64
			SpendUSD     float64
		}
		targets := map[string]*topTarget{}
		err = aggregateByTarget(r.Context(), s, source,
			store.RollupFilter{
				Level: PickLevel(source), From: f.From, To: f.To,
				ProfileIDs: []string{id}, Limit: 100000,
			},
			store.EventFilter{
				From: f.From, To: f.To, ProfileIDs: []string{id}, Limit: 1000,
			},
			func(host, statusClass string, ts time.Time, req, succ, fail, bytesTotal int64, spend float64) {
				if host == "" {
					return
				}
				a := targets[host]
				if a == nil {
					a = &topTarget{Host: host}
					targets[host] = a
				}
				a.RequestCount += req
				a.SuccessCount += succ
				a.BytesTotal += bytesTotal
				a.SpendUSD += spend
				_ = ts
				_ = statusClass
				_ = fail
			})
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		topList := make([]map[string]any, 0, len(targets))
		for _, t := range targets {
			succRate := 0.0
			if t.RequestCount > 0 {
				succRate = float64(t.SuccessCount) / float64(t.RequestCount)
			}
			topList = append(topList, map[string]any{
				"target_host":   t.Host,
				"request_count": t.RequestCount,
				"success_rate":  succRate,
				"bytes_total":   t.BytesTotal,
				"spend_usd":     t.SpendUSD,
			})
		}
		sort.Slice(topList, func(i, j int) bool {
			return topList[i]["request_count"].(int64) > topList[j]["request_count"].(int64)
		})
		if len(topList) > 10 {
			topList = topList[:10]
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"profile": map[string]any{
				"profile_id": profile.ID,
				"label":      profile.Label,
				"vendor":     profile.Vendor,
				"type":       profile.Type,
				"region":     profile.Region,
			},
			"by_status_class": mapValuesAsSlice(byStatus),
			"top_targets":     topList,
			"from":            f.From.Format(time.RFC3339),
			"to":              f.To.Format(time.RFC3339),
			"source":          source,
		})
	}
}

func mapValuesAsSlice(m map[string]map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// aggregateBy walks the appropriate source (events or a rollup table) and calls
// cb for each row with its dimensions and metrics.
func aggregateBy(ctx context.Context, s store.Store, source string,
	rf store.RollupFilter, ef store.EventFilter,
	cb func(profileID, vendor, typ, region, statusClass string, ts time.Time,
		req, succ, fail, bytesTotal int64, spend, latencyAvg float64, p50, p95, p99 int),
) error {
	if source == "events" {
		rows, _, err := s.QueryEvents(ctx, ef)
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
			cb(e.ProfileID, e.Vendor, e.Type, e.Region, e.StatusClass, e.TS,
				1, succ, fail, e.BytesIn+e.BytesOut, e.CostUSD, float64(e.LatencyMS),
				e.LatencyMS, e.LatencyMS, e.LatencyMS)
		}
		return nil
	}
	rows, err := s.QueryRollups(ctx, rf)
	if err != nil {
		return err
	}
	for _, rr := range rows {
		cb(rr.ProfileID, rr.Vendor, rr.Type, rr.Region, rr.StatusClass, rr.TSBucket,
			rr.RequestCount, rr.SuccessCount, rr.FailureCount,
			rr.BytesInTotal+rr.BytesOutTotal, rr.CostUSDTotal, rr.LatencyMSAvg,
			rr.LatencyMSP50, rr.LatencyMSP95, rr.LatencyMSP99)
	}
	return nil
}

// buildSparkline groups RequestsByHourBucket entries into a per-key fixed-length
// `[]int64` slice ordered from oldest to newest. The last entry is the current hour.
func buildSparkline(buckets []store.RequestsByHourBucket, length int, now time.Time) map[string][]int64 {
	out := map[string][]int64{}
	end := now.Truncate(time.Hour)
	for _, b := range buckets {
		offH := int(end.Sub(b.Hour.Truncate(time.Hour)) / time.Hour)
		if offH < 0 || offH >= length {
			continue
		}
		s := out[b.Key]
		if s == nil {
			s = make([]int64, length)
		}
		idx := length - 1 - offH
		s[idx] += b.Count
		out[b.Key] = s
	}
	return out
}

// eventFilterToRollup translates an EventFilter into a RollupFilter with the
// matching dimensions (drops StatusCodes since rollups only have StatusClasses).
func eventFilterToRollup(f store.EventFilter, level string) store.RollupFilter {
	return store.RollupFilter{
		Level:         level,
		From:          f.From,
		To:            f.To,
		ProfileIDs:    f.ProfileIDs,
		Vendors:       f.Vendors,
		Types:         f.Types,
		Regions:       f.Regions,
		Teams:         f.Teams,
		Projects:      f.Projects,
		StatusClasses: f.StatusClasses,
		TargetHost:    f.TargetHost,
		Q:             f.Q,
		Limit:         100000,
	}
}
