package core

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

// ParseProxyAuth extracts the username and password from a Basic Proxy-Authorization
// header. Returns ok=false on missing or malformed.
func ParseProxyAuth(r *http.Request) (user, pass string, ok bool) {
	h := r.Header.Get("Proxy-Authorization")
	if h == "" {
		return "", "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Basic") {
		return "", "", false
	}
	dec, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", false
	}
	colon := strings.IndexByte(string(dec), ':')
	if colon < 0 {
		return "", "", false
	}
	return string(dec[:colon]), string(dec[colon+1:]), true
}

// CheckSecret performs a constant-time comparison.
func CheckSecret(provided, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
