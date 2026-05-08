package audit

import (
	"math"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/anonymous-proxies/proxymetrics/internal/geo"
)

// Expected captures what the user said the proxy should be.
type Expected struct {
	Country    string  // ISO-2 uppercase
	State      string  // raw user input; normalized inside Evaluate
	City       string  // raw user input
	Lat        float64 // expected city's centroid; only used at city level
	Lon        float64
	Type       string // residential | mobile | datacenter
	CheckLevel string // country | state | city
}

// Verdict is the pure outcome of comparing expected vs observed.
type Verdict struct {
	LocationMatch bool
	TypeMatch     bool
}

// Evaluate is a pure function: given an Expected and a geo.Result, decide
// whether location and type pass.
func Evaluate(e Expected, o geo.Result) Verdict {
	return Verdict{
		LocationMatch: locationMatch(e, o),
		TypeMatch:     typeMatch(e.Type, o),
	}
}

func locationMatch(e Expected, o geo.Result) bool {
	switch e.CheckLevel {
	case "country":
		return strings.EqualFold(e.Country, o.Country)
	case "state":
		if !strings.EqualFold(e.Country, o.Country) {
			return false
		}
		return normalizePlace(e.State) == normalizePlace(o.State)
	case "city":
		if !strings.EqualFold(e.Country, o.Country) {
			return false
		}
		if o.Lat == 0 && o.Lon == 0 {
			return false
		}
		return haversineKM(e.Lat, e.Lon, o.Lat, o.Lon) <= 50.0
	}
	return false
}

func typeMatch(expected string, o geo.Result) bool {
	switch expected {
	case "residential":
		return !o.IsDatacenter && !o.IsMobile
	case "mobile":
		return o.IsMobile
	case "datacenter":
		return o.IsDatacenter
	}
	return false
}

// normalizePlace lowercases, strips diacritics, and trims spaces.
func normalizePlace(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		out = s
	}
	return strings.TrimSpace(strings.ToLower(out))
}

// haversineKM returns the great-circle distance in kilometres between two
// (lat, lon) points using the Haversine formula.
func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0 // Earth radius in km
	rad := math.Pi / 180.0
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}
