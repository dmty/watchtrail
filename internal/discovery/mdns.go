// Package discovery advertises the watchtrail HTTP service via mDNS so users
// can reach it at <hostname>.local without configuring DNS. We use
// RegisterProxy (not Register) so that <hostname>.local resolves directly via
// mDNS A/AAAA records, instead of the host machine's existing hostname.
package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/grandcat/zeroconf"
)

// Service is a registered mDNS service handle.
type Service struct {
	srv    *zeroconf.Server
	once   sync.Once
	closed chan struct{}
}

// Register publishes both a service record (hostname._http._tcp.local.) AND a
// host record (hostname.local.) pointing at the local non-loopback IPs.
// Returns a handle whose registration is dropped when ctx is cancelled or
// Shutdown is called, whichever comes first.
func Register(ctx context.Context, hostname string, port int) (*Service, error) {
	ips, err := nonLoopbackIPs()
	if err != nil {
		return nil, fmt.Errorf("enumerate interfaces: %w", err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no non-loopback ips available")
	}
	srv, err := zeroconf.RegisterProxy(hostname, "_http._tcp", "local.", port, hostname, ips, nil, nil)
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

// nonLoopbackIPs returns IPv4 + IPv6 unicast addresses on up, non-loopback
// interfaces, as strings (the form zeroconf.RegisterProxy wants).
func nonLoopbackIPs() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || !ipnet.IP.IsGlobalUnicast() {
				continue
			}
			out = append(out, ipnet.IP.String())
		}
	}
	return out, nil
}
