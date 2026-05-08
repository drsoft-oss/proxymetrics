package profile

import "strings"

// SyntheticID is the deterministic profile ID derived from the credtags
// provider and type. The proxy hot path uses it to stamp events; the registry's
// auto-discovery upsert uses it to key the row. Both must agree.
func SyntheticID(provider, typ string) string {
	switch {
	case provider == "" && typ == "":
		return "unknown"
	case provider == "":
		return typ
	case typ == "":
		return provider
	default:
		return provider + "-" + typ
	}
}

// SyntheticLabel is the human-readable label used when a row is auto-discovered
// from observed traffic. The admin path can later overwrite it.
func SyntheticLabel(provider, typ string) string {
	switch {
	case provider == "" && typ == "":
		return "Unknown"
	case typ == "":
		return capFirst(provider)
	case provider == "":
		return capFirst(typ)
	default:
		return capFirst(provider) + " " + capFirst(typ)
	}
}

func capFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
