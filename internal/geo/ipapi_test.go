package geo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/geo"
)

func TestIPAPIIs_Lookup_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ip":"203.0.113.7",
			"is_datacenter":false,
			"is_mobile":false,
			"is_proxy":true,
			"is_vpn":false,
			"is_tor":false,
			"asn":{"asn":1234,"org":"Example ISP"},
			"company":{"name":"Example Telecom","type":"isp"},
			"location":{"country_code":"RO","state":"Bucharest","city":"Bucharest","latitude":44.43,"longitude":26.10}
		}`))
	}))
	defer srv.Close()

	l := geo.NewIPAPIIs(srv.URL)
	got, err := l.Lookup(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.IP != "203.0.113.7" {
		t.Fatalf("ip: %q", got.IP)
	}
	if got.Country != "RO" || got.City != "Bucharest" {
		t.Fatalf("loc: %+v", got)
	}
	if got.Lat != 44.43 || got.Lon != 26.10 {
		t.Fatalf("coords: %+v", got)
	}
	if got.IsDatacenter || !got.IsProxy {
		t.Fatalf("flags: %+v", got)
	}
	if got.ASN != 1234 || got.Company != "Example Telecom" {
		t.Fatalf("asn/company: %+v", got)
	}
	if got.Source != "ipapi" {
		t.Fatalf("source: %q", got.Source)
	}
}

func TestIPAPIIs_Lookup_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	l := geo.NewIPAPIIs(srv.URL)
	_, err := l.Lookup(context.Background(), http.DefaultClient)
	if err == nil {
		t.Fatal("expected error on 502")
	}
}

func TestIPAPIIs_Lookup_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	l := geo.NewIPAPIIs(srv.URL)
	_, err := l.Lookup(context.Background(), http.DefaultClient)
	if err == nil {
		t.Fatal("expected error on malformed body")
	}
}

func TestIPAPIIs_Lookup_MissingIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"location":{"country_code":"RO"}}`))
	}))
	defer srv.Close()

	l := geo.NewIPAPIIs(srv.URL)
	_, err := l.Lookup(context.Background(), http.DefaultClient)
	if err == nil {
		t.Fatal("expected error when ip is missing")
	}
}
