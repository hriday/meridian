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

// udpTransport broadcasts the DISCOVER from UDP :68 and collects replies for a
// fixed window. Binding :68 needs root, so the constructor maps a permission
// error to ErrNeedsPrivilege.
type udpTransport struct {
	conn   net.PacketConn
	window time.Duration
}

func newTransport(window time.Duration) (*udpTransport, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var serr error
			if err := c.Control(func(fd uintptr) {
				// share :68 with the OS DHCP client, and allow broadcast sends.
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
		if isPermissionErr(err) {
			return nil, ErrNeedsPrivilege
		}
		return nil, err
	}
	return &udpTransport{conn: conn, window: window}, nil
}

func (u *udpTransport) Discover(disco []byte) ([][]byte, error) {
	dst := &net.UDPAddr{IP: net.IPv4bcast, Port: 67}
	if _, err := u.conn.WriteTo(disco, dst); err != nil {
		return nil, err
	}
	_ = u.conn.SetReadDeadline(time.Now().Add(u.window))
	var offers [][]byte
	buf := make([]byte, 1500)
	for {
		n, _, err := u.conn.ReadFrom(buf)
		if err != nil {
			break // deadline reached: done collecting
		}
		pkt := make([]byte, n)
		copy(pkt, buf[:n])
		offers = append(offers, pkt)
	}
	return offers, nil
}

func (u *udpTransport) Close() error { return u.conn.Close() }

func isPermissionErr(err error) bool {
	return errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)
}
