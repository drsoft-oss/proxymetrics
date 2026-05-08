package api

import "time"

// PickSource returns the table name to query for an aggregation over the given
// duration. See spec §8 "Auto-routing".
func PickSource(window time.Duration) string {
	switch {
	case window <= 24*time.Hour:
		return "events"
	case window <= 30*24*time.Hour:
		return "rollups_1hour"
	default:
		return "rollups_1day"
	}
}

// PickLevel returns the rollup level corresponding to a non-events table.
// Returns "" for "events".
func PickLevel(table string) string {
	switch table {
	case "rollups_1min":
		return "1min"
	case "rollups_1hour":
		return "1hour"
	case "rollups_1day":
		return "1day"
	}
	return ""
}
