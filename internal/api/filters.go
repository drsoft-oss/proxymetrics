// Package api implements the REST handlers for sub-project #2.
package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

// ParseEventFilter turns a query string into store.EventFilter.
// `now` is injected for deterministic default-window math (last 24h).
func ParseEventFilter(v url.Values, now time.Time) (store.EventFilter, error) {
	f := store.EventFilter{
		ProfileIDs:    v["profile_id"],
		Vendors:       v["vendor"],
		Types:         v["type"],
		Regions:       v["region"],
		Teams:         v["team"],
		Projects:      v["project"],
		StatusClasses: v["status_class"],
		TargetHost:    v.Get("target_host"),
		Q:             v.Get("q"),
		Sort:          v.Get("sort"),
		Order:         v.Get("order"),
	}

	for _, s := range v["status_code"] {
		n, err := strconv.Atoi(s)
		if err != nil {
			return store.EventFilter{}, fmt.Errorf("status_code: %q is not an integer", s)
		}
		f.StatusCodes = append(f.StatusCodes, n)
	}

	from, to, err := parseTimeRange(v.Get("from"), v.Get("to"), now)
	if err != nil {
		return store.EventFilter{}, err
	}
	f.From, f.To = from, to

	limit, err := parseInt(v.Get("limit"), 50)
	if err != nil {
		return store.EventFilter{}, fmt.Errorf("limit: %w", err)
	}
	if limit > 1000 {
		limit = 1000
	}
	if limit < 1 {
		limit = 50
	}
	f.Limit = limit

	offset, err := parseInt(v.Get("offset"), 0)
	if err != nil {
		return store.EventFilter{}, fmt.Errorf("offset: %w", err)
	}
	if offset < 0 {
		offset = 0
	}
	f.Offset = offset

	return f, nil
}

// ParseRollupFilter turns a query string + level into store.RollupFilter.
func ParseRollupFilter(v url.Values, level string, now time.Time) (store.RollupFilter, error) {
	switch level {
	case "1min", "1hour", "1day":
	default:
		return store.RollupFilter{}, fmt.Errorf("level: invalid value %q", level)
	}

	f := store.RollupFilter{
		Level:         level,
		ProfileIDs:    v["profile_id"],
		Vendors:       v["vendor"],
		Types:         v["type"],
		Regions:       v["region"],
		Teams:         v["team"],
		Projects:      v["project"],
		StatusClasses: v["status_class"],
		TargetHost:    v.Get("target_host"),
		Q:             v.Get("q"),
		Sort:          v.Get("sort"),
		Order:         v.Get("order"),
	}

	from, to, err := parseTimeRange(v.Get("from"), v.Get("to"), now)
	if err != nil {
		return store.RollupFilter{}, err
	}
	f.From, f.To = from, to

	limit, err := parseInt(v.Get("limit"), 50)
	if err != nil {
		return store.RollupFilter{}, fmt.Errorf("limit: %w", err)
	}
	if limit > 1000 {
		limit = 1000
	}
	if limit < 1 {
		limit = 50
	}
	f.Limit = limit

	offset, err := parseInt(v.Get("offset"), 0)
	if err != nil {
		return store.RollupFilter{}, fmt.Errorf("offset: %w", err)
	}
	if offset < 0 {
		offset = 0
	}
	f.Offset = offset

	return f, nil
}

func parseTimeRange(fromS, toS string, now time.Time) (time.Time, time.Time, error) {
	var from, to time.Time
	var err error
	if fromS != "" {
		from, err = time.Parse(time.RFC3339, fromS)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("from: %q is not RFC3339", fromS)
		}
	}
	if toS != "" {
		to, err = time.Parse(time.RFC3339, toS)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("to: %q is not RFC3339", toS)
		}
	}

	switch {
	case fromS == "" && toS == "":
		// Default last-24h window.
		to = now
		from = now.Add(-24 * time.Hour)
	case fromS == "":
		from = to.Add(-24 * time.Hour)
	case toS == "":
		to = now
	}
	return from, to, nil
}

func parseInt(s string, dflt int) (int, error) {
	if s == "" {
		return dflt, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%q is not an integer", s)
	}
	return n, nil
}
