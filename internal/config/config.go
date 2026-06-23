// Package config loads service configuration from a TOML file, applying
// built-in defaults first and environment-variable overrides last.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all tunables for the core service.
type Config struct {
	HTTPAddr               string   `toml:"http_addr"`
	TCPAddr                string   `toml:"tcp_addr"`
	Token                  string   `toml:"token"`
	DBPath                 string   `toml:"db_path"`
	SessionGapSeconds      int      `toml:"session_gap_seconds"`
	CompletionThreshold    float64  `toml:"completion_threshold"`
	ProgressCadenceSeconds int      `toml:"progress_cadence_seconds"`
	ThumbnailSources       []string `toml:"thumbnail_sources"`
	ThumbsDir              string   `toml:"thumbs_dir"`
	DataDir                string   `toml:"data_dir"`
	MDNSEnabled            bool     `toml:"mdns_enabled"`
	MDNSHostname           string   `toml:"mdns_hostname"`
	AuthDisabled           bool     `toml:"auth_disabled"`
}

func defaults() Config {
	return Config{
		HTTPAddr:               "127.0.0.1:8765",
		TCPAddr:                "127.0.0.1:8766",
		Token:                  "",
		DBPath:                 "watchtrail.db",
		SessionGapSeconds:      1800,
		CompletionThreshold:    0.9,
		ProgressCadenceSeconds: 30,
		ThumbnailSources:       []string{"sidecar", "frame"},
		ThumbsDir:              "",
		MDNSEnabled:            true,
		MDNSHostname:           "watchtrail",
	}
}

// Load reads cfgPath over the defaults. A missing file is not an error (pure
// defaults). Environment variables (WATCHTRAIL_*) override file and defaults.
func Load(cfgPath string) (Config, error) {
	cfg := defaults()

	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return Config{}, err
		}
	}

	if v, ok := os.LookupEnv("WATCHTRAIL_HTTP_ADDR"); ok {
		cfg.HTTPAddr = v
	}
	if v, ok := os.LookupEnv("WATCHTRAIL_TCP_ADDR"); ok {
		cfg.TCPAddr = v
	}
	if v, ok := os.LookupEnv("WATCHTRAIL_TOKEN"); ok {
		cfg.Token = v
	}
	if v, ok := os.LookupEnv("WATCHTRAIL_DB_PATH"); ok {
		cfg.DBPath = v
	}
	if v, ok := os.LookupEnv("WATCHTRAIL_THUMBS_DIR"); ok {
		cfg.ThumbsDir = v
	}
	if v, ok := os.LookupEnv("WATCHTRAIL_DATA_DIR"); ok {
		cfg.DataDir = v
	}
	if v, ok := os.LookupEnv("WATCHTRAIL_MDNS_ENABLED"); ok {
		cfg.MDNSEnabled = v == "true" || v == "1"
	}
	if v, ok := os.LookupEnv("WATCHTRAIL_MDNS_HOSTNAME"); ok {
		cfg.MDNSHostname = v
	}
	if v, ok := os.LookupEnv("WATCHTRAIL_AUTH_DISABLED"); ok {
		cfg.AuthDisabled = v == "true" || v == "1"
	}
	// DataDir defaults to dir of DBPath if still unset.
	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Dir(cfg.DBPath)
	}
	return cfg, nil
}
