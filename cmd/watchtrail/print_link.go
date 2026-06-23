package main

import (
	"flag"
	"fmt"
	"net"

	"watchtrail/internal/auth"
	"watchtrail/internal/config"
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
	fmt.Println(buildSetupURL(cfg.HTTPAddr, auth.HexKey(key)))
	return nil
}

// buildSetupURL constructs the magic-link URL a user opens in their browser.
// If addr binds all interfaces, the host is replaced with the mDNS hostname so
// the link is portable across the user's devices.
func buildSetupURL(addr, key string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = addr, ""
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "watchtrail.local"
	}
	if port == "" || port == "80" {
		return fmt.Sprintf("http://%s/?setup=%s", host, key)
	}
	return fmt.Sprintf("http://%s:%s/?setup=%s", host, port, key)
}
