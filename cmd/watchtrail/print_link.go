package main

import (
	"flag"
	"fmt"
	"net"

	"watchtrail/internal/auth"
	"watchtrail/internal/config"
	"watchtrail/internal/tlsca"
)

func runPrintLink(args []string) error {
	fs := flag.NewFlagSet("print-link", flag.ExitOnError)
	cfgPath := fs.String("config", "watchtrail.toml", "path to config file")
	_ = fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	key, _, err := auth.LoadOrCreateKey(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	scheme, addr := "http", cfg.HTTPAddr
	if !cfg.AuthDisabled && tlsca.Enabled(cfg.DataDir) {
		scheme, addr = "https", cfg.TLSAddr
	}
	fmt.Println(buildSetupURL(scheme, addr, auth.HexKey(key)))
	return nil
}

// buildSetupURL constructs the magic-link URL a user opens in their browser.
// A wildcard bind host is replaced with the mDNS hostname so the link is
// portable across the user's devices; default ports for the scheme are omitted.
func buildSetupURL(scheme, addr, key string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = addr, ""
	}
	switch host {
	case "", "0.0.0.0", "::":
		host = "watchtrail.local"
	}
	defaultPort := (scheme == "http" && (port == "" || port == "80")) ||
		(scheme == "https" && port == "443")
	if defaultPort {
		return fmt.Sprintf("%s://%s/?setup=%s", scheme, host, key)
	}
	return fmt.Sprintf("%s://%s:%s/?setup=%s", scheme, host, port, key)
}
