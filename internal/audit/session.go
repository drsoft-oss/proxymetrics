package audit

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// RotateSession returns rawURL with the username's <sessionKey>-<value> pair
// rewritten to <sessionKey>-<newValue>. Token match is case-sensitive and
// matches the first occurrence. Returns an error if the URL is unparseable,
// the username is empty, the key is not present, or the key is the last
// token (no following value to replace).
//
// The password (if any) is preserved verbatim — the rebuilt URL goes through
// url.UserPassword which round-trips the original encoded password.
//
// Designed for the audit hot path: caller calls once per request with a
// fresh newValue from NewSessionValue.
func RotateSession(rawURL, sessionKey, newValue string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if u.User == nil || u.User.Username() == "" {
		return "", errors.New("rotate session: URL has no username to rotate")
	}
	username := u.User.Username()
	password, hasPwd := u.User.Password()

	tokens := strings.Split(username, "-")
	idx := -1
	for i, tok := range tokens {
		if tok == sessionKey {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("rotate session: key %q not found in URL username", sessionKey)
	}
	if idx == len(tokens)-1 {
		return "", fmt.Errorf("rotate session: key %q has no following value", sessionKey)
	}
	tokens[idx+1] = newValue
	newUsername := strings.Join(tokens, "-")
	if hasPwd {
		u.User = url.UserPassword(newUsername, password)
	} else {
		u.User = url.User(newUsername)
	}
	return u.String(), nil
}

// NewSessionValue returns 16 lowercase hex chars from crypto/rand.
// Goroutine-safe. Collision probability across 1000 samples is ~2^-50.
// No dashes — would shift the dash-delimited username parser if injected.
func NewSessionValue() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
