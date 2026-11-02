package netwatch

import (
	"net/netip"
	"os/exec"
	"strings"
)

// gatewayMAC takes the default route of a physical interface (a VPN adds its own
// default route through utun, which says nothing about the local network).
func gatewayMAC() string {
	out, err := exec.Command("/usr/sbin/netstat", "-rn", "-f", "inet").Output()
	if err != nil {
		return ""
	}
	gw := parseNetstatGateway(string(out))
	if gw == "" {
		return ""
	}
	arp, err := exec.Command("/usr/sbin/arp", "-n", gw).Output()
	if err != nil {
		return ""
	}
	return parseArpMAC(string(arp))
}

func parseNetstatGateway(out string) string {
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 || f[0] != "default" {
			continue
		}
		ifname := f[len(f)-1]
		if f[len(f)-1] == "!" || strings.HasPrefix(f[len(f)-2], "en") {
			ifname = f[len(f)-2]
		}
		if !strings.HasPrefix(ifname, "en") {
			continue
		}
		if a, err := netip.ParseAddr(f[1]); err == nil && a.Is4() {
			return f[1]
		}
	}
	return ""
}

// parseArpMAC reads "? (192.168.0.1) at 50:ff:20:1:2:3 on en0 ifscope [ethernet]".
func parseArpMAC(out string) string {
	f := strings.Fields(out)
	for i, w := range f {
		if w == "at" && i+1 < len(f) {
			return f[i+1]
		}
	}
	return ""
}
