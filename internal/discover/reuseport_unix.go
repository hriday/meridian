//go:build !windows

package discover

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// mdnsListenAddr is the address to bind the mDNS receive socket to. On BSD/
// macOS and Linux, binding the multicast group address itself is required so
// the kernel delivers the group's traffic to us alongside the system responder
// (which binds the same group address); a 0.0.0.0 bind loses the delivery race.
const mdnsListenAddr = "224.0.0.251:5353"

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
