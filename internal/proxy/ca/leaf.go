package ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// LeafFactory mints per-host leaf certs signed by the root CA, cached in an LRU.
type LeafFactory struct {
	root  *CA
	cache *lru.Cache[string, leafEntry]
	ttl   time.Duration
	mu    sync.Mutex // protects miss-time minting
}

type leafEntry struct {
	cert     tls.Certificate
	expireAt time.Time
}

// NewLeafFactory returns a factory caching up to capacity entries; entries expire
// after ttl regardless of LRU order. ttl is enforced on Certificate read.
func NewLeafFactory(root *CA, capacity int, ttl time.Duration) *LeafFactory {
	c, _ := lru.New[string, leafEntry](capacity)
	return &LeafFactory{root: root, cache: c, ttl: ttl}
}

// Purge clears the leaf cache (used after CA rotation).
func (f *LeafFactory) Purge() { f.cache.Purge() }

// Certificate returns a leaf TLS cert for host (DNS name or IP literal).
func (f *LeafFactory) Certificate(host string) (*tls.Certificate, error) {
	if e, ok := f.cache.Get(host); ok {
		if time.Now().Before(e.expireAt) {
			c := e.cert
			return &c, nil
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	// Re-check under the lock.
	if e, ok := f.cache.Get(host); ok && time.Now().Before(e.expireAt) {
		c := e.cert
		return &c, nil
	}

	tc, err := f.mint(host)
	if err != nil {
		return nil, err
	}
	f.cache.Add(host, leafEntry{cert: *tc, expireAt: time.Now().Add(f.ttl)})
	return tc, nil
}

func (f *LeafFactory) mint(host string) (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(host); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, f.root.Cert, &key.PublicKey, f.root.Key)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{
		Certificate: [][]byte{der, f.root.CertDER},
		PrivateKey:  key,
		Leaf:        nil,
	}, nil
}
