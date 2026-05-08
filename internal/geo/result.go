// Package geo resolves geo-IP and city-centroid data for the audit module.
//
// Two backends are available behind a small Lookup interface:
//   - IPAPIIs: an HTTPS call to api.ipapi.is, routed through whatever
//     http.Client the caller passes (typically a client that exits via the
//     proxy under test).
//   - MaxMind: a local *.mmdb database lookup, fed an IP discovered via the
//     existing internal/ipcheck package.
//
// Centroid resolution (for converting an expected place into lat/lon)
// uses Nominatim and is cached in DuckDB; it lives alongside the Lookup
// implementations because both produce coordinates.
package geo

// Result is a single geo + connection-type observation.
type Result struct {
	IP           string
	Country      string  // ISO-2 uppercase
	State        string
	City         string
	Lat          float64
	Lon          float64
	IsDatacenter bool
	IsMobile     bool
	IsProxy      bool
	IsVPN        bool
	IsTor        bool
	ASN          int
	Company      string
	Source       string // "ipapi" | "maxmind"
}
