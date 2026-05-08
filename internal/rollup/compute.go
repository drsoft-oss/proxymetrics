// Package rollup contains the rollup scheduler and compute helpers.
package rollup

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

// DBExec is the slice of *sql.DB the rollup package needs. Implemented by
// duckdb.Store via the QueryRollupSource accessor.
type DBExec interface {
	QueryRollupSource(ctx context.Context, q string, args ...any) (*sql.Rows, error)
}

// Computer is the slice of store.Store the rollup package needs.
type Computer interface {
	WriteRollups(ctx context.Context, level string, batch []store.RollupRow) error
}

// truncFn returns the SQL expression that truncates `ts` to the bucket boundary
// for the given level.
func truncFn(level string) (expr string, ok bool) {
	switch level {
	case "1min":
		return "date_trunc('minute', ts)", true
	case "1hour":
		return "date_trunc('hour', ts)", true
	case "1day":
		return "CAST(date_trunc('day', ts) AS DATE)", true
	}
	return "", false
}

// Compute aggregates events in [from, to) into rollup rows for the given level
// and writes them via WriteRollups.
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
  CAST(quantile_cont(latency_ms, 0.50) AS INTEGER),
  CAST(quantile_cont(latency_ms, 0.95) AS INTEGER),
  CAST(quantile_cont(latency_ms, 0.99) AS INTEGER),
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
		var region, targetHost, team, project sql.NullString
		var p50, p95, p99 sql.NullInt64
		if err := rows.Scan(
			&r.TSBucket, &r.ProfileID, &r.Vendor, &r.Type, &region,
			&r.StatusClass, &targetHost, &team, &project,
			&r.RequestCount, &r.BytesInTotal, &r.BytesOutTotal,
			&r.LatencyMSAvg, &p50, &p95, &p99,
			&r.CostUSDTotal, &r.SuccessCount, &r.FailureCount,
		); err != nil {
			return err
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
