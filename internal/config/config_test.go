package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/config"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func loadFromString(t *testing.T, body string) (config.Config, error) {
	t.Helper()
	return config.Load(writeTemp(t, body))
}

func TestLoad_HappyPath_FillsDefaults(t *testing.T) {
	t.Setenv("PROXYMETRICS_SECRET", "s3cret")
	p := writeTemp(t, `
server:
  proxy_listen: ":8080"
  api_listen:   ":8081"
  deployment_secret: "${PROXYMETRICS_SECRET}"
storage:
  data_dir: "./data"
`)
	c, err := config.Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Server.DeploymentSecret != "s3cret" {
		t.Fatalf("env subst: got %q", c.Server.DeploymentSecret)
	}
	if c.Events.ChannelSize != 10000 {
		t.Fatalf("default channel_size: got %d", c.Events.ChannelSize)
	}
	if c.Events.BatchSize != 500 {
		t.Fatalf("default batch_size: got %d", c.Events.BatchSize)
	}
	if c.Events.FlushInterval != time.Second {
		t.Fatalf("default flush_interval: got %v", c.Events.FlushInterval)
	}
	if c.Shutdown.DrainTimeout != 30*time.Second {
		t.Fatalf("default drain_timeout: got %v", c.Shutdown.DrainTimeout)
	}
	if c.Defaults.Team != "default" || c.Defaults.Project != "default" {
		t.Fatalf("defaults: got %q/%q", c.Defaults.Team, c.Defaults.Project)
	}
	if len(c.IPCheck.APIs) != 3 {
		t.Fatalf("default ipcheck apis: got %v", c.IPCheck.APIs)
	}
}

func TestLoad_RejectsMissingRequired(t *testing.T) {
	cases := []struct {
		name, body, wantContain string
	}{
		{"no proxy_listen", `server: {api_listen: ":8081", deployment_secret: x}` + "\nstorage: {data_dir: ./data}\n", "server.proxy_listen"},
		{"no api_listen", `server: {proxy_listen: ":8080", deployment_secret: x}` + "\nstorage: {data_dir: ./data}\n", "server.api_listen"},
		{"no secret", `server: {proxy_listen: ":8080", api_listen: ":8081"}` + "\nstorage: {data_dir: ./data}\n", "server.deployment_secret"},
		{"no data_dir", `server: {proxy_listen: ":8080", api_listen: ":8081", deployment_secret: x}` + "\n", "storage.data_dir"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := writeTemp(t, tc.body)
			_, err := config.Load(p)
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantContain) {
				t.Fatalf("error mismatch: got %q, want substring %q", err, tc.wantContain)
			}
		})
	}
}

func TestLoad_EnvSubstMissing(t *testing.T) {
	// Establish a clean baseline that the test framework will restore on cleanup,
	// regardless of run order. t.Setenv records the prior value (set or unset)
	// and restores it via t.Cleanup when the test finishes.
	t.Setenv("PROXYMETRICS_SECRET", "placeholder")
	os.Unsetenv("PROXYMETRICS_SECRET")

	p := writeTemp(t, `
server:
  proxy_listen: ":8080"
  api_listen:   ":8081"
  deployment_secret: "${PROXYMETRICS_SECRET}"
storage:
  data_dir: "./data"
`)
	_, err := config.Load(p)
	if err == nil || !strings.Contains(err.Error(), "PROXYMETRICS_SECRET") {
		t.Fatalf("want env-missing error naming the var, got: %v", err)
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	p := writeTemp(t, "::: not yaml :::")
	_, err := config.Load(p)
	if err == nil {
		t.Fatal("want yaml parse error")
	}
}

func TestLoad_RejectsUnknownIPCheckRotation(t *testing.T) {
	t.Setenv("PROXYMETRICS_SECRET", "s")
	p := writeTemp(t, `
server: {proxy_listen: ":8080", api_listen: ":8081", deployment_secret: "${PROXYMETRICS_SECRET}"}
storage: {data_dir: "./data"}
ipcheck:
  rotation: "weighted_smartly"
`)
	_, err := config.Load(p)
	if err == nil || !strings.Contains(err.Error(), "rotation") {
		t.Fatalf("want rotation error, got: %v", err)
	}
}

func TestDefaults_GeoAuditNominatim(t *testing.T) {
	c, err := loadFromString(t, `
server:
  proxy_listen: ":18080"
  api_listen: ":18081"
  deployment_secret: x
storage:
  data_dir: /tmp/x
`)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Geo.IPAPI.BaseURL != "https://api.ipapi.is/" {
		t.Fatalf("default ipapi base_url: got %q", c.Geo.IPAPI.BaseURL)
	}
	if c.Geo.Nominatim.BaseURL != "https://nominatim.openstreetmap.org/search" {
		t.Fatalf("default nominatim base_url: got %q", c.Geo.Nominatim.BaseURL)
	}
	if c.Geo.Nominatim.UserAgent != "proxymetrics-audit/1.0" {
		t.Fatalf("default nominatim user_agent: got %q", c.Geo.Nominatim.UserAgent)
	}
	if c.Geo.Nominatim.Timeout != 10*time.Second {
		t.Fatalf("default nominatim timeout: got %v", c.Geo.Nominatim.Timeout)
	}
	if c.Audit.PerRequestTimeout != 30*time.Second {
		t.Fatalf("default audit per_request_timeout: got %v", c.Audit.PerRequestTimeout)
	}
	if c.Audit.MaxInFlight != 5 {
		t.Fatalf("default audit max_in_flight: got %d", c.Audit.MaxInFlight)
	}
	if c.Audit.MinRequestsPerRun != 100 {
		t.Fatalf("default min_requests_per_run: got %d", c.Audit.MinRequestsPerRun)
	}
	if c.Audit.MaxRequestsPerRun != 1000 {
		t.Fatalf("default max_requests_per_run: got %d", c.Audit.MaxRequestsPerRun)
	}
}
