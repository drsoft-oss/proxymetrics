package profile_test

import (
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/profile"
)

func TestSyntheticID(t *testing.T) {
	cases := []struct {
		name           string
		provider, typ  string
		wantID         string
		wantLabel      string
	}{
		{"both", "brightdata", "residential", "brightdata-residential", "Brightdata Residential"},
		{"providerOnly", "brightdata", "", "brightdata", "Brightdata"},
		{"typeOnly", "", "residential", "residential", "Residential"},
		{"neither", "", "", "unknown", "Unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := profile.SyntheticID(c.provider, c.typ); got != c.wantID {
				t.Errorf("SyntheticID(%q,%q): got %q want %q", c.provider, c.typ, got, c.wantID)
			}
			if got := profile.SyntheticLabel(c.provider, c.typ); got != c.wantLabel {
				t.Errorf("SyntheticLabel(%q,%q): got %q want %q", c.provider, c.typ, got, c.wantLabel)
			}
		})
	}
}
