//go:build integration

package integration_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/test/integration"
)

// TestMissingProvider_FailsButRecords verifies that a request whose proxy-auth
// username carries no `provider-` tag (and therefore no resolvable upstream)
// fails cleanly rather than crashing the proxy. The hot path emits an event
// with ProfileID="unknown" and a non-2xx status class so the dashboard can
// surface the misconfiguration.
func TestMissingProvider_FailsButRecords(t *testing.T) {
	upstream := integration.StartFakeUpstream(t)
	srv := integration.Start(t, upstream.URL)

	// Username has no provider/type/price — just a plain id. Parser will
	// pass it through verbatim and upstream lookup will return false.
	proxyURL, _ := url.Parse(fmt.Sprintf("http://no-tags-here:test-secret@%s", srv.ProxyAddr))
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	// Request is expected to fail (no upstream); we don't care about the
	// error, just that the proxy doesn't crash and emits an event.
	resp, err := client.Get(upstream.URL + "/")
	if err == nil {
		resp.Body.Close()
	}

	rows := integration.EventsAfter(t, srv, 5*time.Second)
	for _, e := range rows {
		if e.ProfileID == "unknown" && e.StatusClass != "2xx" {
			return
		}
	}
	t.Fatalf("expected at least one event with ProfileID=unknown and non-2xx, got: %+v", rows)
}
