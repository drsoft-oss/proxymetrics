package store

import "time"

// EventFilter constrains a QueryEvents call.
type EventFilter struct {
	From, To      time.Time
	ProfileIDs    []string
	Vendors       []string
	Types         []string
	Regions       []string
	Teams         []string
	Projects      []string
	StatusClasses []string
	StatusCodes   []int
	TargetHost    string
	Q             string
	Sort, Order   string
	Limit, Offset int
}

// RollupFilter constrains a QueryRollups call.
type RollupFilter struct {
	Level         string // "1min" | "1hour" | "1day"
	From, To      time.Time
	ProfileIDs    []string
	Vendors       []string
	Types         []string
	Regions       []string
	Teams         []string
	Projects      []string
	StatusClasses []string
	TargetHost    string
	Q             string
	Sort, Order   string
	Limit, Offset int
}

// RollupRow is one row returned from a rollup table.
type RollupRow struct {
	TSBucket      time.Time
	ProfileID     string
	Vendor        string
	Type          string
	Region        string
	StatusClass   string
	TargetHost    string
	Team          string
	Project       string
	RequestCount  int64
	BytesInTotal  int64
	BytesOutTotal int64
	LatencyMSAvg  float64
	LatencyMSP50  int
	LatencyMSP95  int
	LatencyMSP99  int
	CostUSDTotal  float64
	SuccessCount  int64
	FailureCount  int64
}

// DBStats is what `db stats` returns.
type DBStats struct {
	SizeBytes     int64
	OldestEventTS time.Time
	NewestEventTS time.Time
	Rows          map[string]int64
}
