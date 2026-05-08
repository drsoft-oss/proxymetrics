package audit

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/geo"
)

// Store wraps a *sql.DB (DuckDB handle) and exposes the audit module's
// persistence operations. The audit package owns the SQL.
type Store struct{ db *sql.DB }

// NewStore constructs a Store. The db is borrowed; the caller closes it.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// InsertRun writes a fresh row in audit_runs (status=running, aggregates 0).
func (s *Store) InsertRun(ctx context.Context, r RunSummary) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_runs
(id, started_at, status, proxy_url, provider,
 expected_country, expected_state, expected_city,
 expected_lat, expected_lon, expected_type, check_level, request_count, session_key)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.StartedAt, string(r.Status), r.ProxyURL, nullIfEmpty(r.Provider),
		nullIfEmpty(r.ExpectedCountry), nullIfEmpty(r.ExpectedState), nullIfEmpty(r.ExpectedCity),
		r.ExpectedLat, r.ExpectedLon, r.ExpectedType, r.CheckLevel, r.RequestCount,
		nullIfNilString(r.SessionKey),
	)
	return err
}

// AppendRequest writes one audit_requests row.
func (s *Store) AppendRequest(ctx context.Context, r RequestRow) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_requests
(run_id, seq, started_at, duration_ms,
 observed_ip, observed_country, observed_state, observed_city,
 observed_lat, observed_lon,
 is_datacenter, is_mobile, is_proxy, is_vpn,
 asn, company, location_match, type_match, error, geo_source, attempts)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.RunID, r.Seq, r.StartedAt, r.DurationMS,
		nullIfEmpty(r.ObservedIP), nullIfEmpty(r.ObservedCountry),
		nullIfEmpty(r.ObservedState), nullIfEmpty(r.ObservedCity),
		nullIfZero(r.ObservedLat), nullIfZero(r.ObservedLon),
		r.IsDatacenter, r.IsMobile, r.IsProxy, r.IsVPN,
		nullIfZeroInt(r.ASN), nullIfEmpty(r.Company),
		r.LocationMatch, r.TypeMatch, nullIfEmpty(r.Error), r.GeoSource,
		atLeastOne(r.Attempts),
	)
	return err
}

// FinaliseRun updates the aggregate columns + status + finished_at on a run.
func (s *Store) FinaliseRun(ctx context.Context, r RunSummary) error {
	_, err := s.db.ExecContext(ctx, `UPDATE audit_runs SET
 finished_at = ?,
 status = ?,
 completed_count = ?,
 location_match_count = ?,
 type_match_count = ?,
 error_count = ?,
 latency_p50_ms = ?,
 latency_p95_ms = ?,
 fallback_used = ?,
 error = ?,
 unique_ip_count = ?
 WHERE id = ?`,
		nullIfNilTime(r.FinishedAt), string(r.Status),
		r.CompletedCount, r.LocationMatchCount, r.TypeMatchCount, r.ErrorCount,
		nullIfNilInt(r.LatencyP50MS), nullIfNilInt(r.LatencyP95MS),
		r.FallbackUsed, nullIfEmpty(r.Error),
		nullIfNilInt(r.UniqueIPCount), r.ID,
	)
	return err
}

// MarkOrphanedRunningFailed sets every status='running' row to 'failed' with a
// canned error. Called once at server startup to clean up stale runs.
func (s *Store) MarkOrphanedRunningFailed(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `UPDATE audit_runs
 SET status='failed', finished_at=CURRENT_TIMESTAMP, error='server restarted during run'
 WHERE status='running'`)
	return err
}

// GetRun returns a single run by id; sql.ErrNoRows on miss.
func (s *Store) GetRun(ctx context.Context, id string) (RunSummary, error) {
	row := s.db.QueryRowContext(ctx, `SELECT
 id, started_at, finished_at, status, proxy_url, provider,
 expected_country, expected_state, expected_city,
 expected_lat, expected_lon, expected_type, check_level, request_count,
 completed_count, location_match_count, type_match_count, error_count,
 latency_p50_ms, latency_p95_ms, fallback_used, error,
 session_key, unique_ip_count
 FROM audit_runs WHERE id = ?`, id)
	return scanRun(row)
}

