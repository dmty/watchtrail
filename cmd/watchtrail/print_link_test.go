package main

import "testing"

func TestBuildSetupURL(t *testing.T) {
	cases := []struct {
		name, addr, key, want string
	}{
		{"loopback default", "127.0.0.1:8765", "abc", "http://127.0.0.1:8765/?setup=abc"},
		{"all interfaces uses mdns", "0.0.0.0:8765", "abc", "http://watchtrail.local:8765/?setup=abc"},
		{"empty host uses mdns", ":8765", "abc", "http://watchtrail.local:8765/?setup=abc"},
		{"ipv6 all uses mdns", "[::]:8765", "abc", "http://watchtrail.local:8765/?setup=abc"},
		{"port 80 elided", "0.0.0.0:80", "abc", "http://watchtrail.local/?setup=abc"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildSetupURL(c.addr, c.key)
			if got != c.want {
				t.Errorf("buildSetupURL(%q, %q) = %q, want %q", c.addr, c.key, got, c.want)
			}
		})
	}
}
