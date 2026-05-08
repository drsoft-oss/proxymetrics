package ui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drsoft-oss/proxymetrics/internal/ui"
)

func TestHandler_ServesIndexAtRoot(t *testing.T) {
	rr := httptest.NewRecorder()
	ui.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("ct: %q", ct)
	}
	body, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(body), "ProxyMetrics") {
		t.Fatalf("body did not contain expected text: %.200q", body)
	}
}

func TestHandler_FallsBackToIndexForUnknownPaths(t *testing.T) {
	rr := httptest.NewRecorder()
	ui.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/providers", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	body, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(body), "ProxyMetrics") {
		t.Fatalf("SPA fallback did not return index.html")
	}
}
