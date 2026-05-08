package audit

import "testing"

func TestGuessProviderFromHost(t *testing.T) {
	cases := []struct {
		name string
		host string
		want string
	}{
		{"exact root domain", "databay.co", "databay"},
		{"single subdomain", "brd.superproxy.io", "brightdata"},
		{"deep subdomain", "gw.eu.smartproxy.com", "smartproxy"},
		{"alternate vendor domain", "zproxy.lum-superproxy.io", "brightdata"},
		{"unknown host", "example.com", ""},
		{"empty host", "", ""},
		{"mixed case host", "BRD.SuperProxy.IO", "brightdata"},
		{"oxylabs", "pr.oxylabs.io", "oxylabs"},
		{"soax", "proxy.soax.com", "soax"},
		{"iproyal", "geo.iproyal.com", "iproyal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GuessProviderFromHost(tc.host); got != tc.want {
				t.Fatalf("GuessProviderFromHost(%q) = %q, want %q", tc.host, got, tc.want)
			}
		})
	}
}

func TestStaticProviders_AllValuesPresent(t *testing.T) {
	got := StaticProviders()
	want := map[string]bool{
		"brightdata": true, "databay": true, "oxylabs": true,
		"smartproxy": true, "soax": true, "iproyal": true,
	}
	for _, p := range got {
		delete(want, p)
	}
	if len(want) != 0 {
		t.Fatalf("StaticProviders missing: %v", want)
	}
}

func TestProviderHostSuffixes_IsACopy(t *testing.T) {
	a := ProviderHostSuffixes()
	a["mutated.example"] = "x"
	b := ProviderHostSuffixes()
	if _, ok := b["mutated.example"]; ok {
		t.Fatal("ProviderHostSuffixes returned a shared map; should be a copy")
	}
}
