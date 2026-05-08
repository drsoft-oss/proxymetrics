package ipcheck_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/ipcheck"
)

func newServer(t *testing.T, body, ct string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.Write([]byte(body))
	}))
}

func TestClient_PlainText(t *testing.T) {
	srv := newServer(t, "203.0.113.42\n", "text/plain")
	defer srv.Close()

	c := ipcheck.New([]string{srv.URL}, http.DefaultClient)
	res, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.IP != "203.0.113.42" {
		t.Fatalf("ip: %q", res.IP)
	}
	if res.APIUsed != srv.URL {
		t.Fatalf("api: %q", res.APIUsed)
	}
}

func TestClient_IPifyJSON(t *testing.T) {
	srv := newServer(t, `{"ip":"198.51.100.7"}`, "application/json")
	defer srv.Close()

	c := ipcheck.New([]string{srv.URL + "?format=json"}, http.DefaultClient)
	res, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.IP != "198.51.100.7" {
		t.Fatalf("ip: %q", res.IP)
	}
}

func TestClient_IfconfigMeJSON(t *testing.T) {
	srv := newServer(t, `{"ip_addr":"192.0.2.55","host":"x"}`, "application/json")
	defer srv.Close()

	c := ipcheck.New([]string{srv.URL}, http.DefaultClient)
	res, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.IP != "192.0.2.55" {
		t.Fatalf("ip: %q", res.IP)
	}
}

func TestClient_Rotation(t *testing.T) {
	a := newServer(t, "1.1.1.1", "text/plain")
	defer a.Close()
	b := newServer(t, "2.2.2.2", "text/plain")
	defer b.Close()

	c := ipcheck.New([]string{a.URL, b.URL}, http.DefaultClient)
	first, _ := c.Check(context.Background())
	second, _ := c.Check(context.Background())
	if first.APIUsed == second.APIUsed {
		t.Fatal("rotation: both calls hit the same API")
	}
}

func TestClient_FirstFails_Failover(t *testing.T) {
	// Server that delays past context deadline.
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer slow.Close()
	good := newServer(t, "9.9.9.9", "text/plain")
	defer good.Close()

	c := ipcheck.New([]string{slow.URL, good.URL}, &http.Client{Timeout: 5 * time.Millisecond})
	res, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.IP != "9.9.9.9" {
		t.Fatalf("expected failover IP, got %q", res.IP)
	}
}
