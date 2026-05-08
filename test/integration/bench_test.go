//go:build integration

package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/test/integration"
)

func BenchmarkMITMOverhead(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(bytes.Repeat([]byte("x"), 4096))
	}))
	defer upstream.Close()

	srv := integration.Start(b, upstream.URL)

	proxyURL := integration.ProxyURL(srv, upstream.URL, "fake", "residential", 100)
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(upstream.URL + "/")
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
