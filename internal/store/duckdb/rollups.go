package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// rollupTableForLevel maps "1min" / "1hour" / "1day" to the actual table name.
// Returns empty string for invalid input — callers should reject.
func rollupTableForLevel(level string) string {
	switch level {
	case "1min":
		return "rollups_1min"
	case "1hour":
		return "rollups_1hour"
	case "1day":
		return "rollups_1day"
	}
	return ""
}

const rollupColumns = `ts_bucket, profile_id, vendor, type, region, status_class,
	target_host, team, project,
	request_count, bytes_in_total, bytes_out_total,
	latency_ms_avg, latency_ms_p50, latency_ms_p95, latency_ms_p99,
	cost_usd_total, success_count, failure_count`

func (s *Store) WriteRollups(ctx context.Context, level string, batch []store.RollupRow) error {
	if len(batch) == 0 {
		return nil
	}
	table := rollupTableForLevel(level)
	if table == "" {
		return fmt.Errorf("rollups: invalid level %q", level)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("rollups tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO `+table+` (`+rollupColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("rollups prepare: %w", err)
	}
	defer stmt.Close()

	for _, r := range batch {
		if _, err := stmt.ExecContext(ctx,
			r.TSBucket, r.ProfileID, r.Vendor, r.Type, nilIfEmpty(r.Region),
			r.StatusClass, nilIfEmpty(r.TargetHost), nilIfEmpty(r.Team), nilIfEmpty(r.Project),
			r.RequestCount, r.BytesInTotal, r.BytesOutTotal,
			r.LatencyMSAvg, nullableInt(r.LatencyMSP50), nullableInt(r.LatencyMSP95), nullableInt(r.LatencyMSP99),
			r.CostUSDTotal, r.SuccessCount, r.FailureCount,
		); err != nil {
			return fmt.Errorf("rollups insert: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) LatestRollupBucket(ctx context.Context, level string) (time.Time, error) {
	table := rollupTableForLevel(level)
	if table == "" {
		return time.Time{}, fmt.Errorf("rollups: invalid level %q", level)
	}
	var ts sql.NullTime
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(ts_bucket) FROM `+table).Scan(&ts); err != nil {
		return time.Time{}, fmt.Errorf("latest rollup bucket: %w", err)
	}
	if !ts.Valid {
		return time.Time{}, nil
	}
	return ts.Time, nil
}

func (s *Store) QueryRollups(ctx context.Context, f store.RollupFilter) ([]store.RollupRow, error) {
	table := rollupTableForLevel(f.Level)
	if table == "" {
		return nil, fmt.Errorf("rollups: invalid level %q", f.Level)
	}

	var (
		where []string
		args  []any
	)
	if !f.From.IsZero() {
		where = append(where, "ts_bucket >= ?")
		args = append(args, f.From)
	}
	if !f.To.IsZero() {
		where = append(where, "ts_bucket < ?")
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
	if f.TargetHost != "" {
		where = append(where, "target_host = ?")
		args = append(args, f.TargetHost)
	} else if f.Q != "" {
		where = append(where, "target_host LIKE ?")
		args = append(args, "%"+f.Q+"%")
	}

	q := `SELECT ` + rollupColumns + ` FROM ` + table
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}

	if f.Sort != "" && allowedSortCol(f.Sort) {
		order := "DESC"
		if strings.EqualFold(f.Order, "asc") {
			order = "ASC"
		}
		q += " ORDER BY " + f.Sort + " " + order
	} else {
		q += " ORDER BY ts_bucket DESC"
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
		return nil, fmt.Errorf("rollups query: %w", err)
	}
	defer rows.Close()

	var out []store.RollupRow
	for rows.Next() {
		var r store.RollupRow
		var region, targetHost, team, project sql.NullString
		var p50, p95, p99 sql.NullInt64
		if err := rows.Scan(
			&r.TSBucket, &r.ProfileID, &r.Vendor, &r.Type, &region,
			&r.StatusClass, &targetHost, &team, &project,
			&r.RequestCount, &r.BytesInTotal, &r.BytesOutTotal,
			&r.LatencyMSAvg, &p50, &p95, &p99,
			&r.CostUSDTotal, &r.SuccessCount, &r.FailureCount,
		); err != nil {
			return nil, err
		}
		r.Region = region.String
		r.TargetHost = targetHost.String
		r.Team = team.String
		r.Project = project.String
		if p50.Valid {
			r.LatencyMSP50 = int(p50.Int64)
		}
		if p95.Valid {
			r.LatencyMSP95 = int(p95.Int64)
		}
		if p99.Valid {
			r.LatencyMSP99 = int(p99.Int64)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func allowedSortCol(name string) bool {
	switch name {
	case "ts_bucket", "request_count", "bytes_in_total", "bytes_out_total",
		"latency_ms_avg", "latency_ms_p50", "latency_ms_p95", "latency_ms_p99",
		"cost_usd_total", "success_count", "failure_count":
		return true
	}
	return false
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
