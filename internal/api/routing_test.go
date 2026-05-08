package api_test

import (
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/api"
)

func TestPickSource(t *testing.T) {
	cases := []struct {
		dur  time.Duration
		want string
	}{
		{1 * time.Hour, "events"},
		{23 * time.Hour, "events"},
		{24 * time.Hour, "events"}, // boundary inclusive on lower side
		{25 * time.Hour, "rollups_1hour"},
		{29 * 24 * time.Hour, "rollups_1hour"},
		{30 * 24 * time.Hour, "rollups_1hour"}, // boundary inclusive
		{31 * 24 * time.Hour, "rollups_1day"},
		{365 * 24 * time.Hour, "rollups_1day"},
	}
	for _, c := range cases {
		got := api.PickSource(c.dur)
		if got != c.want {
			t.Errorf("PickSource(%v) = %q, want %q", c.dur, got, c.want)
		}
	}
}
