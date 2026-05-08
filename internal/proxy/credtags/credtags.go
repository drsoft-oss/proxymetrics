// Package credtags lifts ProxyMetrics-only metadata out of the proxy-auth
// username and returns an upstream-safe username with those tags stripped.
//
// We care about exactly three keys — everything else in the username is the
// upstream provider's business and is forwarded verbatim:
//
//   provider — which upstream vendor this request is for
//   type     — proxy type (residential, datacenter, isp, mobile)
//   price    — integer cents per GB, used to attribute cost
//
// Wire format: dash-delimited "key-value" pairs anywhere in the username.
//
// Example input:
//
//   brd-customer-XYZ-zone-residential-country-us-city-newyork-provider-brightdata-type-residential-price-1200
//
// Yields:
//
//   Tags{Provider: "brightdata", Type: "residential", PriceCentsPerGB: 1200,
//        Stripped: "brd-customer-XYZ-zone-residential-country-us-city-newyork"}
package credtags

import (
	"strconv"
	"strings"
)

// Tags is the result of parsing a proxy-auth username. Only the three
// ProxyMetrics-reserved fields are surfaced; all other username segments
// are preserved in Stripped for the upstream to interpret.
type Tags struct {
	Provider        string
	Type            string
	PriceCentsPerGB int

	// Stripped is the username with provider/type/price pairs removed,
	// safe to forward to the upstream.
	Stripped string
	// Raw is the original input verbatim.
	Raw string
}

var reservedKeys = map[string]struct{}{
	"provider": {},
	"type":     {},
	"price":    {},
}

// Parse walks the dash-delimited username, lifting reserved key/value pairs
// into Tags fields and returning an upstream-safe Stripped username with
// those pairs removed. Unknown tokens — including all other key/value pairs
// the upstream provider may use (customer, zone, country, city, session, …)
// — are preserved verbatim. The parser is deliberately ignorant about
// anything beyond the three reserved keys.
func Parse(user string) Tags {
	t := Tags{Raw: user}
	if user == "" {
		return t
	}

	parts := strings.Split(user, "-")
	keep := make([]string, 0, len(parts))

	i := 0
	for i < len(parts) {
		k := strings.ToLower(parts[i])
		if _, reserved := reservedKeys[k]; reserved && i+1 < len(parts) {
			v := parts[i+1]
			switch k {
			case "provider":
				t.Provider = strings.ToLower(v)
			case "type":
				t.Type = strings.ToLower(v)
			case "price":
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					t.PriceCentsPerGB = n
				}
			}
			i += 2
			continue
		}
		keep = append(keep, parts[i])
		i++
	}

	t.Stripped = strings.Join(keep, "-")
	return t
}
