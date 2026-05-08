package core

import (
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"strings"

	"github.com/zeebo/xxh3"
)

// ErrTimeout is the sentinel used by hot-path callers to indicate the request
// failed due to a deadline rather than a connection failure.
var ErrTimeout = errors.New("core: timeout")

// NormalizeHost lowercases the input and strips scheme + port, returning bare host.
// Unparseable inputs are returned with best-effort cleanup.
func NormalizeHost(in string) string {
	if in == "" {
		return ""
	}
	s := strings.ToLower(in)
	if u, err := url.Parse(s); err == nil && u.Host != "" {
		s = u.Host
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		s = h
	}
	return strings.Trim(s, "[]")
}

// PathHash returns a 16-hex-char xxh3 hash of the URL path (including the query).
func PathHash(path string) string {
	if path == "" {
		return ""
	}
	sum := xxh3.HashString(path)
	var b [8]byte
	for i := 0; i < 8; i++ {
		b[7-i] = byte(sum >> (8 * i))
	}
	return hex.EncodeToString(b[:])
}

// StatusClass derives the canonical bucket name from (status_code, errClassifier).
// errClassifier is one of ErrTimeout, nil, or any other non-nil error → "connection_error".
func StatusClass(code int, errClass error) string {
	if errors.Is(errClass, ErrTimeout) {
		return "timeout"
	}
	if code == 0 {
		return "connection_error"
	}
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	}
	return "connection_error"
}
