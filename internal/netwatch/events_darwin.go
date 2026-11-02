package netwatch

import (
	"context"

	"golang.org/x/sys/unix"
)

// watch reads the routing socket: the kernel writes a message to it on every
// route, address and interface change.
func watch(ctx context.Context, notify func()) {
	fd, err := unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, unix.AF_UNSPEC)
	if err != nil {
		return
	}
	go func() {
		<-ctx.Done()
		unix.Close(fd)
	}()
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := unix.Read(fd, buf); err != nil {
				if err == unix.EINTR {
					continue
				}
				return
			}
			notify()
		}
	}()
}
