package netwatch

import "testing"

func TestLinuxGatewayMAC(t *testing.T) {
	route := "Iface\tDestination\tGateway \tFlags\nwg0\t00000000\t0100080A\t0003\neth0\t00000000\t0100A8C0\t0003\neth0\t0000A8C0\t00000000\t0001\n"
	arp := "IP address       HW type     Flags       HW address            Mask     Device\n192.168.0.1      0x1         0x2         50:ff:20:01:02:03     *        eth0\n"
	if got := linuxGatewayMAC(route, arp); got != "50:ff:20:01:02:03" {
		t.Fatalf("%q", got)
	}
}
