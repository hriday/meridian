//go:build linux

package lldp

import (
	"errors"
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

const supported = true

// afCapture reads raw frames from an AF_PACKET socket bound to an interface.
type afCapture struct {
	fd  int
	buf []byte
}

func newCapture(iface string) (capture, error) {
	proto := int(htons(unix.ETH_P_ALL))
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, proto)
	if err != nil {
		if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM) {
			return nil, ErrNeedsPrivilege
		}
		return nil, fmt.Errorf("couldn't open a raw packet socket: %w", err)
	}
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		unix.Close(fd)
		return nil, err
	}
	if err := unix.Bind(fd, &unix.SockaddrLinklayer{Protocol: htons(unix.ETH_P_ALL), Ifindex: ifi.Index}); err != nil {
		unix.Close(fd)
		return nil, err
	}
	// Poll with a 1s read timeout so Listen can check its deadline/context.
	_ = unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &unix.Timeval{Sec: 1})
	return &afCapture{fd: fd, buf: make([]byte, 65536)}, nil
}

func (c *afCapture) Close() error { return unix.Close(c.fd) }

func (c *afCapture) ReadFrames() ([][]byte, error) {
	n, _, err := unix.Recvfrom(c.fd, c.buf, 0)
	if err != nil {
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EINTR) {
			return nil, errReadTimeout
		}
		return nil, err
	}
	if n <= 0 {
		return nil, errReadTimeout
	}
	f := make([]byte, n)
	copy(f, c.buf[:n])
	return [][]byte{f}, nil
}

func htons(v uint16) uint16 { return (v<<8)&0xff00 | v>>8 }
