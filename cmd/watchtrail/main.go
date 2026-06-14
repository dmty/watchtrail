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
	"syscall"
	"time"

	"watchtrail/internal/config"
	"watchtrail/internal/ingest"
	"watchtrail/internal/store"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		fmt.Fprintln(os.Stderr, "usage: watchtrail serve [-config path]")
		os.Exit(2)
	}

	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfgPath := fs.String("config", "watchtrail.toml", "path to config file")
	_ = fs.Parse(os.Args[2:])

	if err := runServe(*cfgPath); err != nil {
		log.Fatalf("watchtrail: %v", err)
	}
}

func runServe(cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	repo, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer repo.Close()

	pipeline := ingest.NewPipeline(repo, time.Now)

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

	httpErr := make(chan error, 1)
	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: pipeline.HTTPHandler(cfg.Token)}
	go func() {
		log.Printf("HTTP ingest on http://%s/ingest", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErr <- err
		}
	}()

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
