package geo_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/drsoft-oss/proxymetrics/internal/geo"
	"github.com/drsoft-oss/proxymetrics/internal/ipcheck"
)

// fakeIPCheck always returns a fixed IP without making any network calls.
type fakeIPCheck struct{ ip string }

func (f fakeIPCheck) Check(_ context.Context) (ipcheck.Result, error) {
	return ipcheck.Result{IP: f.ip, APIUsed: "fake"}, nil
}

func TestMaxMind_Lookup_KnownIP(t *testing.T) {
	city := filepath.Join("testdata", "GeoIP2-City-Test.mmdb")
	isp := filepath.Join("testdata", "GeoIP2-ISP-Test.mmdb")
	l, err := geo.NewMaxMind(geo.MaxMindConfig{
		CityDB: city, ISPDB: isp, IPCheck: fakeIPCheck{ip: "81.2.69.142"},
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer l.Close()

	got, err := l.Lookup(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got.IP != "81.2.69.142" {
		t.Fatalf("ip: %q", got.IP)
	}
	if got.Country == "" {
		t.Fatalf("country empty: %+v", got)
	}
	if got.Source != "maxmind" {
		t.Fatalf("source: %q", got.Source)
	}
}

func TestMaxMind_Lookup_NoCityDBConfigured(t *testing.T) {
	_, err := geo.NewMaxMind(geo.MaxMindConfig{
		IPCheck: fakeIPCheck{ip: "1.1.1.1"},
	})
	if err == nil {
		t.Fatal("expected error when no city_db configured")
	}
}

func TestMaxMind_NewMaxMind_RejectsBogusISPDB(t *testing.T) {
	city := filepath.Join("testdata", "GeoIP2-City-Test.mmdb")
	bogus := filepath.Join("testdata", "README.md") // not an mmdb
	_, err := geo.NewMaxMind(geo.MaxMindConfig{
		CityDB: city, ISPDB: bogus, IPCheck: fakeIPCheck{ip: "1.1.1.1"},
	})
	if err == nil {
		t.Fatal("expected error opening invalid isp db")
	}
}
