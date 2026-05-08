package geo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/geo"
)

type fakeCache struct {
	stored map[string]geo.Centroid
}

func newFakeCache() *fakeCache { return &fakeCache{stored: map[string]geo.Centroid{}} }
func (f *fakeCache) GetCentroid(_ context.Context, country, state, city string) (geo.Centroid, bool, error) {
	c, ok := f.stored[country+"|"+state+"|"+city]
	return c, ok, nil
}
func (f *fakeCache) PutCentroid(_ context.Context, country, state, city string, lat, lon float64) error {
	f.stored[country+"|"+state+"|"+city] = geo.Centroid{Lat: lat, Lon: lon}
	return nil
}

func TestCentroids_NominatimSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "test-ua/1.0" {
			t.Errorf("user-agent: %q", got)
		}
		_, _ = w.Write([]byte(`[{"lat":"44.43","lon":"26.10"}]`))
	}))
	defer srv.Close()

	r := geo.NewCentroidResolver(geo.CentroidConfig{
		BaseURL:   srv.URL,
		UserAgent: "test-ua/1.0",
		Timeout:   2 * time.Second,
		Cache:     newFakeCache(),
	})
	got, err := r.Resolve(context.Background(), "RO", "Bucharest", "Bucharest")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Lat != 44.43 || got.Lon != 26.10 {
		t.Fatalf("coords: %+v", got)
	}
}

func TestCentroids_CachedHitSkipsNetwork(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(`[{"lat":"44.43","lon":"26.10"}]`))
	}))
	defer srv.Close()

	cache := newFakeCache()
	cache.stored["RO|Bucharest|Bucharest"] = geo.Centroid{Lat: 44.43, Lon: 26.10}

	r := geo.NewCentroidResolver(geo.CentroidConfig{
		BaseURL: srv.URL, UserAgent: "x", Timeout: time.Second, Cache: cache,
	})
	if _, err := r.Resolve(context.Background(), "RO", "Bucharest", "Bucharest"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected cache hit, got %d HTTP calls", calls)
	}
}

func TestCentroids_NominatimEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	r := geo.NewCentroidResolver(geo.CentroidConfig{
		BaseURL: srv.URL, UserAgent: "x", Timeout: time.Second, Cache: newFakeCache(),
	})
	_, err := r.Resolve(context.Background(), "ZZ", "Nowhere", "Nowhere")
	if err == nil {
		t.Fatal("expected error on empty Nominatim result")
	}
}

func TestCentroids_NominatimHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	r := geo.NewCentroidResolver(geo.CentroidConfig{
		BaseURL: srv.URL, UserAgent: "x", Timeout: time.Second, Cache: newFakeCache(),
	})
	_, err := r.Resolve(context.Background(), "RO", "", "Bucharest")
	if err == nil {
		t.Fatal("expected error on 503")
	}
}

func TestCentroids_NewCentroidResolver_PanicsOnNilCache(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when Cache is nil")
		}
	}()
	_ = geo.NewCentroidResolver(geo.CentroidConfig{
		BaseURL: "x", UserAgent: "x",
	})
}
