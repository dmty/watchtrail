package main

import "testing"

func TestBuildSetupURL(t *testing.T) {
	cases := []struct {
		name           string
		scheme, addr, key string
		want           string
	}{
		{"http loopback", "http", "127.0.0.1:8765", "abc", "http://127.0.0.1:8765/?setup=abc"},
		{"http wildcard maps to mdns", "http", "0.0.0.0:8765", "abc", "http://watchtrail.local:8765/?setup=abc"},
		{"http default port omitted", "http", "watchtrail.local:80", "abc", "http://watchtrail.local/?setup=abc"},
		{"https wildcard maps to mdns", "https", ":8443", "abc", "https://watchtrail.local:8443/?setup=abc"},
		{"https default port omitted", "https", "watchtrail.local:443", "abc", "https://watchtrail.local/?setup=abc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildSetupURL(tc.scheme, tc.addr, tc.key); got != tc.want {
				t.Fatalf("buildSetupURL(%q,%q,%q) = %q, want %q", tc.scheme, tc.addr, tc.key, got, tc.want)
			}
		})
	}
}
