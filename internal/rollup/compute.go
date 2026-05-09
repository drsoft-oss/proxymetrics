// Package rollup contains the rollup scheduler and compute helpers.
package rollup

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// DBExec is the slice of *sql.DB the rollup package needs. Implemented by
// sqlite.Store via the QueryRollupSource accessor.
type DBExec interface {
	QueryRollupSource(ctx context.Context, q string, args ...any) (*sql.Rows, error)
}

// Computer is the slice of store.Store the rollup package needs.
type Computer interface {
	WriteRollups(ctx context.Context, level string, batch []store.RollupRow) error
}

// truncFn returns the SQL expression that truncates `ts` to the bucket boundary
// for the given level.
//
// modernc.org/sqlite serializes time.Time via Go's default String() format
// ("2006-01-02 15:04:05.999999999 -0700 MST"), which SQLite's strftime cannot
// parse. The first 19 chars are always "YYYY-MM-DD HH:MM:SS" — substr extracts
// the strftime-compatible prefix. All event timestamps are inserted in UTC, so
// dropping the suffix is lossless for our use.
func truncFn(level string) (expr string, ok bool) {
	switch level {
	case "1min":
		return "strftime('%Y-%m-%d %H:%M:00', substr(ts, 1, 19))", true
	case "1hour":
		return "strftime('%Y-%m-%d %H:00:00', substr(ts, 1, 19))", true
	case "1day":
		return "strftime('%Y-%m-%d 00:00:00', substr(ts, 1, 19))", true
	}
	return "", false
}

// Compute aggregates events in [from, to) into rollup rows for the given level
// and writes them via WriteRollups.
//
// Percentile aggregates (p50/p95/p99) are computed in Go from a comma-separated
// GROUP_CONCAT of latency values, since SQLite has no built-in quantile_cont.
// For typical rollup-window cardinalities this is well below SQLite's
// SQLITE_MAX_LENGTH limit.
func Compute(ctx context.Context, c interface {
	Computer
	DBExec
}, level string, from, to time.Time) error {
	bucketExpr, ok := truncFn(level)
	if !ok {
		return fmt.Errorf("rollup: invalid level %q", level)
	}

	q := fmt.Sprintf(`
SELECT
  %s AS ts_bucket,
  profile_id, vendor, type, region, status_class, target_host, team, project,
  COUNT(*),
  SUM(bytes_in),
  SUM(bytes_out),
  AVG(latency_ms),
  GROUP_CONCAT(latency_ms),
  SUM(cost_usd),
  SUM(CASE WHEN status_class = '2xx' THEN 1 ELSE 0 END),
  SUM(CASE WHEN status_class != '2xx' THEN 1 ELSE 0 END)
FROM events
WHERE ts >= ? AND ts < ?
GROUP BY 1, profile_id, vendor, type, region, status_class, target_host, team, project
`, bucketExpr)

	rows, err := c.QueryRollupSource(ctx, q, from, to)
	if err != nil {
		return fmt.Errorf("rollup compute query: %w", err)
	}
	defer rows.Close()

	var batch []store.RollupRow
	for rows.Next() {
		var r store.RollupRow
		var bucketStr string
		var region, targetHost, team, project sql.NullString
		var latencies sql.NullString
		if err := rows.Scan(
			&bucketStr, &r.ProfileID, &r.Vendor, &r.Type, &region,
			&r.StatusClass, &targetHost, &team, &project,
			&r.RequestCount, &r.BytesInTotal, &r.BytesOutTotal,
			&r.LatencyMSAvg, &latencies,
			&r.CostUSDTotal, &r.SuccessCount, &r.FailureCount,
		); err != nil {
			return err
		}
		// strftime returns TEXT — parse explicitly so we don't rely on driver
		// affinity for an aggregated column.
		bucket, err := time.Parse("2006-01-02 15:04:05", bucketStr)
		if err != nil {
			return fmt.Errorf("rollup parse bucket %q: %w", bucketStr, err)
		}
		r.TSBucket = bucket
		r.Region = region.String
		r.TargetHost = targetHost.String
		r.Team = team.String
		r.Project = project.String
		if latencies.Valid {
			p50, p95, p99 := percentiles(latencies.String)
			r.LatencyMSP50 = p50
			r.LatencyMSP95 = p95
			r.LatencyMSP99 = p99
		}
		batch = append(batch, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(batch) == 0 {
		return nil
	}
	return c.WriteRollups(ctx, level, batch)
}

// percentiles parses a comma-separated list of integer latencies and returns
// p50/p95/p99 using nearest-rank. Returns 0,0,0 for empty input.
func percentiles(csv string) (p50, p95, p99 int) {
	if csv == "" {
		return 0, 0, 0
	}
	parts := strings.Split(csv, ",")
	xs := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			continue
		}
		xs = append(xs, n)
	}
	if len(xs) == 0 {
		return 0, 0, 0
	}
	sort.Ints(xs)
	return rank(xs, 0.50), rank(xs, 0.95), rank(xs, 0.99)
}

// rank returns the nearest-rank percentile of a sorted slice. q in [0,1].
func rank(sorted []int, q float64) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted))*q + 0.5)
	if idx <= 0 {
		idx = 1
	}
	if idx > len(sorted) {
		idx = len(sorted)
	}
	return sorted[idx-1]
}
