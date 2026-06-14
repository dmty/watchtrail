package ingest

import (
	"bufio"
	"context"
	"crypto/subtle"
	"net"
	"sync"
)

const maxLineBytes = 1 << 20 // 1 MiB per event line

// ServeTCP accepts connections on ln until ctx is cancelled, then drains
// in-flight connections before returning. Each connection is a stream of
// newline-delimited JSON events fed to the same pipeline as HTTP. If token != "",
// the first line of each connection must equal it (handshake).
//
// ServeTCP returns only after every spawned connection handler has finished, so
// a caller can safely release pipeline-owned resources (e.g. the store) once it
// returns.
func ServeTCP(ctx context.Context, ln net.Listener, p *Pipeline, token string) {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	var wg sync.WaitGroup
	for {
		conn, err := ln.Accept()
		if err != nil {
			break // listener closed
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleConn(ctx, conn, p, token)
		}()
	}
	wg.Wait()
}

func handleConn(ctx context.Context, conn net.Conn, p *Pipeline, token string) {
	defer conn.Close()

	// Unblock a parked Scan promptly when the server is shutting down.
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-connCtx.Done()
		_ = conn.Close()
	}()

	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	if token != "" {
		if !sc.Scan() {
			return
		}
		if subtle.ConstantTimeCompare(sc.Bytes(), []byte(token)) != 1 {
			return // bad handshake: drop the connection silently
		}
	}

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		raw := make([]byte, len(line)) // Scanner reuses its buffer; copy before use
		copy(raw, line)
		// Per-event errors must not kill the stream; a bad line is skipped.
		_ = p.Process(ctx, raw)
	}
}
