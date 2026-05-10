package core

import "bytes"

// captchaSignature pairs a vendor name with byte fingerprints whose presence
// in a response body indicates a captcha challenge of that vendor.
type captchaSignature struct {
	Kind         string
	Fingerprints [][]byte
}

// captchaSignatures is the closed enumeration spec'd in
// docs/superpowers/specs/2026-05-09-captcha-occurrence-metric-design.md.
// First-match-wins; order is the iteration order.
var captchaSignatures = []captchaSignature{
	{Kind: "recaptcha", Fingerprints: [][]byte{
		[]byte("g-recaptcha"),
		[]byte("recaptcha/api.js"),
		[]byte("www.google.com/recaptcha"),
	}},
	{Kind: "turnstile", Fingerprints: [][]byte{
		[]byte("cf-turnstile"),
		[]byte("challenges.cloudflare.com/turnstile"),
	}},
	{Kind: "hcaptcha", Fingerprints: [][]byte{
		[]byte("h-captcha"),
		[]byte("hcaptcha.com/captcha"),
		[]byte("js.hcaptcha.com"),
	}},
	{Kind: "datadome", Fingerprints: [][]byte{
		[]byte("datadome"),
		[]byte("geo.captcha-delivery.com"),
	}},
	{Kind: "arkose", Fingerprints: [][]byte{
		[]byte("client-api.arkoselabs"),
		[]byte("funcaptcha"),
	}},
}

// captchaLongestFingerprint sizes the tail buffer used by the streaming
// scanner so that signatures spanning a chunk boundary still match.
var captchaLongestFingerprint = func() int {
	n := 0
	for _, sig := range captchaSignatures {
		for _, f := range sig.Fingerprints {
			if len(f) > n {
				n = len(f)
			}
		}
	}
	return n
}()

// captchaMatchAll returns the first matching kind in body, or "" if none.
// The input is lowercased in place — callers must hand a buffer they no
// longer need.
func captchaMatchAll(body []byte) string {
	asciiLowerInPlace(body)
	for _, sig := range captchaSignatures {
		for _, f := range sig.Fingerprints {
			if bytes.Contains(body, f) {
				return sig.Kind
			}
		}
	}
	return ""
}

// asciiLowerInPlace lowercases ASCII A-Z bytes in place. Non-ASCII bytes are
// left untouched — every fingerprint is ASCII, and full Unicode case folding
// would cost more than the matcher saves.
func asciiLowerInPlace(b []byte) {
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
}
