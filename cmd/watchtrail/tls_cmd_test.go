package main

import (
	"os"
	"path/filepath"
	"testing"

	"watchtrail/internal/tlsca"
)

func writeConfig(t *testing.T, dir string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, "watchtrail.toml")
	body := "data_dir = \"" + dir + "\"\ndb_path = \"" + filepath.Join(dir, "watchtrail.db") + "\"\n"
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

func TestEnableTLSGeneratesMaterialsAndDisableRemovesLeaf(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	// installTrust is a package var so the test can stub the privileged step.
	orig := installTrust
	installTrust = func(caPath string) error { return nil }
	defer func() { installTrust = orig }()

	if err := runEnableTLS([]string{"-config", cfgPath}); err != nil {
		t.Fatalf("enable-tls: %v", err)
	}
	if !tlsca.Enabled(dir) {
		t.Fatal("expected TLS enabled after enable-tls")
	}

	if err := runDisableTLS([]string{"-config", cfgPath}); err != nil {
		t.Fatalf("disable-tls: %v", err)
	}
	if tlsca.Enabled(dir) {
		t.Fatal("expected TLS disabled after disable-tls")
	}
	if _, err := os.Stat(tlsca.CACertPath(dir)); err != nil {
		t.Fatalf("CA should survive disable-tls: %v", err)
	}
}

func TestEnableTLSInstallsTrustOnReuse(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir)

	calls := 0
	orig := installTrust
	installTrust = func(caPath string) error { calls++; return nil }
	defer func() { installTrust = orig }()

	// First run creates the CA and installs trust.
	if err := runEnableTLS([]string{"-config", cfgPath}); err != nil {
		t.Fatalf("first enable-tls: %v", err)
	}
	// Second run reuses the existing CA — trust install must STILL run, so a
	// CA that was never successfully trusted gets installed on re-run.
	if err := runEnableTLS([]string{"-config", cfgPath}); err != nil {
		t.Fatalf("second enable-tls: %v", err)
	}
	if calls != 2 {
		t.Fatalf("installTrust called %d times, want 2 (must install on reuse too)", calls)
	}
}

func TestEnableTLSRefusesWhenAuthDisabled(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "watchtrail.toml")
	body := "data_dir = \"" + dir + "\"\nauth_disabled = true\n"
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runEnableTLS([]string{"-config", cfgPath}); err == nil {
		t.Fatal("enable-tls must refuse when auth_disabled=true")
	}
}
