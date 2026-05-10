// Package store defines the persistence interface and data types for ProxyMetrics.
package store

import "time"

// Profile is the registered upstream proxy with vendor metadata and pricing.
type Profile struct {
	ID                string    `json:"id"`
	Label             string    `json:"label"`
	Vendor            string    `json:"vendor"` // brightdata|oxylabs|iproyal|smartproxy|anonymous-proxies|custom
	Type              string    `json:"type"`   // residential|isp|datacenter|mobile
	Region            string    `json:"region,omitempty"`
	UpstreamURL       string    `json:"upstream_url"`
	PricePerGB        *float64  `json:"price_per_gb,omitempty"`
	PricePerGBOverage *float64  `json:"price_per_gb_overage,omitempty"`
	IncludedGB        *float64  `json:"included_gb,omitempty"`
	Currency          string    `json:"currency"`
	DefaultTeam       string    `json:"default_team,omitempty"`
	DefaultProject    string    `json:"default_project,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Event is one row in the events table.
type Event struct {
	TS             time.Time
	RequestID      string
	ProfileID      string
	Vendor         string
	Type           string
	Region         string
	TargetHost     string
	TargetPathHash string
	StatusCode     int
	StatusClass    string
	BytesIn        int64
	BytesOut       int64
	LatencyMS      int
	CostUSD        float64
	Team           string
	Project        string
	CaptchaKind    string // "" = not measured; otherwise: recaptcha|turnstile|hcaptcha|datadome|arkose
}

// TailOptions filters TailEvents.
type TailOptions struct {
	N         int
	ProfileID string // empty = all profiles
}

// DimensionsResult is what Store.DistinctDimensions returns.
// All slices are sorted ascending. ProfileIDs returns id+label pairs because
// the URL filter encodes the id while the chip label shows the name.
type DimensionsResult struct {
	Vendors       []string
	Types         []string
	Regions       []string
	Teams         []string
	Projects      []string
	StatusClasses []string // always {"2xx","3xx","4xx","5xx"} regardless of data
	Profiles      []DimensionProfile
}

// DimensionProfile is one entry in DimensionsResult.Profiles.
type DimensionProfile struct {
	ID    string
	Label string
}

// StatusCodeDistRow is one row of the per-code distribution.
type StatusCodeDistRow struct {
	Code              int
	Class             string
	Requests          int64
	WastedUSD         float64
	SpendUSD          float64
	TopProviderVendor string // vendor with the most requests for this code (may be empty)
}

// StatusCodeDetailResult is the aggregate for a single status code.
type StatusCodeDetailResult struct {
	Code         int
	Class        string
	Requests     int64
	WastedUSD    float64
	SpendUSD     float64
	TopProviders []StatusCodeDetailProvider // up to 5, descending by requests
	TopTargets   []StatusCodeDetailTarget   // up to 5, descending by requests
}

// StatusCodeDetailProvider is one entry in StatusCodeDetailResult.TopProviders.
type StatusCodeDetailProvider struct {
	Vendor    string
	Requests  int64
	WastedUSD float64
	SpendUSD  float64
}

// StatusCodeDetailTarget is one entry in StatusCodeDetailResult.TopTargets.
type StatusCodeDetailTarget struct {
	Host      string
	Requests  int64
	WastedUSD float64
	SpendUSD  float64
}

// RequestsByHourBucket is one element of the trailing-24h sparkline series.
// Key is one of: ProfileID (provider sparklines) or TargetHost (target sparklines).
// Hour is the floor-of-hour timestamp; Count is the request count in that hour.
type RequestsByHourBucket struct {
	Key   string
	Hour  time.Time
	Count int64
}

// ProfileUsage is a per-profile aggregation over the events table
// within a trailing time window.
type ProfileUsage struct {
	ProfileID string
	Requests  int64
	SpendUSD  float64
}

// CaptchaKindDetailResult is the aggregate for one captcha kind in a window.
type CaptchaKindDetailResult struct {
	Kind         string
	Requests     int64
	TopProviders []CaptchaKindDetailProvider
	TopTargets   []CaptchaKindDetailTarget
}

// CaptchaKindDetailProvider is one entry in CaptchaKindDetailResult.TopProviders.
type CaptchaKindDetailProvider struct {
	Vendor   string
	Type     string
	Requests int64
}

// CaptchaKindDetailTarget is one entry in CaptchaKindDetailResult.TopTargets.
type CaptchaKindDetailTarget struct {
	Host     string
	Requests int64
}
