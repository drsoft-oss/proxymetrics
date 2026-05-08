package ca_test

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/proxy/ca"
)

func TestGenerateAndLoad(t *testing.T) {
	dir := t.TempDir()
	gen, err := ca.Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{gen.KeyPath, gen.PEMPath, gen.DERPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing file %s: %v", p, err)
		}
	}
	// Re-load via Load.
	loaded, err := ca.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Fingerprint() != gen.Fingerprint() {
		t.Fatalf("fingerprint mismatch: %q vs %q", loaded.Fingerprint(), gen.Fingerprint())
	}
}

func TestGenerate_Properties(t *testing.T) {
	dir := t.TempDir()
	a, err := ca.Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes, _ := os.ReadFile(a.PEMPath)
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		t.Fatal("not PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if !cert.IsCA {
		t.Fatal("cert is not a CA")
	}
	if !strings.Contains(cert.Subject.CommonName, "ProxyMetrics MITM CA") {
		t.Fatalf("bad CN: %q", cert.Subject.CommonName)
	}
	if cert.NotAfter.Sub(cert.NotBefore) < 9*365*24*time.Hour {
		t.Fatalf("validity too short: %v", cert.NotAfter.Sub(cert.NotBefore))
	}
}

func TestKeyPermissions(t *testing.T) {
	dir := t.TempDir()
	a, err := ca.Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(a.KeyPath)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("key mode = %o, want 0600", st.Mode().Perm())
	}
}

func TestRotateArchives(t *testing.T) {
	dir := t.TempDir()
	first, err := ca.Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := ca.Rotate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.Fingerprint() == first.Fingerprint() {
		t.Fatal("rotate must produce a different cert")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "archive", "*", "proxymetrics-ca.pem"))
	if len(matches) != 1 {
		t.Fatalf("archive: got %v", matches)
	}
}
