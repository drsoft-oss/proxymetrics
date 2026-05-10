package core_test

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/proxy/ca"
	"github.com/drsoft-oss/proxymetrics/internal/proxy/core"
	"github.com/drsoft-oss/proxymetrics/internal/store"
)

type captureEmitter struct {
	mu     sync.Mutex
	events []store.Event
}

func (e *captureEmitter) Submit(ev store.Event) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
	return true
}

func (e *captureEmitter) wait(t *testing.T, n int) []store.Event {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		e.mu.Lock()
		got := append([]store.Event(nil), e.events...)
		e.mu.Unlock()
		if len(got) >= n {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d events; got %d", n, len(e.events))
	return nil
}

type silentLogger struct{}

func (silentLogger) Debug(string, ...any) {}
func (silentLogger) Info(string, ...any)  {}
func (silentLogger) Warn(string, ...any)  {}
func (silentLogger) Error(string, ...any) {}

func TestCore_CaptureCaptchaKind_HTML(t *testing.T) {
	cases := []struct {
		name string
		body string
		ct   string
		want string
	}{
		{"recaptcha", `<div class="g-recaptcha"></div>`, "text/html; charset=utf-8", "recaptcha"},
		{"turnstile", `<div class="cf-turnstile"></div>`, "text/html", "turnstile"},
		{"hcaptcha", `<div class="h-captcha"></div>`, "text/html", "hcaptcha"},
		{"datadome", `<script src="https://geo.captcha-delivery.com/x"></script>`, "text/html", "datadome"},
		{"arkose", `<script src="https://client-api.arkoselabs.com/v2/x/api.js"></script>`, "text/html", "arkose"},
		{"none", `<html><body>hi</body></html>`, "text/html", ""},
		{"binary-skipped", `<div class="g-recaptcha"></div>`, "image/jpeg", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", c.ct)
				fmt.Fprint(w, c.body)
			}))
			t.Cleanup(upstream.Close)

			events := runCaptureProxy(t, upstream)
			if len(events) == 0 {
				t.Fatal("no events captured")
			}
			ev := events[0]
			if ev.CaptchaKind != c.want {
				t.Fatalf("got CaptchaKind=%q, want %q", ev.CaptchaKind, c.want)
			}
		})
	}
}

func TestCore_CaptureCaptchaKind_Gzip(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		gw := gzip.NewWriter(w)
		defer gw.Close()
		_, _ = gw.Write([]byte(`<div class="cf-turnstile"></div>`))
	}))
	t.Cleanup(upstream.Close)

	events := runCaptureProxy(t, upstream)
	if events[0].CaptchaKind != "turnstile" {
		t.Fatalf("got %q want turnstile", events[0].CaptchaKind)
	}
}

func runCaptureProxy(t *testing.T, upstream *httptest.Server) []store.Event {
	t.Helper()
	em := &captureEmitter{}

	root, err := ca.Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	leaf := ca.NewLeafFactory(root, 100, time.Hour)

	proxy, err := core.New(core.Config{
		DeploymentSecret: "secret",
		LeafFactory:      leaf,
		Emitter:          em,
		Cost:             func(store.Profile, int64, time.Time) float64 { return 0 },
		Logger:           silentLogger{},
	})
	if err != nil {
		t.Fatal(err)
	}
	proxySrv := httptest.NewServer(proxy)
	t.Cleanup(proxySrv.Close)

	// Build the proxy-auth username = base64( "http://<credtags-user>:p@<upstream-host>" ).
	// decodeUpstream in core.go expects the username to decode to a full proxy URL.
	// The credtags format for the username portion is "provider~type~~price".
	upstreamHost := strings.TrimPrefix(upstream.URL, "http://")
	upstreamForAuth := "http://provider~residential~~001:p@" + upstreamHost
	user := base64.RawURLEncoding.EncodeToString([]byte(upstreamForAuth))

	// Embed credentials in the proxy URL so Go's http.Transport sends them as
	// Proxy-Authorization (not Authorization). ParseProxyAuth reads Proxy-Authorization.
	proxyURL, _ := url.Parse(proxySrv.URL)
	proxyURL.User = url.UserPassword(user, "secret")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", upstream.URL+"/page", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	return em.wait(t, 1)
}
