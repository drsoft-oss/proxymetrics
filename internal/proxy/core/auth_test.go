package core_test

import (
	"net/http"
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/proxy/core"
)

func TestParseProxyAuth_Valid(t *testing.T) {
	r, _ := http.NewRequest("CONNECT", "https://example.com", nil)
	// base64("payload:secret")
	r.Header.Set("Proxy-Authorization", "Basic cGF5bG9hZDpzZWNyZXQ=")
	user, pass, ok := core.ParseProxyAuth(r)
	if !ok || user != "payload" || pass != "secret" {
		t.Fatalf("got %q/%q ok=%v", user, pass, ok)
	}
}

func TestParseProxyAuth_Missing(t *testing.T) {
	r, _ := http.NewRequest("CONNECT", "https://example.com", nil)
	if _, _, ok := core.ParseProxyAuth(r); ok {
		t.Fatal("expected ok=false on missing header")
	}
}

func TestParseProxyAuth_Malformed(t *testing.T) {
	r, _ := http.NewRequest("CONNECT", "https://example.com", nil)
	r.Header.Set("Proxy-Authorization", "Basic !!!notbase64!!!")
	if _, _, ok := core.ParseProxyAuth(r); ok {
		t.Fatal("expected ok=false on malformed base64")
	}
}

func TestCheckSecret(t *testing.T) {
	if !core.CheckSecret("s3cret", "s3cret") {
		t.Fatal("equal secrets must match")
	}
	if core.CheckSecret("s3cret", "wrong") {
		t.Fatal("unequal secrets must not match")
	}
}
