package netwatch

import (
	"context"

	"golang.org/x/sys/windows"
)

var (
	iphlpapi          = windows.NewLazySystemDLL("iphlpapi.dll")
	notifyAddrChange  = iphlpapi.NewProc("NotifyAddrChange")
	notifyRouteChange = iphlpapi.NewProc("NotifyRouteChange")
)

// watch uses the blocking forms of NotifyAddrChange / NotifyRouteChange
// (NULL handle and overlapped): each call returns once something changes.
// They can't be cancelled, but they live as long as the process anyway.
func watch(_ context.Context, notify func()) {
	for _, p := range []*windows.LazyProc{notifyAddrChange, notifyRouteChange} {
		if p.Find() != nil {
			continue
		}
		go func(p *windows.LazyProc) {
			for {
				if r, _, _ := p.Call(0, 0); r != 0 {
					return // NO_ERROR is 0; anything else means the API isn't usable here
				}
				notify()
			}
		}(p)
	}
}
