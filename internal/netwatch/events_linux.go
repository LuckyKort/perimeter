package netwatch

import (
	"context"

	"golang.org/x/sys/unix"
)

// watch subscribes to rtnetlink: links, IPv4/IPv6 addresses and routes.
func watch(ctx context.Context, notify func()) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW, unix.NETLINK_ROUTE)
	if err != nil {
		return
	}
	groups := uint32(unix.RTMGRP_LINK | unix.RTMGRP_IPV4_IFADDR | unix.RTMGRP_IPV6_IFADDR |
		unix.RTMGRP_IPV4_ROUTE | unix.RTMGRP_IPV6_ROUTE)
	if err := unix.Bind(fd, &unix.SockaddrNetlink{Family: unix.AF_NETLINK, Groups: groups}); err != nil {
		unix.Close(fd)
		return
	}
	go func() {
		<-ctx.Done()
		unix.Close(fd)
	}()
	go func() {
		buf := make([]byte, 8192)
		for {
			if _, _, err := unix.Recvfrom(fd, buf, 0); err != nil {
				if err == unix.EINTR || err == unix.ENOBUFS {
					continue
				}
				return
			}
			notify()
		}
	}()
}
