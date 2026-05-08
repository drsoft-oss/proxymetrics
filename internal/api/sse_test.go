package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/api"
	"github.com/anonymous-proxies/proxymetrics/internal/broadcaster"
	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

func TestSSE_DeliversPublishedEvents(t *testing.T) {
	b := broadcaster.New()
	srv := httptest.NewServer(api.SSEHandler(b))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("ct: %q", resp.Header.Get("Content-Type"))
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		b.Publish(store.Event{RequestID: "abc", ProfileID: "p1", StatusClass: "2xx"})
	}()

	buf := make([]byte, 4096)
	deadline := time.Now().Add(1500 * time.Millisecond)
	var got string
	for time.Now().Before(deadline) && !strings.Contains(got, `"abc"`) {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			got += string(buf[:n])
		}
		if err != nil {
			break
		}
	}
	if !strings.Contains(got, `"request_id":"abc"`) {
		t.Fatalf("missing event in stream: %q", got)
	}
}

func TestSSE_HeadersAreSet(t *testing.T) {
	b := broadcaster.New()
	srv := httptest.NewServer(api.SSEHandler(b))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	for hdr, want := range map[string]string{
		"Content-Type":  "text/event-stream",
		"Cache-Control": "no-cache",
		"Connection":    "keep-alive",
	} {
		if got := resp.Header.Get(hdr); got != want {
			t.Errorf("%s: got %q, want %q", hdr, got, want)
		}
	}
}
