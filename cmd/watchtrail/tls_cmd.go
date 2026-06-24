package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"watchtrail/internal/auth"
	"watchtrail/internal/config"
	"watchtrail/internal/tlsca"
)

// installTrust is a seam so tests can stub the privileged trust-store step.
var installTrust = tlsca.Install

func loadCfg(name string, args []string) (config.Config, error) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	cfgPath := fs.String("config", "watchtrail.toml", "path to config file")
	_ = fs.Parse(args)
	return config.Load(*cfgPath)
}

func runEnableTLS(args []string) error {
	cfg, err := loadCfg("enable-tls", args)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if cfg.AuthDisabled {
		return fmt.Errorf("enable-tls requires auth; set auth_disabled=false first")
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("data dir: %w", err)
	}

	caPath, created, err := tlsca.Enable(cfg.DataDir, tlsca.LANHosts(), time.Now())
	if err != nil {
		return fmt.Errorf("generate certificates: %w", err)
	}
	if created {
		fmt.Printf("generated local CA at %s\n", caPath)
		fmt.Println("installing CA into the host trust store (you may be prompted to authenticate)...")
		if err := installTrust(caPath); err != nil {
			return fmt.Errorf("install trust: %w", err)
		}
	} else {
		fmt.Printf("reused existing CA at %s (re-minted leaf)\n", caPath)
	}

	key, _, err := auth.LoadOrCreateKey(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	fmt.Println("TLS enabled. Restart the daemon, then open:")
	fmt.Println("  " + buildSetupURL("https", cfg.TLSAddr, auth.HexKey(key)))
	fmt.Println("Other devices: trust the CA from http://" + mdnsHost(cfg.HTTPAddr) + "/ca.crt first.")
	return nil
}

func runDisableTLS(args []string) error {
	cfg, err := loadCfg("disable-tls", args)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := tlsca.Disable(cfg.DataDir); err != nil {
		return fmt.Errorf("disable tls: %w", err)
	}
	fmt.Println("TLS disabled. Restart the daemon to serve plain HTTP again.")
	if cmd, err := tlsca.UninstallCommand(tlsca.CACertPath(cfg.DataDir)); err == nil {
		fmt.Println("The CA is left in your trust store. To remove it, run:")
		fmt.Println("  " + cmd)
	}
	return nil
}

// mdnsHost returns the host portion of addr, substituting the mDNS name for
// wildcard binds.
func mdnsHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return "watchtrail.local"
	}
	return host
}
