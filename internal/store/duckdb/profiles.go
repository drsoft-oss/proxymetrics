package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

const profileColumns = `id, label, vendor, type, region, upstream_url,
	price_per_gb, price_per_gb_overage, included_gb, currency,
	default_team, default_project, created_at, updated_at`

func (s *Store) ListProfiles(ctx context.Context) ([]store.Profile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+profileColumns+` FROM profiles ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}
	defer rows.Close()
	var out []store.Profile
	for rows.Next() {
		p, err := scanProfile(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProfile(ctx context.Context, id string) (store.Profile, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+profileColumns+` FROM profiles WHERE id = ?`, id)
	p, err := scanProfile(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return store.Profile{}, store.ErrNotFound
	}
	return p, err
}

func (s *Store) CreateProfile(ctx context.Context, p store.Profile) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO profiles (id, label, vendor, type, region, upstream_url,
			price_per_gb, price_per_gb_overage, included_gb, currency,
			default_team, default_project)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Label, p.Vendor, p.Type, nilIfEmpty(p.Region), p.UpstreamURL,
		p.PricePerGB, p.PricePerGBOverage, p.IncludedGB, defaultStr(p.Currency, "USD"),
		nilIfEmpty(p.DefaultTeam), nilIfEmpty(p.DefaultProject),
	)
	if err != nil {
		if isPKViolation(err) {
			return store.ErrConflict
		}
		return fmt.Errorf("create profile: %w", err)
	}
	return nil
}

func (s *Store) UpdateProfile(ctx context.Context, p store.Profile) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE profiles SET
			label = ?, vendor = ?, type = ?, region = ?, upstream_url = ?,
			price_per_gb = ?, price_per_gb_overage = ?, included_gb = ?, currency = ?,
			default_team = ?, default_project = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		p.Label, p.Vendor, p.Type, nilIfEmpty(p.Region), p.UpstreamURL,
		p.PricePerGB, p.PricePerGBOverage, p.IncludedGB, defaultStr(p.Currency, "USD"),
		nilIfEmpty(p.DefaultTeam), nilIfEmpty(p.DefaultProject),
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

// UpsertProfileObserved inserts a profile row keyed by id, doing nothing if
// the row already exists. It is the auto-discovery path used by the event
// writer when a previously-unseen (vendor, type) combo lands. Existing rows
// are preserved verbatim so the admin path remains the source of truth for
// editable fields like label/price.
func (s *Store) UpsertProfileObserved(ctx context.Context, p store.Profile) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO profiles (id, label, vendor, type, region, upstream_url,
			price_per_gb, price_per_gb_overage, included_gb, currency,
			default_team, default_project)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO NOTHING`,
		p.ID, p.Label, p.Vendor, p.Type, nilIfEmpty(p.Region), p.UpstreamURL,
		p.PricePerGB, p.PricePerGBOverage, p.IncludedGB, defaultStr(p.Currency, "USD"),
		nilIfEmpty(p.DefaultTeam), nilIfEmpty(p.DefaultProject),
	)
	if err != nil {
		return fmt.Errorf("upsert profile observed: %w", err)
	}
	return nil
}

func (s *Store) DeleteProfile(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

// scanProfile is shared between row-at-a-time and single-row queries.
func scanProfile(scan func(...any) error) (store.Profile, error) {
	var p store.Profile
	var region, defTeam, defProj sql.NullString
	if err := scan(
		&p.ID, &p.Label, &p.Vendor, &p.Type, &region, &p.UpstreamURL,
		&p.PricePerGB, &p.PricePerGBOverage, &p.IncludedGB, &p.Currency,
		&defTeam, &defProj, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return store.Profile{}, err
	}
	p.Region = region.String
	p.DefaultTeam = defTeam.String
	p.DefaultProject = defProj.String
	return p, nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func defaultStr(s, dflt string) string {
	if s == "" {
		return dflt
	}
	return s
}

// DuckDB surfaces primary-key violations through error messages.
func isPKViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Duplicate key")
}

// ListProfilesWithUsage returns aggregated request count and USD spend per
// profile_id over the trailing window, computed from the events table.
// Profiles with no events in the window are NOT included; callers LEFT-JOIN
// against the registry to fill 0/0.
func (s *Store) ListProfilesWithUsage(ctx context.Context, window time.Duration) ([]store.ProfileUsage, error) {
	if window <= 0 {
		return nil, fmt.Errorf("list profiles with usage: window must be positive, got %v", window)
	}
	cutoff := time.Now().UTC().Add(-window)
	rows, err := s.db.QueryContext(ctx, `
		SELECT profile_id,
		       COUNT(*)::BIGINT AS requests,
		       COALESCE(SUM(cost_usd), 0)::DOUBLE AS spend_usd
		FROM events
		WHERE ts >= ?
		GROUP BY profile_id`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("list profiles with usage: %w", err)
	}
	defer rows.Close()
	var out []store.ProfileUsage
	for rows.Next() {
		var u store.ProfileUsage
		if err := rows.Scan(&u.ProfileID, &u.Requests, &u.SpendUSD); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
