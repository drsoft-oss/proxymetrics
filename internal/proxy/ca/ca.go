// Package ca generates and manages the MITM root CA + per-host leaf certs.
package ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	keyFilename   = "proxymetrics-ca.key"
	pemFilename   = "proxymetrics-ca.pem"
	derFilename   = "proxymetrics-ca.der"
	archiveSubdir = "archive"
)

// CA holds an in-memory representation of the loaded root CA.
type CA struct {
	Cert    *x509.Certificate
	CertDER []byte
	Key     *ecdsa.PrivateKey
	KeyPath string
	PEMPath string
	DERPath string
}

func paths(dir string) (key, pemP, derP string) {
	return filepath.Join(dir, keyFilename),
		filepath.Join(dir, pemFilename),
		filepath.Join(dir, derFilename)
}

// Exists returns true if all three CA files are present in dir.
func Exists(dir string) bool {
	keyP, pemP, derP := paths(dir)
	for _, p := range []string{keyP, pemP, derP} {
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	return true
}

// Generate creates a new P-256 ECDSA root CA in dir and returns the loaded CA.
// Errors if files already exist.
func Generate(dir string) (*CA, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	keyP, pemP, derP := paths(dir)
	for _, p := range []string{keyP, pemP, derP} {
		if _, err := os.Stat(p); err == nil {
			return nil, fmt.Errorf("ca: file already exists: %s", p)
		}
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ecdsa keygen: %w", err)
	}
	idBytes := make([]byte, 4)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	instanceID := hex.EncodeToString(idBytes)

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "ProxyMetrics MITM CA " + instanceID},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create cert: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyP, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(pemP, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(derP, der, 0o644); err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &CA{Cert: cert, CertDER: der, Key: key, KeyPath: keyP, PEMPath: pemP, DERPath: derP}, nil
}

// Load reads an existing CA from dir.
func Load(dir string) (*CA, error) {
	keyP, pemP, derP := paths(dir)
	keyBytes, err := os.ReadFile(keyP)
	if err != nil {
		return nil, err
	}
	keyBlock, _ := pem.Decode(keyBytes)
	if keyBlock == nil {
		return nil, fmt.Errorf("ca: %s is not PEM", keyP)
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("ca: parse private key: %w", err)
	}
	pemBytes, err := os.ReadFile(pemP)
	if err != nil {
		return nil, err
	}
	certBlock, _ := pem.Decode(pemBytes)
	if certBlock == nil {
		return nil, fmt.Errorf("ca: %s is not PEM", pemP)
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("ca: parse cert: %w", err)
	}
	return &CA{Cert: cert, CertDER: certBlock.Bytes, Key: key, KeyPath: keyP, PEMPath: pemP, DERPath: derP}, nil
}

// Rotate archives the existing CA and generates a fresh one.
func Rotate(dir string) (*CA, error) {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	archDir := filepath.Join(dir, archiveSubdir, ts)
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		return nil, err
	}
	keyP, pemP, derP := paths(dir)
	for _, p := range []string{keyP, pemP, derP} {
		if _, err := os.Stat(p); err == nil {
			if err := os.Rename(p, filepath.Join(archDir, filepath.Base(p))); err != nil {
				return nil, fmt.Errorf("ca: archive %s: %w", p, err)
			}
		}
	}
	return Generate(dir)
}

// Fingerprint returns the SHA-256 fingerprint of the cert as upper-case hex with colons.
func (c *CA) Fingerprint() string {
	sum := sha256.Sum256(c.CertDER)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":")
}
