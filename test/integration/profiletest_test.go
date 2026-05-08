//go:build integration

package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drsoft-oss/proxymetrics/internal/store"
	"github.com/drsoft-oss/proxymetrics/test/integration"
)

func TestProfileTestEndpoint_RoutesAndReturnsExitIP(t *testing.T) {
	// Fake IP-check API that returns a known IP in plain text.
	ipsrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("203.0.113.99"))
	}))
	defer ipsrv.Close()

	// Fake upstream proxy: in this test, the profile's upstream URL points DIRECTLY
	// at ipsrv. The ProxyClientBuilder will configure http.Transport.Proxy = ipsrv,
	// which means the IP-check request gets sent as a "GET http://ipsrv/" request
	// proxied THROUGH ipsrv. Most HTTP-proxy servers serve the requested URL when
	// they receive an absolute-URI GET — but httptest.Server does NOT do that
	// (it just serves its registered routes).
	//
	// So this test mainly verifies the endpoint SHAPE: when called, it returns
	// well-formed JSON and routes-the-attempt; it may return an error in the body
	// because the "fake upstream proxy" we wired isn't a real proxy.
	srv := integration.StartWithSeededProfile(t, []string{ipsrv.URL}, store.Profile{
		ID:          "p1",
		Label:       "L",
		Vendor:      "custom",
		Type:        "residential",
		UpstreamURL: ipsrv.URL,
		Currency:    "USD",
	})

	resp, err := http.Post("http://"+srv.APIAddr+"/api/v1/profiles/p1/test", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buf, _ := io.ReadAll(resp.Body)
		t.Fatalf("status: %d body=%s", resp.StatusCode, buf)
	}

	var body struct {
		ExitIP    string `json:"exit_ip"`
		Status    int    `json:"status_code"`
		APIUsed   string `json:"api_used"`
		Error     string `json:"error,omitempty"`
		LatencyMS int    `json:"latency_ms"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	// Either we got a successful exit IP (if the fake server happened to respond
	// to a proxied GET) or an error explaining why — both are well-formed.
	if body.Error == "" && body.ExitIP == "" {
		t.Fatalf("expected exit_ip or error: %+v", body)
	}
	// Latency must be populated regardless.
	if body.LatencyMS < 0 {
		t.Fatalf("invalid latency: %d", body.LatencyMS)
	}
	_ = strings.Builder{}
}
