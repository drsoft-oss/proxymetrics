package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/proxy/core"
	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// captchasHandler returns one row per (vendor, type) with a per-kind breakdown.
// Rows whose total = 0 are omitted; rollup rows whose captcha_kind = "" are ignored.
func captchasHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		source := PickSource(f.To.Sub(f.From))

		type key struct{ vendor, typ string }
		rows := map[key]map[string]int64{}
		totals := map[key]int64{}

		err = aggregateCaptcha(r.Context(), s, source,
			eventFilterToRollup(f, PickLevel(source)),
			f,
			func(vendor, typ, kind string, req int64) {
				if kind == "" {
					return
				}
				k := key{vendor, typ}
				if rows[k] == nil {
					rows[k] = map[string]int64{}
				}
				rows[k][kind] += req
				totals[k] += req
			})
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		out := make([]map[string]any, 0, len(rows))
		for k, byKind := range rows {
			if totals[k] == 0 {
				continue
			}
			out = append(out, map[string]any{
				"vendor":  k.vendor,
				"type":    k.typ,
				"total":   totals[k],
				"by_kind": byKind,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"rows":   out,
			"from":   f.From.Format(time.RFC3339),
			"to":     f.To.Format(time.RFC3339),
			"source": source,
		})
	}
}

// captchaDistributionHandler returns a flat per-kind series, ignoring "".
func captchaDistributionHandler(s store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		source := PickSource(f.To.Sub(f.From))

		byKind := map[string]int64{}
		err = aggregateCaptcha(r.Context(), s, source,
			eventFilterToRollup(f, PickLevel(source)),
			f,
			func(_, _, kind string, req int64) {
				if kind == "" {
					return
				}
				byKind[kind] += req
			})
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		items := make([]map[string]any, 0, len(byKind))
		for kind, n := range byKind {
			items = append(items, map[string]any{"kind": kind, "requests": n})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":  items,
			"from":   f.From.Format(time.RFC3339),
			"to":     f.To.Format(time.RFC3339),
			"source": source,
		})
	}
}

// captchaDetailHandler is the drill endpoint: GET /api/v1/captchas/{kind}.
func captchaDetailHandler(s store.Store) http.HandlerFunc {
	valid := make(map[string]bool, len(core.CaptchaKinds))
	for _, k := range core.CaptchaKinds {
		valid[k] = true
	}
	return func(w http.ResponseWriter, r *http.Request) {
		kind := strings.TrimPrefix(r.URL.Path, "/api/v1/captchas/")
		if kind == "" || strings.Contains(kind, "/") || !valid[kind] {
			writeProblem(w, http.StatusBadRequest, "bad_request", "kind must be one of recaptcha|turnstile|hcaptcha|datadome|arkose")
			return
		}
		f, err := ParseEventFilter(r.URL.Query(), time.Now().UTC())
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		got, err := s.CaptchaKindDetail(r.Context(), kind, f)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeProblem(w, http.StatusNotFound, "not_found", "kind not seen in the period")
				return
			}
			writeProblem(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		topProviders := make([]map[string]any, 0, len(got.TopProviders))
		for _, p := range got.TopProviders {
			topProviders = append(topProviders, map[string]any{
				"vendor": p.Vendor, "type": p.Type, "requests": p.Requests,
			})
		}
		topTargets := make([]map[string]any, 0, len(got.TopTargets))
		for _, t := range got.TopTargets {
			topTargets = append(topTargets, map[string]any{
				"host": t.Host, "requests": t.Requests,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"kind":          got.Kind,
			"requests":      got.Requests,
			"top_providers": topProviders,
			"top_targets":   topTargets,
			"from":          f.From.Format(time.RFC3339),
			"to":            f.To.Format(time.RFC3339),
		})
	}
}

// aggregateCaptcha walks the appropriate source (events or rollups) and calls
// cb for each row with (vendor, type, captcha_kind, request_count).
func aggregateCaptcha(ctx context.Context, s store.Store, source string,
	rf store.RollupFilter, ef store.EventFilter,
	cb func(vendor, typ, kind string, req int64),
) error {
	if source == "events" {
		rows, _, err := s.QueryEvents(ctx, eventFilterForAggregate(ef))
		if err != nil {
			return err
		}
		for _, e := range rows {
			cb(e.Vendor, e.Type, e.CaptchaKind, 1)
		}
		return nil
	}
	rr, err := s.QueryRollups(ctx, rf)
	if err != nil {
		return err
	}
	for _, r := range rr {
		cb(r.Vendor, r.Type, r.CaptchaKind, r.RequestCount)
	}
	return nil
}
