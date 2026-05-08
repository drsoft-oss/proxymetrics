package core_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/proxy/core"
)

func TestStatusPeek_BasicHTTP(t *testing.T) {
	resp := "HTTP/1.1 403 Forbidden\r\nContent-Length: 5\r\n\r\nhello"
	src := strings.NewReader(resp)
	gotCode := 0
	r := core.NewStatusPeekReader(src, func(code int) { gotCode = code })

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != resp {
		t.Fatalf("body roundtrip mismatch")
	}
	if gotCode != 403 {
		t.Fatalf("status: got %d, want 403", gotCode)
	}
}

func TestStatusPeek_HTTPSResponse_OK(t *testing.T) {
	resp := "HTTP/2 200 \r\nfoo: bar\r\n\r\n"
	r := core.NewStatusPeekReader(strings.NewReader(resp), func(code int) {
		if code != 200 {
			t.Fatalf("HTTP/2: got %d", code)
		}
	})
	io.Copy(io.Discard, r)
}

func TestStatusPeek_NoStatusLine(t *testing.T) {
	called := false
	r := core.NewStatusPeekReader(bytes.NewReader([]byte("not-http")), func(int) { called = true })
	io.Copy(io.Discard, r)
	if called {
		t.Fatal("callback fired on non-HTTP input")
	}
}
