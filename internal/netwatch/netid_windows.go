package netwatch

import (
	"encoding/binary"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var sendARP = iphlpapi.NewProc("SendARP")

const (
	gaaIncludeGateways = 0x0080
	ifTypeTunnel       = 131
	ifTypePPP          = 23
	ifOperUp           = 1
)

// gatewayMAC takes the IPv4 gateway of the first physical adapter that is up
// and asks for its MAC with SendARP (answered from the ARP cache, so it's fast).
func gatewayMAC() string {
	size := uint32(15000)
	var buf []byte
	for i := 0; i < 3; i++ {
		buf = make([]byte, size)
		err := windows.GetAdaptersAddresses(windows.AF_INET, gaaIncludeGateways, 0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])), &size)
		if err == nil {
			break
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			return ""
		}
	}
	for a := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])); a != nil; a = a.Next {
		if a.OperStatus != ifOperUp || a.IfType == ifTypeTunnel || a.IfType == ifTypePPP || a.FirstGatewayAddress == nil {
			continue
		}
		sa, err := a.FirstGatewayAddress.Address.Sockaddr.Sockaddr()
		if err != nil {
			continue
		}
		in4, ok := sa.(*syscall.SockaddrInet4)
		if !ok {
			continue
		}
		mac := make([]byte, 8)
		n := uint32(len(mac))
		dst := binary.LittleEndian.Uint32(in4.Addr[:])
		if r, _, _ := sendARP.Call(uintptr(dst), 0, uintptr(unsafe.Pointer(&mac[0])), uintptr(unsafe.Pointer(&n))); r != 0 || n < 6 {
			continue
		}
		return net.HardwareAddr(mac[:6]).String()
	}
	return ""
}
