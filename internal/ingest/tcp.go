package ingest

import (
	"bufio"
	"context"
	"crypto/subtle"
	"net"
)

const maxLineBytes = 1 << 20 // 1 MiB per event line

// ServeTCP accepts connections on ln until ctx is cancelled. Each connection is
// a stream of newline-delimited JSON events fed to the same pipeline as HTTP.
// If token != "", the first line of each connection must equal it (handshake).
func ServeTCP(ctx context.Context, ln net.Listener, p *Pipeline, token string) {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed
		}
		go handleConn(ctx, conn, p, token)
	}
}

func handleConn(ctx context.Context, conn net.Conn, p *Pipeline, token string) {
	defer conn.Close()
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
		raw := make([]byte, len(line)) // Scanner reuses its buffer; copy before async use
		copy(raw, line)
		// Per-event errors must not kill the stream; a bad line is skipped.
		_ = p.Process(ctx, raw)
	}
}
