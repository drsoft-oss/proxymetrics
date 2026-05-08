package ca_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/proxy/ca"
)

func mux(t *testing.T) (*ca.CA, http.Handler) {
	t.Helper()
	root := setupCA(t)
	return root, ca.HTTPHandler(root)
}

func TestCacert_PEM(t *testing.T) {
	_, h := mux(t)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/cacert", nil))
	if rr.Code != 200 {
		t.Fatalf("status: %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/x-pem-file" {
		t.Fatalf("ct: %q", got)
	}
	body, _ := io.ReadAll(rr.Body)
	if !strings.HasPrefix(string(body), "-----BEGIN CERTIFICATE-----") {
		t.Fatalf("not PEM: %.40q", body)
	}
}

func TestCacert_DER(t *testing.T) {
	_, h := mux(t)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/cacert.crt", nil))
	if rr.Code != 200 {
		t.Fatalf("status: %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/x-x509-ca-cert" {
		t.Fatalf("ct: %q", got)
	}
}

func TestCacert_Fingerprint(t *testing.T) {
	root, h := mux(t)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/cacert/fingerprint", nil))
	if rr.Code != 200 {
		t.Fatalf("status: %d", rr.Code)
	}
	var body struct {
		SHA256     string `json:"sha256"`
		ValidUntil string `json:"valid_until"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.SHA256 != root.Fingerprint() {
		t.Fatalf("fingerprint mismatch")
	}
	if body.ValidUntil == "" {
		t.Fatalf("valid_until missing")
	}
}
