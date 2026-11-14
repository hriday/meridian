//go:build windows

package dhcpscan

import (
	"context"
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const supported = true

// newTransport opens a UDP :68 socket on Windows. Windows has no privileged-port
// restriction, so this works without administrator rights; SO_REUSEADDR lets us
// share the port with the Windows DHCP Client service, and SO_BROADCAST permits
// the broadcast send.
func newTransport(window time.Duration) (*udpTransport, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var serr error
			if err := c.Control(func(fd uintptr) {
				h := windows.Handle(fd)
				// On Windows SO_REUSEADDR allows binding a port already in use —
				// exactly the sharing we need. There is no SO_REUSEPORT.
				_ = windows.SetsockoptInt(h, windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
				serr = windows.SetsockoptInt(h, windows.SOL_SOCKET, windows.SO_BROADCAST, 1)
			}); err != nil {
				return err
			}
			return serr
		},
	}
	conn, err := lc.ListenPacket(context.Background(), "udp4", ":68")
	if err != nil {
		return nil, err
	}
	return &udpTransport{conn: conn, window: window}, nil
}
