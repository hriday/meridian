//go:build windows

package discover

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// mdnsListenAddr is the bind address for the mDNS receive socket. Windows does
// not allow binding a multicast group address, so we bind the wildcard and rely
// on JoinGroup for delivery.
const mdnsListenAddr = "0.0.0.0:5353"

// reusePortControl sets SO_REUSEADDR on Windows, where it provides the
// shared-port semantics needed to bind UDP/5353 alongside other listeners.
func reusePortControl(network, address string, c syscall.RawConn) error {
	var serr error
	if err := c.Control(func(fd uintptr) {
		if e := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1); e != nil {
			serr = e
		}
	}); err != nil {
		return err
	}
	return serr
}
