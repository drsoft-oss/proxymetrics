//go:build integration

package integration

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/cli"
	"github.com/drsoft-oss/proxymetrics/internal/store"
	"github.com/drsoft-oss/proxymetrics/internal/store/duckdb"
)

// Server is the running proxymetrics instance under test.
type Server struct {
	DataDir   string
	ProxyAddr string
	APIAddr   string
	DBPath    string
}

// FakeUpstream is the dummy origin server we point profiles at.
type FakeUpstream struct {
	*httptest.Server
}

func StartFakeUpstream(t testing.TB) *FakeUpstream {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "hello-from-upstream")
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return &FakeUpstream{s}
}

// Start brings up the proxymetrics serve goroutine for this test. Tests build
// their proxy-auth username with ProxyURL(srv, upstreamURL, provider, type,
// priceCents) — there is no per-server upstream configuration; the upstream
// URL is encoded in each request's username.
func Start(t testing.TB, upstreamURL string) *Server {
	return StartWithIPCheck(t, upstreamURL, []string{"http://127.0.0.1:1/dummy"})
}

// StartWithIPCheck is Start but with a custom ipcheck.apis list — used by the
// profile-test integration test.
func StartWithIPCheck(t testing.TB, _ string, ipcheckURLs []string) *Server {
	t.Helper()
	dataDir := t.TempDir()
	return launchInDataDir(t, dataDir, ipcheckURLs)
}

// StartWithSeededProfile is StartWithIPCheck plus a profile row pre-seeded
// directly into the DuckDB file before the server starts. The registry will
// load the seeded row at startup. Used by tests (like the /test endpoint test)
// that need a known profile id with a specific upstream URL — auto-discovery
// from observed traffic alone won't produce the right UpstreamURL for the
// tested code path.
func StartWithSeededProfile(t testing.TB, ipcheckURLs []string, p store.Profile) *Server {
	t.Helper()
	dataDir := t.TempDir()
	seedProfile(t, dataDir, p)
	return launchInDataDir(t, dataDir, ipcheckURLs)
}

func seedProfile(t testing.TB, dataDir string, p store.Profile) {
	t.Helper()
	s, err := duckdb.Open(filepath.Join(dataDir, "events.duckdb"))
	if err != nil {
		t.Fatalf("seed: open duckdb: %v", err)
	}
	if err := s.CreateProfile(context.Background(), p); err != nil {
		s.Close()
		t.Fatalf("seed: create profile: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("seed: close duckdb: %v", err)
	}
}

func launchInDataDir(t testing.TB, dataDir string, ipcheckURLs []string) *Server {
	t.Helper()
	proxyPort := freePort(t)
	apiPort := freePort(t)

	apisYAML := ""
	for _, u := range ipcheckURLs {
		apisYAML += fmt.Sprintf("    - %q\n", u)
	}

	cfgPath := filepath.Join(dataDir, "config.yaml")
	body := fmt.Sprintf(`
server:
  proxy_listen: ":%d"
  api_listen:   ":%d"
  deployment_secret: "test-secret"
storage:
  data_dir: "%s"
events:
  channel_size: 1000
  batch_size: 10
  flush_interval: "100ms"
shutdown:
  drain_timeout: "5s"
defaults:
  team: "default"
  project: "default"
ipcheck:
  apis:
%s  rotation: "round_robin"
logging:
  level: "warn"
  format: "text"
`, proxyPort, apiPort, dataDir, apisYAML)
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	root := cli.Root()
	root.SetArgs([]string{"serve", "-c", cfgPath})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	go func() {
		_ = root.Execute()
	}()

	waitTCP(t, fmt.Sprintf("127.0.0.1:%d", apiPort), 5*time.Second)
	waitTCP(t, fmt.Sprintf("127.0.0.1:%d", proxyPort), 5*time.Second)

	return &Server{
		DataDir:   dataDir,
		ProxyAddr: fmt.Sprintf("127.0.0.1:%d", proxyPort),
		APIAddr:   fmt.Sprintf("127.0.0.1:%d", apiPort),
		DBPath:    filepath.Join(dataDir, "events.duckdb"),
	}
}

// ProxyURL builds the HTTP_PROXY URL clients use to reach proxymetrics. The
// proxy-auth username is the base64-encoded full upstream URL with provider/
// type/price tags injected into its userinfo segment — the same wire format
// scrapers use in production.
func ProxyURL(srv *Server, upstreamURL, provider, typ string, priceCents int) *neturl.URL {
	upstream, _ := neturl.Parse(upstreamURL)
	upstreamUser := fmt.Sprintf("customer-test-provider-%s-type-%s-price-%d", provider, typ, priceCents)
	upstream.User = neturl.UserPassword(upstreamUser, "upstream-pass")
	encoded := base64.RawURLEncoding.EncodeToString([]byte(upstream.String()))
	u, _ := neturl.Parse(fmt.Sprintf("http://%s:test-secret@%s", encoded, srv.ProxyAddr))
	return u
}

func freePort(t testing.TB) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	_, port, _ := net.SplitHostPort(l.Addr().String())
	n, _ := strconv.Atoi(port)
	return n
}

func waitTCP(t testing.TB, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			c.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("listener did not come up: %s", addr)
}

// CertPool returns an x509.CertPool trusting the running server's CA.
func CertPool(t testing.TB, srv *Server) *x509.CertPool {
	t.Helper()
	resp, err := http.Get("http://" + srv.APIAddr + "/cacert")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	block, _ := pem.Decode(body)
	if block == nil {
		t.Fatal("no PEM block in /cacert")
	}
	pool := x509.NewCertPool()
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	pool.AddCert(cert)
	return pool
}

// EventsAfter polls the DuckDB until at least one event is visible, up to timeout.
func EventsAfter(t testing.TB, srv *Server, timeout time.Duration) []store.Event {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		s, err := duckdb.Open(srv.DBPath)
		if err == nil {
			rows, _ := s.TailEvents(context.Background(), store.TailOptions{N: 100})
			s.Close()
			if len(rows) > 0 {
				return rows
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no events written before %v", timeout)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
