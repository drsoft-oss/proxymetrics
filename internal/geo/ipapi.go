package geo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// IPAPIIs queries https://api.ipapi.is/ via whatever http.Client is provided.
// When the client's transport routes through a proxy, the API auto-detects the
// proxy's exit IP and returns it alongside geo + connection-type fields.
type IPAPIIs struct {
	baseURL string
}

// NewIPAPIIs constructs an IPAPIIs lookup with the given base URL (e.g. the
// production "https://api.ipapi.is/", or an httptest server URL in tests).
func NewIPAPIIs(baseURL string) *IPAPIIs {
	return &IPAPIIs{baseURL: baseURL}
}

type ipapiResp struct {
	IP           string `json:"ip"`
	IsDatacenter bool   `json:"is_datacenter"`
	IsMobile     bool   `json:"is_mobile"`
	IsProxy      bool   `json:"is_proxy"`
	IsVPN        bool   `json:"is_vpn"`
	IsTor        bool   `json:"is_tor"`
	ASN          struct {
		ASN int `json:"asn"`
	} `json:"asn"`
	Company struct {
		Name string `json:"name"`
	} `json:"company"`
	Location struct {
		CountryCode string  `json:"country_code"`
		State       string  `json:"state"`
		City        string  `json:"city"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
	} `json:"location"`
}

// Lookup implements Lookup.
func (l *IPAPIIs) Lookup(ctx context.Context, client *http.Client) (Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.baseURL, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return Result{}, fmt.Errorf("ipapi: http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Result{}, err
	}
	var r ipapiResp
	if err := json.Unmarshal(body, &r); err != nil {
		return Result{}, fmt.Errorf("ipapi: decode: %w", err)
	}
	if r.IP == "" {
		return Result{}, errors.New("ipapi: missing ip in response")
	}
	return Result{
		IP:           r.IP,
		Country:      strings.ToUpper(r.Location.CountryCode),
		State:        r.Location.State,
		City:         r.Location.City,
		Lat:          r.Location.Latitude,
		Lon:          r.Location.Longitude,
		IsDatacenter: r.IsDatacenter,
		IsMobile:     r.IsMobile,
		IsProxy:      r.IsProxy,
		IsVPN:        r.IsVPN,
		IsTor:        r.IsTor,
		ASN:          r.ASN.ASN,
		Company:      r.Company.Name,
		Source:       "ipapi",
	}, nil
}
