// Package tlsca manages WatchTrail's local certificate authority: a
// self-signed CA plus a server leaf for watchtrail.local, stored under
// <DataDir>/tls. TLS is "on" exactly when the leaf cert and key both exist.
package tlsca

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	leafLifetime = 390 * 24 * time.Hour // under the ~398-day browser cap
	renewBefore  = 30 * 24 * time.Hour
)

// Dir is the directory holding all TLS materials.
func Dir(dataDir string) string { return filepath.Join(dataDir, "tls") }

func CACertPath(dataDir string) string   { return filepath.Join(Dir(dataDir), "ca.crt") }
func CAKeyPath(dataDir string) string    { return filepath.Join(Dir(dataDir), "ca.key") }
func LeafCertPath(dataDir string) string { return filepath.Join(Dir(dataDir), "leaf.crt") }
func LeafKeyPath(dataDir string) string  { return filepath.Join(Dir(dataDir), "leaf.key") }

// Enabled reports whether both leaf files exist (TLS is configured).
func Enabled(dataDir string) bool {
	if _, err := os.Stat(LeafCertPath(dataDir)); err != nil {
		return false
	}
	if _, err := os.Stat(LeafKeyPath(dataDir)); err != nil {
		return false
	}
	return true
}

// Enable ensures a CA exists (creating it on first call, caCreated=true) and
// (re)mints the leaf for hosts. Returns the CA cert path for trust install.
func Enable(dataDir string, hosts []string, now time.Time) (caCertPath string, caCreated bool, err error) {
	if err := os.MkdirAll(Dir(dataDir), 0o700); err != nil {
		return "", false, fmt.Errorf("tls dir: %w", err)
	}
	caCreated, err = ensureCA(dataDir, now)
	if err != nil {
		return "", false, err
	}
	if err := mintLeaf(dataDir, hosts, now); err != nil {
		return "", false, err
	}
	return CACertPath(dataDir), caCreated, nil
}

// EnsureLeafFresh re-mints the leaf from the existing CA if it is within the
// renewal window. No-op when the leaf is still comfortably valid.
func EnsureLeafFresh(dataDir string, hosts []string, now time.Time) error {
	leaf, err := loadCert(LeafCertPath(dataDir))
	if err != nil {
		return err
	}
	if !needsRenewal(leaf.NotAfter, now) {
		return nil
	}
	return mintLeaf(dataDir, hosts, now)
}

// Disable turns TLS off by removing the leaf; the CA is kept so re-enabling
// needs no re-trust.
func Disable(dataDir string) error {
	for _, p := range []string{LeafCertPath(dataDir), LeafKeyPath(dataDir)} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// CACertBytes returns the CA cert PEM, for serving at GET /ca.crt.
func CACertBytes(dataDir string) ([]byte, error) {
	return os.ReadFile(CACertPath(dataDir))
}

// LANHosts returns the SAN list: fixed loopback names plus this host's
// non-loopback IPv4 addresses.
func LANHosts() []string {
	hosts := []string{"watchtrail.local", "localhost", "127.0.0.1", "::1"}
	ifaces, err := net.Interfaces()
	if err != nil {
		return hosts
	}
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if v4 := ipnet.IP.To4(); v4 != nil && ipnet.IP.IsGlobalUnicast() {
				hosts = append(hosts, v4.String())
			}
		}
	}
	return hosts
}

func needsRenewal(notAfter, now time.Time) bool {
	return notAfter.Sub(now) < renewBefore
}

func genKeyAndSerial() (*ecdsa.PrivateKey, *big.Int, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := randSerial()
	if err != nil {
		return nil, nil, err
	}
	return key, serial, nil
}

func ensureCA(dataDir string, now time.Time) (created bool, err error) {
	if _, statErr := os.Stat(CACertPath(dataDir)); statErr == nil {
		return false, nil
	}
	key, serial, err := genKeyAndSerial()
	if err != nil {
		return false, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "WatchTrail Local CA", Organization: []string{"WatchTrail"}},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return false, err
	}
	if err := writePEM(CACertPath(dataDir), "CERTIFICATE", der, 0o644); err != nil {
		return false, err
	}
	if err := writeKey(CAKeyPath(dataDir), key); err != nil {
		return false, err
	}
	return true, nil
}

func mintLeaf(dataDir string, hosts []string, now time.Time) error {
	caCert, caKey, err := loadCA(dataDir)
	if err != nil {
		return err
	}
	key, serial, err := genKeyAndSerial()
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "watchtrail.local"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(leafLifetime),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return err
	}
	if err := writePEM(LeafCertPath(dataDir), "CERTIFICATE", der, 0o644); err != nil {
		return err
	}
	return writeKey(LeafKeyPath(dataDir), key)
}

func loadCA(dataDir string) (*x509.Certificate, crypto.Signer, error) {
	cert, err := loadCert(CACertPath(dataDir))
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(CAKeyPath(dataDir))
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("ca.key: no PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, nil, fmt.Errorf("ca.key: not a signer")
	}
	return cert, signer, nil
}

func loadCert(path string) (*x509.Certificate, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("%s: no PEM block", path)
	}
	return x509.ParseCertificate(block.Bytes)
}

func writePEM(path, blockType string, der []byte, mode os.FileMode) error {
	out := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	return os.WriteFile(path, out, mode)
}

func writeKey(path string, key *ecdsa.PrivateKey) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	return writePEM(path, "PRIVATE KEY", der, 0o600)
}

func randSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limit)
}
