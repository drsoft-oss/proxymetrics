package credtags_test

import (
	"testing"

	"github.com/drsoft-oss/proxymetrics/internal/proxy/credtags"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want credtags.Tags
	}{
		{
			name: "brightdata typical — only provider/type/price stripped, everything else passes through",
			in:   "brd-customer-XYZ-zone-residential-country-us-city-newyork-provider-brightdata-type-residential-price-1200",
			want: credtags.Tags{
				Provider:        "brightdata",
				Type:            "residential",
				PriceCentsPerGB: 1200,
				Stripped:        "brd-customer-XYZ-zone-residential-country-us-city-newyork",
				Raw:             "brd-customer-XYZ-zone-residential-country-us-city-newyork-provider-brightdata-type-residential-price-1200",
			},
		},
		{
			name: "oxylabs-style with cc/session — passed through unchanged",
			in:   "customer-USER-cc-us-city-newyork-session-abc-provider-oxylabs-type-residential-price-900",
			want: credtags.Tags{
				Provider:        "oxylabs",
				Type:            "residential",
				PriceCentsPerGB: 900,
				Stripped:        "customer-USER-cc-us-city-newyork-session-abc",
				Raw:             "customer-USER-cc-us-city-newyork-session-abc-provider-oxylabs-type-residential-price-900",
			},
		},
		{
			name: "team and project are NOT reserved — they pass through verbatim",
			in:   "user-team-scrapers-project-search-provider-brightdata",
			want: credtags.Tags{
				Provider: "brightdata",
				Stripped: "user-team-scrapers-project-search",
				Raw:      "user-team-scrapers-project-search-provider-brightdata",
			},
		},
		{
			name: "empty input",
			in:   "",
			want: credtags.Tags{},
		},
		{
			name: "no tags, plain username preserved",
			in:   "simpleuser",
			want: credtags.Tags{
				Stripped: "simpleuser",
				Raw:      "simpleuser",
			},
		},
		{
			name: "only reserved tags strip to empty",
			in:   "provider-brightdata-price-1200-type-residential",
			want: credtags.Tags{
				Provider:        "brightdata",
				Type:            "residential",
				PriceCentsPerGB: 1200,
				Stripped:        "",
				Raw:             "provider-brightdata-price-1200-type-residential",
			},
		},
		{
			name: "non-numeric price ignored, key still consumed",
			in:   "provider-brightdata-price-abc",
			want: credtags.Tags{
				Provider:        "brightdata",
				PriceCentsPerGB: 0,
				Stripped:        "",
				Raw:             "provider-brightdata-price-abc",
			},
		},
		{
			name: "trailing reserved key with no value kept verbatim",
			in:   "user-provider",
			want: credtags.Tags{
				Stripped: "user-provider",
				Raw:      "user-provider",
			},
		},
		{
			name: "case-insensitive keys, lowercased provider/type",
			in:   "User-Provider-BrightData-Type-Residential-Country-US-City-NewYork",
			want: credtags.Tags{
				Provider: "brightdata",
				Type:     "residential",
				Stripped: "User-Country-US-City-NewYork",
				Raw:      "User-Provider-BrightData-Type-Residential-Country-US-City-NewYork",
			},
		},
		{
			name: "duplicate provider — last wins, both stripped",
			in:   "provider-brightdata-provider-oxylabs",
			want: credtags.Tags{
				Provider: "oxylabs",
				Stripped: "",
				Raw:      "provider-brightdata-provider-oxylabs",
			},
		},
		{
			name: "no reserved tags at all, unchanged stripped",
			in:   "customer-XYZ-zone-residential-country-us-city-newyork-session-abc",
			want: credtags.Tags{
				Stripped: "customer-XYZ-zone-residential-country-us-city-newyork-session-abc",
				Raw:      "customer-XYZ-zone-residential-country-us-city-newyork-session-abc",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := credtags.Parse(tc.in)
			if got != tc.want {
				t.Errorf("Parse(%q):\n got:  %+v\n want: %+v", tc.in, got, tc.want)
			}
		})
	}
}
