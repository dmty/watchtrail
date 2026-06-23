// Package discovery advertises the watchtrail HTTP service via mDNS so users
// can reach it at <hostname>.local without configuring DNS.
package discovery

import (
	"context"
	"sync"

	"github.com/grandcat/zeroconf"
)

// Service is a registered mDNS service handle.
type Service struct {
	srv    *zeroconf.Server
	once   sync.Once
	closed chan struct{}
}

// Register publishes hostname._http._tcp.local. on every interface and returns
// a handle. The registration is dropped when ctx is cancelled or Shutdown is
// called, whichever comes first.
func Register(ctx context.Context, hostname string, port int) (*Service, error) {
	srv, err := zeroconf.Register(hostname, "_http._tcp", "local.", port, nil, nil)
	if err != nil {
		return nil, err
	}
	s := &Service{srv: srv, closed: make(chan struct{})}
	go func() {
		<-ctx.Done()
		s.Shutdown()
	}()
	return s, nil
}

// Shutdown drops the registration. Safe to call multiple times.
func (s *Service) Shutdown() {
	s.once.Do(func() {
		s.srv.Shutdown()
		close(s.closed)
	})
}

// IsClosed reports whether Shutdown has completed.
func (s *Service) IsClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}
