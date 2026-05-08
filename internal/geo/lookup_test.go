package geo_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/geo"
)

type fakeLookup struct {
	res geo.Result
	err error
}

func (f fakeLookup) Lookup(_ context.Context, _ *http.Client) (geo.Result, error) {
	return f.res, f.err
}

func TestCascade_PrimaryWins(t *testing.T) {
	c := geo.Cascade{
		fakeLookup{res: geo.Result{Source: "primary", IP: "1.1.1.1"}},
		fakeLookup{res: geo.Result{Source: "secondary"}},
	}
	got, err := c.Lookup(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Source != "primary" || got.IP != "1.1.1.1" {
		t.Fatalf("got %+v", got)
	}
}

func TestCascade_FallsBackOnError(t *testing.T) {
	c := geo.Cascade{
		fakeLookup{err: errors.New("boom")},
		fakeLookup{res: geo.Result{Source: "secondary", IP: "2.2.2.2"}},
	}
	got, err := c.Lookup(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Source != "secondary" || got.IP != "2.2.2.2" {
		t.Fatalf("got %+v", got)
	}
}

func TestCascade_AllFail(t *testing.T) {
	c := geo.Cascade{
		fakeLookup{err: errors.New("a")},
		fakeLookup{err: errors.New("b")},
	}
	_, err := c.Lookup(context.Background(), http.DefaultClient)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCascade_Empty(t *testing.T) {
	var c geo.Cascade
	_, err := c.Lookup(context.Background(), http.DefaultClient)
	if err == nil {
		t.Fatal("expected error on empty cascade")
	}
}
