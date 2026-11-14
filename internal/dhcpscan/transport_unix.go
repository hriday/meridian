//go:build darwin || linux

package dhcpscan

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const supported = true

// newTransport opens a UDP :68 socket sharing the port with the OS DHCP client
// and enabling broadcast sends. Binding :68 needs root, so a permission error
// maps to ErrNeedsPrivilege.
func newTransport(window time.Duration) (*udpTransport, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var serr error
			if err := c.Control(func(fd uintptr) {
				_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
				_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
				serr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_BROADCAST, 1)
			}); err != nil {
				return err
			}
			return serr
		},
	}
	conn, err := lc.ListenPacket(context.Background(), "udp4", ":68")
	if err != nil {
		if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
			return nil, ErrNeedsPrivilege
		}
		return nil, err
	}
	return &udpTransport{conn: conn, window: window}, nil
}
