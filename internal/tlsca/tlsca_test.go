package tlsca

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"
)

func TestEnableCreatesVerifiableLeaf(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	caPath, created, err := Enable(dir, []string{"watchtrail.local", "127.0.0.1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected CA created on first Enable")
	}
	if caPath != CACertPath(dir) {
		t.Fatalf("caPath = %q, want %q", caPath, CACertPath(dir))
	}
	if !Enabled(dir) {
		t.Fatal("Enabled should be true after Enable")
	}

	caPEM, err := os.ReadFile(CACertPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("could not append CA")
	}
	leaf := parseLeaf(t, dir)
	if _, err := leaf.Verify(x509.VerifyOptions{
		DNSName:     "watchtrail.local",
		Roots:       pool,
		CurrentTime: now,
	}); err != nil {
		t.Fatalf("leaf does not verify against CA: %v", err)
	}
}

func TestEnableSecondCallReusesCA(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	if _, created, err := Enable(dir, []string{"watchtrail.local"}, now); err != nil || !created {
		t.Fatalf("first Enable: created=%v err=%v", created, err)
	}
	if _, created, err := Enable(dir, []string{"watchtrail.local"}, now); err != nil || created {
		t.Fatalf("second Enable: created=%v err=%v (want created=false)", created, err)
	}
}

func TestEnsureLeafFreshRenews(t *testing.T) {
	dir := t.TempDir()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, _, err := Enable(dir, []string{"watchtrail.local"}, t0); err != nil {
		t.Fatal(err)
	}
	before := parseLeaf(t, dir).NotAfter

	// 370 days later, leaf (390d life) is within the 30d renewal window.
	if err := EnsureLeafFresh(dir, []string{"watchtrail.local"}, t0.AddDate(0, 0, 370)); err != nil {
		t.Fatal(err)
	}
	after := parseLeaf(t, dir).NotAfter
	if !after.After(before) {
		t.Fatalf("expected renewal to extend NotAfter; before=%v after=%v", before, after)
	}
}

func TestEnsureLeafFreshNoopWhenValid(t *testing.T) {
	dir := t.TempDir()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, _, err := Enable(dir, []string{"watchtrail.local"}, t0); err != nil {
		t.Fatal(err)
	}
	before := parseLeaf(t, dir).NotAfter
	if err := EnsureLeafFresh(dir, []string{"watchtrail.local"}, t0.AddDate(0, 0, 10)); err != nil {
		t.Fatal(err)
	}
	if got := parseLeaf(t, dir).NotAfter; !got.Equal(before) {
		t.Fatalf("leaf was renewed unexpectedly: before=%v after=%v", before, got)
	}
}

func TestDisableRemovesLeafKeepsCA(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	if _, _, err := Enable(dir, []string{"watchtrail.local"}, now); err != nil {
		t.Fatal(err)
	}
	if err := Disable(dir); err != nil {
		t.Fatal(err)
	}
	if Enabled(dir) {
		t.Fatal("Enabled should be false after Disable")
	}
	if _, err := os.Stat(CACertPath(dir)); err != nil {
		t.Fatalf("CA cert should survive Disable: %v", err)
	}
}

func TestNeedsRenewal(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !needsRenewal(now.AddDate(0, 0, 10), now) {
		t.Fatal("10d to expiry should renew")
	}
	if needsRenewal(now.AddDate(0, 0, 60), now) {
		t.Fatal("60d to expiry should not renew")
	}
}

func TestLANHostsIncludesBaseNames(t *testing.T) {
	got := LANHosts()
	for _, want := range []string{"watchtrail.local", "localhost", "127.0.0.1", "::1"} {
		found := false
		for _, h := range got {
			if h == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("LANHosts missing %q (got %v)", want, got)
		}
	}
}

func parseLeaf(t *testing.T, dir string) *x509.Certificate {
	t.Helper()
	pemBytes, err := os.ReadFile(LeafCertPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		t.Fatal("no PEM block in leaf cert")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}
