package audit

import "time"

// Spec is what the user submits via POST /api/v1/audits.
//
// Geo expectations are explicit: ExpectedCountry is required, State/City are
// optional. The audit's check level is derived server-side as
// city > state > country (the deepest non-empty field wins). Provider-specific
// proxy-URL key schemes are no longer the source of geo truth; the URL is used
// only to dial the upstream and (best-effort) to surface a `provider` label
// for history rows.
type Spec struct {
	ProxyURL        string `json:"proxy_url"`
	ExpectedCountry string `json:"expected_country"`
	ExpectedState   string `json:"expected_state,omitempty"`
	ExpectedCity    string `json:"expected_city,omitempty"`
	ExpectedType    string `json:"expected_type"` // residential | mobile | datacenter
	RequestCount    int    `json:"request_count"`
	SessionKey      string `json:"session_key,omitempty"`
	Provider        string `json:"provider,omitempty"`
}

// RunStatus is the lifecycle state of a Run.
type RunStatus string

const (
	StatusRunning   RunStatus = "running"
	StatusCompleted RunStatus = "completed"
	StatusCancelled RunStatus = "cancelled"
	StatusFailed    RunStatus = "failed"
)

// RunSummary is a single row in the History table.
type RunSummary struct {
	ID                 string     `json:"id"`
	StartedAt          time.Time  `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	Status             RunStatus  `json:"status"`
	ProxyURL           string     `json:"proxy_url"` // password-redacted
	Provider           string     `json:"provider,omitempty"`
	ExpectedCountry    string     `json:"expected_country,omitempty"`
	ExpectedState      string     `json:"expected_state,omitempty"`
	ExpectedCity       string     `json:"expected_city,omitempty"`
	ExpectedLat        float64    `json:"expected_lat"`
	ExpectedLon        float64    `json:"expected_lon"`
	ExpectedType       string     `json:"expected_type"`
	CheckLevel         string     `json:"check_level"`
	RequestCount       int        `json:"request_count"`
	CompletedCount     int        `json:"completed_count"`
	LocationMatchCount int        `json:"location_match_count"`
	TypeMatchCount     int        `json:"type_match_count"`
	ErrorCount         int        `json:"error_count"`
	LatencyP50MS       *int       `json:"latency_p50_ms,omitempty"`
	LatencyP95MS       *int       `json:"latency_p95_ms,omitempty"`
	FallbackUsed       bool       `json:"fallback_used"`
	SessionKey         *string    `json:"session_key,omitempty"`
	UniqueIPCount      *int       `json:"unique_ip_count,omitempty"`
	Error              string     `json:"error,omitempty"`
}

// RequestRow is a single row in audit_requests.
type RequestRow struct {
	RunID           string    `json:"-"`
	Seq             int       `json:"seq"`
	StartedAt       time.Time `json:"started_at"`
	DurationMS      int       `json:"duration_ms"`
	ObservedIP      string    `json:"observed_ip,omitempty"`
	ObservedCountry string    `json:"observed_country,omitempty"`
	ObservedState   string    `json:"observed_state,omitempty"`
	ObservedCity    string    `json:"observed_city,omitempty"`
	ObservedLat     float64   `json:"observed_lat,omitempty"`
	ObservedLon     float64   `json:"observed_lon,omitempty"`
	IsDatacenter    bool      `json:"is_datacenter"`
	IsMobile        bool      `json:"is_mobile"`
	IsProxy         bool      `json:"is_proxy"`
	IsVPN           bool      `json:"is_vpn"`
	ASN             int       `json:"asn,omitempty"`
	Company         string    `json:"company,omitempty"`
	LocationMatch   bool      `json:"location_match"`
	TypeMatch       bool      `json:"type_match"`
	Error           string    `json:"error,omitempty"`
	GeoSource       string    `json:"geo_source"`
	Attempts        int       `json:"attempts"`
}

// RunDetail is what GET /api/v1/audits/{id} returns.
type RunDetail struct {
	Run      RunSummary   `json:"run"`
	Requests []RequestRow `json:"requests"`
}

// EventType is the SSE event tag.
type EventType string

const (
	EventRequest  EventType = "request"
	EventSummary  EventType = "summary"
	EventFinished EventType = "finished"
)

// Event is an internal channel message published by *Run to its subscribers.
//
// It is NOT marshaled to JSON directly — the SSE handler (audit/sse.go)
// builds the wire payload per Type via a custom encoder, because the
// {request, summary, finished} variants have different on-wire shapes.
// The json tags on Request and Summary are deliberately "-" because of that.
//
// Status is meaningful only when Type == EventFinished (and carries the
// terminal RunStatus).
type Event struct {
	Type    EventType   `json:"type"`
	Request *RequestRow `json:"-"`
	Summary *RunSummary `json:"-"`
	Status  RunStatus   `json:"status,omitempty"`
}
