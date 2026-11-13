package httpx

import (
	"net/http"
	"net/netip"
	"strings"
)

// ClientIP is the request's client address: the connection's remote address, and only when that is a trusted proxy
// X-Forwarded-For, read from the right, where the first address that is not a trusted proxy is the client. For rate
// limits only: a client's address is personal data, never to be stored or logged.
func ClientIP(r *http.Request, trusted []netip.Prefix) string {
	ap, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	ip := ap.Addr().Unmap()
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0 && isTrusted(ip, trusted); i-- {
		hop, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			break
		}
		ip = hop.Unmap()
	}
	return ip.String()
}

func isTrusted(ip netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
