package ca_test

import (
	"crypto/x509"
	"net"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/proxy/ca"
)

func setupCA(t *testing.T) *ca.CA {
	t.Helper()
	a, err := ca.Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestLeafFactory_DNS(t *testing.T) {
	root := setupCA(t)
	f := ca.NewLeafFactory(root, 100, 24*time.Hour)
	tc, err := f.Certificate("example.com")
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(tc.Certificate[0])
	if cert.Subject.CommonName != "example.com" {
		t.Fatalf("CN: %q", cert.Subject.CommonName)
	}
	found := false
	for _, n := range cert.DNSNames {
		if n == "example.com" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing DNS SAN: %v", cert.DNSNames)
	}
}

func TestLeafFactory_IP(t *testing.T) {
	root := setupCA(t)
	f := ca.NewLeafFactory(root, 100, 24*time.Hour)
	tc, err := f.Certificate("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(tc.Certificate[0])
	found := false
	for _, ip := range cert.IPAddresses {
		if ip.Equal(net.ParseIP("127.0.0.1")) {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing IP SAN: %v", cert.IPAddresses)
	}
}

func TestLeafFactory_CacheHit(t *testing.T) {
	root := setupCA(t)
	f := ca.NewLeafFactory(root, 100, 24*time.Hour)
	tc1, _ := f.Certificate("a.example")
	tc2, _ := f.Certificate("a.example")
	// Same backing leaf cert means identical Certificate[0] bytes.
	if string(tc1.Certificate[0]) != string(tc2.Certificate[0]) {
		t.Fatal("expected cache hit to return identical cert")
	}
}

func TestLeafFactory_TTLEviction(t *testing.T) {
	root := setupCA(t)
	f := ca.NewLeafFactory(root, 100, 1*time.Millisecond)
	tc1, _ := f.Certificate("b.example")
	time.Sleep(5 * time.Millisecond)
	tc2, _ := f.Certificate("b.example")
	if string(tc1.Certificate[0]) == string(tc2.Certificate[0]) {
		t.Fatal("expected TTL expiry to mint a new cert")
	}
}
