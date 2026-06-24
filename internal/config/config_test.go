package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsWhenFileMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:8765" {
		t.Errorf("HTTPAddr = %q, want default", cfg.HTTPAddr)
	}
	if cfg.TCPAddr != "127.0.0.1:8766" {
		t.Errorf("TCPAddr = %q, want default", cfg.TCPAddr)
	}
	if cfg.DBPath != "watchtrail.db" {
		t.Errorf("DBPath = %q, want default", cfg.DBPath)
	}
	if cfg.CompletionThreshold != 0.9 {
		t.Errorf("CompletionThreshold = %v, want 0.9", cfg.CompletionThreshold)
	}
	if cfg.SessionGapSeconds != 1800 {
		t.Errorf("SessionGapSeconds = %d, want 1800", cfg.SessionGapSeconds)
	}
	if cfg.ProgressCadenceSeconds != 30 {
		t.Errorf("ProgressCadenceSeconds = %d, want 30", cfg.ProgressCadenceSeconds)
	}
}

func TestFileOverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	body := `
http_addr = "127.0.0.1:9001"
token = "secret-from-file"
db_path = "/tmp/wt.db"
completion_threshold = 0.8
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9001" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.Token != "secret-from-file" {
		t.Errorf("Token = %q", cfg.Token)
	}
	if cfg.CompletionThreshold != 0.8 {
		t.Errorf("CompletionThreshold = %v", cfg.CompletionThreshold)
	}
	if cfg.TCPAddr != "127.0.0.1:8766" {
		t.Errorf("TCPAddr = %q, want default", cfg.TCPAddr)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	if err := os.WriteFile(path, []byte(`token = "from-file"`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WATCHTRAIL_TOKEN", "from-env")
	t.Setenv("WATCHTRAIL_DB_PATH", "/tmp/env.db")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Token != "from-env" {
		t.Errorf("Token = %q, want env override", cfg.Token)
	}
	if cfg.DBPath != "/tmp/env.db" {
		t.Errorf("DBPath = %q, want env override", cfg.DBPath)
	}
}

func TestThumbnailDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.ThumbnailSources) != 2 || cfg.ThumbnailSources[0] != "sidecar" || cfg.ThumbnailSources[1] != "frame" {
		t.Errorf("ThumbnailSources = %v, want [sidecar frame]", cfg.ThumbnailSources)
	}
	if cfg.ThumbsDir != "" {
		t.Errorf("ThumbsDir = %q, want empty", cfg.ThumbsDir)
	}
}

func TestThumbnailOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	body := "thumbnail_sources = [\"frame\"]\nthumbs_dir = \"/var/thumbs\"\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.ThumbnailSources) != 1 || cfg.ThumbnailSources[0] != "frame" {
		t.Errorf("ThumbnailSources = %v", cfg.ThumbnailSources)
	}
	if cfg.ThumbsDir != "/var/thumbs" {
		t.Errorf("ThumbsDir = %q", cfg.ThumbsDir)
	}
}

func TestNewFieldDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MDNSEnabled != true {
		t.Errorf("MDNSEnabled = %v, want true", cfg.MDNSEnabled)
	}
	if cfg.MDNSHostname != "watchtrail" {
		t.Errorf("MDNSHostname = %q, want %q", cfg.MDNSHostname, "watchtrail")
	}
	if cfg.AuthDisabled != false {
		t.Errorf("AuthDisabled = %v, want false", cfg.AuthDisabled)
	}
	// DataDir defaults to dir of DBPath. Default DBPath is "watchtrail.db" (cwd).
	if cfg.DataDir != "." {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, ".")
	}
}

func TestDataDirFollowsDBPathByDefault(t *testing.T) {
	t.Setenv("WATCHTRAIL_DB_PATH", "/srv/wt/wt.db")
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != "/srv/wt" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/srv/wt")
	}
}

func TestDataDirExplicitOverride(t *testing.T) {
	t.Setenv("WATCHTRAIL_DB_PATH", "/srv/wt/wt.db")
	t.Setenv("WATCHTRAIL_DATA_DIR", "/var/lib/watchtrail")
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != "/var/lib/watchtrail" {
		t.Errorf("DataDir = %q, want explicit override", cfg.DataDir)
	}
}

func TestMDNSToggleViaEnv(t *testing.T) {
	t.Setenv("WATCHTRAIL_MDNS_ENABLED", "false")
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MDNSEnabled {
		t.Errorf("MDNSEnabled = true, want false")
	}
}

func TestAuthDisabledViaEnv(t *testing.T) {
	t.Setenv("WATCHTRAIL_AUTH_DISABLED", "true")
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.AuthDisabled {
		t.Errorf("AuthDisabled = false, want true")
	}
}

func TestDefaultTLSAddr(t *testing.T) {
	cfg, err := Load("does-not-exist.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TLSAddr != ":8443" {
		t.Fatalf("TLSAddr = %q, want :8443", cfg.TLSAddr)
	}
}

func TestTLSAddrEnvOverride(t *testing.T) {
	t.Setenv("WATCHTRAIL_TLS_ADDR", "0.0.0.0:9443")
	cfg, err := Load("does-not-exist.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TLSAddr != "0.0.0.0:9443" {
		t.Fatalf("TLSAddr = %q, want 0.0.0.0:9443", cfg.TLSAddr)
	}
}
