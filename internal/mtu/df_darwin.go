//go:build darwin

package mtu

import "golang.org/x/sys/unix"

// setDontFragment sets the Don't-Fragment bit on an IPv4 socket. On Darwin/BSD
// this is IP_DONTFRAG, and it works on an unprivileged datagram ICMP socket.
func setDontFragment(fd int) error {
	return unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_DONTFRAG, 1)
}