// ListRuns returns all runs ordered by started_at DESC, paginated.
func (s *Store) ListRuns(ctx context.Context, limit, offset int) ([]RunSummary, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
 id, started_at, finished_at, status, proxy_url, provider,
 expected_country, expected_state, expected_city,
 expected_lat, expected_lon, expected_type, check_level, request_count,
 completed_count, location_match_count, type_match_count, error_count,
 latency_p50_ms, latency_p95_ms, fallback_used, error,
 session_key, unique_ip_count
 FROM audit_runs ORDER BY started_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RunSummary
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListRequests returns all per-request rows for a run, ordered by seq.
func (s *Store) ListRequests(ctx context.Context, runID string) ([]RequestRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
 run_id, seq, started_at, duration_ms,
 observed_ip, observed_country, observed_state, observed_city,
 observed_lat, observed_lon,
 is_datacenter, is_mobile, is_proxy, is_vpn,
 asn, company, location_match, type_match, error, geo_source, attempts
 FROM audit_requests WHERE run_id = ? ORDER BY seq ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RequestRow
	for rows.Next() {
		var r RequestRow
		var ip, ctry, st, ci, comp, errs sql.NullString
		var lat, lon sql.NullFloat64
		var asn, attempts sql.NullInt64
		if err := rows.Scan(
			&r.RunID, &r.Seq, &r.StartedAt, &r.DurationMS,
			&ip, &ctry, &st, &ci, &lat, &lon,
			&r.IsDatacenter, &r.IsMobile, &r.IsProxy, &r.IsVPN,
			&asn, &comp, &r.LocationMatch, &r.TypeMatch, &errs, &r.GeoSource,
			&attempts,
		); err != nil {
			return nil, err
		}
		r.ObservedIP = ip.String
		r.ObservedCountry = ctry.String
		r.ObservedState = st.String
		r.ObservedCity = ci.String
		r.ObservedLat = lat.Float64
		r.ObservedLon = lon.Float64
		r.ASN = int(asn.Int64)
		r.Company = comp.String
		r.Error = errs.String
		if attempts.Valid {
			r.Attempts = int(attempts.Int64)
		} else {
			r.Attempts = 1
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListDistinctProviders returns the case-folded-distinct provider strings
// that appear in audit_runs (NULLs excluded), sorted ASC. Empty result is
// a valid (non-error) response.
func (s *Store) ListDistinctProviders(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT provider FROM audit_runs WHERE provider IS NOT NULL ORDER BY LOWER(provider)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		if p != "" {
			out = append(out, p)
		}
	}
	return out, rows.Err()
}

// --- centroid cache (implements geo.CentroidCache) ---

func (s *Store) GetCentroid(ctx context.Context, country, state, city string) (geo.Centroid, bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT lat, lon FROM geo_city_centroids
 WHERE country_code=? AND state_norm=? AND city_norm=?`, country, state, city)
	var lat, lon float64
	if err := row.Scan(&lat, &lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return geo.Centroid{}, false, nil
		}
		return geo.Centroid{}, false, err
	}
	return geo.Centroid{Lat: lat, Lon: lon}, true, nil
}

func (s *Store) PutCentroid(ctx context.Context, country, state, city string, lat, lon float64) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO geo_city_centroids
 (country_code, state_norm, city_norm, lat, lon, resolved_at)
 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`, country, state, city, lat, lon)
	return err
}

// --- helpers ---

type rowScanner interface {
	Scan(dst ...any) error
}

func scanRun(s rowScanner) (RunSummary, error) {
	var r RunSummary
	var finishedAt sql.NullTime
	var prov, ec, es, eci sql.NullString
	var p50, p95 sql.NullInt64
	var errs sql.NullString
	var status string
	var sessionKey sql.NullString
	var uniqueIPs sql.NullInt64
	if err := s.Scan(
		&r.ID, &r.StartedAt, &finishedAt, &status, &r.ProxyURL, &prov,
		&ec, &es, &eci, &r.ExpectedLat, &r.ExpectedLon, &r.ExpectedType,
		&r.CheckLevel, &r.RequestCount,
		&r.CompletedCount, &r.LocationMatchCount, &r.TypeMatchCount, &r.ErrorCount,
		&p50, &p95, &r.FallbackUsed, &errs,
		&sessionKey, &uniqueIPs,
	); err != nil {
		return RunSummary{}, err
	}
	r.Status = RunStatus(status)
	if finishedAt.Valid {
		t := finishedAt.Time
		r.FinishedAt = &t
	}
	r.Provider = prov.String
	r.ExpectedCountry = ec.String
	r.ExpectedState = es.String
	r.ExpectedCity = eci.String
	if p50.Valid {
		v := int(p50.Int64)
		r.LatencyP50MS = &v
	}
	if p95.Valid {
		v := int(p95.Int64)
		r.LatencyP95MS = &v
	}
	r.Error = errs.String
	if sessionKey.Valid {
		v := sessionKey.String
		r.SessionKey = &v
	}
	if uniqueIPs.Valid {
		v := int(uniqueIPs.Int64)
		r.UniqueIPCount = &v
	}
	return r, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func nullIfZero(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}
func nullIfZeroInt(n int) any {
	if n == 0 {
		return nil
	}
	return n
}
func nullIfNilInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullIfNilTime(p *time.Time) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullIfNilString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
func atLeastOne(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
