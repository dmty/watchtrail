// Command watchtrail is the WatchTrail core service binary.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"watchtrail/internal/api"
	"watchtrail/internal/auth"
	"watchtrail/internal/config"
	"watchtrail/internal/discovery"
	"watchtrail/internal/events"
	"watchtrail/internal/ingest"
	"watchtrail/internal/sessionize"
	"watchtrail/internal/store"
	"watchtrail/internal/thumb"
	"watchtrail/internal/web"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		cfgPath := fs.String("config", "watchtrail.toml", "path to config file")
		_ = fs.Parse(os.Args[2:])
		if err := runServe(*cfgPath); err != nil {
			log.Fatalf("watchtrail: %v", err)
		}
	case "recent":
		if err := runRecent(os.Args[2:]); err != nil {
			log.Fatalf("watchtrail: %v", err)
		}
	case "item":
		if err := runItem(os.Args[2:]); err != nil {
			log.Fatalf("watchtrail: %v", err)
		}
	case "stats":
		if err := runStats(os.Args[2:]); err != nil {
			log.Fatalf("watchtrail: %v", err)
		}
	case "rebuild-sessions":
		if err := runRebuild(os.Args[2:]); err != nil {
			log.Fatalf("watchtrail: %v", err)
		}
	case "print-link":
		if err := runPrintLink(os.Args[2:]); err != nil {
			log.Fatalf("watchtrail: %v", err)
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: watchtrail <serve|recent|item|stats|rebuild-sessions|print-link> [flags]")
	os.Exit(2)
}

func runServe(cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("data dir: %w", err)
	}

	if cfg.AuthDisabled && !isLoopback(cfg.HTTPAddr) {
		return fmt.Errorf("refusing to bind %s with auth_disabled=true; set auth_disabled=false or bind to 127.0.0.1", cfg.HTTPAddr)
	}

	var (
		authKey     []byte
		authCreated bool
	)
	if !cfg.AuthDisabled {
		authKey, authCreated, err = auth.LoadOrCreateKey(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	repo, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer repo.Close()

	sessCfg := sessionize.Config{
		SessionGap:          time.Duration(cfg.SessionGapSeconds) * time.Second,
		CompletionThreshold: cfg.CompletionThreshold,
		ProgressCadence:     time.Duration(cfg.ProgressCadenceSeconds) * time.Second,
	}
	// One broker shared by the ingest publisher and the dashboard SSE stream.
	broker := events.New()
	pipeline := ingest.NewPipeline(repo, sessCfg, time.Now, broker)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TCP listener.
	ln, err := net.Listen("tcp", cfg.TCPAddr)
	if err != nil {
		return fmt.Errorf("tcp listen %s: %w", cfg.TCPAddr, err)
	}
	log.Printf("TCP line listener on %s", cfg.TCPAddr)
	tcpDone := make(chan struct{})
	go func() {
		ingest.ServeTCP(ctx, ln, pipeline, cfg.Token)
		close(tcpDone)
	}()

	thumbsDir := cfg.ThumbsDir
	if thumbsDir == "" {
		thumbsDir = filepath.Join(filepath.Dir(cfg.DBPath), "thumbs")
	}
	thumbs := thumb.Build(thumbsDir, cfg.ThumbnailSources)

	httpErr := make(chan error, 1)
	webHandler, err := web.Handler(repo, broker, thumbs)
	if err != nil {
		return fmt.Errorf("web: %w", err)
	}
	authMW := func(h http.Handler) http.Handler { return h }
	if !cfg.AuthDisabled {
		authMW = auth.Middleware(authKey)
	}
	root := http.NewServeMux()
	root.Handle("/ingest", pipeline.HTTPHandler(cfg.Token))
	root.Handle("/api/v1/", authMW(api.Handler(repo)))
	root.Handle("/", authMW(webHandler))

	if cfg.MDNSEnabled {
		_, port, err := net.SplitHostPort(cfg.HTTPAddr)
		if err == nil {
			portInt, err := strconv.Atoi(port)
			if err == nil {
				_, err := discovery.Register(ctx, cfg.MDNSHostname, portInt)
				if err != nil {
					log.Printf("mdns: %v (continuing without)", err)
				} else {
					log.Printf("mdns: advertising %s._http._tcp.local on port %d", cfg.MDNSHostname, portInt)
				}
			}
		}
	}

	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: root}
	go func() {
		log.Printf("ingest http://%s/ingest · API http://%s/api/v1 · dashboard http://%s/", cfg.HTTPAddr, cfg.HTTPAddr, cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErr <- err
		}
	}()

	if !cfg.AuthDisabled {
		setupURL := buildSetupURL("http", cfg.HTTPAddr, auth.HexKey(authKey))
		if authCreated {
			log.Printf("setup link (first-run, save this): %s", setupURL)
		} else {
			log.Printf("auth.key loaded from %s — get a fresh setup link with `watchtrail print-link`", auth.KeyPath(cfg.DataDir))
		}
	}

	var runErr error
	select {
	case <-ctx.Done():
	case err := <-httpErr:
		runErr = fmt.Errorf("http server: %w", err)
		stop()
	}

	log.Print("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutCtx); err != nil && runErr == nil {
		runErr = err
	}
	<-tcpDone // wait for TCP connections to drain before repo.Close (deferred)
	return runErr
}

func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	switch host {
	case "127.0.0.1", "localhost", "::1", "[::1]":
		return true
	}
	return false
}
