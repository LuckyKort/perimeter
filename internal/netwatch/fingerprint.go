// Package netwatch notices network changes and finds out the public IP and its country.
package netwatch

import (
	"net"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// Virtual adapters of VMs and containers come and go on their own;
// they must not count as "the network changed".
var skipIface = map[string]*regexp.Regexp{
	"darwin":  regexp.MustCompile(`^(lo|bridge|awdl|llw|anpi|ap|gif|stf|vmenet|vmnet)`),
	"windows": regexp.MustCompile(`(?i)loopback|vethernet|vmware|virtualbox|hyper-v|docker|wsl|bluetooth|npcap|vbox`),
	"linux":   regexp.MustCompile(`^(lo|docker|br-|veth|virbr|vmnet|vboxnet|lxc|lxd|cni|flannel|podman)`),
}

type ifaceAddr struct{ name, ip string }

// Fingerprint describes the local network: every real interface that is up, with its IPv4.
// Turning a VPN on or off, or switching Wi-Fi, changes it.
func Fingerprint() string {
	ifs, err := net.Interfaces()
	if err != nil {
		return "error"
	}
	var addrs []ifaceAddr
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		as, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range as {
			if ipn, ok := a.(*net.IPNet); ok {
				addrs = append(addrs, ifaceAddr{ifc.Name, ipn.IP.String()})
			}
		}
	}
	return fingerprintOf(addrs, runtime.GOOS)
}

func fingerprintOf(addrs []ifaceAddr, goos string) string {
	skip := skipIface[goos]
	var parts []string
	for _, a := range addrs {
		ip := net.ParseIP(a.ip)
		if ip == nil || ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue // IPv6 privacy addresses rotate by themselves; only IPv4 is stable
		}
		if skip != nil && skip.MatchString(a.name) {
			continue
		}
		parts = append(parts, a.name+"="+a.ip)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
