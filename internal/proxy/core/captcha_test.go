package core

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"
)

func TestCaptchaMatch_PerVendor(t *testing.T) {
	cases := []struct {
		name string
		body string
		kind string
	}{
		{"recaptcha-class", `<div class="g-recaptcha" data-sitekey="x"></div>`, "recaptcha"},
		{"recaptcha-script", `<script src="https://www.google.com/recaptcha/api.js"></script>`, "recaptcha"},
		{"turnstile", `<div class="cf-turnstile" data-sitekey="y"></div>`, "turnstile"},
		{"turnstile-script", `<script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script>`, "turnstile"},
		{"hcaptcha", `<div class="h-captcha" data-sitekey="z"></div>`, "hcaptcha"},
		{"hcaptcha-script", `<script src="https://js.hcaptcha.com/1/api.js"></script>`, "hcaptcha"},
		{"datadome", `<script src="https://geo.captcha-delivery.com/captcha/?initialCid=foo"></script>`, "datadome"},
		{"arkose", `<script src="https://client-api.arkoselabs.com/v2/abcd/api.js"></script>`, "arkose"},
		{"arkose-funcaptcha", `<div id="funcaptcha"></div>`, "arkose"},
		{"none", `<html><body><h1>Hello</h1></body></html>`, ""},
		{"case-insensitive", `<DIV CLASS="G-RECAPTCHA"></DIV>`, "recaptcha"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := captchaMatchAll([]byte(c.body))
			if got != c.kind {
				t.Fatalf("got %q want %q", got, c.kind)
			}
		})
	}
}

func TestCaptchaMatch_NoFalsePositiveOnLargeText(t *testing.T) {
	body := make([]byte, 1<<20)
	for i := range body {
		body[i] = 'a'
	}
	if got := captchaMatchAll(body); got != "" {
		t.Fatalf("false positive: %q", got)
	}
}

func TestCaptchaScanner_BoundaryHit(t *testing.T) {
	body := `<html><body><div class="g-recaptcha"></div></body></html>`
	r, sc := wrapForCaptcha(slowReader{src: strings.NewReader(body), step: 1}, "text/html", "")
	if _, err := io.ReadAll(r); err != nil {
		t.Fatal(err)
	}
	if got := sc.Kind(); got != "recaptcha" {
		t.Fatalf("got %q want recaptcha", got)
	}
}

func TestCaptchaScanner_NoMatch(t *testing.T) {
	body := strings.Repeat("hello world ", 1000)
	r, sc := wrapForCaptcha(strings.NewReader(body), "text/html", "")
	if _, err := io.ReadAll(r); err != nil {
		t.Fatal(err)
	}
	if got := sc.Kind(); got != "" {
		t.Fatalf("false positive: %q", got)
	}
}

func TestCaptchaScanner_FirstHitWins(t *testing.T) {
	body := `<div class="g-recaptcha"></div><div class="cf-turnstile"></div>`
	r, sc := wrapForCaptcha(strings.NewReader(body), "text/html", "")
	if _, err := io.ReadAll(r); err != nil {
		t.Fatal(err)
	}
	if got := sc.Kind(); got != "recaptcha" {
		t.Fatalf("got %q want recaptcha", got)
	}
}

func TestCaptchaScanner_GzipBody(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte(`<div class="g-recaptcha"></div>`)); err != nil {
		t.Fatal(err)
	}
	_ = gw.Close()

	r, sc := wrapForCaptcha(bytes.NewReader(buf.Bytes()), "text/html", "gzip")
	if _, err := io.ReadAll(r); err != nil {
		t.Fatal(err)
	}
	if got := sc.Kind(); got != "recaptcha" {
		t.Fatalf("got %q want recaptcha", got)
	}
}

func TestCaptchaScanner_PassthroughBytesUnchanged(t *testing.T) {
	body := `<html><body><div class="g-recaptcha"></div></body></html>`
	r, _ := wrapForCaptcha(strings.NewReader(body), "text/html", "")
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("body mutated:\n got: %q\nwant: %q", got, body)
	}
}

// slowReader returns at most `step` bytes per Read; used to force chunk boundaries.
type slowReader struct {
	src  io.Reader
	step int
}

func (s slowReader) Read(p []byte) (int, error) {
	if s.step > 0 && len(p) > s.step {
		p = p[:s.step]
	}
	return s.src.Read(p)
}
