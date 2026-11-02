package netwatch

import (
	"encoding/binary"
	"encoding/hex"
	"net"
	"os"
	"strings"
)

func gatewayMAC() string {
	route, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	arp, _ := os.ReadFile("/proc/net/arp")
	return linuxGatewayMAC(string(route), string(arp))
}

// linuxGatewayMAC: the default route of a non-tunnel interface, then its MAC from the ARP table.
func linuxGatewayMAC(route, arp string) string {
	gw := ""
	for _, line := range strings.Split(route, "\n")[1:] {
		f := strings.Fields(line)
		if len(f) < 3 || f[1] != "00000000" || f[2] == "00000000" {
			continue
		}
		if strings.HasPrefix(f[0], "tun") || strings.HasPrefix(f[0], "wg") || strings.HasPrefix(f[0], "ppp") {
			continue
		}
		b, err := hex.DecodeString(f[2])
		if err != nil || len(b) != 4 {
			continue
		}
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, binary.LittleEndian.Uint32(b))
		gw = ip.String()
		break
	}
	if gw == "" {
		return ""
	}
	for _, line := range strings.Split(arp, "\n")[1:] {
		f := strings.Fields(line)
		if len(f) >= 4 && f[0] == gw {
			return f[3]
		}
	}
	return ""
}
