// Package pricing computes cost_usd per event, tracking monthly cumulative
// bytes_in per profile in memory.
package pricing

import (
	"context"
	"sync"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// Sum is the boot-time aggregate dependency.
type Sum interface {
	SumBytesInThisMonthByProfile(ctx context.Context) (map[string]int64, error)
}

// Calculator owns the per-profile monthly counter map.
type Calculator struct {
	src Sum

	mu     sync.Mutex
	month  string // "2026-04"
	totals map[string]int64
}

func New(src Sum) *Calculator {
	return &Calculator{
		src:    src,
		totals: map[string]int64{},
	}
}

// Reload bootstraps the in-memory month totals from the store.
func (c *Calculator) Reload(ctx context.Context) error {
	m, err := c.src.SumBytesInThisMonthByProfile(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.month = monthKey(time.Now().UTC())
	c.totals = m
	return nil
}

// Compute returns cost_usd for an event with bytes_in (resolved at write time).
// It also advances the monthly counter for this profile.
func (c *Calculator) Compute(p store.Profile, bytesIn int64, ts time.Time) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	mk := monthKey(ts.UTC())
	if mk != c.month {
		c.month = mk
		c.totals = map[string]int64{}
	}

	cumulative := c.totals[p.ID]
	c.totals[p.ID] = cumulative + bytesIn

	if p.PricePerGB == nil {
		return 0
	}
	rate := *p.PricePerGB
	if p.IncludedGB != nil {
		includedBytes := int64(*p.IncludedGB * (1 << 30))
		if cumulative >= includedBytes {
			if p.PricePerGBOverage != nil {
				rate = *p.PricePerGBOverage
			}
		}
	}
	return float64(bytesIn) / float64(int64(1)<<30) * rate
}

func monthKey(t time.Time) string { return t.Format("2006-01") }
