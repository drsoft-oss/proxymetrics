// Package ipcheck rotates among public IP-echo APIs to determine the exit IP
// returned by an upstream proxy. Used by the profile-test endpoint (and later the
// audit module's Layer 1).
package ipcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
)

// Result is what a single check returned.
type Result struct {
	IP      string
	APIUsed string
}

// Client rotates round-robin across a fixed list of IP-check endpoints.
// On a per-Check failure the client tries the next endpoint until one succeeds.
type Client struct {
	apis []string
	hc   *http.Client
	idx  atomic.Uint32
}

func New(apis []string, hc *http.Client) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{apis: apis, hc: hc}
}

// WithClient returns a copy of the client using the provided http.Client.
// Used by callers that want to route IP checks through a specific transport
// (the profile-test endpoint routes through the upstream proxy).
func (c *Client) WithClient(hc *http.Client) *Client {
	return &Client{apis: c.apis, hc: hc}
}

// Check runs one IP lookup and returns the parsed result + which API was used.
func (c *Client) Check(ctx context.Context) (Result, error) {
	if len(c.apis) == 0 {
		return Result{}, errors.New("ipcheck: no APIs configured")
	}
	start := int(c.idx.Add(1)-1) % len(c.apis)
	var lastErr error
	for i := 0; i < len(c.apis); i++ {
		api := c.apis[(start+i)%len(c.apis)]
		ip, err := c.fetch(ctx, api)
		if err == nil && ip != "" {
			return Result{IP: ip, APIUsed: api}, nil
		}
		lastErr = err
	}
	return Result{}, fmt.Errorf("ipcheck: all APIs failed: %w", lastErr)
}

func (c *Client) fetch(ctx context.Context, api string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return "", err
	}
	return parseIP(resp.Header.Get("Content-Type"), body), nil
}

func parseIP(contentType string, body []byte) string {
	s := strings.TrimSpace(string(body))
	if strings.Contains(contentType, "json") || (len(s) > 0 && s[0] == '{') {
		var generic map[string]any
		if err := json.Unmarshal(body, &generic); err == nil {
			for _, k := range []string{"ip", "ip_addr", "address", "query"} {
				if v, ok := generic[k].(string); ok && net.ParseIP(v) != nil {
					return v
				}
			}
		}
	}
	if net.ParseIP(s) != nil {
		return s
	}
	return ""
}
