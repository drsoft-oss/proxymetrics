package pricing_test

import (
	"context"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/pricing"
	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

const oneGB = 1 << 30

func float64Ptr(v float64) *float64 { return &v }

func TestCost_NilPricePerGB_Zero(t *testing.T) {
	c := pricing.New(stubMonthSum{})
	got := c.Compute(store.Profile{ID: "p"}, oneGB, time.Now().UTC())
	if got != 0 {
		t.Fatalf("nil rate must yield zero cost, got %v", got)
	}
}

func TestCost_BaseRate_BelowIncluded(t *testing.T) {
	c := pricing.New(stubMonthSum{})
	p := store.Profile{ID: "p", PricePerGB: float64Ptr(2.5), IncludedGB: float64Ptr(10)}
	got := c.Compute(p, oneGB, time.Now().UTC())
	if got != 2.5 {
		t.Fatalf("got %v, want 2.5", got)
	}
}

func TestCost_OverageRate_AfterIncluded(t *testing.T) {
	c := pricing.New(stubMonthSum{"p": 11 * oneGB})
	if err := c.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	p := store.Profile{ID: "p", PricePerGB: float64Ptr(2.5), PricePerGBOverage: float64Ptr(5.0), IncludedGB: float64Ptr(10)}
	got := c.Compute(p, oneGB, time.Now().UTC())
	if got != 5.0 {
		t.Fatalf("got %v, want 5.0 (overage)", got)
	}
}

func TestCost_Overage_FallbackToBaseWhenNil(t *testing.T) {
	c := pricing.New(stubMonthSum{"p": 11 * oneGB})
	c.Reload(context.Background())
	p := store.Profile{ID: "p", PricePerGB: float64Ptr(2.5), IncludedGB: float64Ptr(10)} // no overage
	got := c.Compute(p, oneGB, time.Now().UTC())
	if got != 2.5 {
		t.Fatalf("overage-fallback to base failed: got %v", got)
	}
}

func TestCost_MonthRollover(t *testing.T) {
	c := pricing.New(stubMonthSum{})
	p := store.Profile{ID: "p", PricePerGB: float64Ptr(1.0), IncludedGB: float64Ptr(0)}

	// included_gb=0 means "no free tier" — first event already exceeds the included
	// allowance, so the second event hits overage; with overage nil, falls back to base.
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	c.Compute(p, oneGB, now)
	if got := c.Compute(p, oneGB, now); got != 1.0 {
		t.Fatalf("second-event overage-fallback: got %v", got)
	}

	// New month resets cumulative back to 0; first event of new month is base rate.
	next := now.AddDate(0, 1, 0)
	if got := c.Compute(p, oneGB, next); got != 1.0 {
		t.Fatalf("rollover behavior: got %v", got)
	}
}

type stubMonthSum map[string]int64

func (s stubMonthSum) SumBytesInThisMonthByProfile(ctx context.Context) (map[string]int64, error) {
	out := map[string]int64{}
	for k, v := range s {
		out[k] = v
	}
	return out, nil
}
