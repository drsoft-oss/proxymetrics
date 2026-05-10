package core

import (
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
