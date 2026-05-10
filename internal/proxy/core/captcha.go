package core

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"strings"
)

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

// captchaPanicHook is a hook used only by tests to force a panic inside the
// matcher. Production builds leave it nil and the call is elided.
var captchaPanicHook func()

// captchaScannableTypes is the closed list of Content-Type prefixes we scan.
// Anything else gets a no-op scanner — pass-through with no overhead.
var captchaScannableTypes = []string{
	"text/html",
	"application/xhtml+xml",
	"text/plain",
}

const captchaMaxEncBuf = 256 * 1024 // ~enough for any captcha challenge HTML

// captchaScanner reads from an underlying io.Reader, mirrors each chunk into a
// rolling matcher, and surfaces the first detected captcha kind via Kind().
//
// Bytes returned by Read are byte-for-byte the bytes returned by the underlying
// reader. The scanner never mutates, buffers, or delays the body — failure to
// detect must never affect the response stream.
type captchaScanner struct {
	src    io.Reader
	decode func(io.Reader) (io.Reader, error)
	kind   string // "" until first match
	done   bool   // true once kind is set OR we decided not to scan
	tail   []byte // last (longestFingerprint-1) bytes seen, to span chunks
	encBuf []byte // accumulated encoded bytes pending a single-shot decode
}

// wrapForCaptcha constructs a scanner appropriate for `contentType`. If we
// shouldn't scan this response, the returned reader is `r` unchanged and the
// scanner's Kind() always returns "".
func wrapForCaptcha(r io.Reader, contentType, contentEncoding string) (io.Reader, *captchaScanner) {
	if !shouldScanContentType(contentType) {
		return r, &captchaScanner{done: true}
	}
	sc := &captchaScanner{
		src:    r,
		decode: decoderFor(contentEncoding),
	}
	return sc, sc
}

// decoderFor returns a constructor for a streaming decoder appropriate for the
// given Content-Encoding, or nil if the content should be scanned raw.
// brotli is intentionally not supported in this phase — `br` bodies fall to
// the raw-bytes scan path (a documented small accuracy loss to avoid adding a
// new module dependency).
func decoderFor(ce string) func(io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(ce)) {
	case "gzip":
		return func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) }
	case "deflate":
		return func(r io.Reader) (io.Reader, error) { return flate.NewReader(r), nil }
	}
	return nil
}

// shouldScanContentType reports whether ct's prefix (before any ';') matches
// one of captchaScannableTypes. Comparison is case-insensitive.
func shouldScanContentType(ct string) bool {
	if ct == "" {
		return false
	}
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	ct = strings.ToLower(strings.TrimSpace(ct))
	for _, t := range captchaScannableTypes {
		if strings.HasPrefix(ct, t) {
			return true
		}
	}
	return false
}

// Kind returns the matched captcha kind ("" if none yet / never).
func (s *captchaScanner) Kind() string { return s.kind }

// Read forwards to the underlying reader, then mirrors the chunk into the
// matcher. Bytes are not mutated.
func (s *captchaScanner) Read(p []byte) (int, error) {
	n, err := s.src.Read(p)
	if n > 0 && !s.done {
		s.consume(p[:n])
	}
	if err == io.EOF && !s.done && s.decode != nil && len(s.encBuf) > 0 {
		s.flushDecoded()
	}
	return n, err
}

func (s *captchaScanner) consume(chunk []byte) {
	defer func() {
		if r := recover(); r != nil {
			s.done = true
		}
	}()
	if s.decode != nil {
		// Buffer for a single end-of-stream decode; bail to raw if the cap is hit.
		if len(s.encBuf)+len(chunk) > captchaMaxEncBuf {
			s.flushDecoded()
			return
		}
		s.encBuf = append(s.encBuf, chunk...)
		return
	}
	s.scanRaw(chunk)
}

// flushDecoded opens the decoder ONCE over everything we've buffered so far,
// scans the resulting plaintext, and disengages the scanner. Called on
// upstream EOF or when the encoded buffer hits its cap. On any decoder
// error, we fall back to raw scanning of the buffer so we still get a
// best-effort match (and never break the body — the proxy passthrough is
// independent of this code path).
func (s *captchaScanner) flushDecoded() {
	defer func() {
		recover()
		s.encBuf = nil
		s.done = true
	}()
	if s.decode == nil || len(s.encBuf) == 0 {
		return
	}
	dec, err := s.decode(bytes.NewReader(s.encBuf))
	if err != nil {
		s.scanRaw(s.encBuf)
		return
	}
	plain, err := io.ReadAll(dec)
	if err != nil && len(plain) == 0 {
		s.scanRaw(s.encBuf)
		return
	}
	s.scanRaw(plain)
}

// scanRaw runs the matcher over the chunk plus the carried-over tail buffer.
// On a hit, it disengages further scanning. Otherwise it refreshes the tail
// buffer to the last (longestFingerprint-1) bytes so a fingerprint that
// straddles a chunk boundary still matches on the next call.
func (s *captchaScanner) scanRaw(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	span := chunk
	if len(s.tail) > 0 {
		span = append(append([]byte(nil), s.tail...), chunk...)
	}
	if captchaPanicHook != nil {
		captchaPanicHook()
	}
	if k := captchaMatchAll(append([]byte(nil), span...)); k != "" {
		s.kind = k
		s.done = true
		s.tail = nil
		return
	}
	keep := captchaLongestFingerprint - 1
	if keep < 0 {
		keep = 0
	}
	if len(span) > keep {
		s.tail = append(s.tail[:0], span[len(span)-keep:]...)
	} else {
		s.tail = append(s.tail[:0], span...)
	}
}
