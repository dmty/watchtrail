package ingest

import (
	"context"
	"net"
	"testing"
	"time"
)

// startTCP starts ServeTCP on a random loopback port and returns its address.
func startTCP(t *testing.T, p *Pipeline, token string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go ServeTCP(ctx, ln, p, token)
	t.Cleanup(func() { _ = ln.Close() })
	return ln.Addr().String()
}

func TestTCPSingleLineEvent(t *testing.T) {
	p, repo := newTestPipeline(t)
	addr := startTCP(t, p, "")

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(compact(rawEvent) + "\n")); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, repo, 1)
}

func TestTCPHandshakeRequired(t *testing.T) {
	p, repo := newTestPipeline(t)
	addr := startTCP(t, p, "tok")

	// Wrong token: connection should yield no stored events.
	bad, _ := net.Dial("tcp", addr)
	_, _ = bad.Write([]byte("nope\n" + compact(rawEvent) + "\n"))
	_ = bad.Close()

	// Correct token: event is stored.
	good, _ := net.Dial("tcp", addr)
	defer good.Close()
	if _, err := good.Write([]byte("tok\n" + compact(rawEvent) + "\n")); err != nil {
		t.Fatal(err)
	}
	waitForCount(t, repo, 1)
}

// compact strips newlines/tabs so the multi-line literal becomes a single line.
func compact(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' && s[i] != '\t' {
			out = append(out, s[i])
		}
	}
	return string(out)
}

func waitForCount(t *testing.T, repo interface {
	CountEvents(context.Context) (int, error)
}, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if n, _ := repo.CountEvents(context.Background()); n == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	n, _ := repo.CountEvents(context.Background())
	t.Fatalf("CountEvents = %d, want %d within timeout", n, want)
}
