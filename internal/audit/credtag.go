// Package audit orchestrates audit runs (currently the Geo audit).
//
// The package's surface is small:
//   - Manager: starts/cancels/lists runs, owns the in-memory *Run map.
//   - Run: per-run state machine + worker pool; speaks to internal/geo for
//     verdict computation and writes per-request rows to its store.
//   - Verdict: pure (expected, observed) → match flags.
//   - Credtag: parser for the proxy URL credtag format (shared shape with
//     Profiles).
package audit

import (
	"errors"
	"net/url"
	"strings"
)

// Credtag is the parsed view of a credtag-encoded proxy URL.
//
// The underlying wire format is `http://user-KEY1-VAL1-KEY2-VAL2-...:pass@host:port/`
// with the username carrying a flat key/value sequence delimited by `-`.
// All keys past the leading `user-` are optional; only `country` is checked
// at audit-submit time.
type Credtag struct {
	Provider string
	Type     string
	Country  string
	State    string
	City     string
	Password string
	Host     string
	Port     string
}

// HasLevel reports whether the parsed credtag claims the given audit level.
// Pass one of "country", "state", "city".
func (c Credtag) HasLevel(level string) bool {
	switch level {
	case "country":
		return c.Country != ""
	case "state":
		return c.State != ""
	case "city":
		return c.City != ""
	}
	return false
}

// ParseCredtag parses the provided URL string. It accepts the decoded form
// (`http://user-...:pass@host:port`); the base64-wrapped wire form is decoded
// upstream by the proxy core and is not the input here.
func ParseCredtag(raw string) (Credtag, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Credtag{}, err
	}
	if u.User == nil {
		return Credtag{}, errors.New("credtag: url has no userinfo")
	}
	username := u.User.Username()
	if username == "" {
		return Credtag{}, errors.New("credtag: empty username")
	}
	password, _ := u.User.Password()

	tokens := strings.Split(username, "-")
	if len(tokens) < 3 || tokens[0] != "user" {
		return Credtag{}, errors.New("credtag: username does not start with 'user-' or has no key/value pairs")
	}
	pairs := tokens[1:]
	if len(pairs)%2 != 0 {
		return Credtag{}, errors.New("credtag: odd number of key/value tokens")
	}

	out := Credtag{
		Password: password,
		Host:     u.Hostname(),
		Port:     u.Port(),
	}
	saw := 0
	for i := 0; i < len(pairs); i += 2 {
		key, val := pairs[i], pairs[i+1]
		if val == "" {
			return Credtag{}, errors.New("credtag: empty value for key " + key)
		}
		switch key {
		case "provider":
			out.Provider = val
			saw++
		case "type":
			out.Type = val
			saw++
		case "country":
			out.Country = strings.ToUpper(val)
			saw++
		case "state":
			out.State = val
			saw++
		case "city":
			out.City = val
			saw++
		}
	}
	if saw == 0 {
		return Credtag{}, errors.New("credtag: no recognised tags in username")
	}
	return out, nil
}
