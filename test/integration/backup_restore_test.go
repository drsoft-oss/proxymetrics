//go:build integration

package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/store"
	"github.com/anonymous-proxies/proxymetrics/internal/store/duckdb"
)

func TestE2E_BackupRestore_RoundTrip(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "proxymetrics")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/proxymetrics")
	cmd.Dir = projectRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	dataDir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
server:
  proxy_listen: ":0"
  api_listen:   ":0"
  deployment_secret: "x"
storage:
  data_dir: "`+dataDir+`"
events:
  channel_size: 100
  batch_size: 10
  flush_interval: "1s"
shutdown:
  drain_timeout: "5s"
defaults:
  team: "default"
  project: "default"
ipcheck:
  apis: ["http://127.0.0.1:1/dummy"]
  rotation: "round_robin"
logging:
  level: "warn"
  format: "text"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(dataDir, "events.duckdb")
	s, err := duckdb.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.CreateProfile(context.Background(), store.Profile{
		ID: "p1", Label: "L", Vendor: "custom", Type: "residential",
		UpstreamURL: "u", Currency: "USD",
	})
	_ = s.WriteEvents(context.Background(), []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential",
			StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
	})
	s.Close()

	tar := filepath.Join(t.TempDir(), "snap.tgz")
	out, err := exec.Command(bin, "db", "backup", "--out", tar, "-c", cfgPath).CombinedOutput()
	if err != nil {
		t.Fatalf("backup: %v\n%s", err, out)
	}
	if fi, err := os.Stat(tar); err != nil || fi.Size() == 0 {
		t.Fatalf("tar: %v / size=%d", err, fi.Size())
	}

	if err := os.RemoveAll(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err = exec.Command(bin, "db", "restore", "--in", tar, "--yes", "-c", cfgPath).CombinedOutput()
	if err != nil {
		t.Fatalf("restore: %v\n%s", err, out)
	}

	s2, err := duckdb.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	all, _ := s2.ListProfiles(context.Background())
	if len(all) == 0 {
		t.Fatal("profiles after restore: 0")
	}
	var found bool
	for _, p := range all {
		if p.ID == "p1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("p1 not found after restore; profiles: %v", all)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	return filepath.Dir(filepath.Dir(wd))
}
