package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

const eventColumns = `ts, request_id, profile_id, vendor, type, region,
	target_host, target_path_hash, status_code, status_class,
	bytes_in, bytes_out, latency_ms, cost_usd, team, project, captcha_kind`

func (s *Store) WriteEvents(ctx context.Context, batch []store.Event) error {
	if len(batch) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("events tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO events (`+eventColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("events prepare: %w", err)
	}
	defer stmt.Close()

	for _, e := range batch {
		if _, err := stmt.ExecContext(ctx,
			e.TS, e.RequestID, e.ProfileID, e.Vendor, e.Type, nilIfEmpty(e.Region),
			nilIfEmpty(e.TargetHost), nilIfEmpty(e.TargetPathHash),
			nullableInt(e.StatusCode), e.StatusClass,
			e.BytesIn, e.BytesOut, e.LatencyMS, e.CostUSD,
			nilIfEmpty(e.Team), nilIfEmpty(e.Project),
			e.CaptchaKind,
		); err != nil {
			return fmt.Errorf("events insert: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) TailEvents(ctx context.Context, opts store.TailOptions) ([]store.Event, error) {
	n := opts.N
	if n <= 0 {
		n = 50
	}
	q := `SELECT ` + eventColumns + ` FROM events`
	args := []any{}
	if opts.ProfileID != "" {
		q += ` WHERE profile_id = ?`
		args = append(args, opts.ProfileID)
	}
	q += ` ORDER BY ts DESC LIMIT ?`
	args = append(args, n)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("tail: %w", err)
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
			&e.CaptchaKind,
		); err != nil {
			return nil, err
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
	return out, rows.Err()
}

func (s *Store) SumBytesInThisMonthByProfile(ctx context.Context) (map[string]int64, error) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	rows, err := s.db.QueryContext(ctx, `
		SELECT profile_id, COALESCE(SUM(bytes_in), 0)
		FROM events
		WHERE ts >= ?
		GROUP BY profile_id`, monthStart)
	if err != nil {
		return nil, fmt.Errorf("sum bytes: %w", err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var id string
		var sum int64
		if err := rows.Scan(&id, &sum); err != nil {
			return nil, err
		}
		out[id] = sum
	}
	return out, rows.Err()
}

func nullableInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}
