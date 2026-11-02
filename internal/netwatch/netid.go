package netwatch

import (
	"context"
	"net"
	"net/netip"
	"strings"
)

// A local network is recognised by its router's MAC address: it needs no
// permissions (unlike the Wi-Fi name on recent macOS) and is the same at home
// whether you're on Wi-Fi or a cable.

// normMAC accepts "50:ff:20:1:2:3" (macOS arp drops leading zeros) and "50-FF-20-01-02-03".
func normMAC(s string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(s)), func(r rune) bool { return r == ':' || r == '-' })
	if len(parts) != 6 {
		return ""
	}
	zero := true
	for i, p := range parts {
		if len(p) == 1 {
			p = "0" + p
		}
		if len(p) != 2 || strings.Trim(p, "0123456789abcdef") != "" {
			return ""
		}
		if p != "00" {
			zero = false
		}
		parts[i] = p
	}
	if zero {
		return "" // 00:00:00:00:00:00 means "incomplete"
	}
	return strings.Join(parts, ":")
}

// NetworkID returns the router MAC of the physical network, or "" if unknown.
func NetworkID() string { return normMAC(gatewayMAC()) }

// DNSResolver finds the public address of the DNS server that resolves for us:
// Akamai answers whoami.akamai.net with the address the query came from.
func DNSResolver(ctx context.Context) string {
	addrs, err := net.DefaultResolver.LookupHost(ctx, "whoami.akamai.net")
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ip, err := netip.ParseAddr(a); err == nil && ip.Is4() && !ip.IsPrivate() && !ip.IsLoopback() {
			return a
		}
	}
	return ""
}
