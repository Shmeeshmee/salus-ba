package notification

import (
	"net/netip"
	"testing"
)

// Every range that does not reach the public internet, or carries an IPv4 address inside IPv6 that could reach a
// private one (NAT64, 6to4, Teredo, IPv4-compatible), is refused; public addresses of both families are not.
func TestPublicAddr_RefusesEveryNonPublicRange(t *testing.T) {
	for addr, want := range map[string]bool{
		"8.8.8.8":                  true,
		"2a00:1450:4001:80b::200e": true,
		"2606:4700:4700::1111":     true,
		"127.0.0.1":                false,
		"10.0.0.5":                 false,
		"100.64.0.1":               false,
		"169.254.169.254":          false,
		"240.0.0.1":                false, // reserved (class E)
		"255.255.255.255":          false, // limited broadcast
		"::ffff:127.0.0.1":         false, // IPv4-mapped
		"::127.0.0.1":              false, // IPv4-compatible (::/96)
		"64:ff9b::7f00:1":          false, // NAT64, well-known prefix
		"64:ff9b:1::a00:1":         false, // NAT64, local-use prefix
		"2002:7f00:1::1":           false, // 6to4
		"2001::1":                  false, // Teredo
		"2001:db8::1":              false, // documentation
		"100::1":                   false, // discard-only
		"fd00::1":                  false, // unique-local
		"fe80::1":                  false, // link-local
	} {
		if got := publicAddr(netip.MustParseAddr(addr)); got != want {
			t.Errorf("publicAddr(%s) = %v, want %v", addr, got, want)
		}
	}
}
