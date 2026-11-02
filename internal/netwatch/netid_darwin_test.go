package netwatch

import "testing"

func TestParseNetstat(t *testing.T) {
	out := `Routing tables

Internet:
Destination        Gateway            Flags               Netif Expire
default            link#26            UCSIg               utun6
default            192.168.0.1        UGScIg                en0
127                127.0.0.1          UCS                   lo0
`
	if gw := parseNetstatGateway(out); gw != "192.168.0.1" {
		t.Fatalf("gw %q", gw)
	}
	if mac := parseArpMAC("? (192.168.0.1) at 50:ff:20:1:2:3 on en0 ifscope [ethernet]"); mac != "50:ff:20:1:2:3" {
		t.Fatalf("mac %q", mac)
	}
}
