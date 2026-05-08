//go:build integration

package integration_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/test/integration"
)

func TestHappyPath_HTTPRequest_RecordsEvent(t *testing.T) {
	upstream := integration.StartFakeUpstream(t)

	srv := integration.Start(t, upstream.URL)

	// Proxy username carries provider/type/price tags; the rest is opaque
	// upstream-native cred bytes that ProxyMetrics passes through verbatim.
	proxyURL := integration.ProxyURL(srv, upstream.URL, "fake", "residential", 100)
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	resp, err := client.Get(upstream.URL + "/")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "hello-from-upstream") {
		t.Fatalf("unexpected body: %s", body)
	}

	rows := integration.EventsAfter(t, srv, 5*time.Second)
	var found bool
	for _, e := range rows {
		if e.ProfileID == "fake-residential" && e.StatusClass == "2xx" && e.BytesIn > 0 {
			found = true
			if e.Vendor != "fake" || e.Type != "residential" {
				t.Fatalf("synth profile dims: %+v", e)
			}
			break
		}
	}
	if !found {
		t.Fatalf("no matching event: %+v", rows)
	}
}
