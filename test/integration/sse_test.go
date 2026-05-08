//go:build integration

package integration_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/test/integration"
)

func TestE2E_SSEStream_DeliversLiveEvents(t *testing.T) {
	upstream := integration.StartFakeUpstream(t)
	srv := integration.Start(t, upstream.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://"+srv.APIAddr+"/api/v1/events/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	streamCh := make(chan string, 10)
	go func() {
		buf := make([]byte, 4096)
		acc := ""
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				acc += string(buf[:n])
				if strings.Contains(acc, "data:") {
					select {
					case streamCh <- acc:
					default:
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()

	proxyURL := integration.ProxyURL(srv, upstream.URL, "fake", "residential", 100)
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	go func() {
		time.Sleep(100 * time.Millisecond)
		r, err := client.Get(upstream.URL + "/")
		if err == nil {
			io.ReadAll(r.Body)
			r.Body.Close()
		}
	}()

	select {
	case got := <-streamCh:
		if !strings.Contains(got, "data:") {
			t.Fatalf("expected SSE event, got: %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
}
