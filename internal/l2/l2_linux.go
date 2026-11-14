//go:build linux

package l2

import (
	"errors"
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

const supported = true

// Conn is an AF_PACKET raw socket bound to an interface.
type Conn struct {
	fd      int
	ifindex int
	buf     []byte
}

// Open binds a raw packet socket to iface.
func Open(iface string) (*Conn, error) {
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
	_ = unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &unix.Timeval{Sec: 1})
	return &Conn{fd: fd, ifindex: ifi.Index, buf: make([]byte, 65536)}, nil
}

func (c *Conn) Close() error { return unix.Close(c.fd) }

// Write sends one complete Ethernet frame out the bound interface.
func (c *Conn) Write(frame []byte) error {
	sa := &unix.SockaddrLinklayer{Ifindex: c.ifindex, Halen: 6}
	if len(frame) >= 6 {
		copy(sa.Addr[:6], frame[:6]) // destination MAC
	}
	return unix.Sendto(c.fd, frame, 0, sa)
}

// Read returns one frame, or ErrReadTimeout if the poll interval elapsed.
func (c *Conn) Read() ([][]byte, error) {
	n, _, err := unix.Recvfrom(c.fd, c.buf, 0)
	if err != nil {
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EINTR) {
			return nil, ErrReadTimeout
		}
		return nil, err
	}
	if n <= 0 {
		return nil, ErrReadTimeout
	}
	f := make([]byte, n)
	copy(f, c.buf[:n])
	return [][]byte{f}, nil
}

func htons(v uint16) uint16 { return (v<<8)&0xff00 | v>>8 }
