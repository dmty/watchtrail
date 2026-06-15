package main

import (
	"context"
	"net/http"
	"time"

	"watchtrail/internal/config"
	"watchtrail/internal/store"
)

// probeAPI reports whether a read API answers /health quickly. Any transport
// error (refused, DNS, timeout) counts as down; a reachable server — even one
// returning a non-200 — counts as up.
func probeAPI(baseURL string, client *http.Client) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/health", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// newReader picks the read transport: the HTTP API when serve is up, else a
// direct read-only store. The returned closer releases the store when one was
// opened. usedStore reports the fallback for an optional notice.
func newReader(cfgPath string) (rd reader, usedStore bool, closer func(), err error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, false, nil, err
	}
	baseURL := "http://" + cfg.HTTPAddr
	client := &http.Client{Timeout: 5 * time.Second}
	if probeAPI(baseURL, client) {
		return &apiClient{baseURL: baseURL, http: client}, false, func() {}, nil
	}
	repo, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, false, nil, err
	}
	return &storeReader{repo: repo}, true, func() { repo.Close() }, nil
}
