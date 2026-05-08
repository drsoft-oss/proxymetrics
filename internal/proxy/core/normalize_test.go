package core_test

import (
	"testing"

	"github.com/drsoft-oss/proxymetrics/internal/proxy/core"
)

func TestNormalizeHost(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Example.COM:443", "example.com"},
		{"http://Example.com/foo", "example.com"},
		{"https://EXAMPLE.com:8443/path", "example.com"},
		{"sub.api.foo.com:8080", "sub.api.foo.com"},
		{"127.0.0.1:9000", "127.0.0.1"},
		{"[::1]:8080", "::1"},
		{"", ""},
	}
	for _, c := range cases {
		if got := core.NormalizeHost(c.in); got != c.want {
			t.Errorf("NormalizeHost(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPathHash_Stable(t *testing.T) {
	a := core.PathHash("/foo/bar?q=1")
	b := core.PathHash("/foo/bar?q=1")
	c := core.PathHash("/foo/bar?q=2")
	if a == "" || a != b {
		t.Fatalf("PathHash not stable: %q vs %q", a, b)
	}
	if a == c {
		t.Fatalf("PathHash collision on different paths")
	}
}

func TestStatusClass(t *testing.T) {
	cases := []struct {
		code int
		want string
	}{
		{200, "2xx"}, {204, "2xx"},
		{301, "3xx"}, {304, "3xx"},
		{401, "4xx"}, {429, "4xx"},
		{500, "5xx"}, {503, "5xx"},
		{0, "connection_error"},
	}
	for _, c := range cases {
		if got := core.StatusClass(c.code, nil); got != c.want {
			t.Errorf("StatusClass(%d) = %q, want %q", c.code, got, c.want)
		}
	}
	if got := core.StatusClass(0, core.ErrTimeout); got != "timeout" {
		t.Errorf("timeout: got %q", got)
	}
}
