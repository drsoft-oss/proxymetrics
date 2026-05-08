package audit

import "strings"

// providerHostSuffixes maps a registrable-domain suffix (case-insensitive) to
// the canonical provider name we surface in audit history.
//
// Add new vendors here in a single place; the static map is exposed verbatim
// to the UI via /api/v1/audits/providers so prefill stays in sync without a
// duplicated TS table.
var providerHostSuffixes = map[string]string{
	"superproxy.io":     "brightdata",
	"lum-superproxy.io": "brightdata",
	"databay.co":        "databay",
	"oxylabs.io":        "oxylabs",
	"smartproxy.com":    "smartproxy",
	"soax.com":          "soax",
	"iproyal.com":       "iproyal",
}

// GuessProviderFromHost returns the provider name for hosts matching a known
// domain suffix, else "". Match is case-insensitive on label boundaries:
// "brd.superproxy.io" matches the "superproxy.io" suffix.
func GuessProviderFromHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}
	for {
		if v, ok := providerHostSuffixes[host]; ok {
			return v
		}
		i := strings.IndexByte(host, '.')
		if i < 0 {
			return ""
		}
		host = host[i+1:]
	}
}

// StaticProviders returns the unique set of provider names known to the
// static host map, sorted by the underlying map iteration (caller should
// dedupe/sort).
func StaticProviders() []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(providerHostSuffixes))
	for _, v := range providerHostSuffixes {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// ProviderHostSuffixes returns a copy of the static host-suffix map for
// callers that need to ship it to the UI. Mutations on the returned map
// do not affect future calls.
func ProviderHostSuffixes() map[string]string {
	out := make(map[string]string, len(providerHostSuffixes))
	for k, v := range providerHostSuffixes {
		out[k] = v
	}
	return out
}
