package store

import (
	"context"
	"errors"
	"time"
)

// Store is the persistence boundary. One implementation in #1: DuckDB.
type Store interface {
	// --- existing from #1 ---
	ListProfiles(ctx context.Context) ([]Profile, error)
	GetProfile(ctx context.Context, id string) (Profile, error)
	CreateProfile(ctx context.Context, p Profile) error
	UpdateProfile(ctx context.Context, p Profile) error
	DeleteProfile(ctx context.Context, id string) error
	// UpsertProfileObserved inserts a profile keyed by id, leaving an existing
	// row untouched. Used by the auto-discovery path when the proxy observes a
	// new (vendor, type) combo on the wire.
	UpsertProfileObserved(ctx context.Context, p Profile) error
	// ListProfilesWithUsage returns request count and USD spend per profile_id
	// over the trailing window, computed from the events table. Profiles with
	// no events in the window are NOT included; callers LEFT-JOIN against the
	// registry to fill 0/0.
	ListProfilesWithUsage(ctx context.Context, window time.Duration) ([]ProfileUsage, error)

	WriteEvents(ctx context.Context, batch []Event) error
	TailEvents(ctx context.Context, opts TailOptions) ([]Event, error)
	SumBytesInThisMonthByProfile(ctx context.Context) (map[string]int64, error)

	// --- new in #4a ---

	// DistinctDimensions returns the distinct values seen for each dimension
	// in the events table within the [from, to) window. StatusClasses is always
	// the fixed list {"2xx","3xx","4xx","5xx"} regardless of stored data.
	DistinctDimensions(ctx context.Context, from, to time.Time) (DimensionsResult, error)

	// StatusCodeDistribution aggregates events by status_code within the filter window.
	// Rows are returned ordered by Requests descending.
	StatusCodeDistribution(ctx context.Context, f EventFilter) ([]StatusCodeDistRow, error)

	// StatusCodeDetail returns aggregate metrics for one status code, plus its top
	// 5 providers (by requests) and top 5 targets (by requests).
	// Returns ErrNotFound if the code has no events in the filter window.
	StatusCodeDetail(ctx context.Context, code int, f EventFilter) (StatusCodeDetailResult, error)

	// RequestsByHour returns trailing-24h request counts in 1-hour buckets,
	// grouped by the chosen dimension. groupBy must be either "profile_id" or
	// "target_host". `now` defines the trailing window's right edge.
	RequestsByHour(ctx context.Context, groupBy string, now time.Time) ([]RequestsByHourBucket, error)

	Close() error

	// --- new in #2 ---

	// Rollups.
	WriteRollups(ctx context.Context, level string, batch []RollupRow) error
	LatestRollupBucket(ctx context.Context, level string) (time.Time, error) // zero-time if empty
	QueryRollups(ctx context.Context, f RollupFilter) ([]RollupRow, error)

	// Events list with filters + total estimate.
	QueryEvents(ctx context.Context, f EventFilter) (rows []Event, total int64, err error)

	// Retention.
	DeleteEventsBefore(ctx context.Context, ts time.Time) (int64, error)
	DeleteRollupsBefore(ctx context.Context, level string, ts time.Time) (int64, error)

	// Maintenance.
	DBStats(ctx context.Context) (DBStats, error)
	Vacuum(ctx context.Context) error
}

var (
	// ErrNotFound is returned by Store implementations when a lookup by id misses.
	ErrNotFound = errors.New("store: not found")
	// ErrConflict is returned when a write would violate a uniqueness constraint
	// (e.g., creating a profile whose id already exists).
	ErrConflict = errors.New("store: conflict")
)
