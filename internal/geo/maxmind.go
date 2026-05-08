package geo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"

	maxminddb "github.com/oschwald/maxminddb-golang/v2"

	"github.com/anonymous-proxies/proxymetrics/internal/ipcheck"
)

// IPDiscoverer abstracts the existing internal/ipcheck.Client.Check method so
// MaxMind tests can stub IP discovery without spinning up an HTTP server.
type IPDiscoverer interface {
	Check(ctx context.Context) (ipcheck.Result, error)
}

// MaxMindConfig configures a MaxMind lookup. CityDB is required; one of
// ConnectionTypeDB or ISPDB is needed for datacenter detection (preferring
// ConnectionTypeDB if both are set).
type MaxMindConfig struct {
	CityDB           string
	ConnectionTypeDB string
	ISPDB            string
	IPCheck          IPDiscoverer
}

// MaxMind is a fallback Lookup that resolves the exit IP via the configured
// IPDiscoverer (which routes through the caller's http.Client) and looks it up
// locally in MaxMind .mmdb files.
type MaxMind struct {
	city *maxminddb.Reader
	conn *maxminddb.Reader
	isp  *maxminddb.Reader
	ipc  IPDiscoverer
}

// NewMaxMind opens the configured .mmdb files. Returns an error if CityDB
// path is empty or unreadable.
func NewMaxMind(cfg MaxMindConfig) (*MaxMind, error) {
	if cfg.CityDB == "" {
		return nil, errors.New("maxmind: city_db is required")
	}
	if cfg.IPCheck == nil {
		return nil, errors.New("maxmind: ipcheck is required")
	}
	city, err := maxminddb.Open(cfg.CityDB)
	if err != nil {
		return nil, fmt.Errorf("maxmind: open city: %w", err)
	}
	m := &MaxMind{city: city, ipc: cfg.IPCheck}
	if cfg.ConnectionTypeDB != "" {
		conn, err := maxminddb.Open(cfg.ConnectionTypeDB)
		if err != nil {
			_ = city.Close()
			return nil, fmt.Errorf("maxmind: open connection_type: %w", err)
		}
		m.conn = conn
	}
	if cfg.ISPDB != "" {
		isp, err := maxminddb.Open(cfg.ISPDB)
		if err != nil {
			_ = city.Close()
			if m.conn != nil {
				_ = m.conn.Close()
			}
			return nil, fmt.Errorf("maxmind: open isp: %w", err)
		}
		m.isp = isp
	}
	return m, nil
}

// Close releases all opened readers.
func (m *MaxMind) Close() error {
	var firstErr error
	for _, r := range []*maxminddb.Reader{m.city, m.conn, m.isp} {
		if r == nil {
			continue
		}
		if err := r.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type cityRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
}

type connRecord struct {
	ConnectionType string `maxminddb:"connection_type"`
}

type ispRecord struct {
	ISP                 string `maxminddb:"isp"`
	Organization        string `maxminddb:"organization"`
	AutonomousSystemNum uint   `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrg string `maxminddb:"autonomous_system_organization"`
}

// Lookup implements Lookup.
func (m *MaxMind) Lookup(ctx context.Context, _ *http.Client) (Result, error) {
	// We honour the http.Client only via the IPDiscoverer; the .mmdb readers
	// are local-only.
	res, err := m.ipc.Check(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("maxmind: ipcheck: %w", err)
	}
	addr, err := netip.ParseAddr(res.IP)
	if err != nil {
		return Result{}, fmt.Errorf("maxmind: invalid ip %q: %w", res.IP, err)
	}

	out := Result{IP: res.IP, Source: "maxmind"}

	var cr cityRecord
	if err := m.city.Lookup(addr).Decode(&cr); err != nil {
		return Result{}, fmt.Errorf("maxmind: city decode: %w", err)
	}
	out.Country = strings.ToUpper(cr.Country.ISOCode)
	if len(cr.Subdivisions) > 0 {
		out.State = cr.Subdivisions[0].Names["en"]
	}
	out.City = cr.City.Names["en"]
	out.Lat = cr.Location.Latitude
	out.Lon = cr.Location.Longitude

	if m.conn != nil {
		// NOTE: MaxMind's Connection-Type DB only carries Cellular / Cable/DSL /
		// Corporate / Dialup — it does NOT distinguish hosting/datacenter. The
		// checks below are defensive (in case a future MaxMind DB carries Hosting/
		// Datacenter values, or a custom DB does), but on the production
		// Connection-Type DB IsDatacenter will essentially always be false here.
		// ipapi.is is the authoritative source for IsDatacenter; MaxMind fallback
		// is best-effort for geo + IsMobile only.
		var conn connRecord
		if err := m.conn.Lookup(addr).Decode(&conn); err == nil {
			out.IsDatacenter = strings.EqualFold(conn.ConnectionType, "Hosting") ||
				strings.EqualFold(conn.ConnectionType, "Datacenter")
			out.IsMobile = strings.EqualFold(conn.ConnectionType, "Cellular")
		}
	}
	if m.isp != nil {
		var isp ispRecord
		if err := m.isp.Lookup(addr).Decode(&isp); err == nil {
			out.ASN = int(isp.AutonomousSystemNum)
			if isp.ISP != "" {
				out.Company = isp.ISP
			} else if isp.Organization != "" {
				out.Company = isp.Organization
			} else {
				out.Company = isp.AutonomousSystemOrg
			}
		}
	}
	return out, nil
}
