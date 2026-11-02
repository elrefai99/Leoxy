package common

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func trustedNet(t *testing.T) []*net.IPNet {
	t.Helper()
	networks, err := ParseCIDRs([]string{"192.0.2.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	return networks
}

func TestGetClientIPPrefersForwardedForOverRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5")
	req.Header.Set("X-Real-IP", "10.0.0.1")
	if got := GetClientIP(req, trustedNet(t)); got != "203.0.113.5" {
		t.Fatalf("ip = %s, want 203.0.113.5", got)
	}
}

func TestGetClientIPStopsAtMalformedEntry(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, garbage")
	if got := GetClientIP(req, trustedNet(t)); got != "192.0.2.10" {
		t.Fatalf("ip = %s, want peer address", got)
	}
}

func TestGetClientIPFallsBackToRealIPWithoutForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Real-IP", "203.0.113.9")
	if got := GetClientIP(req, trustedNet(t)); got != "203.0.113.9" {
		t.Fatalf("ip = %s, want 203.0.113.9", got)
	}
}

func TestGetClientIPIgnoresHeadersFromUntrustedPeer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.7:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	if got := GetClientIP(req, trustedNet(t)); got != "198.51.100.7" {
		t.Fatalf("ip = %s, want peer address", got)
	}
}

func TestGetClientIP(t *testing.T) {
	trusted, err := ParseCIDRs([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		remote  string
		headers map[string]string
		trusted []*net.IPNet
		want    string
	}{
		{"untrusted peer ignores forwarded header", "192.0.2.10:1234", map[string]string{"X-Forwarded-For": "203.0.113.7"}, trusted, "192.0.2.10"},
		{"no trusted proxies ignores forwarded header", "192.0.2.10:1234", map[string]string{"X-Forwarded-For": "203.0.113.7"}, nil, "192.0.2.10"},
		{"trusted peer uses forwarded client", "10.0.0.5:1234", map[string]string{"X-Forwarded-For": "203.0.113.7"}, trusted, "203.0.113.7"},
		{"trusted hops are skipped from the right", "10.0.0.5:1234", map[string]string{"X-Forwarded-For": "203.0.113.7, 10.0.0.9"}, trusted, "203.0.113.7"},
		{"spoofed leftmost entry is ignored", "10.0.0.5:1234", map[string]string{"X-Forwarded-For": "1.1.1.1, 203.0.113.7, 10.0.0.9"}, trusted, "203.0.113.7"},
		{"X-Real-IP is used when no forwarded header", "10.0.0.5:1234", map[string]string{"X-Real-IP": "198.51.100.9"}, trusted, "198.51.100.9"},
		{"CF-Connecting-IP fallback", "10.0.0.5:1234", map[string]string{"CF-Connecting-IP": "198.51.100.4"}, trusted, "198.51.100.4"},
		{"all hops trusted falls back to peer", "10.0.0.5:1234", map[string]string{"X-Forwarded-For": "10.0.0.8"}, trusted, "10.0.0.5"},
		{"remote address without port", "192.0.2.10", nil, trusted, "192.0.2.10"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &http.Request{RemoteAddr: test.remote, Header: http.Header{}}
			for key, value := range test.headers {
				req.Header.Set(key, value)
			}
			if got := GetClientIP(req, test.trusted); got != test.want {
				t.Fatalf("ip = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseCIDRs(t *testing.T) {
	networks, err := ParseCIDRs([]string{" 10.0.0.0/8 ", "192.0.2.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	if len(networks) != 2 {
		t.Fatalf("networks = %d, want 2", len(networks))
	}
	if _, err := ParseCIDRs([]string{"not-a-cidr"}); err == nil {
		t.Fatal("expected an error for an invalid CIDR")
	}
}
