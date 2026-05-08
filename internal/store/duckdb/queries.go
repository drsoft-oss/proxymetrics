package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

func (s *Store) QueryEvents(ctx context.Context, f store.EventFilter) ([]store.Event, int64, error) {
	where, args := buildEventWhere(f)

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	// Total count over the same filter.
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events"+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("events count: %w", err)
	}

	q := `SELECT ` + eventColumns + ` FROM events` + whereSQL

	if f.Sort != "" && allowedEventSortCol(f.Sort) {
		order := "DESC"
		if strings.EqualFold(f.Order, "asc") {
			order = "ASC"
		}
		q += " ORDER BY " + f.Sort + " " + order
	} else {
		q += " ORDER BY ts DESC"
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	const internalCeiling = 1_000_000
	if limit > internalCeiling {
		limit = internalCeiling
	}
	q += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, max0(f.Offset))

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("events query: %w", err)
	}
	defer rows.Close()

	var out []store.Event
	for rows.Next() {
		var e store.Event
		var region, targetHost, targetPathHash, team, project sql.NullString
		var status sql.NullInt64
		if err := rows.Scan(
			&e.TS, &e.RequestID, &e.ProfileID, &e.Vendor, &e.Type, &region,
			&targetHost, &targetPathHash, &status, &e.StatusClass,
			&e.BytesIn, &e.BytesOut, &e.LatencyMS, &e.CostUSD,
			&team, &project,
		); err != nil {
			return nil, 0, err
		}
		e.Region = region.String
		e.TargetHost = targetHost.String
		e.TargetPathHash = targetPathHash.String
		e.Team = team.String
		e.Project = project.String
		if status.Valid {
			e.StatusCode = int(status.Int64)
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

func buildEventWhere(f store.EventFilter) ([]string, []any) {
	var (
		where []string
		args  []any
	)
	if !f.From.IsZero() {
		where = append(where, "ts >= ?")
		args = append(args, f.From)
	}
	if !f.To.IsZero() {
		where = append(where, "ts < ?")
		args = append(args, f.To)
	}
	for _, w := range []struct {
		col  string
		vals []string
	}{
		{"profile_id", f.ProfileIDs},
		{"vendor", f.Vendors},
		{"type", f.Types},
		{"region", f.Regions},
		{"team", f.Teams},
		{"project", f.Projects},
		{"status_class", f.StatusClasses},
	} {
		if len(w.vals) == 0 {
			continue
		}
		ph := strings.TrimRight(strings.Repeat("?,", len(w.vals)), ",")
		where = append(where, fmt.Sprintf("%s IN (%s)", w.col, ph))
		for _, v := range w.vals {
			args = append(args, v)
		}
	}
	if len(f.StatusCodes) > 0 {
		ph := strings.TrimRight(strings.Repeat("?,", len(f.StatusCodes)), ",")
		where = append(where, fmt.Sprintf("status_code IN (%s)", ph))
		for _, c := range f.StatusCodes {
			args = append(args, c)
		}
	}
	if f.TargetHost != "" {
		where = append(where, "target_host = ?")
		args = append(args, f.TargetHost)
	} else if f.Q != "" {
		where = append(where, "target_host LIKE ?")
		args = append(args, "%"+f.Q+"%")
	}
	return where, args
}

func allowedEventSortCol(name string) bool {
	switch name {
	case "ts", "status_code", "bytes_in", "bytes_out", "latency_ms", "cost_usd":
		return true
	}
	return false
}

func (s *Store) StatusCodeDistribution(ctx context.Context, f store.EventFilter) ([]store.StatusCodeDistRow, error) {
	where, args := buildEventWhere(f)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	// Aggregate per (status_code, status_class).
	q := `SELECT status_code, status_class,
	             COUNT(*) AS reqs,
	             SUM(CASE WHEN status_class != '2xx' THEN cost_usd ELSE 0 END) AS wasted,
	             SUM(cost_usd) AS spend
	      FROM events` + whereSQL + `
	      GROUP BY status_code, status_class
	      ORDER BY reqs DESC, status_code ASC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("status_code dist: %w", err)
	}
	defer rows.Close()

	var out []store.StatusCodeDistRow
	for rows.Next() {
		var r store.StatusCodeDistRow
		var code sql.NullInt64
		if err := rows.Scan(&code, &r.Class, &r.Requests, &r.WastedUSD, &r.SpendUSD); err != nil {
			return nil, err
		}
		if code.Valid {
			r.Code = int(code.Int64)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Compute TopProviderVendor per (code) with a second query: vendor with highest count per code.
	// At most ~50 distinct codes in practice, so we look them up in batch.
	if len(out) == 0 {
		return out, nil
	}
	codes := make([]any, 0, len(out))
	for _, r := range out {
		codes = append(codes, r.Code)
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(codes)), ",")

	// Pull all (code, vendor, count) and pick the top vendor per code in Go.
	q2 := `SELECT status_code, vendor, COUNT(*) AS reqs
	       FROM events` + whereSQL
	if whereSQL == "" {
		q2 += " WHERE status_code IN (" + ph + ")"
	} else {
		q2 += " AND status_code IN (" + ph + ")"
	}
	q2 += " GROUP BY status_code, vendor"

	args2 := append(append([]any{}, args...), codes...)
	r2, err := s.db.QueryContext(ctx, q2, args2...)
	if err != nil {
		return nil, fmt.Errorf("status_code dist top vendor: %w", err)
	}
	defer r2.Close()

	type cv struct {
		vendor string
		reqs   int64
	}
	best := map[int]cv{}
	for r2.Next() {
		var code sql.NullInt64
		var vendor string
		var reqs int64
		if err := r2.Scan(&code, &vendor, &reqs); err != nil {
			return nil, err
		}
		c := 0
		if code.Valid {
			c = int(code.Int64)
		}
		cur, ok := best[c]
		if !ok || reqs > cur.reqs {
			best[c] = cv{vendor: vendor, reqs: reqs}
		}
	}
	if err := r2.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].TopProviderVendor = best[out[i].Code].vendor
	}
	return out, nil
}

func (s *Store) StatusCodeDetail(ctx context.Context, code int, f store.EventFilter) (store.StatusCodeDetailResult, error) {
	where, args := buildEventWhere(f)
	where = append(where, "status_code = ?")
	args = append(args, code)
	whereSQL := " WHERE " + strings.Join(where, " AND ")

	// Totals + class.
	var totals struct {
		class  sql.NullString
		reqs   int64
		wasted float64
		spend  float64
	}
	q := `SELECT MAX(status_class), COUNT(*),
	             COALESCE(SUM(CASE WHEN status_class != '2xx' THEN cost_usd ELSE 0 END), 0),
	             COALESCE(SUM(cost_usd), 0)
	      FROM events` + whereSQL
	if err := s.db.QueryRowContext(ctx, q, args...).Scan(&totals.class, &totals.reqs, &totals.wasted, &totals.spend); err != nil {
		return store.StatusCodeDetailResult{}, fmt.Errorf("status_code detail totals: %w", err)
	}
	if totals.reqs == 0 {
		return store.StatusCodeDetailResult{}, store.ErrNotFound
	}
	anyClass := ""
	if totals.class.Valid {
		anyClass = totals.class.String
	}

	out := store.StatusCodeDetailResult{
		Code:      code,
		Class:     anyClass,
		Requests:  totals.reqs,
		WastedUSD: totals.wasted,
		SpendUSD:  totals.spend,
	}

	// Top providers.
	qp := `SELECT vendor, COUNT(*),
	              SUM(CASE WHEN status_class != '2xx' THEN cost_usd ELSE 0 END),
	              SUM(cost_usd)
	       FROM events` + whereSQL + `
	       GROUP BY vendor
	       ORDER BY 2 DESC
	       LIMIT 5`
	rp, err := s.db.QueryContext(ctx, qp, args...)
	if err != nil {
		return store.StatusCodeDetailResult{}, fmt.Errorf("status_code detail providers: %w", err)
	}
	for rp.Next() {
		var p store.StatusCodeDetailProvider
		if err := rp.Scan(&p.Vendor, &p.Requests, &p.WastedUSD, &p.SpendUSD); err != nil {
			rp.Close()
			return store.StatusCodeDetailResult{}, err
		}
		out.TopProviders = append(out.TopProviders, p)
	}
	rp.Close()
	if err := rp.Err(); err != nil {
		return store.StatusCodeDetailResult{}, err
	}

	// Top targets.
	qt := `SELECT target_host, COUNT(*),
	              SUM(CASE WHEN status_class != '2xx' THEN cost_usd ELSE 0 END),
	              SUM(cost_usd)
	       FROM events` + whereSQL + `
	         AND target_host IS NOT NULL AND target_host != ''
	       GROUP BY target_host
	       ORDER BY 2 DESC
	       LIMIT 5`
	rt, err := s.db.QueryContext(ctx, qt, args...)
	if err != nil {
		return store.StatusCodeDetailResult{}, fmt.Errorf("status_code detail targets: %w", err)
	}
	for rt.Next() {
		var t store.StatusCodeDetailTarget
		if err := rt.Scan(&t.Host, &t.Requests, &t.WastedUSD, &t.SpendUSD); err != nil {
			rt.Close()
			return store.StatusCodeDetailResult{}, err
		}
		out.TopTargets = append(out.TopTargets, t)
	}
	rt.Close()
	if err := rt.Err(); err != nil {
		return store.StatusCodeDetailResult{}, err
	}

	return out, nil
}

func (s *Store) RequestsByHour(ctx context.Context, groupBy string, now time.Time) ([]store.RequestsByHourBucket, error) {
	switch groupBy {
	case "profile_id", "target_host":
	default:
		return nil, fmt.Errorf("RequestsByHour: invalid groupBy %q", groupBy)
	}

	// Trailing 24h window aligned to floor-of-hour buckets. Upper bound is the
	// start of the next hour so the current in-progress hour's bucket is included
	// (rollup ts_bucket is always a floor-of-hour timestamp).
	to := now.Truncate(time.Hour).Add(time.Hour)
	from := to.Add(-24 * time.Hour)

	q := `SELECT ` + groupBy + `, ts_bucket, SUM(request_count)
	      FROM rollups_1hour
	      WHERE ts_bucket >= ? AND ts_bucket < ?
	        AND ` + groupBy + ` IS NOT NULL AND ` + groupBy + ` != ''
	      GROUP BY ` + groupBy + `, ts_bucket`
	rows, err := s.db.QueryContext(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("requests_by_hour: %w", err)
	}
	defer rows.Close()

	var out []store.RequestsByHourBucket
	for rows.Next() {
		var b store.RequestsByHourBucket
		if err := rows.Scan(&b.Key, &b.Hour, &b.Count); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) DistinctDimensions(ctx context.Context, from, to time.Time) (store.DimensionsResult, error) {
	out := store.DimensionsResult{
		StatusClasses: []string{"2xx", "3xx", "4xx", "5xx"},
	}

	cols := []struct {
		col  string
		dest *[]string
	}{
		{"vendor", &out.Vendors},
		{"type", &out.Types},
		{"region", &out.Regions},
		{"team", &out.Teams},
		{"project", &out.Projects},
	}
	for _, c := range cols {
		q := `SELECT DISTINCT ` + c.col + ` FROM events
		      WHERE ts >= ? AND ts < ? AND ` + c.col + ` IS NOT NULL AND ` + c.col + ` != ''
		      ORDER BY ` + c.col
		rows, err := s.db.QueryContext(ctx, q, from, to)
		if err != nil {
			return store.DimensionsResult{}, fmt.Errorf("dimensions %s: %w", c.col, err)
		}
		var vals []string
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				rows.Close()
				return store.DimensionsResult{}, err
			}
			vals = append(vals, v)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return store.DimensionsResult{}, err
		}
		*c.dest = vals
	}

	profiles, err := s.ListProfiles(ctx)
	if err != nil {
		return store.DimensionsResult{}, fmt.Errorf("dimensions profiles: %w", err)
	}
	for _, p := range profiles {
		out.Profiles = append(out.Profiles, store.DimensionProfile{ID: p.ID, Label: p.Label})
	}
	return out, nil
}
