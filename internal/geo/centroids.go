package geo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Centroid is the lat/lon center of a place.
type Centroid struct {
	Lat float64
	Lon float64
}

// CentroidCache persists resolved centroids so repeat audits skip Nominatim.
// The audit module's DuckDB-backed implementation lives in internal/audit/store.go;
// tests use an in-memory fake.
type CentroidCache interface {
	GetCentroid(ctx context.Context, country, state, city string) (Centroid, bool, error)
	PutCentroid(ctx context.Context, country, state, city string, lat, lon float64) error
}

// CentroidConfig configures a CentroidResolver.
type CentroidConfig struct {
	BaseURL   string        // e.g. https://nominatim.openstreetmap.org/search
	UserAgent string        // required by Nominatim's usage policy
	Timeout   time.Duration // per-request timeout
	Cache     CentroidCache
}

// CentroidResolver resolves expected-place tuples to lat/lon via Nominatim,
// caching results.
type CentroidResolver struct {
	cfg CentroidConfig
	hc  *http.Client
}

// NewCentroidResolver constructs a resolver. cfg.Cache must be non-nil; nil
// panics. Production callers wire a DuckDB-backed cache; tests pass an
// in-memory fake. Without a cache the resolver would issue an uncached
// Nominatim request per audit, which would breach Nominatim's usage policy
// immediately.
func NewCentroidResolver(cfg CentroidConfig) *CentroidResolver {
	if cfg.Cache == nil {
		panic("geo: NewCentroidResolver: cfg.Cache is required")
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &CentroidResolver{
		cfg: cfg,
		hc:  &http.Client{Timeout: timeout},
	}
}

// Resolve returns the centroid for (country, state, city). Empty state/city
// strings are dropped from the query so a country-only call works.
func (r *CentroidResolver) Resolve(ctx context.Context, country, state, city string) (Centroid, error) {
	if hit, ok, err := r.cfg.Cache.GetCentroid(ctx, country, state, city); err == nil && ok {
		return hit, nil
	}

	parts := []string{}
	for _, p := range []string{city, state, country} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return Centroid{}, errors.New("centroid: empty query")
	}
	q := url.Values{}
	q.Set("q", strings.Join(parts, ","))
	q.Set("format", "json")
	q.Set("limit", "1")

	full := r.cfg.BaseURL + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return Centroid{}, err
	}
	req.Header.Set("User-Agent", r.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := r.hc.Do(req)
	if err != nil {
		return Centroid{}, fmt.Errorf("centroid: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return Centroid{}, fmt.Errorf("centroid: nominatim http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Centroid{}, err
	}
	var rows []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return Centroid{}, fmt.Errorf("centroid: decode: %w", err)
	}
	if len(rows) == 0 {
		return Centroid{}, errors.New("centroid: no result from nominatim")
	}
	lat, err := strconv.ParseFloat(rows[0].Lat, 64)
	if err != nil {
		return Centroid{}, fmt.Errorf("centroid: bad lat: %w", err)
	}
	lon, err := strconv.ParseFloat(rows[0].Lon, 64)
	if err != nil {
		return Centroid{}, fmt.Errorf("centroid: bad lon: %w", err)
	}
	out := Centroid{Lat: lat, Lon: lon}
	_ = r.cfg.Cache.PutCentroid(ctx, country, state, city, lat, lon)
	return out, nil
}
