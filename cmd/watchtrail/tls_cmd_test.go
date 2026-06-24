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
