package httpx_test

import (
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

func TestClientIP_ReadsXForwardedForOnlyFromTrustedProxies(t *testing.T) {
	proxies := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	for _, tc := range []struct {
		name, remote, forwarded, want string
	}{
		{"direct", "203.0.113.7:4000", "", "203.0.113.7"},
		{"a client's own header is ignored", "203.0.113.7:4000", "198.51.100.1", "203.0.113.7"},
		{"through the proxy", "10.0.0.2:4000", "198.51.100.1", "198.51.100.1"},
		{"the first untrusted hop from the right", "10.0.0.2:4000", "192.0.2.9, 198.51.100.1, 10.0.0.3", "198.51.100.1"},
		{"a garbled hop stops the walk", "10.0.0.2:4000", "198.51.100.1, garbage", "10.0.0.2"},
		{"IPv4 in IPv6", "[::ffff:203.0.113.7]:4000", "", "203.0.113.7"},
		{"no port", "203.0.113.7", "", "203.0.113.7"},
	} {
		r := httptest.NewRequestWithContext(t.Context(), "GET", "/", nil)
		r.RemoteAddr = tc.remote
		if tc.forwarded != "" {
			r.Header.Set("X-Forwarded-For", tc.forwarded)
		}
		if got := httpx.ClientIP(r, proxies); got != tc.want {
			t.Errorf("%s: ClientIP() = %q, want %q", tc.name, got, tc.want)
		}
	}
}
