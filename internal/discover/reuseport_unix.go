//go:build !windows

package discover

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// reusePortControl sets SO_REUSEADDR and SO_REUSEPORT so a socket can bind
// UDP/5353 alongside the system mDNS responder and still receive the group's
// multicast traffic. Without SO_REUSEPORT on BSD/macOS, the responder that
// already owns the port receives every packet and ours receives none.
func reusePortControl(network, address string, c syscall.RawConn) error {
	var serr error
	if err := c.Control(func(fd uintptr) {
		if e := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); e != nil {
			serr = e
			return
		}
		if e := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); e != nil {
			serr = e
		}
	}); err != nil {
		return err
	}
	return serr
}
