package tlsca

import (
	"strings"
	"testing"
)

func TestInstallPlanDarwin(t *testing.T) {
	p, err := installPlan("darwin", "/data/tls/ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	if p.CopyTo != "" {
		t.Fatalf("darwin install should not copy, got CopyTo=%q", p.CopyTo)
	}
	if p.NeedsSudo {
		t.Fatalf("darwin install should not need sudo (GUI prompt), got NeedsSudo=true")
	}
	got := strings.Join(p.Run, " ")
	want := "security add-trusted-cert -r trustRoot /data/tls/ca.crt"
	if got != want {
		t.Fatalf("Run = %q, want %q", got, want)
	}
}

func TestInstallPlanLinux(t *testing.T) {
	p, err := installPlan("linux", "/data/tls/ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	if p.CopyTo != "/usr/local/share/ca-certificates/watchtrail-ca.crt" {
		t.Fatalf("CopyTo = %q", p.CopyTo)
	}
	if strings.Join(p.Run, " ") != "update-ca-certificates" {
		t.Fatalf("Run = %v", p.Run)
	}
}

func TestInstallPlanUnsupported(t *testing.T) {
	if _, err := installPlan("windows", "/data/tls/ca.crt"); err == nil {
		t.Fatal("expected error for unsupported GOOS")
	}
}

func TestUninstallPlanLinux(t *testing.T) {
	p, err := uninstallPlan("linux", "/data/tls/ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	if p.RemoveFile != "/usr/local/share/ca-certificates/watchtrail-ca.crt" {
		t.Fatalf("RemoveFile = %q", p.RemoveFile)
	}
	if strings.Join(p.Run, " ") != "update-ca-certificates --fresh" {
		t.Fatalf("Run = %v", p.Run)
	}
}

func TestUninstallPlanDarwin(t *testing.T) {
	p, err := uninstallPlan("darwin", "/data/tls/ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	want := "security remove-trusted-cert /data/tls/ca.crt"
	if strings.Join(p.Run, " ") != want {
		t.Fatalf("Run = %q, want %q", strings.Join(p.Run, " "), want)
	}
}
